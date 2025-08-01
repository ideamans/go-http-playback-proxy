package main

import (
	"log/slog"
	"time"
	
	"go-http-playback-proxy/pkg/types"
)

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new Logger instance
func NewLogger(handler slog.Handler) *Logger {
	return &Logger{
		Logger: slog.New(handler),
	}
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(method, url string, attrs ...slog.Attr) {
	l.Info("HTTP Request",
		slog.String("method", method),
		slog.String("url", url),
		slog.Group("details", slog.Any("attrs", attrs)))
}

// LogResponse logs HTTP response information
func (l *Logger) LogResponse(method, url string, statusCode int, duration time.Duration, attrs ...slog.Attr) {
	l.Info("HTTP Response",
		slog.String("method", method),
		slog.String("url", url),
		slog.Int("status", statusCode),
		slog.Duration("duration", duration),
		slog.Group("details", slog.Any("attrs", attrs)))
}

// LogDNS logs DNS resolution information
func (l *Logger) LogDNS(host, ip string, duration time.Duration) {
	l.Info("DNS Resolution",
		slog.String("host", host),
		slog.String("ip", ip),
		slog.Duration("duration", duration))
}

// LogCompression logs compression information
func (l *Logger) LogCompression(original, final string, originalSize, finalSize int) {
	l.Info("Content Compression",
		slog.String("original_encoding", original),
		slog.String("final_encoding", final),
		slog.Int("original_size", originalSize),
		slog.Int("final_size", finalSize),
		slog.Float64("compression_ratio", float64(finalSize)/float64(originalSize)))
}

// LogError logs an error with context
func (l *Logger) LogError(message string, err error, attrs ...slog.Attr) {
	if proxyErr, ok := err.(*types.ProxyError); ok {
		l.Error(message,
			slog.String("error_type", string(proxyErr.Type)),
			slog.String("error", proxyErr.Error()),
			slog.Any("context", proxyErr.Context),
			slog.Group("details", slog.Any("attrs", attrs)))
	} else {
		l.Error(message,
			slog.String("error", err.Error()),
			slog.Group("details", slog.Any("attrs", attrs)))
	}
}

// LogInventoryAction logs inventory-related actions
func (l *Logger) LogInventoryAction(action string, path string, resourceCount int) {
	l.Info("Inventory Action",
		slog.String("action", action),
		slog.String("path", path),
		slog.Int("resource_count", resourceCount))
}

// LogPlayback logs playback-related information
func (l *Logger) LogPlayback(url string, fromInventory bool, ttfb time.Duration) {
	l.Info("Playback",
		slog.String("url", url),
		slog.Bool("from_inventory", fromInventory),
		slog.Duration("ttfb", ttfb))
}

// LogRecording logs recording-related information
func (l *Logger) LogRecording(url string, size int, duration time.Duration) {
	l.Info("Recording",
		slog.String("url", url),
		slog.Int("size", size),
		slog.Duration("duration", duration))
}