package config

import (
	"time"
)

// CLI defines command line interface configuration
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

// Config holds all configuration for the proxy
type Config struct {
	Port         int
	InventoryDir string
	LogLevel     string
	Recording    RecordingConfig
	Playback     PlaybackConfig
	Proxy        ProxyConfig
}

// RecordingConfig holds recording-specific configuration
type RecordingConfig struct {
	TargetURL   string
	NoBeautify  bool
	ChunkSize   int
	Timeout     time.Duration
}

// PlaybackConfig holds playback-specific configuration
type PlaybackConfig struct {
	ChunkSize       int
	EnableUpstream  bool
	UpstreamTimeout time.Duration
}

// ProxyConfig holds proxy-specific configuration
type ProxyConfig struct {
	StreamLargeBodies int64
	SSLInsecure       bool
	CAPath            string
	DebugLevel        int
	MaxConnections    int
	DNSTimeout        time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		InventoryDir: "./inventory",
		LogLevel:     "info",
		Recording: RecordingConfig{
			ChunkSize: 32 * 1024, // 32KB chunks
			Timeout:   30 * time.Second,
		},
		Playback: PlaybackConfig{
			ChunkSize:       32 * 1024,
			EnableUpstream:  true,
			UpstreamTimeout: 30 * time.Second,
		},
		Proxy: ProxyConfig{
			StreamLargeBodies: 5 * 1024 * 1024, // 5MB
			SSLInsecure:       true,
			MaxConnections:    10,
			DNSTimeout:        5 * time.Second,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Add validation logic here
	return nil
}