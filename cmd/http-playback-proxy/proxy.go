package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
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

// startPlaybackProxyWithWatch starts the playback proxy with file watching
func startPlaybackProxyWithWatch(p *proxy.Proxy, port int, inventoryDir string) {
	slog.Info("Starting MITM proxy server in playback mode with watch", "port", port)
	slog.Info("Proxy settings", "url", fmt.Sprintf("http://localhost:%d", port))
	slog.Info("Watching inventory directory for changes", "dir", inventoryDir)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create file watcher", "error", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Watch inventory directory recursively
	err = filepath.Walk(inventoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		slog.Error("Failed to add directory to watcher", "error", err)
		os.Exit(1)
	}

	// Reload state management
	var reloadMutex sync.Mutex
	var reloadPending bool
	var lastReloadTime time.Time

	// Reload function
	doReload := func() {
		reloadMutex.Lock()
		defer reloadMutex.Unlock()

		if time.Since(lastReloadTime) < 100*time.Millisecond {
			// Too soon after last reload, mark as pending
			reloadPending = true
			return
		}

		slog.Info("Reloading inventory due to file changes")
		
		// Get the playback plugin from proxy
		for _, addon := range p.Addons {
			if plugin, ok := addon.(*plugins.PlaybackPlugin); ok {
				if err := plugin.ReloadInventory(); err != nil {
					slog.Error("Failed to reload inventory", "error", err)
				} else {
					resourceCount := plugin.GetTransactionCount()
					slog.Info("Inventory reloaded successfully", "resource_count", resourceCount)
				}
				break
			}
		}

		lastReloadTime = time.Now()
		reloadPending = false
	}

	// Check for pending reloads periodically
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		
		for range ticker.C {
			reloadMutex.Lock()
			if reloadPending && time.Since(lastReloadTime) >= 100*time.Millisecond {
				reloadMutex.Unlock()
				doReload()
			} else {
				reloadMutex.Unlock()
			}
		}
	}()

	// Watch for file changes
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Ignore temporary files and directories
				if filepath.Base(event.Name)[0] == '.' {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					slog.Debug("File change detected", "file", event.Name, "op", event.Op.String())
					doReload()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("File watcher error", "error", err)
			}
		}
	}()

	// Signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start proxy in background
	go func() {
		if err := p.Start(); err != nil {
			slog.Error("Proxy start failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-c
	slog.Info("Shutting down...")
	os.Exit(0)
}
