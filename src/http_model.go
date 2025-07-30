package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPMethod represents supported HTTP methods
type HTTPMethod string

const (
	GET     HTTPMethod = "GET"
	POST    HTTPMethod = "POST"
	PUT     HTTPMethod = "PUT"
	PATCH   HTTPMethod = "PATCH"
	DELETE  HTTPMethod = "DELETE"
	HEAD    HTTPMethod = "HEAD"
	OPTIONS HTTPMethod = "OPTIONS"
)

// ContentType represents common content types
type ContentType string

const (
	ContentTypeJSON       ContentType = "application/json"
	ContentTypeXML        ContentType = "application/xml"
	ContentTypeForm       ContentType = "application/x-www-form-urlencoded"
	ContentTypeMultipart  ContentType = "multipart/form-data"
	ContentTypeText       ContentType = "text/plain"
	ContentTypeHTML       ContentType = "text/html"
	ContentTypeOctetStream ContentType = "application/octet-stream"
)

// AuthType represents authentication types
type AuthType string

const (
	AuthNone      AuthType = "none"
	AuthBasic     AuthType = "basic"
	AuthBearer    AuthType = "bearer"
	AuthAPIKey    AuthType = "apikey"
	AuthOAuth2    AuthType = "oauth2"
	AuthCustom    AuthType = "custom"
)

// HTTPRequestModel represents a complete HTTP request
type HTTPRequestModel struct {
	// Basic request properties
	Method      HTTPMethod            `json:"method"`
	URL         string               `json:"url"`
	Headers     map[string]string    `json:"headers,omitempty"`
	QueryParams map[string]string    `json:"query_params,omitempty"`
	
	// Request body
	Body        *RequestBody         `json:"body,omitempty"`
	
	// Authentication
	Auth        *AuthConfig          `json:"auth,omitempty"`
	
	// Request metadata
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	
	// Variables and environment
	Variables   map[string]string    `json:"variables,omitempty"`
	
	// Request configuration
	Timeout     time.Duration        `json:"timeout,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time            `json:"created_at,omitempty"`
	UpdatedAt   time.Time            `json:"updated_at,omitempty"`
}

// RequestBody represents the request body with different content types
type RequestBody struct {
	Type        string            `json:"type"` // "json", "xml", "form", "text", "binary", "none"
	Content     string            `json:"content,omitempty"`
	FormData    map[string]string `json:"form_data,omitempty"`
	Files       []FileUpload      `json:"files,omitempty"`
	ContentType ContentType       `json:"content_type,omitempty"`
}

// FileUpload represents a file to be uploaded
type FileUpload struct {
	FieldName string `json:"field_name"`
	Filename  string `json:"filename"`
	FilePath  string `json:"file_path"`
	MimeType  string `json:"mime_type,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     AuthType          `json:"type"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Token    string            `json:"token,omitempty"`
	APIKey   string            `json:"api_key,omitempty"`
	KeyName  string            `json:"key_name,omitempty"`
	Location string            `json:"location,omitempty"` // "header", "query", "body"
	Custom   map[string]string `json:"custom,omitempty"`
}

// HTTPResponseModel represents a complete HTTP response
type HTTPResponseModel struct {
	// Response status
	StatusCode   int               `json:"status_code"`
	Status       string            `json:"status"`
	StatusClass  StatusClass       `json:"status_class"`
	
	// Response headers
	Headers      map[string]string `json:"headers"`
	ContentType  string            `json:"content_type"`
	ContentLength int64            `json:"content_length"`
	
	// Response body
	Body         string            `json:"body"`
	BodyBytes    []byte            `json:"body_bytes,omitempty"`
	IsJSON       bool              `json:"is_json"`
	IsXML        bool              `json:"is_xml"`
	IsHTML       bool              `json:"is_html"`
	IsText       bool              `json:"is_text"`
	IsBinary     bool              `json:"is_binary"`
	
	// Timing information
	ResponseTime time.Duration     `json:"response_time"`
	DNSTime      time.Duration     `json:"dns_time,omitempty"`
	ConnectTime  time.Duration     `json:"connect_time,omitempty"`
	TLSTime      time.Duration     `json:"tls_time,omitempty"`
	SendTime     time.Duration     `json:"send_time,omitempty"`
	WaitTime     time.Duration     `json:"wait_time,omitempty"`
	ReceiveTime  time.Duration     `json:"receive_time,omitempty"`
	
	// Request/Response metadata
	RequestURL   string            `json:"request_url"`
	FinalURL     string            `json:"final_url,omitempty"` // After redirects
	Redirects    []RedirectInfo    `json:"redirects,omitempty"`
	
	// Error information
	Error        string            `json:"error,omitempty"`
	ErrorType    ErrorType         `json:"error_type,omitempty"`
	
	// Response cookies
	Cookies      []*http.Cookie    `json:"cookies,omitempty"`
	
	// Timestamps
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	
	// Additional metadata
	Protocol     string            `json:"protocol,omitempty"`     // HTTP/1.1, HTTP/2, etc.
	ServerAddr   string            `json:"server_addr,omitempty"`
	LocalAddr    string            `json:"local_addr,omitempty"`
}

