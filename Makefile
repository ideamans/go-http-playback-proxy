.PHONY: build clean run test help

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

# 依存関係の整理
tidy:
	go mod tidy

# ヘルプ
help:
	@echo "使用可能なターゲット:"
	@echo "  build      - バイナリをビルド"
	@echo "  clean      - ビルド成果物を削除"
	@echo "  run        - プロキシを実行（ポート8080）"
	@echo "  run-port   - カスタムポートで実行 (例: make run-port PORT=9000)"
	@echo "  test       - 簡単なテストを実行"
	@echo "  tidy       - go mod tidyを実行"
	@echo "  help       - このヘルプを表示"