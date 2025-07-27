package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lqqyt2423/go-mitmproxy/proxy"
)

// createProxy creates a new MITM proxy instance with common settings
func createProxy(port int) (*proxy.Proxy, error) {
	opts := &proxy.Options{
		Addr:              fmt.Sprintf(":%d", port),
		StreamLargeBodies: 1024 * 1024 * 5, // 5MB以上の大きなボディをストリーミング
		SslInsecure:       true,             // SSL証明書検証を無効化
		CaRootPath:        "",               // デフォルトのCA証明書を使用
		Debug:             0,                // デバッグレベル
	}

	return proxy.NewProxy(opts)
}

// startProxyWithShutdown starts the proxy server with graceful shutdown handling
func startProxyWithShutdown(p *proxy.Proxy, port int) {
	log.Printf("MITM プロキシサーバーを開始します（ポート: %d）", port)
	log.Printf("プロキシ設定: http://localhost:%d", port)

	// シグナルハンドリング
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("シャットダウン中...")
		os.Exit(0)
	}()

	log.Fatal(p.Start())
}

// BaseLogPlugin provides basic logging functionality
type BaseLogPlugin struct {
	proxy.BaseAddon
}

func (p *BaseLogPlugin) ServerConnected(connCtx *proxy.ConnContext) {
	log.Printf("[DNS] Connected to server")
}

func (p *BaseLogPlugin) ClientConnected(clientConn *proxy.ClientConn) {
	log.Printf("[CLIENT] New client connected")
}

func (p *BaseLogPlugin) Request(f *proxy.Flow) {
	if f != nil && f.Request != nil {
		log.Printf("[REQUEST] %s %s", f.Request.Method, f.Request.URL.String())

		// Accept-Encodingヘッダーを確認
		if acceptEncoding := f.Request.Header.Get("Accept-Encoding"); acceptEncoding != "" {
			log.Printf("[COMPRESSION] Client Accept-Encoding: %s", acceptEncoding)
		}
	}
}

func (p *BaseLogPlugin) Response(f *proxy.Flow) {
	if f != nil && f.Response != nil && f.Request != nil {
		log.Printf("[RESPONSE] %s %s %d (Proto: %s)",
			f.Request.Method, f.Request.URL.String(), f.Response.StatusCode, f.Request.Proto)

		// 圧縮情報をログ出力
		if contentEncoding := f.Response.Header.Get("Content-Encoding"); contentEncoding != "" {
			log.Printf("[COMPRESSION] Content-Encoding: %s", contentEncoding)
		}

		// Content-Lengthの情報も確認
		if contentLength := f.Response.Header.Get("Content-Length"); contentLength != "" {
			log.Printf("[SIZE] Content-Length: %s bytes", contentLength)
		}
	}
}