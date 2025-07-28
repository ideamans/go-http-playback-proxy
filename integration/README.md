# HTTP Playback Proxy 統合テスト

このディレクトリには、HTTP Playback Proxy の包括的な統合テストが含まれています。

## 概要

統合テストは以下の3つのフェーズで構成されています：

1. **直接アクセス** - プロキシなしでテストサーバーにアクセスし、基準値を取得
2. **Recording** - プロキシ経由でアクセスし、通信をinventoryに記録
3. **Playback** - inventoryから通信を再生し、結果を検証

## ディレクトリ構造

```
integration/
├── README.md                    # このファイル
├── setup-testdata.sh           # テストデータ生成スクリプト
├── run-integration-tests.sh    # 統合テスト実行スクリプト
├── testdata/                   # テストデータ
│   ├── images/                 # 各種画像ファイル（小・中・大サイズ）
│   ├── html/                   # 多言語HTMLファイル
│   ├── css/                    # 文字コード別CSSファイル
│   ├── js/                     # JavaScriptファイル
│   ├── api/                    # JSON/XMLデータ
│   ├── sample.pdf              # PDFテストファイル
│   └── sample.zip              # ZIPテストファイル
├── testserver/                 # テスト用Webサーバー
│   ├── main.go                 # サーバー実装
│   └── go.mod                  # 依存関係
├── tests/                      # テストコード
│   ├── charset_compression_test.go  # 文字コード・圧縮テスト
│   ├── url_pattern_test.go         # URLパターンテスト
│   ├── performance_test.go         # パフォーマンステスト
│   └── go.mod                      # テストモジュール
└── temp/                       # 一時ファイル（テスト実行時に作成）
```

## テストカテゴリ

### 1. 文字コード・圧縮テスト（並行実行）

**対象**: `charset_compression_test.go`

- **文字コード処理**:
  - UTF-8, Shift_JIS, EUC-JP, ISO-8859-1
  - recording時のUTF-8変換
  - playback時の元文字コード復元
  - Content-Typeヘッダーの正確な再現

- **圧縮形式**:
  - gzip, brotli, deflate, zstd, compress, identity
  - 圧縮状態の保持
  - Content-Encodingヘッダーの処理

### 2. URLパターンテスト（並行実行）

**対象**: `url_pattern_test.go`

- **URLパス変換**:
  - 基本パス、深い階層、拡張子なし
  - 日本語パス・ファイル名
  - 特殊文字・スペース含有

- **クエリパラメータ**:
  - 短いパラメータ（そのまま保存）
  - 長いパラメータ（SHA1ハッシュ化）
  - 日本語・特殊文字パラメータ
  - URLエンコーディング処理

- **HTTPメソッド**:
  - GET, POST, PUT, DELETE, HEAD, OPTIONS

### 3. パフォーマンステスト（直列実行）

**対象**: `performance_test.go`

- **ファイルサイズ**:
  - 小（1KB）、中（100KB）、大（10MB）

- **TTFB制御**:
  - 0ms（即座）〜1000ms（高遅延）

- **転送速度制御**:
  - 100Kbps（低速）〜10Mbps（高速）、無制限

- **検証項目**:
  - TTFBの再現精度（±100ms）
  - 転送速度の再現精度（±20%）
  - チャンク送信タイミングの正確性

## テスト用Webサーバー

**ポート**: 9999  
**機能**:

- **パフォーマンス制御**:
  - `?ttfb=100` - TTFB遅延（ミリ秒）
  - `?speed=1000` - 転送速度制限（Kbps）
  - `?compression=gzip` - 圧縮形式指定

- **エンドポイント**:
  - `/` - インデックスページ
  - `/api/*` - JSON/XMLデータ
  - `/images/*` - 各種画像ファイル
  - `/html/*` - HTML（文字コード別）
  - `/css/*` - CSS（文字コード別）
  - `/js/*` - JavaScript
  - `/performance/*` - パフォーマンステスト用
  - `/status/*` - HTTPステータスコードテスト
  - `/charset/*` - 文字コードテスト

## 実行方法

### 1. 事前準備

```bash
# ImageMagickインストール（テストデータ生成用）
brew install imagemagick

# テストデータ生成
cd integration
./setup-testdata.sh
```

### 2. 全テスト実行

```bash
# 全テスト実行
./run-integration-tests.sh

# オプション指定
./run-integration-tests.sh --verbose
./run-integration-tests.sh --skip-performance
./run-integration-tests.sh --skip-charset --skip-url-patterns
```

### 3. 個別テスト実行

```bash
# テストサーバー起動
cd testserver
go run main.go 9999

# 別ターミナルでテスト実行
cd tests
go test -v -run TestCharsetAndCompression
go test -v -run TestURLPatterns
go test -v -run TestPerformance
```

## テスト条件・因子

### プロトコル条件

- **HTTPステータス**: 200, 301, 302, 404, 500
- **HTTPメソッド**: GET, POST, PUT, DELETE, HEAD, OPTIONS
- **ヘッダー**: Content-Type, Content-Encoding, Content-Length, Transfer-Encoding

### コンテンツ条件

- **ファイル形式**: HTML, CSS, JS, JSON, XML, PDF, ZIP, Images
- **文字コード**: UTF-8, Shift_JIS, EUC-JP, ISO-8859-1
- **圧縮形式**: gzip, brotli, deflate, zstd, compress, identity

### パフォーマンス条件

- **TTFB**: 0ms 〜 1000ms
- **転送速度**: 100Kbps 〜 10Mbps, 無制限
- **ファイルサイズ**: 1KB 〜 10MB

### URL条件

- **パス**: ルート、深い階層、拡張子なし、日本語、特殊文字
- **パラメータ**: なし、短い、長い（ハッシュ化）、日本語、特殊文字

## 期待される結果

### Recording検証

- ✅ ファイルが正しいパスに保存される
- ✅ 文字コードがUTF-8に変換される
- ✅ 圧縮が正しく解除される
- ✅ inventory.jsonにメタデータが記録される
- ✅ Mbps計算が正確に行われる

### Playback検証

- ✅ 元のレスポンスと内容が完全一致する
- ✅ 文字コードが元の形式に復元される
- ✅ 圧縮が正しく適用される
- ✅ TTFBとチャンク送信タイミングが再現される
- ✅ 全HTTPヘッダーが正確に再現される
- ✅ `x-playback-proxy: 1`ヘッダーが付与される

### エラーハンドリング

- ✅ 文字コード変換失敗時の`-failed`サフィックス
- ✅ 未記録リクエストの上流プロキシフォールバック
- ✅ 適切なタイムアウト処理

## 注意事項

- パフォーマンステストは性能測定の正確性のため**直列実行**
- 他のテストは効率性のため**並行実行**
- テスト実行時に一時ディレクトリ（`temp/`）が作成される
- プロセス終了時に自動クリーンアップが実行される
- ImageMagickが必要（テストデータ生成用）

## トラブルシューティング

### テスト失敗時

1. **ログ確認**:
   ```bash
   cat integration/temp/testserver.log
   cat integration/temp/proxy.log  # 存在する場合
   ```

2. **個別実行**:
   ```bash
   cd integration/tests
   go test -v -run TestSpecificTest
   ```

3. **デバッグ出力**:
   ```bash
   ./run-integration-tests.sh --verbose
   ```

### 環境問題

- Go 1.21+ が必要
- ImageMagick インストール確認: `magick --version`
- ポート9999, 8080が使用可能であること
- 十分なディスク容量（テストデータ: ~15MB、ログ: ~数MB）