#!/bin/bash

# 統合テスト実行スクリプト
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TESTSERVER_DIR="$SCRIPT_DIR/testserver"
TESTS_DIR="$SCRIPT_DIR/tests"
TEMP_DIR="$SCRIPT_DIR/temp"

# 色付きログ用
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# クリーンアップ関数
cleanup() {
    log_info "Cleaning up..."
    
    # プロセス終了
    if [ ! -z "$TESTSERVER_PID" ]; then
        log_info "Stopping test server (PID: $TESTSERVER_PID)"
        kill $TESTSERVER_PID 2>/dev/null || true
        wait $TESTSERVER_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$PROXY_PID" ]; then
        log_info "Stopping proxy server (PID: $PROXY_PID)"
        kill $PROXY_PID 2>/dev/null || true
        wait $PROXY_PID 2>/dev/null || true
    fi
    
    # 一時ディレクトリ削除
    if [ -d "$TEMP_DIR" ]; then
        log_info "Removing temporary directory: $TEMP_DIR"
        rm -rf "$TEMP_DIR"
    fi
    
    # コピーしたプロキシバイナリを削除
    if [ -f "$SCRIPT_DIR/http-playback-proxy" ]; then
        log_info "Removing copied proxy binary"
        rm -f "$SCRIPT_DIR/http-playback-proxy"
    fi
}

# シグナルハンドラー設定
trap cleanup EXIT INT TERM

# 引数解析
RUN_PERFORMANCE=true
RUN_CHARSET=true
RUN_URL_PATTERNS=true
RUN_BASIC=true
VERBOSE=false
SKIP_SETUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-performance)
            RUN_PERFORMANCE=false
            shift
            ;;
        --skip-charset)
            RUN_CHARSET=false
            shift
            ;;
        --skip-url-patterns)
            RUN_URL_PATTERNS=false
            shift
            ;;
        --basic-only)
            RUN_PERFORMANCE=false
            RUN_CHARSET=false
            RUN_URL_PATTERNS=false
            shift
            ;;
        --skip-setup)
            SKIP_SETUP=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --skip-performance   Skip performance tests"
            echo "  --skip-charset       Skip charset/compression tests"
            echo "  --skip-url-patterns  Skip URL pattern tests"
            echo "  --basic-only         Run only basic functionality tests"
            echo "  --skip-setup         Skip test data setup (for CI)"
            echo "  --verbose, -v        Verbose output"
            echo "  --help, -h           Show this help"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

log_info "Starting integration tests..."
log_info "Project root: $PROJECT_ROOT"
log_info "Test server dir: $TESTSERVER_DIR"
log_info "Tests dir: $TESTS_DIR"

# 環境チェック
log_info "Checking environment..."

if ! command -v go &> /dev/null; then
    log_error "Go is not installed"
    exit 1
fi

if [ ! -f "$PROJECT_ROOT/cmd/http-playback-proxy/main.go" ]; then
    log_error "Main proxy project not found at $PROJECT_ROOT/cmd/http-playback-proxy"
    exit 1
fi

# テストデータ準備
if [ "$SKIP_SETUP" = false ]; then
    log_info "Setting up test data..."
    if [ ! -d "$SCRIPT_DIR/testdata" ]; then
        log_info "Generating test data..."
        cd "$SCRIPT_DIR"
        ./setup-testdata.sh
        cd - > /dev/null
    else
        log_info "Test data already exists"
    fi
else
    log_info "Skipping test data setup (--skip-setup)"
fi

# 一時ディレクトリ作成
mkdir -p "$TEMP_DIR"

# プロキシプロジェクトのビルド
log_info "Building main proxy..."
cd "$PROJECT_ROOT"
if [ "$VERBOSE" = true ]; then
    go build -o "$TEMP_DIR/http-playback-proxy" ./cmd/http-playback-proxy
else
    go build -o "$TEMP_DIR/http-playback-proxy" ./cmd/http-playback-proxy > /dev/null 2>&1
fi

if [ $? -ne 0 ]; then
    log_error "Failed to build main proxy"
    exit 1
fi
log_success "Main proxy built successfully"

# Copy proxy binary to integration directory for tests
log_info "Copying proxy binary to integration directory..."
cp "$TEMP_DIR/http-playback-proxy" "$SCRIPT_DIR/http-playback-proxy"
if [ $? -ne 0 ]; then
    log_error "Failed to copy proxy binary"
    exit 1
