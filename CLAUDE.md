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

#### 3. 圧縮システム (`encoding.go`)

- 4 つの圧縮形式をサポート:
  - **Gzip**: RFC 1952 準拠
  - **Deflate**: RFC 1951 準拠
  - **Brotli**: Google 開発
  - **Zstd**: Facebook 開発（klauspost/compress）
  - **Identity**: 無圧縮パススルー

#### 4. 文字コード変換システム (`charset.go`)

- HTML/CSS の文字コード自動検出・変換
- Content-Type ヘッダーと<meta charset>の両方に対応
- UTF-8 以外のコンテンツを UTF-8 で保存、再生時に復元
- 対応文字コード: Shift_JIS, EUC-JP, ISO-8859-1 等
- 変換失敗時の安全な処理（-failed サフィックス）

#### 5. フォーマット判定システム (`formatting.go`)

- コンテンツタイプの正確な判定
- HTML, CSS, JavaScript の識別
- 文字コード処理が必要なフォーマットの特定

#### 6. リソースパス変換システム (`resource.go`)

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
- `golang.org/x/text`: 文字コード変換
- 標準ライブラリのみで最小構成

### ファイル構成

#### コア機能

- `main.go`: プロキシモード分岐とメイン実装
- `proxy.go`: 共通プロキシ機能と基本ログ処理
- `recording.go`: 録画モード実装（RecordingPlugin）
- `playback.go`: 再生モード実装（PlaybackPlugin、上流プロキシ対応）
- `inventory.go`: データ永続化システム（PersistenceManager、PlaybackManager）
- `types.go`: TypeScript 互換型定義システム
- `resource.go`: URL-ファイルパス変換システム

#### 文字コード処理

- `charset.go`: HTML/CSS の文字コード検出・変換
- `encoding.go`: HTTP Content-Encoding 処理（gzip, deflate, brotli, zstd）
- `formatting.go`: コンテンツフォーマット判定（HTML, CSS, JavaScript 等）

#### テスト・自動化

- `*_test.go`: 包括的テストスイート
- `lighthouse.sh`: Lighthouse パフォーマンステスト自動化スクリプト
- `Makefile`: ビルド自動化
- `integration/`: 統合テストとテストデータ

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

### テスト覆盖

#### 単体テスト

- 圧縮/展開システム（encoding_test.go）
- 文字コード変換（charset_test.go）
- URL-ファイルパス変換（resource_test.go）
- 型システム（types_test.go）
- インベントリ管理（inventory_test.go）
- 録画・再生機能（recording_test.go, playback_test.go）

#### 統合テスト（integration/）

- 実際の HTTP 通信の録画・再生
- 文字コード変換の E2E テスト
- パフォーマンステスト
- 多言語コンテンツ処理

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

- **TTFB**: 実際の記録時間を再現（TTFBMS フィールド）
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
targetOffset := time.Duration(resource.TTFBMS)*time.Millisecond + chunkTime
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

## 文字コード処理システム

### 概要

HTML と CSS の文字コード処理を自動化。録画時は UTF-8 で保存し、再生時に元の文字コードを復元します。

### 実装詳細（`charset.go`）

#### 文字コード検出

1. **HTTP ヘッダー**: `Content-Type: text/html; charset=shift_jis`
2. **HTML メタタグ**: `<meta charset="shift_jis">`
3. **CSS ルール**: `@charset "shift_jis";`

#### 録画時処理

```go
// Content-Encoding 展開後
charset := DetectCharset(contentType, body)
if charset != "" && charset != "utf-8" {
    utf8Body, err := ConvertToUTF8(body, charset)
    if err != nil {
        resource.ContentCharset = charset + "-failed"
    } else {
        body = utf8Body
        resource.ContentCharset = charset
    }
}
```

#### 再生時処理

```go
// Content-Encoding 圧縮前
if resource.ContentCharset != "" && !strings.HasSuffix(resource.ContentCharset, "-failed") {
    originalBody, err := ConvertFromUTF8(body, resource.ContentCharset)
    if err == nil {
        body = originalBody
        // Content-Type ヘッダーも更新
        updateContentTypeCharset(response, resource.ContentCharset)
    }
}
```

### Resource フィールド

- **ContentTypeCharset**: HTTP ヘッダーの charset 値
- **ContentCharset**: 実際に使用する charset（検出結果）

### 対応文字コード

- **Shift_JIS**: 日本語（レガシー）
- **EUC-JP**: 日本語（Unix 系）
- **ISO-8859-1**: 西欧言語
- **UTF-8**: デフォルト

## Make タスク

### 基本タスク

```bash
# ビルド（テストバイナリも生成）
make build

# 単体テスト実行
make test

# 統合テスト環境セットアップ
make setup-integration

# 統合テスト実行
make test-integration

# Lighthouse パフォーマンステスト
make lighthouse
```

### 使用例

```bash
# 完全なテスト実行
make build
make setup-integration
make test
make test-integration
make lighthouse
```

## TODO

- [ ] CI/CD パイプライン構築
- [ ] Docker コンテナ対応
- [ ] WebSocket プロキシ対応
- [ ] HTTP/2 サポート検討
