package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	request "kalo/src/panels/request"
	response "kalo/src/panels/response"
)


type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HTTPClient) ExecuteRequest(bruReq *request.BruRequest) (*response.HTTPResponse, error) {
	if bruReq == nil {
		return nil, fmt.Errorf("request is nil")
	}

	start := time.Now()

	// Substitute environment variables
	processedURL := c.substituteVars(bruReq.HTTP.URL, bruReq.Vars)
	
	// Parse URL and add query parameters
	parsedURL, err := url.Parse(processedURL)
	if err != nil {
		return &response.HTTPResponse{Error: fmt.Sprintf("Invalid URL: %v", err)}, nil
	}

	// Add query parameters
	if len(bruReq.Query) > 0 {
		query := parsedURL.Query()
		for key, value := range bruReq.Query {
			processedValue := c.substituteVars(value, bruReq.Vars)
			query.Add(key, processedValue)
		}
		parsedURL.RawQuery = query.Encode()
	}

	// Prepare request body
	var body io.Reader
	if bruReq.Body.Type != "" && bruReq.Body.Data != "" {
		processedBody := c.substituteVars(bruReq.Body.Data, bruReq.Vars)
		
		switch bruReq.Body.Type {
		case "json":
			// Wrap the body data in proper JSON structure if needed
			if !strings.HasPrefix(strings.TrimSpace(processedBody), "{") {
				processedBody = "{" + processedBody + "}"
			}
			body = bytes.NewBufferString(processedBody)
		case "text", "raw":
			body = bytes.NewBufferString(processedBody)
		default:
			body = bytes.NewBufferString(processedBody)
		}
	}

	// Create HTTP request
	req, err := http.NewRequest(bruReq.HTTP.Method, parsedURL.String(), body)
	if err != nil {
		return &response.HTTPResponse{Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	// Add headers
	for key, value := range bruReq.Headers {
		processedValue := c.substituteVars(value, bruReq.Vars)
		req.Header.Set(key, processedValue)
	}

	// Add authentication
	if bruReq.Auth.Type != "" {
		err := c.addAuth(req, bruReq.Auth, bruReq.Vars)
		if err != nil {
			return &response.HTTPResponse{Error: fmt.Sprintf("Auth error: %v", err)}, nil
		}
	}

	// Execute request
	resp, err := c.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &response.HTTPResponse{
			Error:        fmt.Sprintf("Request failed: %v", err),
			ResponseTime: responseTime,
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &response.HTTPResponse{
			StatusCode:   resp.StatusCode,
			Status:       resp.Status,
			Error:        fmt.Sprintf("Failed to read response body: %v", err),
			ResponseTime: responseTime,
		}, nil
	}

	// Extract response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Format response body
	bodyStr := string(bodyBytes)
	contentType := resp.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/json")
	
	if isJSON {
		// Pretty print JSON
		var jsonObj interface{}
		if err := json.Unmarshal(bodyBytes, &jsonObj); err == nil {
			if prettyBytes, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
				bodyStr = string(prettyBytes)
			}
		}
	}

	return &response.HTTPResponse{
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Headers:      headers,
		Body:         bodyStr,
		ResponseTime: responseTime,
		IsJSON:       isJSON,
	}, nil
}

func (c *HTTPClient) substituteVars(text string, vars map[string]string) string {
	// Replace {{VARIABLE}} patterns with actual values
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	
	return re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])
		
		if value, exists := vars[varName]; exists {
			return value
		}
		
		// Return original if variable not found
		return match
	})
}

func (c *HTTPClient) addAuth(req *http.Request, auth request.BruAuth, vars map[string]string) error {
	switch auth.Type {
	case "bearer":
		if token, exists := auth.Values["token"]; exists {
			processedToken := c.substituteVars(token, vars)
			req.Header.Set("Authorization", "Bearer "+processedToken)
		}
	case "basic":
		if username, userExists := auth.Values["username"]; userExists {
			if password, passExists := auth.Values["password"]; passExists {
				processedUser := c.substituteVars(username, vars)
				processedPass := c.substituteVars(password, vars)
				req.SetBasicAuth(processedUser, processedPass)
			}
		}
	case "apikey":
		if key, keyExists := auth.Values["key"]; keyExists {
			if value, valueExists := auth.Values["value"]; valueExists {
				processedKey := c.substituteVars(key, vars)
				processedValue := c.substituteVars(value, vars)
				
				// Add as header by default, could be query param based on config
				req.Header.Set(processedKey, processedValue)
			}
		}
	}
	
	return nil
}

func (c *HTTPClient) FormatResponseForDisplay(httpResp *response.HTTPResponse) string {
	if httpResp == nil {
		return "No response"
	}

	if httpResp.Error != "" {
		return fmt.Sprintf("Error: %s", httpResp.Error)
	}

	// Return only the response body since headers are displayed separately
	return httpResp.Body
}