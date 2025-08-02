package plugins

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
	"go-http-playback-proxy/pkg/inventory"
	"go-http-playback-proxy/pkg/types"
)

// RecordingPlugin handles recording mode functionality
type RecordingPlugin struct {
	BaseLogPlugin
	targetURL    string
	targetDomain string
	transactions []types.RecordingTransaction
	mutex        sync.RWMutex
	inventoryDir string
	noBeautify   bool
}

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
		transactions: make([]types.RecordingTransaction, 0),
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
		transaction := types.RecordingTransaction{
			Method:         f.Request.Method,
			URL:            f.Request.URL.String(),
			RequestStarted: time.Now(),
			RawHeaders:     make(types.HttpHeaders),
		}

		// Store transaction for later retrieval
		p.mutex.Lock()
		if len(p.transactions) < 10000 { // Prevent memory issues
			p.transactions = append(p.transactions, transaction)
			slog.Debug("Transaction started", "method", transaction.Method, "url", transaction.URL, "count", len(p.transactions))
		}
		p.mutex.Unlock()
	}
}

func (p *RecordingPlugin) Response(f *proxy.Flow) {
	p.BaseLogPlugin.Response(f)

	slog.Debug("Response called", "hasFlow", f != nil, "hasResponse", f != nil && f.Response != nil, "hasRequest", f != nil && f.Request != nil)

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

				// Record body
				if f.Response.Body != nil {
					transaction.Body = f.Response.Body
				}

				// Record response finish time
				transaction.ResponseFinished = time.Now()

				// Track metrics
				duration := transaction.ResponseFinished.Sub(transaction.RequestStarted)
				success := transaction.StatusCode != nil && *transaction.StatusCode < 400
				
				if globalMetrics != nil {
					globalMetrics.RecordRequest(transaction.Method, transaction.URL, duration, success)
					globalMetrics.RecordBytesRecorded(int64(len(transaction.Body)))
				}

				// Log transaction
				statusCode := "N/A"
				if transaction.StatusCode != nil {
					statusCode = fmt.Sprintf("%d", *transaction.StatusCode)
				}
				slog.Debug("RECORDED", 
					"method", transaction.Method,
					"url", transaction.URL,
					"status", statusCode,
					"duration_ms", duration.Milliseconds(),
					"body_size", len(transaction.Body),
				)
				break
			}
		}
		p.mutex.Unlock()
	}
}

// SaveInventory saves the recorded transactions to inventory
func (p *RecordingPlugin) SaveInventory() error {
	p.mutex.RLock()
	transactions := make([]types.RecordingTransaction, len(p.transactions))
	copy(transactions, p.transactions)
	p.mutex.RUnlock()

	if len(transactions) == 0 {
		slog.Warn("No transactions recorded to save")
		return nil
	}

	pm := inventory.NewPersistenceManager(p.inventoryDir)
	err := pm.SaveRecordedTransactionsWithOptions(transactions, p.targetURL, p.noBeautify)
	if err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	slog.Info("Inventory saved", "transactions", len(transactions), "directory", p.inventoryDir)
	return nil
}

// SetupSignalHandling sets up signal handling for graceful shutdown
func (p *RecordingPlugin) SetupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received interrupt signal, saving inventory...")
		if err := p.SaveInventory(); err != nil {
			slog.Error("Failed to save inventory on shutdown", "error", err)
		}
		os.Exit(0)
	}()
}

// GetTransactionCount returns the number of recorded transactions
func (p *RecordingPlugin) GetTransactionCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.transactions)
}