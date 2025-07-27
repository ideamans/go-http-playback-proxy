# Go HTTP Playback Proxy

Go 言語製の MITM プロキシです。HTTP/HTTPS 通信の記録・再生機能を持ち、ブラウザのプロキシとして機能します。

## 概要

- **MITM プロキシ**: HTTP/HTTPS 通信を完全に監視・記録
- **圧縮保持**: レスポンスの圧縮状態を維持してパフォーマンス最適化
- **DNS 監視**: DNS 解決プロセスの詳細ログ
- **TypeScript 互換**: 型定義が TypeScript と完全互換
- **URL-ファイルパス変換**: HTTP リクエストを適切なファイルパスに変換

## アーキテクチャ

### プロキシモード

- **recording**: 指定 URL への通信を記録
- **playback**: 記録した通信を再生

### コア機能

#### 1. MITM プロキシ (`main.go`)

- HTTP/1.1 強制で HTTP/2 エラーを回避
- 自己署名証明書による HTTPS 対応
- SSL/TLS エラー完全無視
- 接続プール・Keep-Alive 最適化
- DNS 解決時間と IP アドレスの詳細ログ

#### 2. 型定義システム (`types.go`)

- TypeScript との完全互換型定義
- JSON シリアライゼーション対応（選択的）
- 以下の主要型をサポート:
  - `Resource`: HTTP リソースの詳細情報（JSON 対応、Mbps 転送速度含む）
  - `Domain`: ドメイン名と IP アドレス（JSON 対応）
  - `Inventory`: リソースとドメインのコレクション（JSON 対応）
  - `RecordingTransaction`: 記録用の HTTP トランザクション型（JSON 非対応）
  - `PlaybackTransaction`: 再生用トランザクション型（JSON 非対応）
  - `BodyChunk`: レスポンスボディのチャンク（JSON 非対応、TargetOffset 付き）

#### 3. 圧縮システム (`coding.go`)

- 6 つの圧縮形式をサポート:
  - **Gzip**: RFC 1952 準拠 (46.55%圧縮率)
  - **Deflate**: RFC 1951 準拠 (40.00%圧縮率)
  - **Brotli**: Google 開発 (35.27%圧縮率 - 最高効率)
  - **Zstd**: Facebook 開発 (43.64%圧縮率)
  - **Compress**: Unix LZW (68.00%圧縮率)
  - **Identity**: 無圧縮パススルー

#### 4. リソースパス変換システム (`resource.go`)

- HTTP メソッド・URL をファイルパスに変換
- 逆変換でファイルパスから HTTP リクエストを復元
- URL パラメータの適切なエンコーディング
- 日本語・多言語パラメータ完全対応
- 長いパラメータの SHA1 ハッシュ化

**変換例:**

```
GET https://example.com/api?user=123&action=view
→ get/https/example.com/api/index~user=123&action=view.html

GET https://example.com/search?q=東京&lang=ja
→ get/https/example.com/search/index~q=%E6%9D%B1%E4%BA%AC&lang=ja.html

GET https://example.com/image.jpg?param=value
→ get/https/example.com/image~param=value.jpg
```

**特徴:**

- ディレクトリパスは自動的に `/index.html` を付与
- URL パラメータは `~` で区切り、`index.html` のファイル名部分に埋め込み
- `=` と `&` はエンコードせず可読性を維持
- スペースは `%20`、日本語は適切な percent encoding
- 32 文字を超える長いパラメータは SHA1 ハッシュで短縮

## 使用方法

### 基本起動

```bash
# ビルド
make build

# recordingモード（指定URLへの通信を ./inventory に記録）
./http-playback-proxy --port 8080 recording https://www.example.com/

# playbackモード（./inventory から再生、未記録は上流プロキシ）
./http-playback-proxy --port 8080 playback
```

### プロキシ設定

ブラウザの HTTP/HTTPS プロキシを `localhost:8080` に設定

### Chrome 起動例

```bash
google-chrome --proxy-server=localhost:8080 --ignore-certificate-errors --ignore-ssl-errors
```

### URL-ファイルパス変換 API

```go
// URL をファイルパスに変換
filePath, err := MethodURLToFilePath("GET", "https://example.com/api?user=123")
// → "get/https/example.com/api/index~user=123.html"

// ファイルパスを URL に逆変換
method, url, err := FilePathToMethodURL("get/https/example.com/api/index~user=123.html")
// → "GET", "https://example.com/api?user=123"

// 安全なファイルパスに変換（Windows互換）
safePath, err := GetResourceFilePath("GET", "https://example.com/CON.txt")
// → "get/https/example.com/_CON.txt"
```

## パフォーマンス最適化

### 圧縮保持

- `DisableCompression=true` で Transport の自動展開を無効化
- Content-Encoding ヘッダーの強制保持
- 圧縮データをそのまま転送して CPU 負荷軽減

### 接続最適化

- TLS セッションキャッシュ (256 エントリ)
- 接続プール (ホスト毎 10 接続)
- TCP_NODELAY 有効化

### DNS 最適化

- DNS 解決時間の詳細ログ
- 接続タイムアウト短縮 (5 秒)

## ログ出力例

