.PHONY: build test setup-integration test-integration lighthouse clean

# デフォルトターゲット
.DEFAULT_GOAL := build

# バイナリ名
BINARY_NAME := http-playback-proxy

# ビルド
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/http-playback-proxy
	@if [ -d "./cmd/http-playback-proxy" ]; then \
		go test -c -o $(BINARY_NAME).test ./cmd/http-playback-proxy 2>/dev/null || true; \
	fi

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
test-integration:
	@echo "統合テストを実行しています..."
	@echo "Current directory: $$(pwd)"
	@echo "Directory contents:"
	@ls -la
	@echo "Building binary first..."
	$(MAKE) build
	@cd integration && ./run-integration-tests.sh --skip-setup --basic-only --verbose

# 統合テスト実行（全テスト）
test-integration-all:
	@echo "全ての統合テストを実行しています..."
	@echo "Current directory: $$(pwd)"
	@echo "Building binary first..."
	$(MAKE) build
	@cd integration && ./run-integration-tests.sh --skip-setup --verbose

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