// StatusClass represents HTTP status code classes
type StatusClass string

const (
	StatusInformational StatusClass = "1xx" // 100-199
	StatusSuccess       StatusClass = "2xx" // 200-299
	StatusRedirection   StatusClass = "3xx" // 300-399
	StatusClientError   StatusClass = "4xx" // 400-499
	StatusServerError   StatusClass = "5xx" // 500-599
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorNone        ErrorType = "none"
	ErrorNetwork     ErrorType = "network"
	ErrorTimeout     ErrorType = "timeout"
	ErrorDNS         ErrorType = "dns"
	ErrorTLS         ErrorType = "tls"
	ErrorAuth        ErrorType = "auth"
	ErrorRedirect    ErrorType = "redirect"
	ErrorParse       ErrorType = "parse"
	ErrorValidation  ErrorType = "validation"
	ErrorUnknown     ErrorType = "unknown"
)

// RedirectInfo represents information about a redirect
type RedirectInfo struct {
	FromURL    string    `json:"from_url"`
	ToURL      string    `json:"to_url"`
	StatusCode int       `json:"status_code"`
	Timestamp  time.Time `json:"timestamp"`
}

// RequestResponsePair represents a request-response pair for history/logging
type RequestResponsePair struct {
	ID       string             `json:"id"`
	Request  *HTTPRequestModel  `json:"request"`
	Response *HTTPResponseModel `json:"response"`
	Success  bool               `json:"success"`
	
	// Execution metadata
	ExecutedAt time.Time          `json:"executed_at"`
	Duration   time.Duration      `json:"duration"`
	
	// Test results (if any)
	TestResults []TestResult      `json:"test_results,omitempty"`
	
	// Environment context
	Environment string            `json:"environment,omitempty"`
	Collection  string            `json:"collection,omitempty"`
}

