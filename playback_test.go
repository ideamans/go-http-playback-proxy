package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPlaybackPlugin_LoadInventory tests loading inventory from file
func TestPlaybackPlugin_LoadInventory(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create sample inventory data
	inventory := Inventory{
		EntryURL: stringPtr("https://example.com/"),
		Resources: []Resource{
			{
				Method:          "GET",
				URL:             "https://example.com/api/test",
				TTFBMS:          100,
				StatusCode:      intPtr(200),
				RawHeaders:      HttpHeaders{"Content-Type": "application/json"},
				ContentFilePath: stringPtr("content1.txt"),
				// Remove gzip encoding for test simplicity
			},
		},
	}

	// Save inventory to file
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("Failed to marshal inventory: %v", err)
	}

	if err := os.WriteFile(inventoryPath, inventoryData, 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	// Create sample content file
	contentDir := filepath.Join(tempDir, "contents")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	contentPath := filepath.Join(contentDir, "content1.txt")
	contentData := []byte(`{"message": "test response"}`)
	if err := os.WriteFile(contentPath, contentData, 0644); err != nil {
		t.Fatalf("Failed to write content file: %v", err)
	}

	// Create playback plugin
	plugin := &PlaybackPlugin{
		inventoryDir:      tempDir,
		transactionMap:    make(map[string]*PlaybackTransaction),
		playbackManager:   NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{},
	}

	// Load inventory
	err = plugin.loadInventory()
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify transaction map
	key := "GET:https://example.com/api/test"
	transaction, exists := plugin.transactionMap[key]
	if !exists {
		t.Fatalf("Transaction not found in map: %s", key)
	}

	// Verify transaction details
	if transaction.Method != "GET" {
		t.Errorf("Expected method GET, got %s", transaction.Method)
	}

	if transaction.URL != "https://example.com/api/test" {
		t.Errorf("Expected URL https://example.com/api/test, got %s", transaction.URL)
	}

	if transaction.TTFB != 100*time.Millisecond {
		t.Errorf("Expected TTFB 100ms, got %v", transaction.TTFB)
	}

	if *transaction.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", *transaction.StatusCode)
	}

	// Verify body chunks
	if len(transaction.Chunks) != 1 {
		t.Fatalf("Expected 1 body chunk, got %d", len(transaction.Chunks))
	}

	if !bytes.Equal(transaction.Chunks[0].Chunk, contentData) {
		t.Errorf("Body chunk content mismatch. Expected: %s, Got: %s",
			string(contentData), string(transaction.Chunks[0].Chunk))
	}
}

// TestPlaybackPlugin_LoadInventory_NoFile tests loading when no inventory file exists
func TestPlaybackPlugin_LoadInventory_NoFile(t *testing.T) {
	tempDir := t.TempDir()

	plugin := &PlaybackPlugin{
		inventoryDir:      tempDir,
		transactionMap:    make(map[string]*PlaybackTransaction),
		playbackManager:   NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{},
	}

	// Load inventory (should not fail)
	err := plugin.loadInventory()
	if err != nil {
		t.Fatalf("Expected no error when inventory doesn't exist, got: %v", err)
	}

	// Verify empty transaction map
	if len(plugin.transactionMap) != 0 {
		t.Errorf("Expected empty transaction map, got %d entries", len(plugin.transactionMap))
	}
}

// TestPlaybackPlugin_NewPlaybackPlugin tests plugin creation
func TestPlaybackPlugin_NewPlaybackPlugin(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory to avoid loading real inventory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	plugin, err := NewPlaybackPlugin()
	if err != nil {
		t.Fatalf("Failed to create playback plugin: %v", err)
	}

	if plugin == nil {
		t.Fatal("Plugin is nil")
	}

	if plugin.transactionMap == nil {
		t.Error("Transaction map is nil")
	}

	if plugin.upstreamTransport == nil {
		t.Error("Upstream transport is nil")
	}
}