```
[REQUEST] GET https://example.com/api
[DNS] Resolving example.com
[DNS] Resolved example.com -> 192.168.1.1:443 (took 45ms)
[RESPONSE] GET https://example.com/api 200 OK (Proto: HTTP/1.1)
[COMPRESSION] Original Content-Encoding: gzip
[COMPRESSION] Final Content-Encoding: gzip
[SIZE] Content-Length: 1234 bytes
```

## 技術仕様

### 依存関係

- `github.com/lqqyt2423/go-mitmproxy/proxy`: MITM プロキシ基盤
- `github.com/andybalholm/brotli`: Brotli 圧縮
- `github.com/klauspost/compress`: Zstd 圧縮

### ファイル構成

- `main.go`: プロキシモード分岐とメイン実装
- `proxy.go`: 共通プロキシ機能と基本ログ処理
- `recording.go`: 録画モード実装（RecordingPlugin）
- `playback.go`: 再生モード実装（PlaybackPlugin、上流プロキシ対応）
- `inventory.go`: データ永続化システム（PersistenceManager、PlaybackManager）
- `types.go`: TypeScript 互換型定義システム
- `coding.go`: マルチフォーマット圧縮/展開システム
- `resource.go`: URL-ファイルパス変換システム
- `lighthouse.sh`: Lighthouse パフォーマンステスト自動化スクリプト
- `*_test.go`: 包括的テストスイート
- `Makefile`: ビルド自動化

### プロトコル対応

- HTTP/1.1 (HTTP/2 は安定性のため無効化)
- TLS 1.2/1.3 with セッション再利用
- IPv4/IPv6 DNS 解決

### URL パラメータ処理

- パラメータ長制限: 32 文字（設定可能）
- ハッシュアルゴリズム: SHA1 + Base64
- ハッシュ長: 8 文字（設定可能）
- エンコーディング: UTF-8 percent encoding
- 対応文字: 日本語、中国語、韓国語等のマルチバイト文字

### テスト覆盖率

- 33 個のテストケース
- 圧縮システム: 6 つの形式 × 複数レベル
- URL 変換システム: 基本・パラメータ・国際化・逆変換
- 型システム: JSON シリアライゼーション

### 制限事項

- 開発・テスト用途を想定
- 自己署名証明書使用のため本番環境非推奨
- HTTP/2 は互換性のため無効化

## 録画・再生システム

### 録画モード（Recording Mode）

**実装ファイル**: `recording.go`

- `RecordingPlugin` による HTTP トランザクション記録
- `PersistenceManager` でインベントリ永続化
- `RecordingTransaction` 型でリクエスト・レスポンス情報を保存
- 自動 Mbps 計算（転送時間とデータサイズから算出）
- Ctrl+C でシグナル処理によるインベントリ保存
- 圧縮されたレスポンスを自動デコードして保存

**保存構造**:

```
./inventory/
├── inventory.json     # Resource配列とDomain配列のメタデータ
└── contents/          # デコードされたレスポンスボディ
    └── get/https/example.com/index.html
```

### 再生モード（Playback Mode）

**実装ファイル**: `playback.go`

- `PlaybackPlugin` による高速トランザクション再生
- `PlaybackManager` でインベントリ読み込み・チャンク生成
- `PlaybackTransaction` 型で時間精度の高い再生制御
- 未記録リクエストは上流プロキシで透過転送
- `x-playback-proxy: 1` ヘッダー付与で再生判別

**時間制御システム**:

- **TTFB**: 実際の記録時間を再現（TTFBMs フィールド）
- **チャンク送信**: `TargetOffset` による精密タイミング制御
- **転送速度**: 記録時の Mbps で実際のネットワーク速度を再現

### チャンク時間計算アルゴリズム

**録画時の Mbps 計算** (`inventory.go:103-114`):

```go
transferDuration := transaction.ResponseFinished.Sub(transaction.ResponseStarted)
totalBits := float64(len(transaction.Body) * 8)
transferSeconds := transferDuration.Seconds()
mbpsValue := totalBits / (transferSeconds * 1024 * 1024)
```

**再生時のチャンク時間計算** (`inventory.go:407-413`):

```go
// チャンクの進行率計算
chunkProgress := float64(end) / float64(totalSize)
chunkTime := time.Duration(float64(totalTransferTime) * chunkProgress)

// TTFB + チャンク時間 = 絶対送信タイミング
targetOffset := time.Duration(resource.TTFBMs)*time.Millisecond + chunkTime
```

**再生時の待機制御** (`playback.go:156-189`):

```go
targetSendTime = requestStartTime.Add(chunk.TargetOffset)
if now.Before(targetSendTime) {
    waitTime := targetSendTime.Sub(now)
    time.Sleep(waitTime)
}
```

### データフロー

1. **録画**: `RecordingTransaction` → `Resource` + 計算済み Mbps → `inventory.json`
2. **再生**: `inventory.json` → `Resource` → `PlaybackTransaction` + チャンク + `TargetOffset`
3. **送信**: リクエスト開始時刻 + `TargetOffset` = 精密送信タイミング

この実装により、元の HTTP 通信のパフォーマンス特性を正確に再現できます。

# TODO

- [ ] optimize の recording と playback への組み込み
- [ ] charset の自動制御
- [ ] CI/CD
- [ ] 結合テスト
