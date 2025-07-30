package main

import (
	"fmt"
	"time"
)

// ExampleHTTPModels demonstrates how to use the HTTP request and response models
func ExampleHTTPModels() {
	
	// Example 1: Simple GET request
	getRequest := &HTTPRequestModel{
		Method: GET,
		URL:    "https://api.github.com/users/octocat",
		Name:   "Get GitHub User",
		Headers: map[string]string{
			"User-Agent": "Kalo HTTP Client",
			"Accept":     "application/json",
		},
		Tags:      []string{"github", "api", "user"},
		CreatedAt: time.Now(),
	}
	
	// Example 2: POST request with JSON body
	postRequest := &HTTPRequestModel{
		Method: POST,
		URL:    "https://api.example.com/users",
		Name:   "Create User",
	}
	
	// Set JSON body
	userData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}
	postRequest.SetJSONBody(userData)
	
	// Add authentication
	postRequest.SetBearerToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
	
	// Example 3: Form data request
	formRequest := &HTTPRequestModel{
		Method: POST,
		URL:    "https://httpbin.org/post",
		Name:   "Submit Form",
	}
	
	formData := map[string]string{
		"username": "testuser",
		"password": "secret123",
		"email":    "test@example.com",
	}
	formRequest.SetFormBody(formData)
	
	// Example 4: Request with query parameters
	searchRequest := &HTTPRequestModel{
		Method: GET,
		URL:    "https://api.github.com/search/repositories",
		Name:   "Search Repositories",
	}
	searchRequest.SetQueryParam("q", "language:go")
	searchRequest.SetQueryParam("sort", "stars")
	searchRequest.SetQueryParam("order", "desc")
	
	// Example 5: Request with basic authentication
	authRequest := &HTTPRequestModel{
		Method: GET,
		URL:    "https://httpbin.org/basic-auth/user/pass",
		Name:   "Basic Auth Test",
	}
	authRequest.SetBasicAuth("user", "pass")
	
	// Example 6: Request with API key authentication
	apiRequest := &HTTPRequestModel{
		Method: GET,
		URL:    "https://api.openweathermap.org/data/2.5/weather",
		Name:   "Get Weather Data",
	}
	apiRequest.SetAPIKey("appid", "your-api-key-here", "query")
	apiRequest.SetQueryParam("q", "London")
	
	// Example Response Models
	
	// Example 1: Successful JSON response
	successResponse := &HTTPResponseModel{
		StatusCode:   200,
		Status:       "200 OK",
		StatusClass:  StatusSuccess,
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"Content-Length": "156",
			"Server":         "nginx/1.18.0",
		},
		Body: `{
			"id": 1,
			"login": "octocat",
			"name": "The Octocat",
			"company": "GitHub",
			"blog": "https://github.com/blog",
			"location": "San Francisco"
		}`,
		IsJSON:       true,
		ResponseTime: 245 * time.Millisecond,
		RequestURL:   "https://api.github.com/users/octocat",
		StartTime:    time.Now().Add(-245 * time.Millisecond),
		EndTime:      time.Now(),
		Protocol:     "HTTP/2",
	}
	
	// Example 2: Error response
	errorResponse := &HTTPResponseModel{
		StatusCode:   404,
		Status:       "404 Not Found",
		StatusClass:  StatusClientError,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Server":       "nginx/1.18.0",
		},
		Body: `{
			"message": "Not Found",
			"documentation_url": "https://docs.github.com/rest"
		}`,
		IsJSON:       true,
		ResponseTime: 123 * time.Millisecond,
		RequestURL:   "https://api.github.com/users/nonexistent",
		StartTime:    time.Now().Add(-123 * time.Millisecond),
		EndTime:      time.Now(),
		Protocol:     "HTTP/2",
	}
	
	// Example 3: Network error response
	networkErrorResponse := &HTTPResponseModel{
		StatusCode:   0,
		Status:       "",
		Error:        "dial tcp: lookup api.example.com: no such host",
		ErrorType:    ErrorDNS,
		ResponseTime: 5 * time.Second,
		RequestURL:   "https://api.example.com/users",
		StartTime:    time.Now().Add(-5 * time.Second),
		EndTime:      time.Now(),
	}
	
	// Example 4: Response with redirects
	redirectResponse := &HTTPResponseModel{
		StatusCode:  200,
		Status:      "200 OK",
		StatusClass: StatusSuccess,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body:         "<html><body>Final destination</body></html>",
		IsHTML:       true,
		ResponseTime: 456 * time.Millisecond,
		RequestURL:   "http://example.com/redirect",
		FinalURL:     "https://example.com/final",
		Redirects: []RedirectInfo{
			{
				FromURL:    "http://example.com/redirect",
				ToURL:      "https://example.com/redirect",
				StatusCode: 301,
				Timestamp:  time.Now().Add(-400 * time.Millisecond),
			},
			{
				FromURL:    "https://example.com/redirect",
				ToURL:      "https://example.com/final",
				StatusCode: 302,
				Timestamp:  time.Now().Add(-300 * time.Millisecond),
			},
		},
		StartTime: time.Now().Add(-456 * time.Millisecond),
		EndTime:   time.Now(),
		Protocol:  "HTTP/1.1",
	}
	
	// Example Request-Response Pairs for history
	
	pair1 := &RequestResponsePair{
		ID:          "req_123456789",
		Request:     getRequest,
		Response:    successResponse,
		Success:     true,
		ExecutedAt:  time.Now(),
		Duration:    successResponse.ResponseTime,
		Environment: "production",
		Collection:  "GitHub API",
	}
	
	pair2 := &RequestResponsePair{
		ID:          "req_987654321",
		Request:     postRequest,
		Response:    errorResponse,
		Success:     false,
		ExecutedAt:  time.Now(),
		Duration:    errorResponse.ResponseTime,
		Environment: "staging",
		Collection:  "User Management",
		TestResults: []TestResult{
			{
				Name:     "Status Code Check",
				Passed:   false,
				Message:  "Expected 201 but got 404",
				Expected: 201,
				Actual:   404,
			},
			{
				Name:     "Response Time Check",
				Passed:   true,
				Message:  "Response time within acceptable range",
				Expected: "< 500ms",
				Actual:   "123ms",
			},
		},
	}
	
	// Demonstrate usage
	fmt.Printf("GET Request: %s %s\n", getRequest.Method, getRequest.URL)
	fmt.Printf("POST Request Body Type: %s\n", postRequest.Body.Type)
	fmt.Printf("Success Response Status: %s (%s)\n", successResponse.Status, successResponse.StatusClass)
	fmt.Printf("Error Response: %s\n", errorResponse.Status)
	fmt.Printf("Network Error: %s (%s)\n", networkErrorResponse.Error, networkErrorResponse.ErrorType)
	fmt.Printf("Redirect Response Redirects: %d\n", len(redirectResponse.Redirects))
	fmt.Printf("Request-Response Pair Success: %t\n", pair1.Success)
	fmt.Printf("Test Results: %d tests, %d passed\n", 
		len(pair2.TestResults), 
		countPassedTests(pair2.TestResults))
}

