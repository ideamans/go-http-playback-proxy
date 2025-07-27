package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PersistenceManager handles saving recorded resources to disk
type PersistenceManager struct {
	BaseDir string
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(baseDir string) *PersistenceManager {
	return &PersistenceManager{
		BaseDir: baseDir,
	}
}

// SaveRecordedTransactions saves RecordingTransaction and Domains to the specified directory
func (pm *PersistenceManager) SaveRecordedTransactions(
	transactions []RecordingTransaction,
	domains []Domain,
	entryURL string,
) error {
	var resources []Resource

	// Convert each RecordingTransaction to Resource
	for _, transaction := range transactions {
		resource, err := pm.convertRecordingTransactionToResource(&transaction)
		if err != nil {
			return fmt.Errorf("failed to convert recording transaction: %w", err)
		}

		// Save decoded body to contents file
		if resource.ContentFilePath != nil {
			contentsFilePath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
			err = pm.saveDecodedBody(contentsFilePath, &transaction)
			if err != nil {
				return fmt.Errorf("failed to save decoded body: %w", err)
			}
		}

		resources = append(resources, *resource)
	}

	// Create inventory
	inventory := Inventory{
		EntryURL:  &entryURL,
		Domains:   domains,
		Resources: resources,
	}

	// Save inventory.json
	inventoryPath := filepath.Join(pm.BaseDir, "inventory.json")
	err := pm.saveInventoryJSON(inventoryPath, &inventory)
	if err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	return nil
}

// convertRecordingTransactionToResource converts RecordingTransaction to Resource
func (pm *PersistenceManager) convertRecordingTransactionToResource(
	transaction *RecordingTransaction,
) (*Resource, error) {
	// Calculate TTFB (Time To First Byte)
	ttfbMs := transaction.ResponseStarted.Sub(transaction.RequestStarted).Milliseconds()

	// Get file path using resource.go functions
	relativePath, err := GetResourceFilePath(transaction.Method, transaction.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource file path: %w", err)
	}

	// Parse content type from headers
	contentType := transaction.RawHeaders["Content-Type"]
	var contentTypeMime, contentTypeCharset *string
	if contentType != "" {
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err == nil {
			contentTypeMime = &mediaType
			if charset, ok := params["charset"]; ok {
				contentTypeCharset = &charset
			}
		}
	}

	// Parse Content-Encoding
	var contentEncoding *ContentEncodingType
	if encoding := transaction.RawHeaders["Content-Encoding"]; encoding != "" {
		encodingType := ContentEncodingType(strings.ToLower(encoding))
		contentEncoding = &encodingType
	}

	// Calculate Mbps (Megabits per second) from transfer time and body size
	var mbps *float64
	if !transaction.ResponseFinished.IsZero() && len(transaction.Body) > 0 {
		transferDuration := transaction.ResponseFinished.Sub(transaction.ResponseStarted)
		if transferDuration > 0 {
			// Convert bytes to bits, then to Mbps
			totalBits := float64(len(transaction.Body) * 8)
			transferSeconds := transferDuration.Seconds()
			mbpsValue := totalBits / (transferSeconds * 1024 * 1024) // Convert to Mbps
			mbps = &mbpsValue
		}
	}

	resource := &Resource{
		Method:             transaction.Method,
		URL:                transaction.URL,
		TTFBMs:             ttfbMs,
		MBPS:               mbps,
		StatusCode:         transaction.StatusCode,
		ErrorMessage:       transaction.ErrorMessage,
		RawHeaders:         transaction.RawHeaders,
		ContentEncoding:    contentEncoding,
		ContentTypeMime:    contentTypeMime,
		ContentTypeCharset: contentTypeCharset,
		ContentFilePath:    &relativePath,
	}

	return resource, nil
}

// saveDecodedBody saves the decoded body content to the specified path
func (pm *PersistenceManager) saveDecodedBody(filePath string, transaction *RecordingTransaction) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Decode the body if it's compressed
	bodyData := transaction.Body
	if contentEncoding := transaction.RawHeaders["Content-Encoding"]; contentEncoding != "" {
		encodingType := ContentEncodingType(strings.ToLower(contentEncoding))

		// Only decode if it's not identity encoding
		if encodingType != ContentEncodingIdentity && encodingType != "" {
			decodedData, err := DecodeData(bodyData, encodingType)
			if err != nil {
				// If decoding fails, save the original data and log the error
				fmt.Printf("Warning: failed to decode %s content, saving raw data: %v\n", encodingType, err)
			} else {
				bodyData = decodedData
			}
		}
	}

	// Write the decoded body to file
	err = os.WriteFile(filePath, bodyData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// saveInventoryJSON saves the inventory as JSON
func (pm *PersistenceManager) saveInventoryJSON(filePath string, inventory *Inventory) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal inventory to JSON with indentation
	jsonData, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal inventory to JSON: %w", err)
	}

	// Write JSON to file
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write inventory file %s: %w", filePath, err)
	}

	return nil
}

