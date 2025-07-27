package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

func TestRecordingPlugin_BasicFunctionality(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recording_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create recording plugin
	targetURL := "https://example.com"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Override inventory directory for testing
	plugin.inventoryDir = tempDir

	// Simulate a request/response flow
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    parseURL(t, "https://example.com/test"),
			Header: make(http.Header),
		},
	}

	// Simulate request processing
	plugin.Request(flow)

	// Check that transaction was recorded
	plugin.mutex.RLock()
	if len(plugin.transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(plugin.transactions))
	}
	plugin.mutex.RUnlock()

	// Simulate response
	flow.Response = &proxy.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":     []string{"text/html"},
			"Content-Encoding": []string{"gzip"},
		},
		Body: []byte("test response body"),
	}

	// Process response
	plugin.Response(flow)

	// Verify transaction was updated
	plugin.mutex.RLock()
	transaction := plugin.transactions[0]
	if transaction.StatusCode == nil || *transaction.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %v", transaction.StatusCode)
	}
	if len(transaction.Body) == 0 {
		t.Error("Expected response body to be recorded")
	}
	if transaction.RawHeaders["Content-Type"] != "text/html" {
		t.Error("Expected Content-Type header to be recorded")
	}
	plugin.mutex.RUnlock()
}

func TestRecordingPlugin_SaveInventory(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recording_save_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create recording plugin
	targetURL := "https://example.com"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Override inventory directory for testing
	plugin.inventoryDir = tempDir

	// Add test transactions
	statusCode := 200
	transaction1 := RecordingTransaction{
		Method:           "GET",
		URL:              "https://example.com/page1",
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(50 * time.Millisecond),
		ResponseFinished: time.Now().Add(100 * time.Millisecond),
		StatusCode:       &statusCode,
		RawHeaders: HttpHeaders{
			"Content-Type": "text/html",
		},
		Body: []byte("page1 content"),
	}

	transaction2 := RecordingTransaction{
		Method:           "GET",
		URL:              "https://example.com/style.css",
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(30 * time.Millisecond),
		ResponseFinished: time.Now().Add(80 * time.Millisecond),
		StatusCode:       &statusCode,
		RawHeaders: HttpHeaders{
			"Content-Type": "text/css",
		},
		Body: []byte("css content"),
	}

	plugin.transactions = []RecordingTransaction{transaction1, transaction2}

	// Add test domains
	plugin.domains = []Domain{
		{Name: "example.com", IPAddress: "192.168.1.1"},
	}

	// Save inventory
	err = plugin.SaveInventory()
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Check if inventory.json was created
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		t.Fatal("inventory.json was not created")
	}

	// Read and verify inventory
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory file: %v", err)
	}

	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to parse inventory JSON: %v", err)
	}

	// Verify inventory contents
	if inventory.EntryURL == nil || *inventory.EntryURL != targetURL {
		t.Errorf("Expected entry URL %s, got %v", targetURL, inventory.EntryURL)
	}

	if len(inventory.Domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(inventory.Domains))
	}

	if inventory.Domains[0].Name != "example.com" {
		t.Errorf("Expected domain name example.com, got %s", inventory.Domains[0].Name)
	}

	if len(inventory.Resources) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(inventory.Resources))
	}

	// Check first resource
	resource1 := inventory.Resources[0]
	if resource1.Method != "GET" {
		t.Errorf("Expected method GET, got %s", resource1.Method)
	}
	if resource1.URL != "https://example.com/page1" {
		t.Errorf("Expected URL https://example.com/page1, got %s", resource1.URL)
	}
	if resource1.StatusCode == nil || *resource1.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %v", resource1.StatusCode)
	}

	// Check that content files were created
	for _, resource := range inventory.Resources {
		if resource.ContentFilePath != nil {
			contentPath := filepath.Join(tempDir, "contents", *resource.ContentFilePath)
			if _, err := os.Stat(contentPath); os.IsNotExist(err) {
				t.Errorf("Content file not created: %s", contentPath)
			}
		}
	}
}

func TestRecordingPlugin_MultipleRequestResponse(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recording_multi_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create recording plugin
	targetURL := "https://example.com"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Override inventory directory for testing
	plugin.inventoryDir = tempDir

	// Test multiple request/response cycles
	testCases := []struct {
		url        string
		statusCode int
		contentType string
		body       string
	}{
		{"https://example.com/", 200, "text/html", "<html>home page</html>"},
		{"https://example.com/style.css", 200, "text/css", "body { margin: 0; }"},
		{"https://example.com/script.js", 200, "application/javascript", "console.log('hello');"},
		{"https://example.com/api/data", 200, "application/json", `{"message": "success"}`},
	}

	for i, tc := range testCases {
		// Create flow
		flow := &proxy.Flow{
			Request: &proxy.Request{
				Method: "GET",
				URL:    parseURL(t, tc.url),
				Header: make(http.Header),
			},
		}

		// Process request
		plugin.Request(flow)

		// Process response
		flow.Response = &proxy.Response{
			StatusCode: tc.statusCode,
			Header: http.Header{
				"Content-Type": []string{tc.contentType},
			},
			Body: []byte(tc.body),
		}

		plugin.Response(flow)

		// Verify transaction count
		plugin.mutex.RLock()
		if len(plugin.transactions) != i+1 {
			t.Fatalf("Expected %d transactions, got %d", i+1, len(plugin.transactions))
		}
		plugin.mutex.RUnlock()
	}

	// Save inventory
	err = plugin.SaveInventory()
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Verify final inventory
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory file: %v", err)
	}

	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to parse inventory JSON: %v", err)
	}

	if len(inventory.Resources) != len(testCases) {
		t.Fatalf("Expected %d resources, got %d", len(testCases), len(inventory.Resources))
	}

	// Verify each resource
	for i, tc := range testCases {
		resource := inventory.Resources[i]
		if resource.URL != tc.url {
			t.Errorf("Resource %d: expected URL %s, got %s", i, tc.url, resource.URL)
		}
		if resource.StatusCode == nil || *resource.StatusCode != tc.statusCode {
			t.Errorf("Resource %d: expected status %d, got %v", i, tc.statusCode, resource.StatusCode)
		}
		if resource.ContentTypeMime == nil || *resource.ContentTypeMime != tc.contentType {
			t.Errorf("Resource %d: expected content type %s, got %v", i, tc.contentType, resource.ContentTypeMime)
		}
	}
}

