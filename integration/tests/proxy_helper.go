package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// プロキシ制御ヘルパー
type ProxyController struct {
	Port         int
	ProxyPath    string
	InventoryDir string
	Process      *exec.Cmd
}

// テスト用の型定義は後で統一定義される

// HTTPレスポンス構造体
type HTTPResponse struct {
	StatusCode      int
	Headers         map[string]string
	Body            []byte
	ContentType     string
	ContentEncoding string
	ContentLength   int64
}

// テスト用型定義（メインプロジェクトと互換性を保つ）
type Inventory struct {
	EntryURL  *string    `json:"entryUrl,omitempty"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Method             string            `json:"method"`
	URL                string            `json:"url"`
	StatusCode         *int              `json:"statusCode,omitempty"`
	TTFBMS             int64             `json:"ttfbMs"`
	Mbps               *float64          `json:"mbps,omitempty"`
	ContentType        string            `json:"contentType,omitempty"`
	ContentTypeMime    *string           `json:"contentTypeMime,omitempty"`
	ContentEncoding    *string           `json:"contentEncoding,omitempty"`
	ContentCharset     *string           `json:"contentCharset,omitempty"`
	ContentTypeCharset *string           `json:"contentTypeCharset,omitempty"`
	Minify             *bool             `json:"minify,omitempty"`
	ErrorMessage       *string           `json:"errorMessage,omitempty"`
	RawHeaders         map[string]string `json:"rawHeaders,omitempty"`
	ContentFilePath    *string           `json:"contentFilePath,omitempty"`
}

func NewProxyController(port int, proxyPath, inventoryDir string) *ProxyController {
	return &ProxyController{
		Port:         port,
		ProxyPath:    proxyPath,
		InventoryDir: inventoryDir,
	}
}

// Recording モードでプロキシを起動
func (pc *ProxyController) StartRecording(targetURL string) error {
	// inventory ディレクトリを作成（再帰的に全て作成）
	if err := os.MkdirAll(pc.InventoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create inventory directory: %v", err)
	}

	// contents ディレクトリも事前に作成
	contentsDir := filepath.Join(pc.InventoryDir, "contents")
	if err := os.MkdirAll(contentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create contents directory: %v", err)
	}

	// プロキシを recording モードで起動
	cmd := exec.Command(pc.ProxyPath,
		"--port", fmt.Sprintf("%d", pc.Port),
		"--inventory-dir", pc.InventoryDir,
		"recording", targetURL)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start recording proxy: %v", err)
	}

	pc.Process = cmd

	// プロキシの起動を待機
	if err := pc.waitForProxy(); err != nil {
		pc.Stop()
		return err
	}

	return nil
}

// Playback モードでプロキシを起動
func (pc *ProxyController) StartPlayback() error {
	// inventory ファイルの存在確認
	inventoryPath := filepath.Join(pc.InventoryDir, "inventory.json")
	if _, err := os.Stat(inventoryPath); err != nil {
		return fmt.Errorf("inventory not found: %v", err)
	}

	// プロキシを playback モードで起動
	cmd := exec.Command(pc.ProxyPath,
		"--port", fmt.Sprintf("%d", pc.Port),
		"--inventory-dir", pc.InventoryDir,
		"playback")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start playback proxy: %v", err)
	}

	pc.Process = cmd

	// プロキシの起動を待機
	if err := pc.waitForProxy(); err != nil {
		pc.Stop()
		return err
	}

	return nil
}

// プロキシを停止
func (pc *ProxyController) Stop() error {
	if pc.Process == nil {
		return nil
	}

	// SIGINT でプロセスを終了（graceful shutdown）
	if err := pc.Process.Process.Signal(syscall.SIGINT); err != nil {
		// SIGINT が効かない場合は SIGKILL
		pc.Process.Process.Kill()
	}

	// プロセス終了を待機（タイムアウト付き）
	done := make(chan error, 1)
	go func() {
		done <- pc.Process.Wait()
	}()

	select {
	case <-done:
		// プロセスが正常終了
	case <-time.After(3 * time.Second):
		// タイムアウト: 強制終了
		pc.Process.Process.Kill()
		<-done
	}

	pc.Process = nil

	return nil
}

// プロキシの起動を待機
func (pc *ProxyController) waitForProxy() error {
	for i := 0; i < 15; i++ { // 15秒まで待機（短縮）
		// プロセスがまだ実行中か確認
		if pc.Process != nil && pc.Process.ProcessState != nil && pc.Process.ProcessState.Exited() {
			return fmt.Errorf("proxy process exited unexpectedly")
		}

		// ポートが開いているか直接確認
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", pc.Port), 500*time.Millisecond)
		if err == nil {
			conn.Close()

			// プロキシ経由でローカルテストサーバーに簡単なテスト
			client := &http.Client{Timeout: 3 * time.Second}
			proxyURL := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("localhost:%d", pc.Port),
			}
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client.Transport = transport

			// ローカルテストサーバーをテスト（より信頼性が高い）
			req, err := http.NewRequest("GET", "http://localhost:9999/", nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
					return nil // プロキシが応答した
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("proxy did not start within 15 seconds")
}

// プロキシ経由でHTTPリクエストを実行
func (pc *ProxyController) MakeRequest(method, urlStr string, headers map[string]string) (*HTTPResponse, error) {
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", pc.Port),
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	// ヘッダーを設定
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// レスポンスヘッダーを map に変換
	responseHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			responseHeaders[k] = v[0]
		}
	}

	return &HTTPResponse{
		StatusCode:      resp.StatusCode,
		Headers:         responseHeaders,
		Body:            body,
		ContentType:     resp.Header.Get("Content-Type"),
		ContentEncoding: resp.Header.Get("Content-Encoding"),
		ContentLength:   resp.ContentLength,
	}, nil
}

// 直接アクセス（プロキシなし）
func MakeDirectRequest(method, urlStr string, headers map[string]string) (*HTTPResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	// ヘッダーを設定
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// レスポンスヘッダーを map に変換
	responseHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			responseHeaders[k] = v[0]
		}
	}

	return &HTTPResponse{
		StatusCode:      resp.StatusCode,
		Headers:         responseHeaders,
		Body:            body,
		ContentType:     resp.Header.Get("Content-Type"),
		ContentEncoding: resp.Header.Get("Content-Encoding"),
		ContentLength:   resp.ContentLength,
	}, nil
}

// inventory.json を読み込み
func (pc *ProxyController) LoadInventory() (*Inventory, error) {
	// プロキシは inventory.json をディレクトリ直下に保存
	inventoryPath := filepath.Join(pc.InventoryDir, "inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, err
	}

	var inventory Inventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		return nil, err
	}

	return &inventory, nil
}

// コンテンツファイルを読み込み
func (pc *ProxyController) LoadContent(method, urlStr string) ([]byte, error) {
	// URL をファイルパスに変換（実際の resource.go の関数を使用予定）
	// 今は簡易実装
	filename := fmt.Sprintf("%s_%s.content", method, sanitizeURL(urlStr))
	contentPath := filepath.Join(pc.InventoryDir, "contents", filename)

	return os.ReadFile(contentPath)
}

// URL をファイル名として安全な文字列に変換
func sanitizeURL(urlStr string) string {
	safe := ""
	for _, r := range urlStr {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			safe += string(r)
		} else {
			safe += "_"
		}
	}
	return safe
}

// 並行実行用のヘルパー関数

// GetAvailablePort 利用可能なポート番号を取得
func GetAvailablePort() (int, error) {
	// ランダムシードを初期化（一度だけ）
	rand.Seed(time.Now().UnixNano())

	// ランダムな範囲から開始（10000-20000に拡張）
	start := 10000 + rand.Intn(10000)

	for i := 0; i < 200; i++ { // 最大200個のポートを試行
		port := start + i
		if port > 65535 {
			port = 10000 + (port - 65536)
		}

		// ポートが利用可能かチェック
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available port found")
}

// CreateTempInventoryDir 一時的なインベントリディレクトリを作成
func CreateTempInventoryDir(testName string) (string, error) {
	// ユニークなディレクトリ名を生成
	timestamp := time.Now().UnixNano()
	safeName := sanitizeFileName(testName)
	dirName := fmt.Sprintf("test_%s_%d", safeName, timestamp)

	tempDir := filepath.Join("..", "temp", "parallel_tests", dirName)

	// ディレクトリを作成
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	return tempDir, nil
}

// NewParallelProxyController 並行実行用のProxyControllerを作成
func NewParallelProxyController(proxyPath, testName string) (*ProxyController, error) {
	// 利用可能なポートを取得
	port, err := GetAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get available port: %v", err)
	}

	// 一時ディレクトリを作成
	inventoryDir, err := CreateTempInventoryDir(testName)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	return &ProxyController{
		Port:         port,
		ProxyPath:    proxyPath,
		InventoryDir: inventoryDir,
	}, nil
}

// copyInventoryDir インベントリディレクトリをコピー
func copyInventoryDir(srcDir, dstDir string) error {
	// ソースディレクトリが存在することを確認
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", srcDir)
	}

	// 再帰的にファイルをコピー
	return filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 相対パスを計算
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			// ディレクトリを作成
			return os.MkdirAll(dstPath, info.Mode())
		} else {
			// ファイルをコピー
			return copyFile(srcPath, dstPath)
		}
	})
}

// copyFile ファイルをコピー
func copyFile(src, dst string) error {
	// ソースファイルを開く
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 宛先ディレクトリを作成
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// 宛先ファイルを作成
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// ファイル内容をコピー
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MakeProxyRequest プロキシ経由でHTTPリクエストを送信
func MakeProxyRequest(method, urlStr string, headers map[string]string, proxyPort int) (*HTTPResponse, error) {
	// プロキシURLを構築
	proxyURL := fmt.Sprintf("http://localhost:%d", proxyPort)
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %v", err)
	}

	// プロキシ設定付きのHTTPクライアントを作成
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}

	// リクエストを作成
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	// ヘッダーを設定
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// リクエストを送信
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスボディを読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// レスポンスヘッダーをマップに変換
	headers_response := make(map[string]string)
	for name, values := range resp.Header {
		if len(values) > 0 {
			headers_response[name] = values[0]
		}
	}

	return &HTTPResponse{
		StatusCode:      resp.StatusCode,
		Headers:         headers_response,
		Body:            body,
		ContentType:     resp.Header.Get("Content-Type"),
		ContentEncoding: resp.Header.Get("Content-Encoding"),
	}, nil
}

// LoadInventory インベントリファイルを読み込み
func LoadInventory(inventoryDir string) (*Inventory, error) {
	inventoryPath := filepath.Join(inventoryDir, "inventory.json")

	// ファイルの存在確認
	if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
		// デバッグ情報: ディレクトリの内容を確認
		if files, err := os.ReadDir(inventoryDir); err == nil {
			fileList := make([]string, len(files))
			for i, file := range files {
				fileList[i] = file.Name()
			}
			return nil, fmt.Errorf("inventory file not found: %s (directory exists with files: %v)", inventoryPath, fileList)
		}
		return nil, fmt.Errorf("inventory file not found: %s (directory may not exist)", inventoryPath)
	}

	// ファイルを読み込み
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %v", err)
	}

	// JSONを解析
	var inventory Inventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		return nil, fmt.Errorf("failed to parse inventory JSON: %v", err)
	}

	return &inventory, nil
}

// sanitizeFileName ファイル名として安全な文字列に変換
func sanitizeFileName(name string) string {
	safe := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			safe += string(r)
		} else {
			safe += "_"
		}
	}
	return safe
}

// startPlaybackProxyWithArgs starts the playback proxy with custom arguments
func startPlaybackProxyWithArgs(t *testing.T, args ...string) *exec.Cmd {
	proxyPath := "../http-playback-proxy"
	if _, err := os.Stat(proxyPath); os.IsNotExist(err) {
		t.Fatalf("Proxy binary not found at %s", proxyPath)
	}

	// Parse args to separate global options from command-specific options
	var globalArgs []string
	var playbackArgs []string
	isPlaybackArg := false
	
	for i := 0; i < len(args); i++ {
		if args[i] == "--watch" {
			isPlaybackArg = true
			playbackArgs = append(playbackArgs, args[i])
		} else if i < len(args)-1 && (args[i] == "-i" || args[i] == "--inventory-dir" || args[i] == "-p" || args[i] == "--port") {
			// These are global options that come before the command
			globalArgs = append(globalArgs, args[i], args[i+1])
			i++ // Skip the next argument as it's the value
		} else {
			if isPlaybackArg {
				playbackArgs = append(playbackArgs, args[i])
			} else {
				globalArgs = append(globalArgs, args[i])
			}
		}
	}
	
	// Build command line: global args, then "playback", then playback-specific args
	cmdArgs := append(globalArgs, "playback")
	cmdArgs = append(cmdArgs, playbackArgs...)
	
	cmd := exec.Command(proxyPath, cmdArgs...)
	
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}

	return cmd
}

// stopProxy stops a running proxy command
func stopProxy(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGINT)
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited normally
		case <-time.After(3 * time.Second):
			// Force kill after timeout
			cmd.Process.Kill()
			<-done
		}
	}
}
