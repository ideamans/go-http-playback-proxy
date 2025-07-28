.PHONY: build clean run test test-integration help

# デフォルトターゲット
.DEFAULT_GOAL := build

# バイナリ名
BINARY_NAME := http-playback-proxy

# ビルド
build:
	go build -o $(BINARY_NAME) .

# クリーンアップ
clean:
	rm -f $(BINARY_NAME)
	go clean

# プロキシを実行（デフォルトポート8080）
run: build
	./$(BINARY_NAME)

# カスタムポートで実行
run-port:
	@if [ -z "$(PORT)" ]; then echo "Usage: make run-port PORT=9000"; exit 1; fi
	./$(BINARY_NAME) -port $(PORT)

# テスト実行
test: build
	@echo "HTTPリクエストテスト..."
	@./$(BINARY_NAME) & \
	PROXY_PID=$$!; \
	sleep 1; \
	curl -x localhost:8080 http://httpbin.org/get --max-time 10 || true; \
	kill $$PROXY_PID

# 統合テスト実行
test-integration: build
	@echo "統合テストを実行しています..."
	@cd integration && ./run-integration-tests.sh

# CI用統合テスト（パフォーマンステストをスキップ）
test-integration-ci: build
	@echo "CI用統合テストを実行しています（パフォーマンステストをスキップ）..."
	@cd integration && ./run-integration-tests.sh --skip-performance

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

# 依存関係の整理
tidy:
	go mod tidy

# ヘルプ
help:
	@echo "使用可能なターゲット:"
	@echo "  build              - バイナリをビルド"
	@echo "  clean              - ビルド成果物を削除"
	@echo "  run                - プロキシを実行（ポート8080）"
	@echo "  run-port           - カスタムポートで実行 (例: make run-port PORT=9000)"
	@echo "  test               - 簡単なテストを実行"
	@echo "  test-integration   - 統合テストを実行（全テスト）"
	@echo "  test-integration-ci - CI用統合テスト（パフォーマンステストをスキップ）"
	@echo "  setup-integration  - 統合テスト環境をセットアップ"
	@echo "  tidy               - go mod tidyを実行"
	@echo "  help               - このヘルプを表示"