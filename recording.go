package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// RecordingPlugin handles recording mode functionality
type RecordingPlugin struct {
	BaseLogPlugin
	targetURL    string
	targetDomain string
	transactions []RecordingTransaction
	mutex        sync.RWMutex
	inventoryDir string
	noBeautify   bool
}

// RecordingTransaction represents a transaction being recorded
// NewRecordingPlugin creates a new recording plugin
func NewRecordingPlugin(targetURL string) (*RecordingPlugin, error) {
	return NewRecordingPluginWithInventoryDir(targetURL, "./inventory", false)
}

// NewRecordingPluginWithInventoryDir creates a new recording plugin with custom inventory directory
func NewRecordingPluginWithInventoryDir(targetURL string, inventoryDir string, noBeautify bool) (*RecordingPlugin, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	plugin := &RecordingPlugin{
		targetURL:    targetURL,
		targetDomain: parsedURL.Host,
		transactions: make([]RecordingTransaction, 0),
		inventoryDir: inventoryDir,
		noBeautify:   noBeautify,
	}

	// Create inventory directory if it doesn't exist
	if err := os.MkdirAll(plugin.inventoryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create inventory directory: %w", err)
	}

	return plugin, nil
}

func (p *RecordingPlugin) ServerConnected(connCtx *proxy.ConnContext) {
	p.BaseLogPlugin.ServerConnected(connCtx)
}


func (p *RecordingPlugin) Request(f *proxy.Flow) {
	p.BaseLogPlugin.Request(f)

	if f != nil && f.Request != nil {
		// Start recording transaction
		transaction := RecordingTransaction{
			Method:         f.Request.Method,
			URL:            f.Request.URL.String(),
			RequestStarted: time.Now(),
			RawHeaders:     make(HttpHeaders),
		}

		// Store transaction for later retrieval
		p.mutex.Lock()
		if len(p.transactions) < 10000 { // Prevent memory issues
			p.transactions = append(p.transactions, transaction)
		}
		p.mutex.Unlock()
	}
}

func (p *RecordingPlugin) Response(f *proxy.Flow) {
	p.BaseLogPlugin.Response(f)

	if f != nil && f.Response != nil && f.Request != nil {
		// Find the most recent transaction for this request
		p.mutex.Lock()
		for i := len(p.transactions) - 1; i >= 0; i-- {
			transaction := &p.transactions[i]
			if transaction.Method == f.Request.Method && transaction.URL == f.Request.URL.String() && transaction.ResponseStarted.IsZero() {
				responseStartTime := time.Now()
				transaction.ResponseStarted = responseStartTime

				// Record response details
				transaction.StatusCode = &f.Response.StatusCode

				// Copy headers
				for name, values := range f.Response.Header {
					if len(values) > 0 {
						transaction.RawHeaders[name] = values[0]
					}
				}

				// Record response body
				if len(f.Response.Body) > 0 {
					transaction.Body = make([]byte, len(f.Response.Body))
					copy(transaction.Body, f.Response.Body)
				}

				// Calculate realistic ResponseFinished time based on actual response duration
				// For realistic timing, use the duration from RequestStarted to now
				actualDuration := time.Since(transaction.RequestStarted)
				if actualDuration > 100*time.Millisecond {
					// If actual duration is significant, use it for realistic timing
					transaction.ResponseFinished = responseStartTime.Add(actualDuration - 50*time.Millisecond)
				} else {
					// For very fast responses, add a minimal realistic delay
					transaction.ResponseFinished = responseStartTime.Add(10 * time.Millisecond)
				}

				slog.Debug("Recorded transaction",
					"method", transaction.Method,
					"url", transaction.URL,
					"bytes", len(transaction.Body))
				break
			}
		}
		p.mutex.Unlock()
	}
}

// SaveInventory saves the recorded data to the inventory directory
func (p *RecordingPlugin) SaveInventory() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// Use PersistenceManager to save transactions
	persistenceManager := NewPersistenceManager(p.inventoryDir)
	
	err := persistenceManager.SaveRecordedTransactionsWithOptions(p.transactions, p.targetURL, p.noBeautify)
	if err != nil {
		return fmt.Errorf("failed to save recorded transactions: %w", err)
	}

	slog.Debug("Saved inventory",
		"transactions", len(p.transactions),
		"directory", p.inventoryDir)

	return nil
}

// StartRecording starts the recording mode proxy
func StartRecording(targetURL string, port int, inventoryDir string, noBeautify bool) error {
	// Create recording plugin
	recordingPlugin, err := NewRecordingPluginWithInventoryDir(targetURL, inventoryDir, noBeautify)
	if err != nil {
		return fmt.Errorf("failed to create recording plugin: %w", err)
	}

	// Create proxy
	p, err := createProxy(port)
	if err != nil {
		return fmt.Errorf("failed to create proxy: %w", err)
	}

	// Add recording plugin
	p.AddAddon(recordingPlugin)

	// Setup graceful shutdown to save inventory
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		slog.Info("Shutdown signal received, saving inventory...")
		if err := recordingPlugin.SaveInventory(); err != nil {
			slog.Error("Failed to save inventory", "error", err)
		} else {
			slog.Info("Inventory saved successfully")
		}
		
		// Give time for file operations to complete
		time.Sleep(1 * time.Second)
		slog.Info("Shutdown complete")
		os.Exit(0)
	}()

	// Start proxy manually (don't use startProxyWithShutdown to avoid conflicting signal handlers)
	slog.Info("Starting MITM proxy server", "port", port)
	slog.Info("Proxy settings", "url", fmt.Sprintf("http://localhost:%d", port))
	
	if err := p.Start(); err != nil {
		slog.Error("Proxy start failed", "error", err)
		os.Exit(1)
	}
	return nil
}
