package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// OpenAPI 3.x data structures
type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    OpenAPIInfo            `json:"info"`
	Servers []OpenAPIServer        `json:"servers,omitempty"`
	Paths   map[string]OpenAPIPath `json:"paths"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type OpenAPIPath struct {
	Get    *OpenAPIOperation `json:"get,omitempty"`
	Post   *OpenAPIOperation `json:"post,omitempty"`
	Put    *OpenAPIOperation `json:"put,omitempty"`
	Patch  *OpenAPIOperation `json:"patch,omitempty"`
	Delete *OpenAPIOperation `json:"delete,omitempty"`
}

type OpenAPIOperation struct {
	OperationID string                    `json:"operationId,omitempty"`
	Summary     string                    `json:"summary,omitempty"`
	Description string                    `json:"description,omitempty"`
	Parameters  []OpenAPIParameter        `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody       `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
}

type OpenAPIParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, header, path, cookie
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

type OpenAPIRequestBody struct {
	Description string                       `json:"description,omitempty"`
	Content     map[string]OpenAPIMediaType  `json:"content,omitempty"`
	Required    bool                         `json:"required,omitempty"`
}

type OpenAPIMediaType struct {
	Schema  interface{} `json:"schema,omitempty"`
	Example interface{} `json:"example,omitempty"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

// ImportOpenAPIFromURL downloads and imports an OpenAPI spec from a URL
func ImportOpenAPIFromURL(url, collectionName string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenAPI spec: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch OpenAPI spec: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI spec: %v", err)
	}

	return ImportOpenAPIFromBytes(data, collectionName)
}

// ImportOpenAPIFromFile imports an OpenAPI spec from a local file
func ImportOpenAPIFromFile(filePath, collectionName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	return ImportOpenAPIFromBytes(data, collectionName)
}

// ImportOpenAPIFromBytes parses OpenAPI spec from bytes and creates Bruno collection
func ImportOpenAPIFromBytes(data []byte, collectionName string) error {
	var spec OpenAPISpec
	
	// Try JSON first
	err := json.Unmarshal(data, &spec)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI spec as JSON: %v", err)
	}

	// Validate it's OpenAPI 3.x
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return fmt.Errorf("unsupported OpenAPI version: %s (only 3.x is supported)", spec.OpenAPI)
	}

	return convertOpenAPIToBruno(&spec, collectionName)
}

// convertOpenAPIToBruno converts the parsed OpenAPI spec to Bruno collection
func convertOpenAPIToBruno(spec *OpenAPISpec, collectionName string) error {
	// Get collections directory
	collectionsDir, err := getCollectionsDir()
	if err != nil {
		return fmt.Errorf("failed to get collections directory: %v", err)
	}

	// Use collection name or default from spec
	if collectionName == "" {
		collectionName = sanitizeFilename(spec.Info.Title)
		if collectionName == "" {
			collectionName = "openapi-import"
		}
	}

	// Create collection directory
	collectionPath := filepath.Join(collectionsDir, collectionName)
	err = os.MkdirAll(collectionPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create collection directory: %v", err)
	}

	// Get base URL from servers
	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = spec.Servers[0].URL
	}

	// Convert each path/operation to Bruno requests
	requestCount := 0
	for path, pathItem := range spec.Paths {
		operations := map[string]*OpenAPIOperation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"PATCH":  pathItem.Patch,
			"DELETE": pathItem.Delete,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			requestCount++
			
			// Generate Bruno request
			bruContent, err := generateBrunoRequest(method, path, operation, baseURL, requestCount)
			if err != nil {
				fmt.Printf("Warning: failed to generate request for %s %s: %v\n", method, path, err)
				continue
			}

			// Generate filename
			filename := generateBrunoFilename(method, path, operation)
			filePath := filepath.Join(collectionPath, filename)

			// Write .bru file
			err = os.WriteFile(filePath, []byte(bruContent), 0644)
			if err != nil {
				fmt.Printf("Warning: failed to write %s: %v\n", filename, err)
				continue
			}
		}
	}

	fmt.Printf("Successfully imported %d requests to collection '%s'\n", requestCount, collectionName)
	return nil
}

