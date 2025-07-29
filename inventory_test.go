package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestPersistenceManager_SaveRecordedTransactions(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "inventory_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create persistence manager
	pm := NewPersistenceManager(tempDir)

	// Create test data
	method := "GET"
	url := "https://example.com/api/data?param=value"
	statusCode := 200

	headers := HttpHeaders{
		"Content-Type":     "application/json; charset=utf-8",
		"Content-Encoding": "gzip",
	}

	body := []byte("test body content")

	recordingTransaction := RecordingTransaction{
		Method:           method,
		URL:              url,
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(50 * time.Millisecond),
		ResponseFinished: time.Now().Add(100 * time.Millisecond),
		StatusCode:       &statusCode,
		RawHeaders:       headers,
		Body:             body,
	}

	domains := []Domain{
		{Name: "example.com", IPAddress: "192.168.1.1"},
	}

	transactions := []RecordingTransaction{recordingTransaction}

	// Test saving
	err = pm.SaveRecordedTransactions(transactions, domains, url)
	if err != nil {
		t.Fatalf("Failed to save recorded transactions: %v", err)
	}

	// Check if inventory.json was created
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		t.Fatal("inventory.json was not created")
	}

	// Check if contents file was created
	expectedPath, err := GetResourceFilePath(method, url)
	if err != nil {
		t.Fatalf("Failed to get resource file path: %v", err)
	}

	contentsPath := filepath.Join(tempDir, "contents", expectedPath)
	if _, err := os.Stat(contentsPath); os.IsNotExist(err) {
		t.Fatalf("Contents file was not created at %s", contentsPath)
	}

	// Verify contents file content
	savedContent, err := os.ReadFile(contentsPath)
	if err != nil {
		t.Fatalf("Failed to read saved content: %v", err)
	}

	if string(savedContent) != string(body) {
		t.Errorf("Saved content mismatch. Expected: %s, Got: %s", string(body), string(savedContent))
	}
}

func TestRecordingTransaction_Creation(t *testing.T) {
	// Test creating RecordingTransaction directly
	method := "GET"
	url := "https://example.com/test"
	statusCode := 200
	body := []byte("test response body")
	requestStart := time.Now()
	responseStart := requestStart.Add(50 * time.Millisecond)
	responseFinish := responseStart.Add(100 * time.Millisecond)

	headers := HttpHeaders{
		"Content-Type":     "text/html; charset=utf-8",
		"Content-Encoding": "gzip",
		"Content-Length":   "1234",
	}

	// Create RecordingTransaction
	transaction := RecordingTransaction{
		Method:           method,
		URL:              url,
		RequestStarted:   requestStart,
		ResponseStarted:  responseStart,
		ResponseFinished: responseFinish,
		StatusCode:       &statusCode,
		RawHeaders:       headers,
		Body:             body,
	}

	// Verify timing
	if !transaction.RequestStarted.Equal(requestStart) {
		t.Error("Request start time mismatch")
	}
	if !transaction.ResponseStarted.Equal(responseStart) {
		t.Error("Response start time mismatch")
	}
	if !transaction.ResponseFinished.Equal(responseFinish) {
		t.Error("Response finish time mismatch")
	}

	// Verify method and URL
	if transaction.Method != method {
		t.Error("Method mismatch")
	}
	if transaction.URL != url {
		t.Error("URL mismatch")
	}

	// Verify status code
	if transaction.StatusCode == nil || *transaction.StatusCode != statusCode {
		t.Error("Status code mismatch")
	}

	// Verify headers
	if transaction.RawHeaders["Content-Type"] != "text/html; charset=utf-8" {
		t.Error("Content-Type header mismatch")
	}
	if transaction.RawHeaders["Content-Encoding"] != "gzip" {
		t.Error("Content-Encoding header mismatch")
	}

	// Verify body
	if string(transaction.Body) != string(body) {
		t.Error("Body content mismatch")
	}
}

