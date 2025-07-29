package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

var (
	port = flag.Int("port", 8080, "プロキシサーバーのポート番号")
	inventoryDir = flag.String("inventory", "./inventory", "inventoryディレクトリのパス")
	noBeautify = flag.Bool("no-beautify", false, "HTML・CSS・JavaScriptのBeautifyを無効化")
)


func printUsage() {
	fmt.Fprintf(os.Stderr, "使用方法:\n")
	fmt.Fprintf(os.Stderr, "  %s recording <URL> [--port <port>] [--inventory <dir>] [--no-beautify]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s playback [--port <port>] [--inventory <dir>]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n例:\n")
	fmt.Fprintf(os.Stderr, "  %s recording https://www.ideamans.com/ --port 8080 --inventory ./test_inventory\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s recording https://www.ideamans.com/ --port 8080 --inventory ./test_inventory --no-beautify\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s playback --port 8080 --inventory ./test_inventory\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nオプション:\n")
	flag.PrintDefaults()
}

func main() {
	// カスタムUsageを設定
	flag.Usage = printUsage
	
	// フラグを解析
	flag.Parse()
	
	// 引数の解析
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "エラー: モードを指定してください (recording または playback)\n\n")
		printUsage()
		os.Exit(1)
	}
	
	mode := args[0]
	var targetURL string
	
	// 手動で --no-beautify フラグを検出
	for i, arg := range args {
		if arg == "--no-beautify" {
			*noBeautify = true
			// フラグを引数リストから削除
			args = append(args[:i], args[i+1:]...)
			break
		}
	}
	
	// 手動で --inventory フラグを検出
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--inventory" {
			*inventoryDir = args[i+1]
			// フラグと値を引数リストから削除
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	
	// 手動で --port フラグを検出
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--port" {
			if portVal, err := strconv.Atoi(args[i+1]); err == nil {
				*port = portVal
			}
			// フラグと値を引数リストから削除
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	
	
	switch mode {
	case "recording":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "エラー: recordingモードではURLを指定してください\n\n")
			printUsage()
			os.Exit(1)
		}
		targetURL = args[1]
		log.Printf("モード: recording, 対象URL: %s, ポート: %d, inventory: %s, no-beautify: %t", targetURL, *port, *inventoryDir, *noBeautify)
		
		// Start recording mode
		if err := StartRecording(targetURL, *port, *inventoryDir, *noBeautify); err != nil {
			log.Fatalf("Recording mode failed: %v", err)
		}
		
	case "playback":
		log.Printf("モード: playback, ポート: %d, inventory: %s", *port, *inventoryDir)
		
		// Start playback mode
		if err := StartPlayback(*port, *inventoryDir); err != nil {
			log.Fatalf("Playback mode failed: %v", err)
		}
		
	default:
		fmt.Fprintf(os.Stderr, "エラー: 無効なモードです: %s (recording または playback を指定してください)\n\n", mode)
		printUsage()
		os.Exit(1)
	}
}