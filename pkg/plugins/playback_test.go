package plugins

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"go-http-playback-proxy/pkg/inventory"
	"go-http-playback-proxy/pkg/testutil"
	"go-http-playback-proxy/pkg/types"
)

// TestPlaybackPlugin_LoadInventory tests loading inventory from file
func TestPlaybackPlugin_LoadInventory(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create sample inventory data
	inv := types.Inventory{
		EntryURL: testutil.StringPtr("https://example.com/"),
		Resources: []types.Resource{
			{
				Method:          "GET",
				URL:             "https://example.com/api/test",
				TTFBMS:          100,
				StatusCode:      testutil.IntPtr(200),
				RawHeaders:      types.HttpHeaders{"Content-Type": "application/json"},
				ContentFilePath: testutil.StringPtr("content1.txt"),
			},
		},
	}

	// Save inventory to file
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	inventoryData, err := json.Marshal(inv)
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
		transactionMap:    make(map[string]*types.PlaybackTransaction),
		playbackManager:   inventory.NewPlaybackManager(tempDir),
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
		t.Errorf("Expected chunk content %q, got %q", string(contentData), string(transaction.Chunks[0].Chunk))
	}
}

// TestPlaybackPlugin_NoInventory tests plugin behavior when no inventory exists
func TestPlaybackPlugin_NoInventory(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create playback plugin - no inventory exists
	plugin := &PlaybackPlugin{
		inventoryDir:      tempDir,
		transactionMap:    make(map[string]*types.PlaybackTransaction),
		playbackManager:   inventory.NewPlaybackManager(tempDir),
		upstreamTransport: &http.Transport{},
	}

	// Load inventory - should not error
	err := plugin.loadInventory()
	if err != nil {
		t.Fatalf("Expected no error when inventory doesn't exist, got: %v", err)
	}

	// Transaction map should be empty
	if len(plugin.transactionMap) != 0 {
		t.Errorf("Expected empty transaction map, got %d entries", len(plugin.transactionMap))
	}
}