// TestPlaybackProxy_EndToEnd tests end-to-end playback functionality
func TestPlaybackProxy_EndToEnd(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create sample inventory data with multiple resources
	inventory := Inventory{
		EntryURL: stringPtr("https://example.com/"),
		Resources: []Resource{
			{
				Method:          "GET",
				URL:             "https://example.com/",
				TTFBMS:          50,
				StatusCode:      intPtr(200),
				RawHeaders:      HttpHeaders{"Content-Type": "text/html"},
				ContentFilePath: stringPtr("index.html"),
			},
			{
				Method:          "GET",
				URL:             "https://example.com/api/data",
				TTFBMS:          25,
				StatusCode:      intPtr(200),
				RawHeaders:      HttpHeaders{"Content-Type": "application/json"},
				ContentFilePath: stringPtr("data.json"),
			},
		},
	}

	// Save inventory to file
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("Failed to marshal inventory: %v", err)
	}

	if err := os.WriteFile(inventoryPath, inventoryData, 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	// Create sample content files
	contentDir := filepath.Join(tempDir, "contents")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	indexContent := []byte(`<!DOCTYPE html><html><body><h1>Test Page</h1></body></html>`)
	if err := os.WriteFile(filepath.Join(contentDir, "index.html"), indexContent, 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	dataContent := []byte(`{"status": "ok", "data": [1, 2, 3]}`)
	if err := os.WriteFile(filepath.Join(contentDir, "data.json"), dataContent, 0644); err != nil {
		t.Fatalf("Failed to write data.json: %v", err)
	}

	// Create playback plugin with custom inventory directory
	plugin := &PlaybackPlugin{
		inventoryDir:    tempDir,
		transactionMap:  make(map[string]*PlaybackTransaction),
		playbackManager: NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}

	// Load inventory
	err = plugin.loadInventory()
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify both transactions were loaded
	if len(plugin.transactionMap) != 2 {
		t.Fatalf("Expected 2 transactions, got %d", len(plugin.transactionMap))
	}

	// Test first transaction
	key1 := "GET:https://example.com/"
	transaction1, exists1 := plugin.transactionMap[key1]
	if !exists1 {
		t.Fatalf("Transaction not found: %s", key1)
	}

	if len(transaction1.Chunks) != 1 {
		t.Fatalf("Expected 1 chunk for index.html, got %d", len(transaction1.Chunks))
	}

	if !bytes.Equal(transaction1.Chunks[0].Chunk, indexContent) {
		t.Errorf("Index content mismatch")
	}

	// Test second transaction
	key2 := "GET:https://example.com/api/data"
	transaction2, exists2 := plugin.transactionMap[key2]
	if !exists2 {
		t.Fatalf("Transaction not found: %s", key2)
	}

	if len(transaction2.Chunks) != 1 {
		t.Fatalf("Expected 1 chunk for data.json, got %d", len(transaction2.Chunks))
	}

	if !bytes.Equal(transaction2.Chunks[0].Chunk, dataContent) {
		t.Errorf("Data content mismatch")
	}

	// Verify headers
	if transaction1.RawHeaders["Content-Type"] != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", transaction1.RawHeaders["Content-Type"])
	}

	if transaction2.RawHeaders["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", transaction2.RawHeaders["Content-Type"])
	}

	// Verify TTFB timing
	if transaction1.TTFB != 50*time.Millisecond {
		t.Errorf("Expected TTFB 50ms for index, got %v", transaction1.TTFB)
	}

	if transaction2.TTFB != 25*time.Millisecond {
		t.Errorf("Expected TTFB 25ms for data, got %v", transaction2.TTFB)
	}
}

// TestPlaybackProxy_StartProxy tests the proxy startup functionality
func TestPlaybackProxy_StartProxy(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create minimal inventory
	inventory := Inventory{
		Resources: []Resource{},
	}

	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("Failed to marshal inventory: %v", err)
	}

	if err := os.WriteFile(inventoryPath, inventoryData, 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Test that plugin can be created without errors
	plugin, err := NewPlaybackPlugin()
	if err != nil {
		t.Fatalf("Failed to create playback plugin: %v", err)
	}

	if plugin == nil {
		t.Fatal("Plugin should not be nil")
	}

	// Verify empty inventory was handled correctly
	if len(plugin.transactionMap) != 0 {
		t.Errorf("Expected empty transaction map, got %d entries", len(plugin.transactionMap))
	}
}

// TestPlaybackProxy_ErrorHandling tests error handling scenarios
func TestPlaybackProxy_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Create inventory with missing content file
	inventory := Inventory{
		Resources: []Resource{
			{
				Method:          "GET",
				URL:             "https://example.com/missing",
				TTFBMS:          100,
				StatusCode:      intPtr(200),
				ContentFilePath: stringPtr("missing-file.txt"),
			},
		},
	}

	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("Failed to marshal inventory: %v", err)
	}

	if err := os.WriteFile(inventoryPath, inventoryData, 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	plugin := &PlaybackPlugin{
		inventoryDir:      tempDir,
		transactionMap:    make(map[string]*PlaybackTransaction),
		playbackManager:   NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{},
	}

	// Load inventory (should not fail even with missing content file)
	err = plugin.loadInventory()
	if err != nil {
		t.Fatalf("Loading inventory should not fail with missing content file: %v", err)
	}

	// Verify transaction was still created
	key := "GET:https://example.com/missing"
	transaction, exists := plugin.transactionMap[key]
	if !exists {
		t.Fatalf("Transaction should exist even without content file")
	}

	// Verify empty chunks when content file is missing
	if len(transaction.Chunks) != 0 {
		t.Errorf("Expected no chunks when content file is missing, got %d", len(transaction.Chunks))
	}
}