// AppendRecordedTransaction appends a new transaction to an existing inventory
func (pm *PersistenceManager) AppendRecordedTransaction(
	transaction *RecordingTransaction,
	domains []Domain,
	entryURL string,
) error {
	inventoryPath := filepath.Join(pm.BaseDir, "inventory.json")

	// Load existing inventory if it exists
	var inventory Inventory
	if _, err := os.Stat(inventoryPath); err == nil {
		// File exists, load it
		data, err := os.ReadFile(inventoryPath)
		if err != nil {
			return fmt.Errorf("failed to read existing inventory: %w", err)
		}

		err = json.Unmarshal(data, &inventory)
		if err != nil {
			return fmt.Errorf("failed to parse existing inventory: %w", err)
		}
	} else {
		// File doesn't exist, create new inventory
		inventory = Inventory{
			EntryURL:  &entryURL,
			Domains:   domains,
			Resources: []Resource{},
		}
	}

	// Convert and add new resource
	resource, err := pm.convertRecordingTransactionToResource(transaction)
	if err != nil {
		return fmt.Errorf("failed to convert recording transaction: %w", err)
	}

	// Save decoded body to contents file
	if resource.ContentFilePath != nil {
		contentsFilePath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
		err = pm.saveDecodedBody(contentsFilePath, transaction)
		if err != nil {
			return fmt.Errorf("failed to save decoded body: %w", err)
		}
	}

	// Add resource to inventory
	inventory.Resources = append(inventory.Resources, *resource)

	// Merge domains (avoid duplicates)
	inventory.Domains = pm.mergeDomains(inventory.Domains, domains)

	// Save updated inventory
	err = pm.saveInventoryJSON(inventoryPath, &inventory)
	if err != nil {
		return fmt.Errorf("failed to save updated inventory: %w", err)
	}

	return nil
}

// mergeDomains merges two domain slices, avoiding duplicates
func (pm *PersistenceManager) mergeDomains(existing, new []Domain) []Domain {
	domainMap := make(map[string]Domain)

	// Add existing domains
	for _, domain := range existing {
		domainMap[domain.Name] = domain
	}

	// Add new domains (will overwrite if same name)
	for _, domain := range new {
		domainMap[domain.Name] = domain
	}

	// Convert back to slice
	result := make([]Domain, 0, len(domainMap))
	for _, domain := range domainMap {
		result = append(result, domain)
	}

	return result
}

// PlaybackManager handles generating playback transactions from inventory
type PlaybackManager struct {
	BaseDir   string
	ChunkSize int // Size of each body chunk in bytes (default: 16KB)
}

// NewPlaybackManager creates a new playback manager
func NewPlaybackManager(baseDir string) *PlaybackManager {
	return &PlaybackManager{
		BaseDir:   baseDir,
		ChunkSize: 16 * 1024, // 16KB default chunk size
	}
}

// LoadPlaybackTransactions loads inventory and generates playback transactions
func (pm *PlaybackManager) LoadPlaybackTransactions() ([]PlaybackTransaction, error) {
	// Load inventory.json
	inventoryPath := filepath.Join(pm.BaseDir, "inventory.json")
	inventory, err := pm.loadInventory(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}

	var transactions []PlaybackTransaction

	// Process each resource
	for _, resource := range inventory.Resources {
		transaction, err := pm.convertResourceToTransaction(&resource)
		if err != nil {
			fmt.Printf("Warning: failed to convert resource %s: %v\n", resource.URL, err)
			continue
		}
		transactions = append(transactions, *transaction)
	}

	return transactions, nil
}

// loadInventory loads and parses inventory.json
func (pm *PlaybackManager) loadInventory(inventoryPath string) (*Inventory, error) {
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	var inventory Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inventory JSON: %w", err)
	}

	return &inventory, nil
}

