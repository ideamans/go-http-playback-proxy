package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"go-http-playback-proxy/pkg/config"
)

func main() {
	var cli config.CLI
	ctx := kong.Parse(&cli,
		kong.Name("http-playback-proxy"),
		kong.Description("HTTP/HTTPS通信の記録・再生プロキシ"),
		kong.UsageOnError(),
	)

	// Create proxy builder
	builder := NewProxyBuilder().
		WithPort(cli.Port).
		WithInventoryDir(cli.InventoryDir).
		WithLogLevel(cli.LogLevel)

	// Execute command
	switch ctx.Command() {
	case "recording <url>":
		if err := executeRecording(builder, cli.Recording.URL, cli.Recording.NoBeautify); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
	case "playback":
		if err := executePlayback(builder, cli.Playback.Watch); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
	default:
		panic("Unknown command")
	}
}

func executeRecording(builder *ProxyBuilder, targetURL string, noBeautify bool) error {
	// Build recording proxy
	p, plugin, err := builder.BuildRecordingProxy(targetURL, noBeautify)
	if err != nil {
		return err
	}
	
	// Start proxy with recording plugin
	startRecordingProxyWithShutdown(p, plugin, builder.GetPort())
	return nil
}

func executePlayback(builder *ProxyBuilder, watch bool) error {
	// Build playback proxy
	p, err := builder.BuildPlaybackProxy()
	if err != nil {
		return err
	}
	
	// Start proxy
	if watch {
		startPlaybackProxyWithWatch(p, builder.GetPort(), builder.GetInventoryDir())
	} else {
		startProxyWithShutdown(p, builder.GetPort())
	}
	return nil
}