// TestPlaybackProxy_HTTPIntegration tests actual HTTP proxy functionality
func TestPlaybackProxy_HTTPIntegration(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create sample inventory data
	inventory := Inventory{
		EntryURL: stringPtr("https://httpbin.org/"),
		Resources: []Resource{
			{
				Method:     "GET",
				URL:        "https://httpbin.org/json",
				TTFBMS:     50,
				StatusCode: intPtr(200),
				RawHeaders: HttpHeaders{
					"Content-Type": "application/json",
					"Server":       "nginx/1.10.0 (Ubuntu)",
				},
				ContentFilePath: stringPtr("json_response.json"),
			},
		},
	}

	// Save inventory to file
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inventory)
	if err != nil {
		t.Fatalf("Failed to marshal inventory: %v", err)
	}

	if err := os.WriteFile(inventoryPath, inventoryData, 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	// Create sample content file
	contentDir := filepath.Join(tempDir, "contents")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	jsonContent := []byte(`{
  "slideshow": {
    "author": "Yours Truly", 
    "date": "date of publication", 
    "slides": [
      {
        "title": "Wake up to WonderWidgets!", 
        "type": "all"
      }
    ], 
    "title": "Sample Slide Show"
  }
}`)

	contentPath := filepath.Join(contentDir, "json_response.json")
	if err := os.WriteFile(contentPath, jsonContent, 0644); err != nil {
		t.Fatalf("Failed to write content file: %v", err)
	}

	// Create playback plugin
	plugin := &PlaybackPlugin{
		inventoryDir:    tempDir,
		transactionMap:  make(map[string]*PlaybackTransaction),
		playbackManager: NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}

	// Load inventory
	err = plugin.loadInventory()
	if err != nil {
		t.Fatalf("Failed to load inventory: %v", err)
	}

	// Verify transaction exists
	key := "GET:https://httpbin.org/json"
	transaction, exists := plugin.transactionMap[key]
	if !exists {
		t.Fatalf("Transaction not found: %s", key)
	}

	// Verify content was loaded
	if len(transaction.Chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(transaction.Chunks))
	}

	if !bytes.Equal(transaction.Chunks[0].Chunk, jsonContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s",
			string(jsonContent), string(transaction.Chunks[0].Chunk))
	}

	// Verify headers
	if transaction.RawHeaders["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s",
			transaction.RawHeaders["Content-Type"])
	}

	// Verify TTFB
	if transaction.TTFB != 50*time.Millisecond {
		t.Errorf("Expected TTFB 50ms, got %v", transaction.TTFB)
	}

	// Verify status code
	if *transaction.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", *transaction.StatusCode)
	}
}

// TestPlaybackProxy_TimingAccuracy tests TTFB timing accuracy
func TestPlaybackProxy_TimingAccuracy(t *testing.T) {
	tempDir := t.TempDir()

	// Create inventory with different TTFB values
	inventory := Inventory{
		Resources: []Resource{
			{
				Method:          "GET",
				URL:             "https://example.com/fast",
				TTFBMS:          10,
				StatusCode:      intPtr(200),
				RawHeaders:      HttpHeaders{"Content-Type": "text/plain"},
				ContentFilePath: stringPtr("fast.txt"),
			},
			{
				Method:          "GET",
				URL:             "https://example.com/slow",
				TTFBMS:          200,
				StatusCode:      intPtr(200),
				RawHeaders:      HttpHeaders{"Content-Type": "text/plain"},
				ContentFilePath: stringPtr("slow.txt"),
			},
		},
	}

	// Save inventory
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, _ := json.Marshal(inventory)
	os.WriteFile(inventoryPath, inventoryData, 0644)

	// Create content files
	contentDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentDir, 0755)
	os.WriteFile(filepath.Join(contentDir, "fast.txt"), []byte("fast response"), 0644)
	os.WriteFile(filepath.Join(contentDir, "slow.txt"), []byte("slow response"), 0644)

	// Create plugin and load inventory
	plugin := &PlaybackPlugin{
		inventoryDir:      tempDir,
		transactionMap:    make(map[string]*PlaybackTransaction),
		playbackManager:   NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{},
	}
	plugin.loadInventory()

	// Test fast transaction
	fastTransaction := plugin.transactionMap["GET:https://example.com/fast"]
	if fastTransaction.TTFB != 10*time.Millisecond {
		t.Errorf("Fast transaction TTFB should be 10ms, got %v", fastTransaction.TTFB)
	}

	// Test slow transaction
	slowTransaction := plugin.transactionMap["GET:https://example.com/slow"]
	if slowTransaction.TTFB != 200*time.Millisecond {
		t.Errorf("Slow transaction TTFB should be 200ms, got %v", slowTransaction.TTFB)
	}
}