// convertResourceToTransaction converts a Resource to PlaybackTransaction
func (pm *PlaybackManager) convertResourceToTransaction(resource *Resource) (*PlaybackTransaction, error) {
	// Load content based on priority: ContentUTF8 > ContentBase64 > ContentFilePath
	var compressedBody []byte
	var err error

	if resource.ContentUTF8 != nil {
		// Use ContentUTF8 directly as decoded content
		decodedBody := []byte(*resource.ContentUTF8)
		compressedBody, err = pm.compressContent(decodedBody, resource)
		if err != nil {
			fmt.Printf("Warning: failed to compress ContentUTF8 for %s: %v\n", resource.URL, err)
			compressedBody = decodedBody // Use uncompressed if compression fails
		}
	} else if resource.ContentBase64 != nil {
		// Decode ContentBase64 and use as content
		decodedBody, err := pm.decodeBase64Content(*resource.ContentBase64)
		if err != nil {
			fmt.Printf("Warning: failed to decode ContentBase64 for %s: %v\n", resource.URL, err)
			compressedBody = []byte{}
		} else {
			compressedBody, err = pm.compressContent(decodedBody, resource)
			if err != nil {
				fmt.Printf("Warning: failed to compress ContentBase64 for %s: %v\n", resource.URL, err)
				compressedBody = decodedBody // Use uncompressed if compression fails
			}
		}
	} else if resource.ContentFilePath != nil {
		// Load from file path (existing behavior)
		compressedBody, err = pm.loadAndCompressContent(resource)
		if err != nil {
			// Log warning but continue with empty body instead of failing
			fmt.Printf("Warning: failed to load content for %s: %v\n", resource.URL, err)
			compressedBody = []byte{}
		}
	} else {
		// No content available, use empty body
		compressedBody = []byte{}
	}

	// Create chunks with timing
	chunks := pm.createBodyChunks(compressedBody, resource)

	// Update Content-Length header
	rawHeaders := make(HttpHeaders)
	for k, v := range resource.RawHeaders {
		rawHeaders[k] = v
	}
	if len(compressedBody) > 0 {
		rawHeaders["Content-Length"] = strconv.Itoa(len(compressedBody))
	}

	transaction := &PlaybackTransaction{
		Method:       resource.Method,
		URL:          resource.URL,
		TTFB:         time.Duration(resource.TTFBMs) * time.Millisecond,
		StatusCode:   resource.StatusCode,
		ErrorMessage: resource.ErrorMessage,
		RawHeaders:   rawHeaders,
		Chunks:       chunks,
	}

	return transaction, nil
}

// loadAndCompressContent loads content file and re-compresses it
func (pm *PlaybackManager) loadAndCompressContent(resource *Resource) ([]byte, error) {
	// Load the decoded content file
	contentPath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
	decodedBody, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read content file %s: %w", contentPath, err)
	}

	// If no content encoding specified, return as-is
	if resource.ContentEncoding == nil || *resource.ContentEncoding == ContentEncodingIdentity {
		return decodedBody, nil
	}

	// Re-compress the content using the original encoding
	compressedBody, err := EncodeData(decodedBody, *resource.ContentEncoding, 6) // Use default compression level
	if err != nil {
		return nil, fmt.Errorf("failed to re-compress content with %s: %w", *resource.ContentEncoding, err)
	}

	return compressedBody, nil
}

// createBodyChunks creates body chunks with calculated timing
func (pm *PlaybackManager) createBodyChunks(body []byte, resource *Resource) []BodyChunk {
	if len(body) == 0 {
		return []BodyChunk{}
	}

	var chunks []BodyChunk
	totalSize := len(body)

	// Calculate total transfer time from Mbps if available
	var totalTransferTime time.Duration
	if resource.MBPS != nil && *resource.MBPS > 0 {
		// Convert bytes to bits, then calculate time
		totalBits := float64(totalSize * 8)
		totalSeconds := totalBits / (*resource.MBPS * 1024 * 1024) // Mbps to bits per second
		totalTransferTime = time.Duration(totalSeconds * float64(time.Second))
	} else {
		// Default to 100ms total transfer time if no Mbps specified
		totalTransferTime = 100 * time.Millisecond
	}

	// Split body into chunks
	for i := 0; i < totalSize; i += pm.ChunkSize {
		end := i + pm.ChunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := body[i:end]

		// Calculate target time for this chunk
		// Time is proportional to the chunk's position in the total body
		chunkProgress := float64(end) / float64(totalSize)
		chunkTime := time.Duration(float64(totalTransferTime) * chunkProgress)

		// Target offset is TTFB + chunk time from request start
		targetOffset := time.Duration(resource.TTFBMs)*time.Millisecond + chunkTime

		// For backward compatibility, also set TargetTime (will be recalculated during playback)
		targetTime := time.Now().Add(targetOffset)

		chunks = append(chunks, BodyChunk{
			Chunk:        chunk,
			TargetTime:   targetTime,
			TargetOffset: targetOffset,
		})
	}

	return chunks
}

// SetChunkSize sets the chunk size for body chunking
func (pm *PlaybackManager) SetChunkSize(size int) {
	if size > 0 {
		pm.ChunkSize = size
	}
}

// decodeBase64Content decodes base64 content
func (pm *PlaybackManager) decodeBase64Content(base64Content string) ([]byte, error) {
	decodedData, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	return decodedData, nil
}

// compressContent compresses content based on resource's content encoding
func (pm *PlaybackManager) compressContent(decodedBody []byte, resource *Resource) ([]byte, error) {
	// If no content encoding specified, return as-is
	if resource.ContentEncoding == nil || *resource.ContentEncoding == ContentEncodingIdentity {
		return decodedBody, nil
	}

	// Compress the content using the original encoding
	compressedBody, err := EncodeData(decodedBody, *resource.ContentEncoding, 6) // Use default compression level
	if err != nil {
		return nil, fmt.Errorf("failed to compress content with %s: %w", *resource.ContentEncoding, err)
	}

	return compressedBody, nil
}
