.PHONY: build test setup-integration test-integration lighthouse clean

# デフォルトターゲット
.DEFAULT_GOAL := build

# バイナリ名
BINARY_NAME := http-playback-proxy

# ビルド
build:
	go build -o $(BINARY_NAME) ./cmd/http-playback-proxy
	go test -c -o $(BINARY_NAME).test ./cmd/http-playback-proxy

# テスト実行
test:
	go test -v ./...

# 統合テストの事前準備
setup-integration:
	@echo "統合テスト環境をセットアップしています..."
	@if ! command -v magick >/dev/null 2>&1; then \
		echo "Error: ImageMagickが必要です。以下でインストールしてください:"; \
		echo "  macOS: brew install imagemagick"; \
		echo "  Ubuntu: apt-get install imagemagick"; \
		echo "  CentOS/RHEL: yum install ImageMagick"; \
		exit 1; \
	fi
	@cd integration && ./setup-testdata.sh
	@echo "統合テスト環境のセットアップが完了しました"

# 統合テスト実行
test-integration: build
	@echo "統合テストを実行しています..."
	@cd integration && ./run-integration-tests.sh --skip-setup --basic-only

# Lighthouse パフォーマンステスト
lighthouse: build
	@echo "Lighthouseテストを実行しています..."
	@./lighthouse.sh

# クリーンアップ
clean:
	@echo "ビルド成果物をクリーンアップしています..."
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME).test
	@echo "クリーンアップが完了しました"