func TestPersistenceManager_AppendRecordedTransaction(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "inventory_append_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPersistenceManager(tempDir)

	// First transaction
	statusCode1 := 200
	transaction1 := RecordingTransaction{
		Method:           "GET",
		URL:              "https://example.com/page1",
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(50 * time.Millisecond),
		ResponseFinished: time.Now().Add(100 * time.Millisecond),
		StatusCode:       &statusCode1,
		RawHeaders: HttpHeaders{
			"Content-Type": "text/html",
		},
		Body: []byte("page1 content"),
	}
	domains1 := []Domain{{Name: "example.com", IPAddress: "192.168.1.1"}}

	// Second transaction
	statusCode2 := 200
	transaction2 := RecordingTransaction{
		Method:           "GET",
		URL:              "https://example.com/page2",
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(30 * time.Millisecond),
		ResponseFinished: time.Now().Add(80 * time.Millisecond),
		StatusCode:       &statusCode2,
		RawHeaders: HttpHeaders{
			"Content-Type": "application/json",
		},
		Body: []byte("page2 content"),
	}
	domains2 := []Domain{{Name: "api.example.com", IPAddress: "192.168.1.2"}}

	// Append first transaction
	err = pm.AppendRecordedTransaction(&transaction1, domains1, "https://example.com/page1")
	if err != nil {
		t.Fatalf("Failed to append first transaction: %v", err)
	}

	// Append second transaction
	err = pm.AppendRecordedTransaction(&transaction2, domains2, "https://example.com/page2")
	if err != nil {
		t.Fatalf("Failed to append second transaction: %v", err)
	}

	// Check inventory contains both resources
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory: %v", err)
	}

	// Basic checks that both URLs are in the JSON
	inventoryContent := string(data)
	if !contains(inventoryContent, "page1") {
		t.Error("First resource not found in inventory")
	}
	if !contains(inventoryContent, "page2") {
		t.Error("Second resource not found in inventory")
	}
	if !contains(inventoryContent, "example.com") {
		t.Error("First domain not found in inventory")
	}
	if !contains(inventoryContent, "api.example.com") {
		t.Error("Second domain not found in inventory")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}

func TestPlaybackManager_LoadPlaybackTransactions(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "playback_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test content
	testContent := "Hello, World! This is test content for playback."

	// Create persistence manager and save test data
	pm := NewPersistenceManager(tempDir)

	method := "GET"
	url := "https://example.com/test"
	mbps := 10.0 // 10 Mbps
	statusCode := 200

	headers := HttpHeaders{
		"Content-Type":     "text/plain",
		"Content-Encoding": "gzip",
	}

	// Encode the test content with gzip
	encoder, err := CreateEncoder(ContentEncodingGzip, 6)
	if err != nil {
		t.Fatalf("Failed to create gzip encoder: %v", err)
	}

	encodedContent, err := encoder.Encode([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to encode content: %v", err)
	}

	recordingTransaction := RecordingTransaction{
		Method:           method,
		URL:              url,
		RequestStarted:   time.Now(),
		ResponseStarted:  time.Now().Add(50 * time.Millisecond),
		ResponseFinished: time.Now().Add(100 * time.Millisecond),
		StatusCode:       &statusCode,
		RawHeaders:       headers,
		Body:             encodedContent,
	}

	domains := []Domain{
		{Name: "example.com", IPAddress: "192.168.1.1"},
	}

	transactions := []RecordingTransaction{recordingTransaction}

	// Save the recording transactions
	err = pm.SaveRecordedTransactions(transactions, domains, url)
	if err != nil {
		t.Fatalf("Failed to save recorded resources: %v", err)
	}

	// Load inventory and update resource with Mbps
	inventoryPath := filepath.Join(tempDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatalf("Failed to read inventory: %v", err)
	}

	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		t.Fatalf("Failed to parse inventory: %v", err)
	}

	// Add Mbps to the resource
	if len(inventory.Resources) > 0 {
		inventory.Resources[0].MBPS = &mbps
	}

	// Save updated inventory
	updatedData, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal updated inventory: %v", err)
	}

	err = os.WriteFile(inventoryPath, updatedData, 0644)
	if err != nil {
		t.Fatalf("Failed to write updated inventory: %v", err)
	}

	// Now test playback loading
	playbackManager := NewPlaybackManager(tempDir)
	playbackTransactions, err := playbackManager.LoadPlaybackTransactions()
	if err != nil {
		t.Fatalf("Failed to load playback transactions: %v", err)
	}

	if len(playbackTransactions) != 1 {
		t.Fatalf("Expected 1 transaction, got %d", len(playbackTransactions))
	}

	transaction := playbackTransactions[0]

	// Verify basic properties
	if transaction.Method != method {
		t.Errorf("Method mismatch. Expected: %s, Got: %s", method, transaction.Method)
	}
	if transaction.URL != url {
		t.Errorf("URL mismatch. Expected: %s, Got: %s", url, transaction.URL)
	}
	if transaction.TTFB != 50*time.Millisecond {
		t.Errorf("TTFB mismatch. Expected: %v, Got: %v", 50*time.Millisecond, transaction.TTFB)
	}

	// Verify chunks
	if len(transaction.Chunks) == 0 {
		t.Fatal("No chunks generated")
	}

	// Verify total body size matches re-compressed content
	totalSize := 0
	for _, chunk := range transaction.Chunks {
		totalSize += len(chunk.Chunk)
	}

	if totalSize != len(encodedContent) {
		t.Errorf("Total chunk size mismatch. Expected: %d, Got: %d", len(encodedContent), totalSize)
	}

	// Verify Content-Length header was updated
	contentLength := transaction.RawHeaders["Content-Length"]
	expectedLength := strconv.Itoa(len(encodedContent))
	if contentLength != expectedLength {
		t.Errorf("Content-Length header mismatch. Expected: %s, Got: %s", expectedLength, contentLength)
	}

	// Verify chunk timing (should be increasing)
	for i := 1; i < len(transaction.Chunks); i++ {
		if !transaction.Chunks[i].TargetTime.After(transaction.Chunks[i-1].TargetTime) {
			t.Error("Chunk target times should be increasing")
		}
	}
}

