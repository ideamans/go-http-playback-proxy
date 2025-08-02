package interfaces

import (
	"context"
	"net/http"
	"time"
)

// InventoryManager manages inventory storage and retrieval
type InventoryManager interface {
	Load() (interface{}, error)
	Save(inventory interface{}) error
	GetResource(method, url string) (interface{}, error)
	AddResource(resource interface{}) error
}

// ContentProcessor processes content (encoding, formatting, etc.)
type ContentProcessor interface {
	Process(ctx context.Context, body []byte, contentType string) ([]byte, error)
	ShouldProcess(contentType string) bool
}

// Encoder handles content encoding
type Encoder interface {
	Encode(data []byte) ([]byte, error)
}

// Decoder handles content decoding
type Decoder interface {
	Decode(data []byte) ([]byte, error)
}

// ProxyPlugin defines the interface for proxy plugins
type ProxyPlugin interface {
	Request(ctx context.Context, req *http.Request) (*http.Request, *http.Response)
	Response(ctx context.Context, req *http.Request, resp *http.Response) *http.Response
}

// Logger defines logging interface
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// MetricsCollector collects metrics
type MetricsCollector interface {
	RecordRequest(method, url string, duration time.Duration, success bool)
	RecordBytesRecorded(bytes int64)
	RecordBytesPlayed(bytes int64)
	RecordError(err error)
	GetStats() interface{}
}

// ResourceConverter converts between URLs and file paths
type ResourceConverter interface {
	MethodURLToFilePath(method, url string) (string, error)
	FilePathToMethodURL(filePath string) (method, url string, err error)
}