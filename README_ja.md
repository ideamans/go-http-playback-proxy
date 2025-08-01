# Go HTTP Playback Proxy

HTTP/HTTPS 通信の記録・再生機能を持つ Go 言語製 MITM プロキシです。パフォーマンステストや通信解析のためのブラウザプロキシとして機能します。

## 概要

- **MITM プロキシ**: HTTP/HTTPS 通信の完全な監視・記録
- **圧縮保持**: パフォーマンス最適化のためレスポンスの圧縮状態を維持
- **DNS 監視**: DNS 解決プロセスの詳細なログ記録
- **TypeScript 互換**: TypeScript と完全互換の型定義
- **URL-ファイルパス変換**: HTTP リクエストを適切なファイルパスに変換
- **コンテンツ最適化**: HTML/CSS/JavaScript の整形・圧縮機能

## クイックスタート

```bash
# ビルド
make build

# 録画モード - 指定 URL への通信を記録
./http-playback-proxy recording https://www.example.com/

# 再生モード - 記録した通信を再生
./http-playback-proxy playback

# オプション付き実行
./http-playback-proxy --port 8080 --inventory-dir ./inventory recording https://www.example.com/
```

## インストール

### 前提条件

- Go 1.22 以上
- Make（オプション、ビルド自動化用）

### ソースからビルド

```bash
git clone https://github.com/ideamans/go-http-playback-proxy.git
cd go-http-playback-proxy
make build
```

### バイナリのダウンロード

[リリースページ](https://github.com/ideamans/go-http-playback-proxy/releases)から以下のプラットフォーム向けビルド済みバイナリが利用可能です：
- Linux (amd64, arm64)
- macOS (amd64, arm64) 
- Windows (amd64)
- FreeBSD (amd64)

## 使用方法

### コマンドラインオプション

```bash
./http-playback-proxy [オプション] <コマンド>

コマンド:
  recording <url>  指定 URL への通信を記録
  playback        記録した通信を再生

オプション:
  --port, -p          プロキシサーバーのポート番号 (デフォルト: 8080)
  --inventory-dir, -i inventoryディレクトリのパス (デフォルト: ./inventory)
  --log-level, -l     ログレベル (debug, info, warn, error) (デフォルト: info)

録画オプション:
  --no-beautify       HTML/CSS/JavaScript の整形を無効化
```

### ブラウザ設定

ブラウザの HTTP/HTTPS プロキシを `localhost:8080` に設定します。

#### Chrome の例

```bash
google-chrome --proxy-server=localhost:8080 --ignore-certificate-errors --ignore-ssl-errors
```

### 録画モード

指定した URL パターンへのすべての HTTP/HTTPS 通信を記録します：

```bash
./http-playback-proxy recording https://www.example.com/
```

記録データの保存先：
```
./inventory/
├── inventory.json     # リソースのメタデータとドメイン情報
└── contents/          # デコードされたレスポンスボディ
    └── get/https/example.com/index.html
```

### 再生モード

記録した通信を正確なタイミングで再生します：

```bash
./http-playback-proxy playback
```

特徴：
- オリジナルの TTFB（Time To First Byte）を保持
- 転送速度（Mbps）を維持
- レスポンスに `x-playback-proxy: 1` ヘッダーを追加
- 未記録のリクエストは上流プロキシにフォールバック

## 機能

### コンテンツエンコーディング対応

複数の圧縮形式をサポート：
- **Gzip**: RFC 1952 準拠
- **Deflate**: RFC 1951 準拠
- **Brotli**: Google の圧縮アルゴリズム
- **Zstd**: Facebook の Zstandard 圧縮
- **Identity**: 無圧縮パススルー

### 文字エンコーディング対応

文字エンコーディングの自動検出と変換：
- HTTP ヘッダーと HTML メタタグから charset を検出
- 保存時に UTF-8 に変換
- 再生時に元のエンコーディングを復元
- 対応: Shift_JIS, EUC-JP, ISO-8859-1, UTF-8

### コンテンツ最適化

オプションの整形と圧縮：
- **HTML**: gohtml による整形
- **CSS**: 手動インデント整形
- **JavaScript**: jsbeautifier-go による整形
- **圧縮**: tdewolff/minify による全形式の圧縮

### URL-ファイルパス変換

URL とファイルパス間のインテリジェントな変換：

```
GET https://example.com/api?user=123&action=view
→ get/https/example.com/api/index~user=123&action=view.html

GET https://example.com/image.jpg?param=value
→ get/https/example.com/image~param=value.jpg
```

特徴：
- ディレクトリパスは自動的に `/index.html` を付与
- クエリパラメータは `~` 区切りで保持
- 長いパラメータ（32文字超）は SHA1 でハッシュ化
- 国際文字の完全な Unicode サポート

## パフォーマンス

### 接続最適化

- TLS セッションキャッシュ（256 エントリ）
- 接続プーリング（ホストあたり 10 接続）
- TCP_NODELAY 有効
- Keep-Alive 最適化

### 圧縮処理

- オリジナルの圧縮を保持
- `DisableCompression=true` で自動展開を防止
- 圧縮状態を維持して CPU オーバーヘッドを削減

### 正確な再生タイミング

- TTFB を正確に記録・再生
- オリジナルの転送速度（Mbps）を維持
- リアルなネットワーク動作のためのチャンクベースタイミング

## 開発

### テスト

```bash
# 単体テスト実行
make test

# 統合テスト実行
make test-integration

# すべてのテスト実行
make test-all

# Lighthouse によるパフォーマンステスト
make lighthouse
```

### API 使用例

```go
// URL をファイルパスに変換
filePath, err := MethodURLToFilePath("GET", "https://example.com/api?user=123")
// → "get/https/example.com/api/index~user=123.html"

// ファイルパスを URL に逆変換
method, url, err := FilePathToMethodURL("get/https/example.com/api/index~user=123.html")
// → "GET", "https://example.com/api?user=123"
```

## CI/CD

GitHub Actions ワークフロー：
- **CI**: main/develop ブランチへのプッシュ時にテスト実行
- **リリース**: バージョンタグ時に GoReleaser で自動リリース

## 制限事項

- 開発・テスト用途向けに設計
- 自己署名証明書を使用（本番環境非推奨）
- 互換性のため HTTP/2 は無効化
- WebSocket はまだ未対応

## コントリビューション

1. リポジトリをフォーク
2. フィーチャーブランチを作成（`git checkout -b feature/amazing-feature`）
3. 変更をコミット（`git commit -m 'Add some amazing feature'`）
4. ブランチにプッシュ（`git push origin feature/amazing-feature`）
5. プルリクエストを作成

## ライセンス

このプロジェクトは MIT ライセンスの下でライセンスされています。詳細は LICENSE ファイルを参照してください。

## 謝辞

- MITM プロキシ機能は [go-mitmproxy](https://github.com/lqqyt2423/go-mitmproxy) をベースに構築
- 圧縮、エンコーディング、最適化に優れた Go ライブラリを使用