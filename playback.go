package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// PlaybackPlugin handles playback mode functionality
type PlaybackPlugin struct {
	BaseLogPlugin
	inventoryDir      string
	transactionMap    map[string]*PlaybackTransaction
	upstreamTransport *http.Transport
	playbackManager   *PlaybackManager
	mutex             sync.RWMutex
}

// NewPlaybackPlugin creates a new playback plugin
func NewPlaybackPlugin() (*PlaybackPlugin, error) {
	return NewPlaybackPluginWithInventoryDir("./inventory")
}

// NewPlaybackPluginWithInventoryDir creates a new playback plugin with custom inventory directory
func NewPlaybackPluginWithInventoryDir(inventoryDir string) (*PlaybackPlugin, error) {
	plugin := &PlaybackPlugin{
		inventoryDir:   inventoryDir,
		transactionMap: make(map[string]*PlaybackTransaction),
		playbackManager: NewPlaybackManager(inventoryDir),
		upstreamTransport: &http.Transport{
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true, // 圧縮を無効化してオリジナルの状態を保持
		},
	}

	if err := plugin.loadInventory(); err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}

	return plugin, nil
}

// loadInventory loads the inventory and creates the transaction map
func (p *PlaybackPlugin) loadInventory() error {
	inventoryPath := filepath.Join(p.inventoryDir, "inventory.json")
	
	// Check if inventory exists
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		log.Printf("[PLAYBACK] No inventory found at %s, will proxy all requests upstream", inventoryPath)
		return nil
	}

	// Load transactions using PlaybackManager (handles proper chunking)
	transactions, err := p.playbackManager.LoadPlaybackTransactions()
	if err != nil {
		return fmt.Errorf("failed to load playback transactions: %w", err)
	}

	log.Printf("[PLAYBACK] PlaybackManager loaded %d transactions", len(transactions))

	// Convert transactions to map for fast lookup
	for _, transaction := range transactions {
		key := fmt.Sprintf("%s:%s", transaction.Method, transaction.URL)
		
		// Check for duplicate keys
		if _, exists := p.transactionMap[key]; exists {
			log.Printf("[PLAYBACK] WARNING: Duplicate key detected: %s", key)
		}
		
		// Create a copy to store in the map
		transactionCopy := transaction
		p.transactionMap[key] = &transactionCopy
	}

	// Check for specific URL
	gtmKey := "GET:https://www.googletagmanager.com/gtag/js?id=G-VDRYPM3MEG"
	if transaction, exists := p.transactionMap[gtmKey]; exists {
		log.Printf("[PLAYBACK] ✓ Google Tag Manager found: %d chunks", len(transaction.Chunks))
	} else {
		log.Printf("[PLAYBACK] ✗ Google Tag Manager NOT found in transaction map")
	}

	log.Printf("[PLAYBACK] Loaded %d transactions from inventory", len(p.transactionMap))
	return nil
}


func (p *PlaybackPlugin) Request(f *proxy.Flow) {
	p.BaseLogPlugin.Request(f)

	if f.Request == nil {
		return
	}

	key := fmt.Sprintf("%s:%s", f.Request.Method, f.Request.URL.String())
	
	p.mutex.RLock()
	transaction, exists := p.transactionMap[key]
	p.mutex.RUnlock()

	if exists {
		log.Printf("[PLAYBACK] Found matching transaction for: %s", key)
		// Playback from recorded transaction
		p.playbackTransaction(f, transaction)
	} else {
		log.Printf("[PLAYBACK] No matching transaction for: %s, proxying upstream", key)
		// Also log some available keys for debugging
		p.mutex.RLock()
		count := 0
		for availableKey := range p.transactionMap {
			if count < 3 { // Show first 3 keys for debugging
				log.Printf("[PLAYBACK] Available key: %s", availableKey)
				count++
			}
		}
		p.mutex.RUnlock()
		// Proxy to upstream server
		p.proxyUpstream(f)
	}
}

