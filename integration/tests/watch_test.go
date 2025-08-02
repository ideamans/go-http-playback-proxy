package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-http-playback-proxy/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlaybackWatchMode tests the watch mode functionality
func TestPlaybackWatchMode(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	tempDir := t.TempDir()
	inventoryDir := filepath.Join(tempDir, "inventory")
	contentsDir := filepath.Join(inventoryDir, "contents")
	require.NoError(t, os.MkdirAll(contentsDir, 0755))

	// Create initial inventory
	statusCode := 200
	mbps := 10.0
	contentType := "text/plain"
	contentFilePath := "get/http/example.com/test/index.html"
	inventory := &types.Inventory{
		Resources: []types.Resource{
			{
				Method:          "GET",
				URL:             "http://example.com/test",
				TTFBMS:          100,
				MBPS:            &mbps,
				StatusCode:      &statusCode,
				ContentTypeMime: &contentType,
				ContentFilePath: &contentFilePath,
			},
		},
	}

	// Write initial inventory
	inventoryPath := filepath.Join(inventoryDir, "inventory.json")
	data, err := json.MarshalIndent(inventory, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inventoryPath, data, 0644))

	// Write initial content
	contentPath := filepath.Join(contentsDir, "get", "http", "example.com", "test", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(contentPath), 0755))
	require.NoError(t, os.WriteFile(contentPath, []byte("Initial content"), 0644))

	// Start playback proxy with watch mode
	proxyPort := 10050
	proxyAddr := fmt.Sprintf("localhost:%d", proxyPort)
	cmd := startPlaybackProxyWithArgs(t, "--watch", "-i", inventoryDir, "-p", fmt.Sprintf("%d", proxyPort))
	defer stopProxy(cmd)

	// Give proxy time to start and initial load
	time.Sleep(2 * time.Second)

	// Test initial response
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   proxyAddr,
			}),
		},
		Timeout: 5 * time.Second,
	}

	// Make initial request
	resp, err := client.Get("http://example.com/test")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Initial content", string(body))
	assert.Equal(t, 200, resp.StatusCode)

	// Update content file
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, os.WriteFile(contentPath, []byte("Updated content"), 0644))
	
	// Wait for file watcher to detect change and reload
	time.Sleep(1 * time.Second)

	// Test updated response
	resp, err = client.Get("http://example.com/test")
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Updated content", string(body))

	// Update inventory to add new resource
	statusCode2 := 201
	mbps2 := 20.0
	contentType2 := "application/json"
	contentFilePath2 := "get/http/example.com/new/index.html"
	inventory.Resources = append(inventory.Resources, types.Resource{
		Method:          "GET",
		URL:             "http://example.com/new",
		TTFBMS:          50,
		MBPS:            &mbps2,
		StatusCode:      &statusCode2,
		ContentTypeMime: &contentType2,
		ContentFilePath: &contentFilePath2,
	})

	data, err = json.MarshalIndent(inventory, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inventoryPath, data, 0644))

	// Write content for new resource
	newContentPath := filepath.Join(contentsDir, "get", "http", "example.com", "new", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(newContentPath), 0755))
	require.NoError(t, os.WriteFile(newContentPath, []byte(`{"message":"new resource"}`), 0644))

	// Wait for reload
	time.Sleep(1 * time.Second)

	// Test new resource
	resp, err = client.Get("http://example.com/new")
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, `{"message":"new resource"}`, string(body))
	assert.Equal(t, 201, resp.StatusCode)

	// Test rapid sequential updates
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("Rapid update %d", i)
		require.NoError(t, os.WriteFile(contentPath, []byte(content), 0644))
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for final reload
	time.Sleep(1 * time.Second)

	// Test final state
	resp, err = client.Get("http://example.com/test")
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Rapid update 4", string(body))
}

// TestPlaybackWatchModeWithDelete tests watch mode handling of file deletion
func TestPlaybackWatchModeWithDelete(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	tempDir := t.TempDir()
	inventoryDir := filepath.Join(tempDir, "inventory")
	contentsDir := filepath.Join(inventoryDir, "contents")
	require.NoError(t, os.MkdirAll(contentsDir, 0755))

	// Create initial inventory with two resources
	statusCode := 200
	mbps := 10.0
	contentType := "text/plain"
	contentFilePath1 := "get/http/example.com/resource1/index.html"
	contentFilePath2 := "get/http/example.com/resource2/index.html"
	inventory := &types.Inventory{
		Resources: []types.Resource{
			{
				Method:          "GET",
				URL:             "http://example.com/resource1",
				TTFBMS:          100,
				MBPS:            &mbps,
				StatusCode:      &statusCode,
				ContentTypeMime: &contentType,
				ContentFilePath: &contentFilePath1,
			},
			{
				Method:          "GET",
				URL:             "http://example.com/resource2",
				TTFBMS:          100,
				MBPS:            &mbps,
				StatusCode:      &statusCode,
				ContentTypeMime: &contentType,
				ContentFilePath: &contentFilePath2,
			},
		},
	}

	// Write initial inventory
	inventoryPath := filepath.Join(inventoryDir, "inventory.json")
	data, err := json.MarshalIndent(inventory, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inventoryPath, data, 0644))

	// Write content for both resources
	content1Path := filepath.Join(contentsDir, "get", "http", "example.com", "resource1", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(content1Path), 0755))
	require.NoError(t, os.WriteFile(content1Path, []byte("Resource 1 content"), 0644))

	content2Path := filepath.Join(contentsDir, "get", "http", "example.com", "resource2", "index.html")
	require.NoError(t, os.MkdirAll(filepath.Dir(content2Path), 0755))
	require.NoError(t, os.WriteFile(content2Path, []byte("Resource 2 content"), 0644))

	// Start playback proxy with watch mode
	proxyPort := 10051
	proxyAddr := fmt.Sprintf("localhost:%d", proxyPort)
	cmd := startPlaybackProxyWithArgs(t, "--watch", "-i", inventoryDir, "-p", fmt.Sprintf("%d", proxyPort))
	defer stopProxy(cmd)

	// Give proxy time to start
	time.Sleep(2 * time.Second)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   proxyAddr,
			}),
		},
		Timeout: 5 * time.Second,
	}

	// Test both resources are available
	resp, err := client.Get("http://example.com/resource1")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = client.Get("http://example.com/resource2")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// Remove resource2 from inventory
	inventory.Resources = inventory.Resources[:1]
	data, err = json.MarshalIndent(inventory, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(inventoryPath, data, 0644))

	// Wait for reload
	time.Sleep(1 * time.Second)

	// Test resource1 is still available
	resp, err = client.Get("http://example.com/resource1")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// Test resource2 should now proxy upstream
	// Since there's no upstream server, we expect either a connection error, 404, or 502
	resp, err = client.Get("http://example.com/resource2")
	if err == nil {
		resp.Body.Close()
		// If we get a response, it should be 404 (Not Found) or 502 (Bad Gateway)
		assert.True(t, resp.StatusCode == 404 || resp.StatusCode == 502, 
			"Expected status 404 or 502, got %d", resp.StatusCode)
	} else {
		// Connection error is also acceptable since there's no upstream
		assert.Contains(t, err.Error(), "deadline exceeded")
	}
}