package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/MatusOllah/slogcolor"
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"go-http-playback-proxy/pkg/httputil"
	"go-http-playback-proxy/pkg/plugins"
	"go-http-playback-proxy/pkg/types"
)

// ProxyBuilder helps build proxy instances with configuration
type ProxyBuilder struct {
	port         int
	inventoryDir string
	logLevel     string
	logger       *Logger
}

// NewProxyBuilder creates a new proxy builder
func NewProxyBuilder() *ProxyBuilder {
	return &ProxyBuilder{
		port:         8080,
		inventoryDir: "./inventory",
		logLevel:     "info",
	}
}

// WithPort sets the proxy port
func (b *ProxyBuilder) WithPort(port int) *ProxyBuilder {
	b.port = port
	return b
}

// WithInventoryDir sets the inventory directory
func (b *ProxyBuilder) WithInventoryDir(dir string) *ProxyBuilder {
	b.inventoryDir = dir
	return b
}

// WithLogLevel sets the log level
func (b *ProxyBuilder) WithLogLevel(level string) *ProxyBuilder {
	b.logLevel = level
	return b
}

// Build creates the proxy instance
func (b *ProxyBuilder) Build() (*proxy.Proxy, error) {
	// Setup logger first
	if err := b.setupLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	// Set global metrics for plugins
	plugins.SetGlobalMetrics(globalMetrics)

	// Create proxy using httputil
	opts := &httputil.ProxyOptions{
		Port:              b.port,
		StreamLargeBodies: 1024 * 1024 * 5, // 5MB
		SslInsecure:       true,
		CaRootPath:        "",
		Debug:             0,
	}
	
	p, err := httputil.CreateProxy(opts)
	if err != nil {
		return nil, types.NewNetworkError("failed to create proxy", err)
	}

	return p, nil
}

// BuildRecordingProxy creates a recording proxy
func (b *ProxyBuilder) BuildRecordingProxy(targetURL string, noBeautify bool) (*proxy.Proxy, *plugins.RecordingPlugin, error) {
	p, err := b.Build()
	if err != nil {
		return nil, nil, err
	}

	// Create recording plugin
	plugin, err := plugins.NewRecordingPluginWithInventoryDir(targetURL, b.inventoryDir, noBeautify)
	if err != nil {
		return nil, nil, types.NewValidationError("failed to create recording plugin", err)
	}

	// Add the plugin
	p.AddAddon(plugin)

	b.logger.LogInventoryAction("recording_start", b.inventoryDir, 0)
	b.logger.Info("Recording mode initialized",
		slog.String("target_url", targetURL),
		slog.String("inventory_dir", b.inventoryDir),
		slog.Bool("beautify", !noBeautify))

	return p, plugin, nil
}

// BuildPlaybackProxy creates a playback proxy
func (b *ProxyBuilder) BuildPlaybackProxy() (*proxy.Proxy, error) {
	p, err := b.Build()
	if err != nil {
		return nil, err
	}

	// Create playback plugin
	plugin, err := plugins.NewPlaybackPluginWithInventoryDir(b.inventoryDir)
	if err != nil {
		return nil, types.NewInventoryError("failed to create playback plugin", err)
	}

	// Add the plugin
	p.AddAddon(plugin)

	// Get resource count from plugin
	resourceCount := plugin.GetTransactionCount()

	b.logger.LogInventoryAction("playback_start", b.inventoryDir, resourceCount)
	b.logger.Info("Playback mode initialized",
		slog.String("inventory_dir", b.inventoryDir),
		slog.Int("resource_count", resourceCount))

	return p, nil
}

// GetLogger returns the configured logger
func (b *ProxyBuilder) GetLogger() *Logger {
	return b.logger
}

// GetPort returns the configured port
func (b *ProxyBuilder) GetPort() int {
	return b.port
}

// setupLogger configures the logger
func (b *ProxyBuilder) setupLogger() error {
	// Parse log level
	var level slog.Level
	switch b.logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler
	handler := slogcolor.NewHandler(os.Stderr, &slogcolor.Options{
		Level:       level,
		TimeFormat:  "15:04:05",
		SrcFileMode: slogcolor.ShortFile,
	})

	// Create logger
	b.logger = NewLogger(handler)
	slog.SetDefault(b.logger.Logger)

	// Redirect logrus logs to slog
	SetupLogrusRedirect()

	return nil
}