package httputil

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// ProxyOptions defines options for creating a proxy
type ProxyOptions struct {
	Port              int
	StreamLargeBodies int64
	SslInsecure       bool
	CaRootPath        string
	Debug             int
}

// DefaultProxyOptions returns default proxy options
func DefaultProxyOptions(port int) *ProxyOptions {
	return &ProxyOptions{
		Port:              port,
		StreamLargeBodies: 1024 * 1024 * 5, // 5MB
		SslInsecure:       true,
		CaRootPath:        "",
		Debug:             0,
	}
}

// CreateProxy creates a new MITM proxy instance with common settings
func CreateProxy(opts *ProxyOptions) (*proxy.Proxy, error) {
	proxyOpts := &proxy.Options{
		Addr:              fmt.Sprintf(":%d", opts.Port),
		StreamLargeBodies: opts.StreamLargeBodies,
		SslInsecure:       opts.SslInsecure,
		CaRootPath:        opts.CaRootPath,
		Debug:             opts.Debug,
	}

	return proxy.NewProxy(proxyOpts)
}

// StartProxyWithShutdown starts the proxy server with graceful shutdown handling
func StartProxyWithShutdown(p *proxy.Proxy, port int) {
	slog.Info("Starting MITM proxy server", "port", port)
	slog.Info("Proxy settings", "url", fmt.Sprintf("http://localhost:%d", port))

	// シグナルハンドリング
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		slog.Info("Shutting down...")
		os.Exit(0)
	}()

	if err := p.Start(); err != nil {
		slog.Error("Proxy start failed", "error", err)
		os.Exit(1)
	}
}