func TestPlaybackManager_ChunkCreation(t *testing.T) {
	pm := NewPlaybackManager("")
	pm.SetChunkSize(10) // 10 bytes per chunk for testing

	// Create test resource
	mbps := 8.0 // 8 Mbps
	resource := &Resource{
		TTFBMS: 100, // 100ms TTFB
		MBPS:   &mbps,
	}

	// Test body
	testBody := []byte("This is a test body content!")
	t.Logf("Test body length: %d", len(testBody))

	chunks := pm.createBodyChunks(testBody, resource)
	t.Logf("Number of chunks: %d", len(chunks))

	// Verify total size matches
	totalSize := 0
	for i, chunk := range chunks {
		t.Logf("Chunk %d size: %d", i, len(chunk.Chunk))
		totalSize += len(chunk.Chunk)
	}

	if totalSize != len(testBody) {
		t.Errorf("Total chunk size mismatch. Expected: %d, Got: %d", len(testBody), totalSize)
	}

	// Verify chunks are reasonable size (at most chunkSize)
	for i, chunk := range chunks {
		if len(chunk.Chunk) > pm.ChunkSize {
			t.Errorf("Chunk %d too large. Expected <= %d, Got: %d", i, pm.ChunkSize, len(chunk.Chunk))
		}
		if len(chunk.Chunk) == 0 {
			t.Errorf("Chunk %d is empty", i)
		}
	}

	// Verify timing progression
	for i := 1; i < len(chunks); i++ {
		if !chunks[i].TargetTime.After(chunks[i-1].TargetTime) {
			t.Error("Chunk target times should be increasing")
		}
	}
}

func TestPlaybackManager_ContentUTF8(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "content_utf8_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Test ContentUTF8 priority over ContentFilePath
	utf8Content := "Hello, 世界! This is UTF-8 content."
	resource := &Resource{
		Method:          "GET",
		URL:             "https://example.com/utf8",
		TTFBMS:          100,
		ContentUTF8:     &utf8Content,
		ContentFilePath: stringPtr("should-not-be-used.txt"), // This should be ignored
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with ContentUTF8: %v", err)
	}

	// Verify content
	if len(transaction.Chunks) == 0 {
		t.Fatalf("Expected chunks to be created from ContentUTF8")
	}

	// Reconstruct body from chunks
	var reconstructed []byte
	for _, chunk := range transaction.Chunks {
		reconstructed = append(reconstructed, chunk.Chunk...)
	}

	if string(reconstructed) != utf8Content {
		t.Errorf("ContentUTF8 content mismatch. Expected: %q, Got: %q", utf8Content, string(reconstructed))
	}
}

