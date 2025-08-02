package main

import (
	"sync"
	"sync/atomic"
	"time"
	
	"go-http-playback-proxy/pkg/types"
)

// ResponseTimeHistogram tracks response time distribution
type ResponseTimeHistogram struct {
	buckets []time.Duration
	counts  []atomic.Int64
}

// NewResponseTimeHistogram creates a new histogram with default buckets
func NewResponseTimeHistogram() *ResponseTimeHistogram {
	return &ResponseTimeHistogram{
		buckets: []time.Duration{
			10 * time.Millisecond,
			50 * time.Millisecond,
			100 * time.Millisecond,
			250 * time.Millisecond,
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
		},
		counts: make([]atomic.Int64, 10), // 9 buckets + 1 for > 10s
	}
}

// Record adds a response time to the histogram
func (h *ResponseTimeHistogram) Record(duration time.Duration) {
	for i, bucket := range h.buckets {
		if duration <= bucket {
			h.counts[i].Add(1)
			return
		}
	}
	// Greater than all buckets
	h.counts[len(h.counts)-1].Add(1)
}

// GetStats returns histogram statistics
func (h *ResponseTimeHistogram) GetStats() map[string]int64 {
	stats := make(map[string]int64)
	for i, bucket := range h.buckets {
		stats[bucket.String()] = h.counts[i].Load()
	}
	stats[">10s"] = h.counts[len(h.counts)-1].Load()
	return stats
}

// Metrics collects proxy performance metrics
type Metrics struct {
	// Request counts
	totalRequests     atomic.Int64
	successfulRequests atomic.Int64
	failedRequests    atomic.Int64
	
	// Data transfer
	bytesRecorded  atomic.Int64
	bytesPlayed    atomic.Int64
	
	// Error counts by type
	networkErrors   atomic.Int64
	inventoryErrors atomic.Int64
	encodingErrors  atomic.Int64
	
	// Response times
	mu        sync.RWMutex
	histogram map[string]*ResponseTimeHistogram
	
	// Start time for uptime calculation
	startTime time.Time
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		histogram: make(map[string]*ResponseTimeHistogram),
		startTime: time.Now(),
	}
}

// RecordRequest records a request with its response time
func (m *Metrics) RecordRequest(method, url string, duration time.Duration, success bool) {
	m.totalRequests.Add(1)
	
	if success {
		m.successfulRequests.Add(1)
	} else {
		m.failedRequests.Add(1)
	}
	
	// Record response time in histogram
	key := method + " " + url
	m.mu.Lock()
	hist, exists := m.histogram[key]
	if !exists {
		hist = NewResponseTimeHistogram()
		m.histogram[key] = hist
	}
	m.mu.Unlock()
	
	hist.Record(duration)
}

// RecordBytesRecorded records bytes saved during recording
func (m *Metrics) RecordBytesRecorded(bytes int64) {
	m.bytesRecorded.Add(bytes)
}

// RecordBytesPlayed records bytes served during playback
func (m *Metrics) RecordBytesPlayed(bytes int64) {
	m.bytesPlayed.Add(bytes)
}

// RecordError records an error by type
func (m *Metrics) RecordError(err error) {
	if err == nil {
		return
	}
	
	switch {
	case types.IsErrorType(err, types.ErrorTypeNetwork):
		m.networkErrors.Add(1)
	case types.IsErrorType(err, types.ErrorTypeInventory):
		m.inventoryErrors.Add(1)
	case types.IsErrorType(err, types.ErrorTypeEncoding):
		m.encodingErrors.Add(1)
	}
}

// GetStats returns current metrics
func (m *Metrics) GetStats() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"uptime":              time.Since(m.startTime).String(),
		"total_requests":      m.totalRequests.Load(),
		"successful_requests": m.successfulRequests.Load(),
		"failed_requests":     m.failedRequests.Load(),
		"bytes_recorded":      m.bytesRecorded.Load(),
		"bytes_played":        m.bytesPlayed.Load(),
		"errors": map[string]int64{
			"network":   m.networkErrors.Load(),
			"inventory": m.inventoryErrors.Load(),
			"encoding":  m.encodingErrors.Load(),
		},
	}
	
	// Add top 10 endpoints by request count
	topEndpoints := make(map[string]map[string]int64)
	for endpoint, hist := range m.histogram {
		topEndpoints[endpoint] = hist.GetStats()
	}
	stats["response_times"] = topEndpoints
	
	return stats
}

// GetSuccessRate returns the success rate percentage
func (m *Metrics) GetSuccessRate() float64 {
	total := m.totalRequests.Load()
	if total == 0 {
		return 100.0
	}
	successful := m.successfulRequests.Load()
	return float64(successful) / float64(total) * 100.0
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.totalRequests.Store(0)
	m.successfulRequests.Store(0)
	m.failedRequests.Store(0)
	m.bytesRecorded.Store(0)
	m.bytesPlayed.Store(0)
	m.networkErrors.Store(0)
	m.inventoryErrors.Store(0)
	m.encodingErrors.Store(0)
	
	m.mu.Lock()
	m.histogram = make(map[string]*ResponseTimeHistogram)
	m.mu.Unlock()
	
	m.startTime = time.Now()
}

// Global metrics instance
var globalMetrics = NewMetrics()