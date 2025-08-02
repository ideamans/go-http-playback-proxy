package plugins

import (
	"log/slog"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"go-http-playback-proxy/pkg/interfaces"
)

// Global metrics instance - should be injected via dependency injection in the future
var globalMetrics interfaces.MetricsCollector

// SetGlobalMetrics sets the global metrics collector
func SetGlobalMetrics(m interfaces.MetricsCollector) {
	globalMetrics = m
}

// BaseLogPlugin provides basic logging functionality
type BaseLogPlugin struct {
	proxy.BaseAddon
}

func (p *BaseLogPlugin) ServerConnected(connCtx *proxy.ConnContext) {
	slog.Debug("Connected to server", "type", "DNS")
}

func (p *BaseLogPlugin) ClientConnected(clientConn *proxy.ClientConn) {
	slog.Debug("New client connected", "type", "CLIENT")
}

func (p *BaseLogPlugin) Request(f *proxy.Flow) {
	if f != nil && f.Request != nil {
		slog.Debug("Request", "method", f.Request.Method, "url", f.Request.URL.String())

		// Accept-Encodingヘッダーを確認
		if acceptEncoding := f.Request.Header.Get("Accept-Encoding"); acceptEncoding != "" {
			slog.Debug("Client Accept-Encoding", "encoding", acceptEncoding)
		}
	}
}

func (p *BaseLogPlugin) Response(f *proxy.Flow) {
	if f != nil && f.Response != nil && f.Request != nil {
		slog.Debug("Response",
			"method", f.Request.Method,
			"url", f.Request.URL.String(),
			"status", f.Response.StatusCode,
			"proto", f.Request.Proto)

		// 圧縮情報をログ出力
		if contentEncoding := f.Response.Header.Get("Content-Encoding"); contentEncoding != "" {
			slog.Debug("Content-Encoding", "encoding", contentEncoding)
		}

		// Content-Lengthの情報も確認
		if contentLength := f.Response.Header.Get("Content-Length"); contentLength != "" {
			slog.Debug("Content-Length", "bytes", contentLength)
		}
	}
}