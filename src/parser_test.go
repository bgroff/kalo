package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"kalo/src/panels"
)

func TestBruParser(t *testing.T) {
	testFiles := []string{
		"../examples/get-users.bru",
		"../examples/create-user.bru",
		"../examples/update-user.bru",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			file, err := os.Open(filename)
			if err != nil {
				t.Fatalf("Failed to open %s: %v", filename, err)
			}
			defer file.Close()

			parser := NewBruParser(file)
			request, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", filename, err)
			}

			if request == nil {
				t.Fatalf("Parser returned nil request for %s", filename)
			}

			jsonData, err := json.MarshalIndent(request, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal request to JSON: %v", err)
			}

			fmt.Printf("Parsed %s:\n%s\n\n", filename, string(jsonData))

			validateRequest(t, request, filename)
		})
	}
}

func validateRequest(t *testing.T, request *panels.BruRequest, filename string) {
	if request.Meta.Name == "" {
		t.Errorf("Missing meta.name in %s", filename)
	}

	if request.HTTP.Method == "" {
		t.Errorf("Missing HTTP method in %s", filename)
	}

	if request.HTTP.URL == "" {
		t.Errorf("Missing HTTP URL in %s", filename)
	}

	switch filepath.Base(filename) {
	case "get-users.bru":
		if request.HTTP.Method != "GET" {
			t.Errorf("Expected GET method, got %s", request.HTTP.Method)
		}
		if request.Meta.Name != "Get Users" {
			t.Errorf("Expected name 'Get Users', got %s", request.Meta.Name)
		}
		if len(request.Query) == 0 {
			t.Error("Expected query parameters")
		}

	case "create-user.bru":
		if request.HTTP.Method != "POST" {
			t.Errorf("Expected POST method, got %s", request.HTTP.Method)
		}
		if request.Body.Type != "json" {
			t.Errorf("Expected json body type, got %s", request.Body.Type)
		}
		if request.Auth.Type != "bearer" {
			t.Errorf("Expected bearer auth, got %s", request.Auth.Type)
		}
		if !strings.Contains(request.Body.Data, "John Doe") {
			t.Error("Expected body to contain 'John Doe'")
		}

	case "update-user.bru":
		if request.HTTP.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", request.HTTP.Method)
		}
		if !strings.Contains(request.Body.Data, "Jane Doe") {
			t.Error("Expected body to contain 'Jane Doe'")
		}
	}
}

func TestParseSimpleBru(t *testing.T) {
	bruContent := `meta {
  name: Simple Test
  type: http
  seq: 1
}

get {
  url: https://example.com/api
  body: none
  auth: none
}

headers {
  Accept: application/json
}`

	parser := NewBruParser(strings.NewReader(bruContent))
	request, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse simple bru: %v", err)
	}

	if request.Meta.Name != "Simple Test" {
		t.Errorf("Expected name 'Simple Test', got %s", request.Meta.Name)
	}

	if request.HTTP.Method != "GET" {
		t.Errorf("Expected GET method, got %s", request.HTTP.Method)
	}

	if request.HTTP.URL != "https://example.com/api" {
		t.Errorf("Expected URL 'https://example.com/api', got %s", request.HTTP.URL)
	}

	if request.Headers["Accept"] != "application/json" {
		t.Errorf("Expected Accept header 'application/json', got %s", request.Headers["Accept"])
	}
}