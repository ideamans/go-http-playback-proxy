package inventory

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"go-http-playback-proxy/pkg/charset"
	"go-http-playback-proxy/pkg/encoding"
	"go-http-playback-proxy/pkg/formatting"
	"go-http-playback-proxy/pkg/resource"
	"go-http-playback-proxy/pkg/types"
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

// SaveRecordedTransactions saves RecordingTransaction to the specified directory
func (pm *PersistenceManager) SaveRecordedTransactions(
	transactions []types.RecordingTransaction,
	entryURL string,
) error {
	return pm.SaveRecordedTransactionsWithOptions(transactions, entryURL, false)
}

// SaveRecordedTransactionsWithOptions saves RecordingTransaction to the specified directory with options
func (pm *PersistenceManager) SaveRecordedTransactionsWithOptions(
	transactions []types.RecordingTransaction,
	entryURL string,
	noBeautify bool,
) error {
	// Use map to ensure unique resources by method+URL
	resourceMap := make(map[string]*types.Resource)

	// Convert each RecordingTransaction to Resource
	for _, transaction := range transactions {
		resource, err := pm.convertRecordingTransactionToResource(&transaction)
		if err != nil {
			return fmt.Errorf("failed to convert recording transaction: %w", err)
		}

		// Create unique key from method and URL
		key := fmt.Sprintf("%s:%s", resource.Method, resource.URL)

		// Check if we already have this resource
		if existingResource, exists := resourceMap[key]; exists {
			// Update existing resource if this one is newer or has more data
			if resource.Timestamp.After(existingResource.Timestamp) ||
				(resource.MBPS != nil && *resource.MBPS > 0 && (existingResource.MBPS == nil || *existingResource.MBPS == 0)) {
				resourceMap[key] = resource
			}
			// Skip saving body if we're not updating the resource
			continue
		}

		// Save decoded body to contents file and get charset information
		if resource.ContentFilePath != nil {
			contentsFilePath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
			httpCharset, contentCharset, err := pm.saveDecodedBodyWithOptions(contentsFilePath, &transaction, noBeautify)
			if err != nil {
				return fmt.Errorf("failed to save decoded body: %w", err)
			}

			// Update resource with charset information
			if httpCharset != "" {
				resource.ContentTypeCharset = &httpCharset
			}
			if contentCharset != "" {
				resource.ContentCharset = &contentCharset
			}
		}

		resourceMap[key] = resource
	}

	// Convert map to slice
	var resources []types.Resource
	for _, resource := range resourceMap {
		resources = append(resources, *resource)
	}

	// Create inventory
	inventory := types.Inventory{
		EntryURL:  &entryURL,
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
	transaction *types.RecordingTransaction,
) (*types.Resource, error) {
	// Calculate TTFB (Time To First Byte)
	var ttfbMS int64
	if !transaction.ResponseStarted.IsZero() && !transaction.RequestStarted.IsZero() {
		ttfbMS = transaction.ResponseStarted.Sub(transaction.RequestStarted).Milliseconds()
		// Sanity check: TTFB should be positive and reasonable (< 1 hour)
		if ttfbMS < 0 || ttfbMS > 3600000 {
			slog.Warn("Invalid TTFB, setting to 0", "ttfb_ms", ttfbMS)
			ttfbMS = 0
		}
	}

	// Calculate Mbps
	var mbpsValue float64
	if !transaction.ResponseStarted.IsZero() && !transaction.ResponseFinished.IsZero() {
		transferDuration := transaction.ResponseFinished.Sub(transaction.ResponseStarted)
		if transferDuration > 0 && len(transaction.Body) > 0 {
			// Convert bytes to bits, then to megabits
			totalBits := float64(len(transaction.Body) * 8)
			transferSeconds := transferDuration.Seconds()
			mbpsValue = totalBits / (transferSeconds * 1024 * 1024)
		}
	}

	// Get Content-Type details
	contentType := transaction.RawHeaders["Content-Type"]
	var contentTypeMime string
	var contentTypeCharset string
	if contentType != "" {
		// Parse Content-Type header
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err == nil {
			contentTypeMime = mediaType
			if charset, ok := params["charset"]; ok {
				contentTypeCharset = charset
			}
		} else {
			// Fallback to simple parsing
			contentTypeMime = contentType
		}
	}

	// Get Content-Encoding
	var contentEncoding *types.ContentEncodingType
	if ce := transaction.RawHeaders["Content-Encoding"]; ce != "" {
		encoding := types.ContentEncodingType(strings.ToLower(ce))
		contentEncoding = &encoding
	}

	// Determine content file path
	contentFilePath, err := resource.GetResourceFilePath(transaction.Method, transaction.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource file path: %w", err)
	}

	resource := &types.Resource{
		Method:          transaction.Method,
		URL:             transaction.URL,
		StatusCode:      transaction.StatusCode,
		ErrorMessage:    transaction.ErrorMessage,
		RawHeaders:      transaction.RawHeaders,
		TTFBMS:          ttfbMS,
		MBPS:            &mbpsValue,
		ContentEncoding: contentEncoding,
		ContentFilePath: &contentFilePath,
		Timestamp:       transaction.RequestStarted,
	}

	// Only set content type fields if they have values
	if contentTypeMime != "" {
		resource.ContentTypeMime = &contentTypeMime
	}
	if contentTypeCharset != "" {
		resource.ContentTypeCharset = &contentTypeCharset
	}

	return resource, nil
}

// AppendRecordedTransaction appends a single RecordingTransaction to an existing inventory
func (pm *PersistenceManager) AppendRecordedTransaction(transaction *types.RecordingTransaction) error {
	inventoryPath := filepath.Join(pm.BaseDir, "inventory.json")

	// Load existing inventory
	var inventory types.Inventory
	if _, err := os.Stat(inventoryPath); err == nil {
		// File exists, load it
		data, err := os.ReadFile(inventoryPath)
		if err != nil {
			return fmt.Errorf("failed to read inventory file: %w", err)
		}
		if err := json.Unmarshal(data, &inventory); err != nil {
			return fmt.Errorf("failed to unmarshal inventory: %w", err)
		}
	}

	// Convert and add the new transaction
	resource, err := pm.convertRecordingTransactionToResource(transaction)
	if err != nil {
		return fmt.Errorf("failed to convert recording transaction: %w", err)
	}

	// Create unique key from method and URL
	key := fmt.Sprintf("%s:%s", resource.Method, resource.URL)

	// Check if we already have this resource and update or add
	updated := false
	for i, existingResource := range inventory.Resources {
		existingKey := fmt.Sprintf("%s:%s", existingResource.Method, existingResource.URL)
		if existingKey == key {
			// Update existing resource if this one is newer or has more data
			if resource.Timestamp.After(existingResource.Timestamp) ||
				(resource.MBPS != nil && *resource.MBPS > 0 && (existingResource.MBPS == nil || *existingResource.MBPS == 0)) {
				inventory.Resources[i] = *resource
				updated = true
			} else {
				// Skip if existing resource is newer or has better data
				return nil
			}
			break
		}
	}

	// Save decoded body only if we're adding or updating the resource
	if resource.ContentFilePath != nil {
		contentsFilePath := filepath.Join(pm.BaseDir, "contents", *resource.ContentFilePath)
		httpCharset, contentCharset, err := pm.saveDecodedBody(contentsFilePath, transaction)
		if err != nil {
			return fmt.Errorf("failed to save decoded body: %w", err)
		}

		// Update resource with charset information
		if httpCharset != "" {
			resource.ContentTypeCharset = &httpCharset
		}
		if contentCharset != "" {
			resource.ContentCharset = &contentCharset
		}
	}

	// Add to inventory if not updated
	if !updated {
		inventory.Resources = append(inventory.Resources, *resource)
	}

	// Save updated inventory
	if err := pm.saveInventoryJSON(inventoryPath, &inventory); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	return nil
}

// saveDecodedBody saves the decoded body to a file and returns charset information
func (pm *PersistenceManager) saveDecodedBody(filePath string, transaction *types.RecordingTransaction) (httpCharset, contentCharset string, err error) {
	return pm.saveDecodedBodyWithOptions(filePath, transaction, false)
}

// saveDecodedBodyWithOptions saves the decoded body to a file with options and returns charset information
func (pm *PersistenceManager) saveDecodedBodyWithOptions(filePath string, transaction *types.RecordingTransaction, noBeautify bool) (httpCharset, contentCharset string, err error) {
	// Decode the body if it's compressed
	bodyData := transaction.Body
	if contentEncoding := transaction.RawHeaders["Content-Encoding"]; contentEncoding != "" {
		encodingType := types.ContentEncodingType(strings.ToLower(contentEncoding))

		// Only decode if it's not identity encoding
		if encodingType != types.ContentEncodingIdentity && encodingType != "" {
			decodedData, err := encoding.DecodeData(bodyData, encodingType)
			if err != nil {
				// If decoding fails, save the original data and log the error
				fmt.Printf("Warning: failed to decode %s content, saving raw data: %v\n", encodingType, err)
			} else {
				bodyData = decodedData
			}
		}
	}

	// Process charset conversion for HTML/CSS content
	contentType := transaction.RawHeaders["Content-Type"]
	processedBody, httpCharset, contentCharset, err := charset.ProcessCharsetForRecording(contentType, bodyData)
	if err != nil {
		// Log the error but continue with original body
		fmt.Printf("Warning: charset processing failed: %v\n", err)
		processedBody = bodyData
	}

	// Apply beautification if content type is appropriate and not disabled
	if !noBeautify && contentType != "" {
		optimizer := formatting.NewContentOptimizer()
		if optimizer.Accept(contentType) {
			beautified, err := optimizer.Beautify(contentType, string(processedBody))
			if err != nil {
				// Log the error but continue with original body
				fmt.Printf("Warning: beautification failed: %v\n", err)
			} else {
				processedBody = []byte(beautified)
			}
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the decoded body to file
	if err := os.WriteFile(filePath, processedBody, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}

	return httpCharset, contentCharset, nil
}

// saveInventoryJSON saves the inventory to a JSON file
func (pm *PersistenceManager) saveInventoryJSON(filePath string, inventory *types.Inventory) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal inventory to JSON
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}