func TestRecordingPlugin_DomainTracking(t *testing.T) {
	// Create recording plugin
	targetURL := "https://example.com"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Test domain recording
	plugin.recordDomainIP("example.com", "192.168.1.1")
	plugin.recordDomainIP("cdn.example.com", "192.168.1.2")
	plugin.recordDomainIP("example.com", "192.168.1.1") // duplicate should not create new entry

	// Verify domains
	if len(plugin.domains) != 2 {
		t.Fatalf("Expected 2 domains, got %d", len(plugin.domains))
	}

	domainMap := make(map[string]string)
	for _, domain := range plugin.domains {
		domainMap[domain.Name] = domain.IPAddress
	}

	if ip, exists := domainMap["example.com"]; !exists || ip != "192.168.1.1" {
		t.Errorf("Expected example.com -> 192.168.1.1, got %s", ip)
	}

	if ip, exists := domainMap["cdn.example.com"]; !exists || ip != "192.168.1.2" {
		t.Errorf("Expected cdn.example.com -> 192.168.1.2, got %s", ip)
	}
}

func TestRecordingPlugin_EmptyResponse(t *testing.T) {
	// Create recording plugin
	targetURL := "https://example.com"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Create flow with empty response
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    parseURL(t, "https://example.com/empty"),
			Header: make(http.Header),
		},
	}

	// Process request
	plugin.Request(flow)

	// Process empty response
	flow.Response = &proxy.Response{
		StatusCode: 204, // No Content
		Header:     make(http.Header),
		Body:       []byte{}, // Empty body
	}

	plugin.Response(flow)

	// Verify transaction was recorded
	plugin.mutex.RLock()
	if len(plugin.transactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(plugin.transactions))
	}

	transaction := plugin.transactions[0]
	if transaction.StatusCode == nil || *transaction.StatusCode != 204 {
		t.Errorf("Expected status code 204, got %v", transaction.StatusCode)
	}
	if len(transaction.Body) != 0 {
		t.Errorf("Expected empty body, got %d bytes", len(transaction.Body))
	}
	plugin.mutex.RUnlock()
}

// Helper function to parse URL for testing
func parseURL(t *testing.T, urlStr string) *url.URL {
	t.Helper()
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("Failed to parse URL %s: %v", urlStr, err)
	}
	return parsedURL
}

// Test end-to-end functionality with a mock HTTP server
func TestRecordingPlugin_Integration(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recording_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create recording plugin
	targetURL := "https://httpbin.org"
	plugin, err := NewRecordingPlugin(targetURL)
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}

	// Override inventory directory for testing
	plugin.inventoryDir = tempDir

	// Create a realistic flow
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL:    parseURL(t, "https://httpbin.org/get"),
			Header: http.Header{
				"User-Agent": []string{"test-client/1.0"},
				"Accept":     []string{"application/json"},
			},
		},
	}

	// Process request
	plugin.Request(flow)

	// Simulate a realistic response
	responseBody := `{
		"args": {},
		"headers": {
			"Accept": "application/json",
			"User-Agent": "test-client/1.0"
		},
		"origin": "127.0.0.1",
		"url": "https://httpbin.org/get"
	}`

	// Simulate some delay to test TTFB calculation
	time.Sleep(10 * time.Millisecond)

	flow.Response = &proxy.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":     []string{"application/json"},
			"Content-Length":   []string{string(rune(len(responseBody)))},
			"Server":           []string{"nginx/1.10.0"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: []byte(responseBody),
	}

	// Process response
	plugin.Response(flow)

	// Add domain
	plugin.recordDomainIP("httpbin.org", "54.230.96.147")

	// Save inventory
	err = plugin.SaveInventory()
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Verify inventory was saved correctly
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory file: %v", err)
	}

	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to parse inventory JSON: %v", err)
	}

	// Verify inventory structure
	if len(inventory.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(inventory.Resources))
	}

	resource := inventory.Resources[0]
	if resource.Method != "GET" {
		t.Errorf("Expected method GET, got %s", resource.Method)
	}

	if resource.ContentTypeMime == nil || *resource.ContentTypeMime != "application/json" {
		t.Errorf("Expected content type application/json, got %v", resource.ContentTypeMime)
	}

	// Verify TTFB was calculated
	if resource.TTFBMs <= 0 {
		t.Errorf("Expected positive TTFB, got %d", resource.TTFBMs)
	}

	// Verify domain was recorded
	if len(inventory.Domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(inventory.Domains))
	}

	domain := inventory.Domains[0]
	if domain.Name != "httpbin.org" {
		t.Errorf("Expected domain httpbin.org, got %s", domain.Name)
	}

	t.Logf("Integration test completed successfully. Inventory saved with %d resources and %d domains",
		len(inventory.Resources), len(inventory.Domains))
}