fi

# テストサーバーのビルドと起動
log_info "Building and starting test server..."
cd "$TESTSERVER_DIR"

# go.modのダウンロード
if [ "$VERBOSE" = true ]; then
    go mod download
else
    go mod download > /dev/null 2>&1
fi

# テストサーバービルド
if [ "$VERBOSE" = true ]; then
    go build -o "$TEMP_DIR/testserver" .
else
    go build -o "$TEMP_DIR/testserver" . > /dev/null 2>&1
fi

if [ $? -ne 0 ]; then
    log_error "Failed to build test server"
    exit 1
fi

# テストサーバー起動
log_info "Starting test server on port 9999..."
"$TEMP_DIR/testserver" 9999 > "$TEMP_DIR/testserver.log" 2>&1 &
TESTSERVER_PID=$!

# サーバー起動待ち
sleep 3
if ! kill -0 $TESTSERVER_PID 2>/dev/null; then
    log_error "Test server failed to start"
    cat "$TEMP_DIR/testserver.log"
    exit 1
fi

# ヘルスチェック
log_info "Checking test server health..."
for i in {1..10}; do
    if curl -s http://localhost:9999/ > /dev/null 2>&1; then
        log_success "Test server is ready"
        break
    fi
    if [ $i -eq 10 ]; then
        log_error "Test server health check failed"
        exit 1
    fi
    sleep 1
done

# Go テストモジュール初期化
cd "$TESTS_DIR"
if [ ! -f go.mod ]; then
    log_info "Initializing test module..."
    go mod init integration-tests > /dev/null 2>&1
fi

# 依存関係を更新
go mod tidy > /dev/null 2>&1

# テストキャッシュをクリア
log_info "Clearing test cache..."
go clean -testcache

# テスト実行
log_info "Running integration tests..."

TEST_FLAGS=""
if [ "$VERBOSE" = true ]; then
    TEST_FLAGS="-v"
fi

FAILED_TESTS=""

# 基本テスト
log_info "Running basic functionality tests..."
if go test $TEST_FLAGS -timeout 60s -run TestBasicFunctionality .; then
    log_success "Basic functionality tests passed"
else
    log_error "Basic functionality tests failed"
    FAILED_TESTS="$FAILED_TESTS basic"
fi

# Watchモードテスト
if [ "$RUN_BASIC" = true ]; then
    log_info "Running watch mode tests..."
    if go test $TEST_FLAGS -timeout 60s -run TestPlaybackWatchMode .; then
        log_success "Watch mode tests passed"
    else
        log_error "Watch mode tests failed"
        FAILED_TESTS="$FAILED_TESTS watch"
    fi
fi

# 包括的統合テスト（3段階: 直接・recording・playback）
if [ "$RUN_CHARSET" = true ] || [ "$RUN_URL_PATTERNS" = true ]; then
    log_info "Running comprehensive integration tests..."
    if go test $TEST_FLAGS -timeout 300s -run TestComprehensiveIntegration .; then
        log_success "Comprehensive integration tests passed"
    else
        log_error "Comprehensive integration tests failed"
        FAILED_TESTS="$FAILED_TESTS comprehensive"
    fi
fi

# パフォーマンステスト（直列・3段階）
if [ "$RUN_PERFORMANCE" = true ]; then
    log_info "Running performance tests (sequential)..."
    log_warning "Performance tests may take several minutes..."
    if go test $TEST_FLAGS -timeout 600s -run TestPerformanceComprehensive .; then
        log_success "Performance tests passed"
    else
        log_error "Performance tests failed"
        FAILED_TESTS="$FAILED_TESTS performance"
    fi
fi


# 結果サマリー
echo ""
log_info "=========================================="
log_info "Integration Test Results"
log_info "=========================================="

if [ -z "$FAILED_TESTS" ]; then
    log_success "All tests passed!"
    exit 0
else
    log_error "Some tests failed: $FAILED_TESTS"
    echo ""
    log_info "Check logs for details:"
    echo "  Test server log: $TEMP_DIR/testserver.log"
    if [ -f "$TEMP_DIR/proxy.log" ]; then
        echo "  Proxy log: $TEMP_DIR/proxy.log"
    fi
    exit 1
fi