// playbackTransaction replays a recorded transaction with timing control
func (p *PlaybackPlugin) playbackTransaction(f *proxy.Flow, transaction *PlaybackTransaction) {
	startTime := time.Now()
	
	log.Printf("[PLAYBACK] Replaying: %s %s (TTFB: %v)", 
		transaction.Method, transaction.URL, transaction.TTFB)

	// Create response
	response := &proxy.Response{
		StatusCode: 200, // Default status code
		Header:     make(http.Header),
	}

	if transaction.StatusCode != nil {
		response.StatusCode = *transaction.StatusCode
	}

	// Set headers
	for name, value := range transaction.RawHeaders {
		response.Header.Set(name, value)
	}

	// Add playback indicator header
	response.Header.Set("x-playback-proxy", "1")

	// Handle response body with timing
	if len(transaction.Chunks) > 0 {
		// Process chunks with timing consideration (TTFB timing is handled per chunk)
		var bodyBuffer bytes.Buffer
		requestStartTime := startTime // リクエスト開始時刻
		
		for i, chunk := range transaction.Chunks {
			// Calculate when this chunk should be sent based on request start time
			var targetSendTime time.Time
			if chunk.TargetOffset > 0 {
				// Use TargetOffset for precise timing from request start
				targetSendTime = requestStartTime.Add(chunk.TargetOffset)
			} else {
				// Fallback: use TTFB for first chunk, or proportional timing for others
				if i == 0 {
					targetSendTime = requestStartTime.Add(transaction.TTFB)
				} else {
					// For backward compatibility, calculate proportional timing
					proportionalDelay := transaction.TTFB + time.Duration(i)*50*time.Millisecond
					targetSendTime = requestStartTime.Add(proportionalDelay)
				}
			}
			
			// Check if we need to wait
			now := time.Now()
			if now.Before(targetSendTime) {
				waitTime := targetSendTime.Sub(now)
				log.Printf("[PLAYBACK] Waiting %v for chunk %d/%d of %s (offset: %v)", 
					waitTime, i+1, len(transaction.Chunks), transaction.URL, chunk.TargetOffset)
				time.Sleep(waitTime)
			} else {
				log.Printf("[PLAYBACK] Target time already passed for chunk %d/%d of %s (behind by %v, offset: %v)", 
					i+1, len(transaction.Chunks), transaction.URL, now.Sub(targetSendTime), chunk.TargetOffset)
			}
			
			// Add chunk to body buffer
			bodyBuffer.Write(chunk.Chunk)
		}

		response.Body = bodyBuffer.Bytes()
		log.Printf("[PLAYBACK] Combined %d chunks into %d bytes for %s", 
			len(transaction.Chunks), bodyBuffer.Len(), transaction.URL)
	} else {
		response.Body = []byte{}
	}

	// Set the response
	f.Response = response

	elapsed := time.Since(startTime)
	log.Printf("[PLAYBACK] Completed replay: %s %s (took %v)", 
		transaction.Method, transaction.URL, elapsed)
}

// proxyUpstream forwards the request to the upstream server
func (p *PlaybackPlugin) proxyUpstream(f *proxy.Flow) {
	log.Printf("[PLAYBACK] Proxying upstream: %s %s", f.Request.Method, f.Request.URL.String())

	// Create HTTP client with our transport
	client := &http.Client{
		Transport: p.upstreamTransport,
		Timeout:   30 * time.Second,
	}

	// Create request body reader
	var bodyReader io.Reader
	if len(f.Request.Body) > 0 {
		bodyReader = bytes.NewReader(f.Request.Body)
	}

	// Create request
	req, err := http.NewRequest(f.Request.Method, f.Request.URL.String(), bodyReader)
	if err != nil {
		p.createErrorResponse(f, 500, fmt.Sprintf("Failed to create upstream request: %v", err))
		return
	}

	// Copy headers
	for name, values := range f.Request.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		p.createErrorResponse(f, 502, fmt.Sprintf("Upstream request failed: %v", err))
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		p.createErrorResponse(f, 502, fmt.Sprintf("Failed to read upstream response: %v", err))
		return
	}

	// Create proxy response
	response := &proxy.Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}

	// Set response
	f.Response = response
	
	log.Printf("[PLAYBACK] Upstream response: %s %s %d", 
		f.Request.Method, f.Request.URL.String(), resp.StatusCode)
}

// createErrorResponse creates an error response
func (p *PlaybackPlugin) createErrorResponse(f *proxy.Flow, statusCode int, message string) {
	response := &proxy.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       []byte(message),
	}

	response.Header.Set("Content-Type", "text/plain")
	f.Response = response

	log.Printf("[PLAYBACK] Error response: %d %s", statusCode, message)
}

// StartPlayback starts the playback mode proxy
func StartPlayback(port int, inventoryDir string) error {
	// Create playback plugin
	playbackPlugin, err := NewPlaybackPluginWithInventoryDir(inventoryDir)
	if err != nil {
		return fmt.Errorf("failed to create playback plugin: %w", err)
	}

	// Create proxy
	p, err := createProxy(port)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	// Add playback plugin
	p.AddAddon(playbackPlugin)

	// Start proxy
	startProxyWithShutdown(p, port)
	return nil
}