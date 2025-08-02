package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"go-http-playback-proxy/pkg/httputil"
	"go-http-playback-proxy/pkg/plugins"
)

// createProxy creates a new MITM proxy instance with common settings
func createProxy(port int) (*proxy.Proxy, error) {
	opts := httputil.DefaultProxyOptions(port)
	return httputil.CreateProxy(opts)
}

// startProxyWithShutdown starts the proxy server with graceful shutdown handling
func startProxyWithShutdown(p *proxy.Proxy, port int) {
	httputil.StartProxyWithShutdown(p, port)
}

// startRecordingProxyWithShutdown starts the recording proxy with proper shutdown handling
func startRecordingProxyWithShutdown(p *proxy.Proxy, plugin *plugins.RecordingPlugin, port int) {
	slog.Info("Starting MITM proxy server in recording mode", "port", port)
	slog.Info("Proxy settings", "url", fmt.Sprintf("http://localhost:%d", port))

	// シグナルハンドリング - 録画プラグインのインベントリ保存を優先
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		slog.Info("Shutting down...")
		
		// First save the inventory
		if err := plugin.SaveInventory(); err != nil {
			slog.Error("Failed to save inventory on shutdown", "error", err)
		}
		
		os.Exit(0)
	}()

	if err := p.Start(); err != nil {
		slog.Error("Proxy start failed", "error", err)
		os.Exit(1)
	}
}

