package inventory

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-http-playback-proxy/pkg/charset"
	"go-http-playback-proxy/pkg/encoding"
	"go-http-playback-proxy/pkg/formatting"
	"go-http-playback-proxy/pkg/types"
)

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
func (pm *PlaybackManager) LoadPlaybackTransactions() ([]types.PlaybackTransaction, error) {
	// Load inventory.json
	inventoryPath := filepath.Join(pm.BaseDir, "inventory.json")
	inventory, err := pm.loadInventory(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}

	var transactions []types.PlaybackTransaction

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
func (pm *PlaybackManager) loadInventory(inventoryPath string) (*types.Inventory, error) {
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	var inventory types.Inventory
	err = json.Unmarshal(data, &inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inventory JSON: %w", err)
	}

	return &inventory, nil
}

// convertResourceToTransaction converts a Resource to PlaybackTransaction
func (pm *PlaybackManager) convertResourceToTransaction(resource *types.Resource) (*types.PlaybackTransaction, error) {
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

	// Update Content-Length header and charset
	rawHeaders := make(types.HttpHeaders)
	for k, v := range resource.RawHeaders {
		rawHeaders[k] = v
	}
	if len(compressedBody) > 0 {
		rawHeaders["Content-Length"] = strconv.Itoa(len(compressedBody))
	}

	// Update Content-Type header with charset if restored
	if resource.ContentCharset != nil && *resource.ContentCharset != "" && !strings.HasSuffix(*resource.ContentCharset, "-failed") {
		if contentType, exists := rawHeaders["Content-Type"]; exists {
			// Remove existing charset if present
			if idx := strings.Index(strings.ToLower(contentType), "charset="); idx != -1 {
				before := contentType[:idx]
				after := contentType[idx:]
				if semiIdx := strings.Index(after, ";"); semiIdx != -1 {
					after = after[semiIdx:]
				} else {
					after = ""
				}
				contentType = strings.TrimSpace(before) + after
			}

			// Add charset
			if !strings.HasSuffix(contentType, ";") && contentType != "" {
				contentType += "; "
			}
			contentType += fmt.Sprintf("charset=%s", *resource.ContentCharset)
			rawHeaders["Content-Type"] = contentType
		}
	}

	transaction := &types.PlaybackTransaction{
		Method:       resource.Method,
		URL:          resource.URL,
		TTFB:         time.Duration(resource.TTFBMS) * time.Millisecond,
		StatusCode:   resource.StatusCode,
		ErrorMessage: resource.ErrorMessage,
		RawHeaders:   rawHeaders,
		Chunks:       chunks,
	}

	return transaction, nil
}

// loadAndCompressContent loads content file and re-compresses it
func (pm *PlaybackManager) loadAndCompressContent(resource *types.Resource) ([]byte, error) {
	// Load the decoded content file
	contentPath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
	decodedBody, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read content file %s: %w", contentPath, err)
	}

	// Apply minify optimization if ResourceMinify is true and supported content type
	if resource.Minify != nil && *resource.Minify && resource.ContentTypeMime != nil {
		optimizer := formatting.NewContentOptimizer()
		if optimizer.Accept(*resource.ContentTypeMime) {
			minified, minifyErr := optimizer.Minify(*resource.ContentTypeMime, string(decodedBody))
			if minifyErr != nil {
				fmt.Printf("Warning: minify processing failed for %s, using original data: %v\n", resource.URL, minifyErr)
			} else {
				decodedBody = []byte(minified)
			}
		}
	}

	// Process charset restoration if needed
	if resource.ContentCharset != nil && *resource.ContentCharset != "" {
		// Create a temporary http.Header for charset processing
		headers := make(http.Header)
		if resource.ContentTypeMime != nil {
			contentType := *resource.ContentTypeMime
			if resource.ContentTypeCharset != nil && *resource.ContentTypeCharset != "" {
				contentType += "; charset=" + *resource.ContentTypeCharset
			}
			headers.Set("Content-Type", contentType)
		}

		restoredBody, err := charset.ProcessCharsetForPlayback(decodedBody, *resource.ContentCharset, headers)
		if err != nil {
			fmt.Printf("Warning: failed to restore charset for %s: %v\n", resource.URL, err)
			// Continue with UTF-8 content if restoration fails
		} else {
			decodedBody = restoredBody
		}
	}

	// If no content encoding specified, return as-is
	if resource.ContentEncoding == nil || *resource.ContentEncoding == types.ContentEncodingIdentity {
		return decodedBody, nil
	}

	// Re-compress the content using the original encoding
	compressedBody, err := encoding.EncodeData(decodedBody, *resource.ContentEncoding, 6) // Use default compression level
	if err != nil {
		return nil, fmt.Errorf("failed to re-compress content with %s: %w", *resource.ContentEncoding, err)
	}

	return compressedBody, nil
}

// createBodyChunks creates body chunks with calculated timing
func (pm *PlaybackManager) createBodyChunks(body []byte, resource *types.Resource) []types.BodyChunk {
	if len(body) == 0 {
		return []types.BodyChunk{}
	}

	var chunks []types.BodyChunk
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
		targetOffset := time.Duration(resource.TTFBMS)*time.Millisecond + chunkTime

		// For backward compatibility, also set TargetTime (will be recalculated during playback)
		targetTime := time.Now().Add(targetOffset)

		chunks = append(chunks, types.BodyChunk{
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
func (pm *PlaybackManager) compressContent(decodedBody []byte, resource *types.Resource) ([]byte, error) {
	// If no content encoding specified, return as-is
	if resource.ContentEncoding == nil || *resource.ContentEncoding == types.ContentEncodingIdentity {
		return decodedBody, nil
	}

	// Re-compress the content using the original encoding
	compressedBody, err := encoding.EncodeData(decodedBody, *resource.ContentEncoding, 6) // Use default compression level
	if err != nil {
		return nil, fmt.Errorf("failed to compress content with %s: %w", *resource.ContentEncoding, err)
	}

	return compressedBody, nil
}