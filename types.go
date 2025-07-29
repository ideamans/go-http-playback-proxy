package main

import (
	"time"
)

// HttpHeaders represents HTTP headers as key-value pairs
type HttpHeaders map[string]string

// ContentEncodingType represents supported content encoding types
type ContentEncodingType string

const (
	ContentEncodingGzip     ContentEncodingType = "gzip"
	ContentEncodingCompress ContentEncodingType = "compress"
	ContentEncodingDeflate  ContentEncodingType = "deflate"
	ContentEncodingBr       ContentEncodingType = "br"
	ContentEncodingZstd     ContentEncodingType = "zstd"
	ContentEncodingIdentity ContentEncodingType = "identity"
)

// DeviceType represents the device type
type DeviceType string

const (
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeMobile  DeviceType = "mobile"
)

// Resource represents an HTTP resource with all its metadata
type Resource struct {
	Method             string               `json:"method"`
	URL                string               `json:"url"`
	TTFBMS             int64                `json:"ttfbMs"`
	MBPS               *float64             `json:"mbps,omitempty"`
	StatusCode         *int                 `json:"statusCode,omitempty"`
	ErrorMessage       *string              `json:"errorMessage,omitempty"`
	RawHeaders         HttpHeaders          `json:"rawHeaders,omitempty"`
	ContentEncoding    *ContentEncodingType `json:"contentEncoding,omitempty"`
	ContentTypeMime    *string              `json:"contentTypeMime,omitempty"`
	ContentTypeCharset *string              `json:"contentTypeCharset,omitempty"`
	ContentCharset     *string              `json:"contentCharset,omitempty"`
	ContentFilePath    *string              `json:"contentFilePath,omitempty"`
	ContentUTF8        *string              `json:"contentUtf8,omitempty"`
	ContentBase64      *string              `json:"contentBase64,omitempty"`
	Minify             *bool                `json:"minify,omitempty"`
}

// Domain represents a domain with its IP address
type Domain struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
	LatencyMS int64  `json:"latencyMs,omitempty"` // Latency in milliseconds
}

// Inventory represents a collection of resources and domains
type Inventory struct {
	EntryURL   *string     `json:"entryUrl,omitempty"`
	DeviceType *DeviceType `json:"deviceType,omitempty"`
	Domains    []Domain    `json:"domains"`
	Resources  []Resource  `json:"resources"`
}

// BodyChunk represents a chunk of response body with timing information
type BodyChunk struct {
	Chunk      []byte
	TargetTime time.Time
	// TargetOffset represents the time offset from request start when this chunk should be sent
	TargetOffset time.Duration
}

// PlaybackResource represents a complete HTTP transaction for playback
type PlaybackResource struct {
	Method       string
	URL          string
	TTFB         time.Duration
	StatusCode   *int
	ErrorMessage *string
	RawHeaders   HttpHeaders
	Chunks       []BodyChunk
}

// RecordingResource represents an HTTP resource during recording
type RecordingTransaction struct {
	Method           string
	URL              string
	RequestStarted   time.Time
	ResponseStarted  time.Time
	ResponseFinished time.Time
	StatusCode       *int
	ErrorMessage     *string
	RawHeaders       HttpHeaders
	Body             []byte
}

// PlaybackTransaction represents a complete HTTP transaction for playback with all data
type PlaybackTransaction struct {
	Method       string
	URL          string
	TTFB         time.Duration
	StatusCode   *int
	ErrorMessage *string
	RawHeaders   HttpHeaders
	Chunks       []BodyChunk
}