// TestResult represents the result of running a test on the response
type TestResult struct {
	Name     string    `json:"name"`
	Passed   bool      `json:"passed"`
	Message  string    `json:"message,omitempty"`
	Expected interface{} `json:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
}

// Helper methods for HTTPRequestModel

// SetHeader sets a header value
func (r *HTTPRequestModel) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
}

// GetHeader gets a header value
func (r *HTTPRequestModel) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers[key]
}

// SetQueryParam sets a query parameter
func (r *HTTPRequestModel) SetQueryParam(key, value string) {
	if r.QueryParams == nil {
		r.QueryParams = make(map[string]string)
	}
	r.QueryParams[key] = value
}

// GetQueryParam gets a query parameter
func (r *HTTPRequestModel) GetQueryParam(key string) string {
	if r.QueryParams == nil {
		return ""
	}
	return r.QueryParams[key]
}

// SetJSONBody sets the request body as JSON
func (r *HTTPRequestModel) SetJSONBody(data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	r.Body = &RequestBody{
		Type:        "json",
		Content:     string(jsonBytes),
		ContentType: ContentTypeJSON,
	}
	r.SetHeader("Content-Type", string(ContentTypeJSON))
	return nil
}

// SetTextBody sets the request body as plain text
func (r *HTTPRequestModel) SetTextBody(content string) {
	r.Body = &RequestBody{
		Type:        "text",
		Content:     content,
		ContentType: ContentTypeText,
	}
	r.SetHeader("Content-Type", string(ContentTypeText))
}

// SetFormBody sets the request body as form data
func (r *HTTPRequestModel) SetFormBody(data map[string]string) {
	r.Body = &RequestBody{
		Type:        "form",
		FormData:    data,
		ContentType: ContentTypeForm,
	}
	r.SetHeader("Content-Type", string(ContentTypeForm))
}

// SetBasicAuth sets basic authentication
func (r *HTTPRequestModel) SetBasicAuth(username, password string) {
	r.Auth = &AuthConfig{
		Type:     AuthBasic,
		Username: username,
		Password: password,
	}
}

// SetBearerToken sets bearer token authentication
func (r *HTTPRequestModel) SetBearerToken(token string) {
	r.Auth = &AuthConfig{
		Type:  AuthBearer,
		Token: token,
	}
}

// SetAPIKey sets API key authentication
func (r *HTTPRequestModel) SetAPIKey(keyName, keyValue, location string) {
	r.Auth = &AuthConfig{
		Type:     AuthAPIKey,
		KeyName:  keyName,
		APIKey:   keyValue,
		Location: location,
	}
}

// Helper methods for HTTPResponseModel

// GetStatusClass returns the status class based on status code
func (r *HTTPResponseModel) GetStatusClass() StatusClass {
	switch r.StatusCode / 100 {
	case 1:
		return StatusInformational
	case 2:
		return StatusSuccess
	case 3:
		return StatusRedirection
	case 4:
		return StatusClientError
	case 5:
		return StatusServerError
	default:
		return StatusClientError
	}
}

// IsSuccess returns true if the response indicates success (2xx)
func (r *HTTPResponseModel) IsSuccess() bool {
	return r.GetStatusClass() == StatusSuccess
}

// IsError returns true if the response indicates an error (4xx or 5xx)
func (r *HTTPResponseModel) IsError() bool {
	class := r.GetStatusClass()
	return class == StatusClientError || class == StatusServerError
}

// GetContentType returns the content type from headers
func (r *HTTPResponseModel) GetContentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	
	if contentType, exists := r.Headers["Content-Type"]; exists {
		// Extract just the media type, ignoring charset and other parameters
		parts := strings.Split(contentType, ";")
		return strings.TrimSpace(parts[0])
	}
	
	return ""
}

// ParsedBody attempts to parse the response body based on content type
func (r *HTTPResponseModel) ParsedBody() (interface{}, error) {
	contentType := r.GetContentType()
	
	switch {
	case strings.Contains(contentType, "application/json"):
		var jsonData interface{}
		err := json.Unmarshal([]byte(r.Body), &jsonData)
		return jsonData, err
	case strings.Contains(contentType, "application/xml"):
		// XML parsing would require additional library
		return r.Body, nil
	default:
		return r.Body, nil
	}
}

// GetFormattedDuration returns a human-readable duration string
func (r *HTTPResponseModel) GetFormattedDuration() string {
	if r.ResponseTime < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(r.ResponseTime.Nanoseconds())/1000)
	} else if r.ResponseTime < time.Second {
		return fmt.Sprintf("%.2fms", float64(r.ResponseTime.Nanoseconds())/1000000)
	} else {
		return fmt.Sprintf("%.2fs", r.ResponseTime.Seconds())
	}
}

// Validation methods

// Validate validates the HTTP request model
func (r *HTTPRequestModel) Validate() error {
	if r.Method == "" {
		return fmt.Errorf("HTTP method is required")
	}
	
	if r.URL == "" {
		return fmt.Errorf("URL is required")
	}
	
	// Validate URL format
	if _, err := url.Parse(r.URL); err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	
	// Validate method
	validMethods := []HTTPMethod{GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS}
	validMethod := false
	for _, method := range validMethods {
		if r.Method == method {
			validMethod = true
			break
		}
	}
	if !validMethod {
		return fmt.Errorf("invalid HTTP method: %s", r.Method)
	}
	
	return nil
}

// Clone creates a deep copy of the request model
func (r *HTTPRequestModel) Clone() *HTTPRequestModel {
	clone := &HTTPRequestModel{
		Method:      r.Method,
		URL:         r.URL,
		Name:        r.Name,
		Description: r.Description,
		Timeout:     r.Timeout,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	
	// Clone headers
	if r.Headers != nil {
		clone.Headers = make(map[string]string)
		for k, v := range r.Headers {
			clone.Headers[k] = v
		}
	}
	
	// Clone query params
	if r.QueryParams != nil {
		clone.QueryParams = make(map[string]string)
		for k, v := range r.QueryParams {
			clone.QueryParams[k] = v
		}
	}
	
	// Clone variables
	if r.Variables != nil {
		clone.Variables = make(map[string]string)
		for k, v := range r.Variables {
			clone.Variables[k] = v
		}
	}
	
	// Clone tags
	if r.Tags != nil {
		clone.Tags = make([]string, len(r.Tags))
		copy(clone.Tags, r.Tags)
	}
	
	// Clone body
	if r.Body != nil {
		clone.Body = &RequestBody{
			Type:        r.Body.Type,
			Content:     r.Body.Content,
			ContentType: r.Body.ContentType,
		}
		
		if r.Body.FormData != nil {
			clone.Body.FormData = make(map[string]string)
			for k, v := range r.Body.FormData {
				clone.Body.FormData[k] = v
			}
		}
		
		if r.Body.Files != nil {
			clone.Body.Files = make([]FileUpload, len(r.Body.Files))
			copy(clone.Body.Files, r.Body.Files)
		}
	}
	
	// Clone auth
	if r.Auth != nil {
		clone.Auth = &AuthConfig{
			Type:     r.Auth.Type,
			Username: r.Auth.Username,
			Password: r.Auth.Password,
			Token:    r.Auth.Token,
			APIKey:   r.Auth.APIKey,
			KeyName:  r.Auth.KeyName,
			Location: r.Auth.Location,
		}
		
		if r.Auth.Custom != nil {
			clone.Auth.Custom = make(map[string]string)
			for k, v := range r.Auth.Custom {
				clone.Auth.Custom[k] = v
			}
		}
	}
	
	return clone
}