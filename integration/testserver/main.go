package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const (
	TestDataDir = "../testdata"
	DefaultPort = 9999
)

type TestServer struct {
	testDataDir string
}

func NewTestServer() *TestServer {
	return &TestServer{
		testDataDir: TestDataDir,
	}
}

func (ts *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[REQUEST] %s %s", r.Method, r.URL.Path)

	// パフォーマンス制御パラメータ
	ttfb := ts.getTTFB(r)
	speed := ts.getSpeed(r)
	compression := ts.getCompression(r)

	// TTFB遅延
	if ttfb > 0 {
		time.Sleep(ttfb)
	}

	// ルーティング
	switch {
	case r.URL.Path == "/":
		ts.serveIndex(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		ts.serveAPI(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/images/"):
		ts.serveImages(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/html/"):
		ts.serveHTML(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/css/"):
		ts.serveCSS(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/js/"):
		ts.serveJS(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/performance/"):
		ts.servePerformanceTest(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/status/"):
		ts.serveStatusTest(w, r, compression, speed)
	case strings.HasPrefix(r.URL.Path, "/charset/"):
		ts.serveCharsetTest(w, r, compression, speed)
	default:
		// 存在しないパスの場合の処理
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// APIパスの場合は汎用応答
			ts.serveGenericAPI(w, r, compression, speed)
		} else {
			// その他は404
			http.NotFound(w, r)
		}
	}
}

func (ts *TestServer) getTTFB(r *http.Request) time.Duration {
	if ttfbStr := r.URL.Query().Get("ttfb"); ttfbStr != "" {
		if ms, err := strconv.Atoi(ttfbStr); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return 0
}

func (ts *TestServer) getSpeed(r *http.Request) int {
	speedStr := r.URL.Query().Get("speed")
	log.Printf("[SPEED] Query parameter 'speed' = '%s' from URL: %s", speedStr, r.URL.String())
	
	if speedStr != "" {
		if speed, err := strconv.Atoi(speedStr); err == nil {
			log.Printf("[SPEED] Parsed speed: %d Kbps", speed)
			return speed // Kbps
		} else {
			log.Printf("[SPEED] Failed to parse speed: %v", err)
		}
	}
	log.Printf("[SPEED] Returning unlimited speed (0)")
	return 0 // 無制限
}

func (ts *TestServer) getCompression(r *http.Request) string {
	if comp := r.URL.Query().Get("compression"); comp != "" {
		return comp
	}
	// Accept-Encodingヘッダーから判定
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if strings.Contains(acceptEncoding, "br") {
		return "br"
	}
	if strings.Contains(acceptEncoding, "gzip") {
		return "gzip"
	}
	if strings.Contains(acceptEncoding, "deflate") {
		return "deflate"
	}
	return "identity"
}

func (ts *TestServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>HTTP Playback Proxy Test Server</title>
</head>
<body>
    <h1>テストサーバー</h1>
    <h2>テストエンドポイント</h2>
    <ul>
        <li><a href="/api/users.json">JSON API</a></li>
        <li><a href="/api/large_data.json">大きなJSON</a></li>
        <li><a href="/images/small.jpg">小さな画像</a></li>
        <li><a href="/images/large.jpg">大きな画像</a></li>
        <li><a href="/html/utf8.html">UTF-8 HTML</a></li>
        <li><a href="/html/shift_jis.html">Shift_JIS HTML</a></li>
        <li><a href="/css/utf8.css">UTF-8 CSS</a></li>
        <li><a href="/js/utf8.js">JavaScript</a></li>
    </ul>
    
    <h2>パフォーマンステスト</h2>
    <ul>
        <li><a href="/performance/small?ttfb=100&speed=1000">小ファイル・低速</a></li>
        <li><a href="/performance/medium?ttfb=500&speed=5000">中ファイル・中速</a></li>
        <li><a href="/performance/large?ttfb=1000&speed=10000">大ファイル・高速</a></li>
    </ul>
    
    <h2>ステータスコードテスト</h2>
    <ul>
        <li><a href="/status/200">200 OK</a></li>
        <li><a href="/status/301">301 Moved Permanently</a></li>
        <li><a href="/status/404">404 Not Found</a></li>
        <li><a href="/status/500">500 Internal Server Error</a></li>
    </ul>
    
    <h2>文字コードテスト</h2>
    <ul>
        <li><a href="/charset/utf8">UTF-8</a></li>
        <li><a href="/charset/shift_jis">Shift_JIS</a></li>
        <li><a href="/charset/euc_jp">EUC-JP</a></li>
        <li><a href="/charset/iso8859">ISO-8859-1</a></li>
    </ul>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (ts *TestServer) serveAPI(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/")
	filePath := filepath.Join(ts.testDataDir, "api", filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		// ファイルが存在しない場合は汎用APIレスポンスにフォールバック
		ts.serveGenericAPI(w, r, compression, speed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveImages(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	filename := strings.TrimPrefix(r.URL.Path, "/images/")
	filePath := filepath.Join(ts.testDataDir, "images", filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// Content-Typeを拡張子から判定
	ext := filepath.Ext(filename)
	var contentType string
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
	}
	
	w.Header().Set("Content-Type", contentType)
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveHTML(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	filename := strings.TrimPrefix(r.URL.Path, "/html/")
	filePath := filepath.Join(ts.testDataDir, "html", filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// 文字コードを設定
	var charset string
	switch filename {
	case "shift_jis.html":
		charset = "Shift_JIS"
	case "euc_jp.html":
		charset = "EUC-JP"
	case "iso8859.html":
		charset = "ISO-8859-1"
	default:
		charset = "UTF-8"
	}
	
	w.Header().Set("Content-Type", fmt.Sprintf("text/html; charset=%s", charset))
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveCSS(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	filename := strings.TrimPrefix(r.URL.Path, "/css/")
	filePath := filepath.Join(ts.testDataDir, "css", filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// 文字コードを設定
	var charset string
	switch filename {
	case "shift_jis.css":
		charset = "Shift_JIS"
	default:
		charset = "UTF-8"
	}
	
	w.Header().Set("Content-Type", fmt.Sprintf("text/css; charset=%s", charset))
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveJS(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	filename := strings.TrimPrefix(r.URL.Path, "/js/")
	filePath := filepath.Join(ts.testDataDir, "js", filename)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// 文字コードを設定
	var charset string
	switch filename {
	case "shift_jis.js":
		charset = "Shift_JIS"
	default:
		charset = "UTF-8"
	}
	
	w.Header().Set("Content-Type", fmt.Sprintf("application/javascript; charset=%s", charset))
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) servePerformanceTest(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	size := strings.TrimPrefix(r.URL.Path, "/performance/")
	
	var data []byte
	switch size {
	case "small":
		data = make([]byte, 1024) // 1KB
		for i := range data {
			data[i] = byte(i % 256)
		}
	case "medium":
		data = make([]byte, 100*1024) // 100KB
		for i := range data {
			data[i] = byte(i % 256)
		}
	case "large":
		data = make([]byte, 10*1024*1024) // 10MB
		for i := range data {
			data[i] = byte(i % 256)
		}
	default:
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveStatusTest(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	statusStr := strings.TrimPrefix(r.URL.Path, "/status/")
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	w.WriteHeader(status)
	
	response := map[string]interface{}{
		"status": status,
		"message": http.StatusText(status),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	data, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) serveCharsetTest(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	charset := strings.TrimPrefix(r.URL.Path, "/charset/")
	
	var filename string
	switch charset {
	case "utf8":
		filename = "utf8.html"
	case "shift_jis":
		filename = "shift_jis.html"
	case "euc_jp":
		filename = "euc_jp.html"
	case "iso8859":
		filename = "iso8859.html"
	default:
		http.NotFound(w, r)
		return
	}
	
	filePath := filepath.Join(ts.testDataDir, "html", filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	
	// Content-Typeを正しく設定
	var contentType string
	switch charset {
	case "shift_jis":
		contentType = "text/html; charset=Shift_JIS"
	case "euc_jp":
		contentType = "text/html; charset=EUC-JP"
	case "iso8859":
		contentType = "text/html; charset=ISO-8859-1"
	default:
		contentType = "text/html; charset=UTF-8"
	}
	
	w.Header().Set("Content-Type", contentType)
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func (ts *TestServer) writeWithCompressionAndSpeed(w http.ResponseWriter, data []byte, compression string, speed int) {
	// 圧縮処理
	compressedData := ts.compressData(data, compression)
	if compression != "identity" {
		w.Header().Set("Content-Encoding", compression)
	}
	
	// Content-Lengthを設定
	w.Header().Set("Content-Length", strconv.Itoa(len(compressedData)))
	
	// 速度制御
	if speed > 0 {
		ts.writeWithSpeedLimit(w, compressedData, speed)
	} else {
		w.Write(compressedData)
	}
}

func (ts *TestServer) compressData(data []byte, compression string) []byte {
	switch compression {
	case "gzip":
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(data)
		gz.Close()
		return buf.Bytes()
	case "deflate":
		var buf bytes.Buffer
		zw, _ := zstd.NewWriter(&buf)
		zw.Write(data)
		zw.Close()
		return buf.Bytes()
	case "br":
		var buf bytes.Buffer
		bw := brotli.NewWriter(&buf)
		bw.Write(data)
		bw.Close()
		return buf.Bytes()
	default:
		return data
	}
}

func (ts *TestServer) writeWithSpeedLimit(w http.ResponseWriter, data []byte, speedKbps int) {
	log.Printf("[SPEED] writeWithSpeedLimit called: %d bytes, %d Kbps", len(data), speedKbps)
	
	if speedKbps <= 0 {
		// 速度制限なしの場合はそのまま送信
		log.Printf("[SPEED] No speed limit, sending %d bytes immediately", len(data))
		w.Write(data)
		return
	}

	// 1秒あたりのバイト数を計算
	bytesPerSecond := speedKbps * 1024 / 8
	log.Printf("[SPEED] Speed limit: %d Kbps = %d bytes/sec", speedKbps, bytesPerSecond)
	
	// データサイズに応じた適応的速度制御
	var intervalMs int
	var chunkSize int
	
	if len(data) < 10*1024 { // 10KB未満の小さなファイル
		// 小さなファイルでは細かいチャンクで時間をかける
		intervalMs = 100 // 100ms間隔
		expectedTransferTimeMs := (len(data) * 8 * 1000) / (speedKbps * 1024) // 期待転送時間（ミリ秒）
		if expectedTransferTimeMs < 100 {
			expectedTransferTimeMs = 100 // 最低100ms
		}
		chunkSize = len(data) / (expectedTransferTimeMs / intervalMs) // チャンク数から逆算
		if chunkSize < 1 {
			chunkSize = 1
		}
		if chunkSize > len(data)/3 { // 最低3チャンクに分割
			chunkSize = len(data) / 3
			if chunkSize < 1 {
				chunkSize = 1
			}
		}
	} else { // 大きなファイル
		// 従来の50ms間隔制御
		intervalMs = 50
		chunkSize = bytesPerSecond * intervalMs / 1000 // 50msあたりのバイト数
		if chunkSize < 1 {
			chunkSize = 1
		}
		// 最小1KB、最大10KBのチャンクサイズ
		minChunkSize := 1024    // 1KB
		maxChunkSize := 10 * 1024 // 10KB
		if chunkSize < minChunkSize {
			chunkSize = minChunkSize
		}
		if chunkSize > maxChunkSize {
			chunkSize = maxChunkSize
		}
	}
	
	expectedChunks := (len(data) + chunkSize - 1) / chunkSize
	expectedDurationMs := expectedChunks * intervalMs
	log.Printf("[SPEED] Adaptive control: %d bytes file, chunk=%d, interval=%dms, chunks=%d, duration=%dms", 
		len(data), chunkSize, intervalMs, expectedChunks, expectedDurationMs)
	
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		
		w.Write(data[i:end])
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		
		// 次のチャンクまで待機（50ms）
		if end < len(data) {
			time.Sleep(time.Duration(intervalMs) * time.Millisecond)
		}
	}
}

// 汎用APIハンドラー（存在しないパスも200で応答）
func (ts *TestServer) serveGenericAPI(w http.ResponseWriter, r *http.Request, compression string, speed int) {
	// 基本的なJSON応答を生成
	response := map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
		"headers": func() map[string]string {
			headers := make(map[string]string)
			for k, v := range r.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			return headers
		}(),
		"timestamp": time.Now().Format(time.RFC3339),
		"message": "Generic API response for testing",
	}
	
	data, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	ts.writeWithCompressionAndSpeed(w, data, compression, speed)
}

func main() {
	port := DefaultPort
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}
	
	server := NewTestServer()
	addr := fmt.Sprintf(":%d", port)
	
	log.Printf("Starting test server on %s", addr)
	log.Printf("Test data directory: %s", TestDataDir)
	
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}