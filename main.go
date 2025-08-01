package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/MatusOllah/slogcolor"
)

type CLI struct {
	Port         int    `short:"p" default:"8080" help:"プロキシサーバーのポート番号"`
	InventoryDir string `short:"i" default:"./inventory" help:"inventoryディレクトリのパス"`
	LogLevel     string `short:"l" default:"info" help:"ログレベル (debug, info, warn, error)" env:"LOG_LEVEL"`

	Recording struct {
		URL        string `arg:"" required:"" help:"記録対象のURL"`
		NoBeautify bool   `help:"HTML・CSS・JavaScriptのBeautifyを無効化"`
	} `cmd:"" help:"指定URLへの通信を記録"`

	Playback struct {
	} `cmd:"" help:"記録した通信を再生"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("http-playback-proxy"),
		kong.Description("HTTP/HTTPS通信の記録・再生プロキシ"),
		kong.UsageOnError(),
	)

	// ログレベル設定
	var level slog.Level
	switch strings.ToLower(cli.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		fmt.Fprintf(os.Stderr, "Warning: Unknown log level '%s', using default 'info'\n", cli.LogLevel)
		level = slog.LevelInfo
	}

	// ログハンドラ設定
	handler := slogcolor.NewHandler(os.Stderr, &slogcolor.Options{
		Level: level,
		TimeFormat: "15:04:05",
		SrcFileMode: slogcolor.ShortFile,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// コマンド実行
	switch ctx.Command() {
	case "recording <url>":
		slog.Info("Starting recording mode",
			"url", cli.Recording.URL,
			"port", cli.Port,
			"inventory", cli.InventoryDir,
			"no-beautify", cli.Recording.NoBeautify,
		)
		
		if err := StartRecording(cli.Recording.URL, cli.Port, cli.InventoryDir, cli.Recording.NoBeautify); err != nil {
			slog.Error("Recording mode failed", "error", err)
			os.Exit(1)
		}
		
	case "playback":
		slog.Info("Starting playback mode",
			"port", cli.Port,
			"inventory", cli.InventoryDir,
		)
		
		if err := StartPlayback(cli.Port, cli.InventoryDir); err != nil {
			slog.Error("Playback mode failed", "error", err)
			os.Exit(1)
		}
		
	default:
		panic("Unknown command")
	}
}