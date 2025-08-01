package plugins

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"go-http-playback-proxy/pkg/types"
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

	// Add response
	flow.Response = &proxy.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       []byte("test response"),
	}
	flow.Response.Header.Set("Content-Type", "text/plain")

	// Simulate response processing
	plugin.Response(flow)

	// Check transaction was updated
	plugin.mutex.RLock()
	transaction := plugin.transactions[0]
	plugin.mutex.RUnlock()

	if transaction.StatusCode == nil || *transaction.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %v", transaction.StatusCode)
	}

	// Save inventory
	err = plugin.SaveInventory()
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Check inventory file exists
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		t.Fatalf("Inventory file not created")
	}

	// Load and verify inventory
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory: %v", err)
	}

	var inventory types.Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to unmarshal inventory: %v", err)
	}

	if len(inventory.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(inventory.Resources))
	}

	resource := inventory.Resources[0]
	if resource.Method != "GET" {
		t.Errorf("Expected method GET, got %s", resource.Method)
	}
	if resource.URL != "https://example.com/test" {
		t.Errorf("Expected URL https://example.com/test, got %s", resource.URL)
	}
}

func TestRecordingPlugin_MultipleTransactions(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recording_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create recording plugin
	plugin, err := NewRecordingPlugin("https://example.com")
	if err != nil {
		t.Fatalf("Failed to create recording plugin: %v", err)
	}
	plugin.inventoryDir = tempDir

	// Record multiple transactions
	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/api/data",
	}

	for i, urlStr := range urls {
		flow := &proxy.Flow{
			Request: &proxy.Request{
				Method: "GET",
				URL:    parseURL(t, urlStr),
				Header: make(http.Header),
			},
		}

		plugin.Request(flow)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		flow.Response = &proxy.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       []byte("response " + string(rune(i))),
		}

		plugin.Response(flow)
	}

	// Save and verify
	err = plugin.SaveInventory()
	if err != nil {
		t.Fatalf("Failed to save inventory: %v", err)
	}

	// Load inventory
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory: %v", err)
	}

	var inventory types.Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to unmarshal inventory: %v", err)
	}

	if len(inventory.Resources) != len(urls) {
		t.Fatalf("Expected %d resources, got %d", len(urls), len(inventory.Resources))
	}

	// Verify each resource
	for i, resource := range inventory.Resources {
		if resource.URL != urls[i] {
			t.Errorf("Resource %d: expected URL %s, got %s", i, urls[i], resource.URL)
		}
	}
}

// Helper function to parse URL
func parseURL(t *testing.T, urlStr string) *url.URL {
	u, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("Failed to parse URL %s: %v", urlStr, err)
	}
	return u
}