// Helper function to count passed tests
func countPassedTests(results []TestResult) int {
	count := 0
	for _, result := range results {
		if result.Passed {
			count++
		}
	}
	return count
}

// ExampleRequestValidation demonstrates request validation
func ExampleRequestValidation() {
	// Valid request
	validRequest := &HTTPRequestModel{
		Method: GET,
		URL:    "https://api.example.com/users",
	}
	
	if err := validRequest.Validate(); err != nil {
		fmt.Printf("Valid request failed validation: %v\n", err)
	} else {
		fmt.Println("Request is valid")
	}
	
	// Invalid request - missing URL
	invalidRequest := &HTTPRequestModel{
		Method: POST,
		// URL missing
	}
	
	if err := invalidRequest.Validate(); err != nil {
		fmt.Printf("Invalid request validation error: %v\n", err)
	}
	
	// Invalid request - bad method
	badMethodRequest := &HTTPRequestModel{
		Method: HTTPMethod("INVALID"),
		URL:    "https://api.example.com/users",
	}
	
	if err := badMethodRequest.Validate(); err != nil {
		fmt.Printf("Bad method request validation error: %v\n", err)
	}
}

// ExampleResponseHelpers demonstrates response helper methods
func ExampleResponseHelpers() {
	response := &HTTPResponseModel{
		StatusCode:   201,
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8",
			"Location":     "/users/123",
		},
		Body:         `{"id": 123, "name": "John Doe"}`,
		ResponseTime: 150 * time.Millisecond,
	}
	
	fmt.Printf("Status Class: %s\n", response.GetStatusClass())
	fmt.Printf("Is Success: %t\n", response.IsSuccess())
	fmt.Printf("Is Error: %t\n", response.IsError())
	fmt.Printf("Content Type: %s\n", response.GetContentType())
	fmt.Printf("Formatted Duration: %s\n", response.GetFormattedDuration())
	
	// Parse JSON body
	if parsedBody, err := response.ParsedBody(); err == nil {
		fmt.Printf("Parsed Body: %+v\n", parsedBody)
	}
}

// ExampleRequestCloning demonstrates request cloning
func ExampleRequestCloning() {
	original := &HTTPRequestModel{
		Method: POST,
		URL:    "https://api.example.com/users",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		},
		Tags: []string{"api", "users"},
	}
	original.SetJSONBody(map[string]string{"name": "John"})
	
	// Clone the request
	cloned := original.Clone()
	
	// Modify the clone
	cloned.URL = "https://api.example.com/posts"
	cloned.SetHeader("X-Custom", "value")
	
	fmt.Printf("Original URL: %s\n", original.URL)
	fmt.Printf("Cloned URL: %s\n", cloned.URL)
	fmt.Printf("Original has X-Custom header: %t\n", original.GetHeader("X-Custom") != "")
	fmt.Printf("Clone has X-Custom header: %t\n", cloned.GetHeader("X-Custom") != "")
}