func TestPlaybackManager_ContentBase64(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "content_base64_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Test ContentBase64 when ContentUTF8 is not available
	originalContent := "Binary content that was base64 encoded"
	base64Content := base64.StdEncoding.EncodeToString([]byte(originalContent))

	resource := &Resource{
		Method:          "GET",
		URL:             "https://example.com/base64",
		TTFBMS:          100,
		ContentBase64:   &base64Content,
		ContentFilePath: stringPtr("should-not-be-used.txt"), // This should be ignored
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with ContentBase64: %v", err)
	}

	// Verify content
	if len(transaction.Chunks) == 0 {
		t.Fatalf("Expected chunks to be created from ContentBase64")
	}

	// Reconstruct body from chunks
	var reconstructed []byte
	for _, chunk := range transaction.Chunks {
		reconstructed = append(reconstructed, chunk.Chunk...)
	}

	if string(reconstructed) != originalContent {
		t.Errorf("ContentBase64 content mismatch. Expected: %q, Got: %q", originalContent, string(reconstructed))
	}
}

func TestPlaybackManager_ContentPriority(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "content_priority_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Create a file content that should NOT be used
	contentDir := filepath.Join(tempDir, "contents")
	os.MkdirAll(contentDir, 0755)
	fileContent := "This is file content that should be ignored"
	contentPath := filepath.Join(contentDir, "test.txt")
	os.WriteFile(contentPath, []byte(fileContent), 0644)

	// Test priority: ContentUTF8 > ContentBase64 > ContentFilePath
	utf8Content := "UTF-8 has highest priority"
	base64Content := base64.StdEncoding.EncodeToString([]byte("Base64 content"))

	resource := &Resource{
		Method:          "GET",
		URL:             "https://example.com/priority",
		TTFBMS:          100,
		ContentUTF8:     &utf8Content,          // Highest priority
		ContentBase64:   &base64Content,        // Should be ignored
		ContentFilePath: stringPtr("test.txt"), // Should be ignored
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with all content types: %v", err)
	}

	// Verify ContentUTF8 was used
	var reconstructed []byte
	for _, chunk := range transaction.Chunks {
		reconstructed = append(reconstructed, chunk.Chunk...)
	}

	if string(reconstructed) != utf8Content {
		t.Errorf("Expected ContentUTF8 to have priority. Expected: %q, Got: %q", utf8Content, string(reconstructed))
	}
}

func TestPlaybackManager_EmptyContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "empty_content_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Test with no content fields
	resource := &Resource{
		Method: "GET",
		URL:    "https://example.com/empty",
		TTFBMS: 100,
		// No ContentUTF8, ContentBase64, or ContentFilePath
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with no content: %v", err)
	}

	// Verify empty content results in empty chunks
	if len(transaction.Chunks) != 0 {
		t.Errorf("Expected no chunks for empty content, got %d chunks", len(transaction.Chunks))
	}
}

func TestPlaybackManager_InvalidBase64(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "invalid_base64_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Test with invalid base64 content
	invalidBase64 := "This is not valid base64!!!"

	resource := &Resource{
		Method:        "GET",
		URL:           "https://example.com/invalid-base64",
		TTFBMS:        100,
		ContentBase64: &invalidBase64,
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with invalid base64: %v", err)
	}

	// Should fall back to empty content
	if len(transaction.Chunks) != 0 {
		t.Errorf("Expected no chunks for invalid base64, got %d chunks", len(transaction.Chunks))
	}
}

func TestPlaybackManager_ContentCompression(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "content_compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pm := NewPlaybackManager(tempDir)

	// Test ContentUTF8 with gzip compression
	utf8Content := "This content should be gzip compressed"
	gzipEncoding := ContentEncodingGzip

	resource := &Resource{
		Method:          "GET",
		URL:             "https://example.com/compressed",
		TTFBMS:          100,
		ContentUTF8:     &utf8Content,
		ContentEncoding: &gzipEncoding,
	}

	transaction, err := pm.convertResourceToTransaction(resource)
	if err != nil {
		t.Fatalf("Failed to convert resource with compression: %v", err)
	}

	// Verify chunks were created
	if len(transaction.Chunks) == 0 {
		t.Fatalf("Expected chunks to be created")
	}

	// Reconstruct compressed body
	var compressedBody []byte
	for _, chunk := range transaction.Chunks {
		compressedBody = append(compressedBody, chunk.Chunk...)
	}

	// Decompress and verify
	decoder, err := CreateDecoder(ContentEncodingGzip)
	if err != nil {
		t.Fatalf("Failed to create gzip decoder: %v", err)
	}

	decompressedBody, err := decoder.Decode(compressedBody)
	if err != nil {
		t.Fatalf("Failed to decompress body: %v", err)
	}

	if string(decompressedBody) != utf8Content {
		t.Errorf("Decompressed content mismatch. Expected: %q, Got: %q", utf8Content, string(decompressedBody))
	}
}