// generateBrunoRequest creates the Bruno .bru file content
func generateBrunoRequest(method, path string, operation *OpenAPIOperation, baseURL string, seq int) (string, error) {
	var content strings.Builder

	// Determine request name
	name := operation.Summary
	if name == "" {
		name = operation.OperationID
	}
	if name == "" {
		name = fmt.Sprintf("%s %s", method, path)
	}

	// Meta block
	content.WriteString("meta {\n")
	content.WriteString(fmt.Sprintf("  name: %s\n", name))
	content.WriteString("  type: http\n")
	content.WriteString(fmt.Sprintf("  seq: %d\n", seq))
	content.WriteString("}\n\n")

	// HTTP method block
	fullURL := constructFullURL(baseURL, path)
	content.WriteString(fmt.Sprintf("%s {\n", strings.ToLower(method)))
	content.WriteString(fmt.Sprintf("  url: %s\n", fullURL))
	content.WriteString("}\n")

	// Query parameters
	queryParams := extractQueryParameters(operation.Parameters)
	if len(queryParams) > 0 {
		content.WriteString("\nquery {\n")
		for _, param := range queryParams {
			example := getParameterExample(param)
			content.WriteString(fmt.Sprintf("  %s: %s\n", param.Name, example))
		}
		content.WriteString("}\n")
	}

	// Headers
	headerParams := extractHeaderParameters(operation.Parameters)
	if len(headerParams) > 0 {
		content.WriteString("\nheaders {\n")
		for _, param := range headerParams {
			example := getParameterExample(param)
			content.WriteString(fmt.Sprintf("  %s: %s\n", param.Name, example))
		}
		content.WriteString("}\n")
	}

	// Request body (for POST, PUT, PATCH)
	if operation.RequestBody != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		bodyContent := generateRequestBody(operation.RequestBody)
		if bodyContent != "" {
			content.WriteString("\nbody:json {\n")
			content.WriteString(bodyContent)
			content.WriteString("}\n")
		}
	}

	return content.String(), nil
}

// Helper functions
func constructFullURL(baseURL, path string) string {
	if baseURL == "" {
		return path
	}
	
	// Remove trailing slash from baseURL and leading slash from path if both exist
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	return baseURL + path
}

func extractQueryParameters(parameters []OpenAPIParameter) []OpenAPIParameter {
	var queryParams []OpenAPIParameter
	for _, param := range parameters {
		if param.In == "query" {
			queryParams = append(queryParams, param)
		}
	}
	sort.Slice(queryParams, func(i, j int) bool {
		return queryParams[i].Name < queryParams[j].Name
	})
	return queryParams
}

func extractHeaderParameters(parameters []OpenAPIParameter) []OpenAPIParameter {
	var headerParams []OpenAPIParameter
	for _, param := range parameters {
		if param.In == "header" {
			headerParams = append(headerParams, param)
		}
	}
	sort.Slice(headerParams, func(i, j int) bool {
		return headerParams[i].Name < headerParams[j].Name
	})
	return headerParams
}

func getParameterExample(param OpenAPIParameter) string {
	if param.Example != nil {
		return fmt.Sprintf("%v", param.Example)
	}
	
	// Generate example based on type/name
	switch strings.ToLower(param.Name) {
	case "id":
		return "1"
	case "limit":
		return "10"
	case "offset":
		return "0"
	case "page":
		return "1"
	default:
		if param.Required {
			return fmt.Sprintf("{{%s}}", param.Name)
		}
		return ""
	}
}

func generateRequestBody(requestBody *OpenAPIRequestBody) string {
	// Look for JSON content first
	if mediaType, exists := requestBody.Content["application/json"]; exists {
		if mediaType.Example != nil {
			// Use the provided example
			exampleBytes, err := json.MarshalIndent(mediaType.Example, "  ", "  ")
			if err == nil {
				return string(exampleBytes) + "\n"
			}
		}
		
		// Generate a simple example
		return "{\n  // Add your JSON payload here\n}\n"
	}
	
	// Fallback for other content types
	return "{\n  // Add your request body here\n}\n"
}

func generateBrunoFilename(method, path string, operation *OpenAPIOperation) string {
	// Use operationId if available
	if operation.OperationID != "" {
		return sanitizeFilename(operation.OperationID) + ".bru"
	}
	
	// Use summary if available
	if operation.Summary != "" {
		return sanitizeFilename(operation.Summary) + ".bru"
	}
	
	// Generate from method and path
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	var name strings.Builder
	name.WriteString(strings.ToLower(method))
	
	for _, part := range pathParts {
		if part != "" && !strings.Contains(part, "{") {
			name.WriteString("-")
			name.WriteString(part)
		}
	}
	
	return sanitizeFilename(name.String()) + ".bru"
}

func sanitizeFilename(filename string) string {
	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	sanitized := reg.ReplaceAllString(filename, "-")
	
	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	sanitized = reg.ReplaceAllString(sanitized, "-")
	
	// Trim hyphens from start and end
	sanitized = strings.Trim(sanitized, "-")
	
	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "request"
	}
	
	return sanitized
}