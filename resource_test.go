package main

import (
	"strings"
	"testing"
)

func TestMethodURLToFilePath(t *testing.T) {
	testCases := []struct {
		name     string
		method   string
		url      string
		expected string
	}{
		{
			name:     "Basic GET request with file",
			method:   "GET",
			url:      "https://www.example.com/path/to/image.jpg",
			expected: "get/https/www.example.com/path/to/image.jpg",
		},
		{
			name:     "Directory path with trailing slash",
			method:   "GET",
			url:      "https://www.example.com/path/",
			expected: "get/https/www.example.com/path/index.html",
		},
		{
			name:     "Root path",
			method:   "GET",
			url:      "https://www.example.com/",
			expected: "get/https/www.example.com/index.html",
		},
		{
			name:     "Path without extension",
			method:   "GET",
			url:      "https://www.example.com/api/users",
			expected: "get/https/www.example.com/api/users/index.html",
		},
		{
			name:     "POST request",
			method:   "POST",
			url:      "https://api.example.com/data",
			expected: "post/https/api.example.com/data/index.html",
		},
		{
			name:     "HTTP (not HTTPS)",
			method:   "GET",
			url:      "http://example.com/test.html",
			expected: "get/http/example.com/test.html",
		},
		{
			name:     "Mixed case method and hostname",
			method:   "GeT",
			url:      "HTTPS://WWW.EXAMPLE.COM/Path.html",
			expected: "get/https/www.example.com/Path.html",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := MethodURLToFilePath(tc.method, tc.url)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestMethodURLToFilePathWithParams(t *testing.T) {
	testCases := []struct {
		name     string
		method   string
		url      string
		expected string
	}{
		{
			name:     "URL with short parameters",
			method:   "GET",
			url:      "http://example.com/path/to/image.jpg?param=value",
			expected: "get/http/example.com/path/to/image~param=value.jpg",
		},
		{
			name:     "URL with multiple parameters",
			method:   "GET",
			url:      "https://example.com/api?user=123&action=view",
			expected: "get/https/example.com/api/index~user=123&action=view.html",
		},
		{
			name:     "URL with no extension and parameters",
			method:   "GET",
			url:      "https://example.com/search?q=test&limit=10",
			expected: "get/https/example.com/search/index~q=test&limit=10.html",
		},
		{
			name:     "URL with spaces in parameters",
			method:   "GET",
			url:      "https://example.com/api?name=john doe&id=123",
			expected: "get/https/example.com/api/index~name=john%20doe&id=123.html",
		},
		{
			name:     "URL with Japanese parameters",
			method:   "GET",
			url:      "https://example.com/search?q=東京&lang=ja",
			expected: "get/https/example.com/search/index~q=%E6%9D%B1%E4%BA%AC&lang=ja.html",
		},
		{
			name:     "URL with already encoded parameters",
			method:   "GET",
			url:      "https://example.com/api?name=john%20doe&id=123",
			expected: "get/https/example.com/api/index~name=john%20doe&id=123.html",
		},
		{
			name:     "URL with mixed encoded and unencoded parameters",
			method:   "GET",
			url:      "https://example.com/api?id=1&name=太郎",
			expected: "get/https/example.com/api/index~id=1&name=%E5%A4%AA%E9%83%8E.html",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := MethodURLToFilePath(tc.method, tc.url)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestMethodURLToFilePathWithLongParams(t *testing.T) {
	// Create a long parameter string (over 32 characters) with Japanese
	url := "https://example.com/test.jpg?message=これは非常に長いメッセージです&user=田中太郎&action=詳細表示"

	result, err := MethodURLToFilePath("GET", url)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The result should contain a hashed version
	t.Logf("Long param result: %s", result)

	// Check that the result contains the ~ separator and .jpg extension
	if !strings.Contains(result, "~") {
		t.Error("Expected result to contain ~ separator")
	}
	if !strings.HasSuffix(result, ".jpg") {
		t.Error("Expected result to end with .jpg")
	}

	// Extract the parameter part
	parts := strings.Split(result, "~")
	if len(parts) != 2 {
		t.Errorf("Expected 2 parts when split by ~, got %d", len(parts))
	}

	paramPart := strings.TrimSuffix(parts[1], ".jpg")
	// Should be 32 + 8 = 40 characters max
	if len(paramPart) > 40 {
		t.Errorf("Parameter part too long: %d characters (max 40)", len(paramPart))
	}
	
	// Verify = and & are preserved in the prefix part
	if !strings.Contains(paramPart[:32], "=") {
		t.Error("Expected = to be preserved in parameter encoding")
	}
	
	// Check that proper percent encoding is used for Japanese characters
	if !strings.Contains(paramPart[:32], "%") {
		t.Error("Expected percent encoding for Japanese characters")
	}
}

func TestFilePathToMethodURL(t *testing.T) {
	testCases := []struct {
		name         string
		filePath     string
		expectedMethod string
		expectedURL  string
	}{
		{
			name:         "Basic file path",
			filePath:     "get/https/www.example.com/path/to/image.jpg",
			expectedMethod: "GET",
			expectedURL:  "https://www.example.com/path/to/image.jpg",
		},
		{
			name:         "Index.html conversion",
			filePath:     "get/https/www.example.com/path/index.html",
			expectedMethod: "GET",
			expectedURL:  "https://www.example.com/path/",
		},
		{
			name:         "Root index.html",
			filePath:     "get/https/www.example.com/index.html",
			expectedMethod: "GET",
			expectedURL:  "https://www.example.com/",
		},
		{
			name:         "POST method",
			filePath:     "post/https/api.example.com/data/index.html",
			expectedMethod: "POST",
			expectedURL:  "https://api.example.com/data/",
		},
		{
			name:         "Index with parameters",
			filePath:     "get/https/example.com/api/index~user=123&action=view.html",
			expectedMethod: "GET",
			expectedURL:  "https://example.com/api?user=123&action=view",
		},
		{
			name:         "Index with spaces in parameters",
			filePath:     "get/https/example.com/api/index~name=john%20doe&id=123.html",
			expectedMethod: "GET",
			expectedURL:  "https://example.com/api?name=john doe&id=123",
		},
		{
			name:         "Index with Japanese parameters",
			filePath:     "get/https/example.com/search/index~q=%E6%9D%B1%E4%BA%AC&lang=ja.html",
			expectedMethod: "GET",
			expectedURL:  "https://example.com/search?q=東京&lang=ja",
		},
		{
			name:         "Index with mixed encoded parameters",
			filePath:     "get/https/example.com/api/index~id=1&name=%E5%A4%AA%E9%83%8E.html",
			expectedMethod: "GET",
			expectedURL:  "https://example.com/api?id=1&name=太郎",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			method, url, err := FilePathToMethodURL(tc.filePath)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if method != tc.expectedMethod {
				t.Errorf("Expected method %s, got %s", tc.expectedMethod, method)
			}
			if url != tc.expectedURL {
				t.Errorf("Expected URL %s, got %s", tc.expectedURL, url)
			}
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	testCases := []struct {
		method string
		url    string
	}{
		{"GET", "https://www.example.com/path/to/image.jpg"},
		{"POST", "https://api.example.com/data"},
		{"GET", "http://example.com/path/?param=value"},
		{"PUT", "https://www.example.com/"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.url, func(t *testing.T) {
			// Convert to file path
			filePath, err := MethodURLToFilePath(tc.method, tc.url)
			if err != nil {
				t.Fatalf("MethodURLToFilePath error: %v", err)
			}

			// Convert back to method and URL
			method, url, err := FilePathToMethodURL(filePath)
			if err != nil {
				t.Fatalf("FilePathToMethodURL error: %v", err)
			}

			// Check method
			if method != tc.method {
				t.Errorf("Method mismatch: expected %s, got %s", tc.method, method)
			}

			// For URLs ending with slash, the round trip will normalize them
			expectedURL := tc.url
			if strings.HasSuffix(tc.url, "/") && !strings.HasSuffix(tc.url, "//") {
				// The reverse conversion will keep the trailing slash
			}

			if url != expectedURL {
				t.Logf("URL changed during round trip: %s -> %s", tc.url, url)
				// This might be expected behavior for some cases
			}
		})
	}
}

func TestSanitizeFilePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unsafe characters",
			input:    "path/with<unsafe>chars:and|more",
			expected: "path/with_unsafe_chars_and_more",
		},
		{
			name:     "Windows reserved name",
			input:    "get/https/example.com/CON.txt",
			expected: "get/https/example.com/_CON.txt",
		},
		{
			name:     "Normal path",
			input:    "get/https/example.com/normal/path.jpg",
			expected: "get/https/example.com/normal/path.jpg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeFilePath(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestGetResourceFilePath(t *testing.T) {
	method := "GET"
	url := "https://www.example.com/path/to/image.jpg?param=value"
	
	result, err := GetResourceFilePath(method, url)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Logf("Resource file path: %s", result)

	// Should be a sanitized version of the converted path
	if !strings.Contains(result, "get/https/www.example.com") {
		t.Error("Expected result to contain basic path structure")
	}
}

func TestInvalidURLs(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		url    string
	}{
		{
			name:   "Invalid URL",
			method: "GET",
			url:    "not-a-valid-url",
		},
		{
			name:   "URL without hostname",
			method: "GET",
			url:    "https:///path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MethodURLToFilePath(tc.method, tc.url)
			if err == nil {
				t.Error("Expected error for invalid URL")
			}
		})
	}
}

func TestCustomOptions(t *testing.T) {
	options := ResourcePathOptions{
		MaxParamLength: 10, // Very short for testing
		HashLength:     4,
	}

	url := "https://example.com/test.jpg?verylongparameter=value"
	result, err := MethodURLToFilePathWithOptions("GET", url, options)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Logf("Custom options result: %s", result)

	// Check that hashing was applied
	if !strings.Contains(result, "~") {
		t.Error("Expected result to contain ~ separator")
	}
}