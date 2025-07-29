package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
	
	switch mode {
	case "recording":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "エラー: recordingモードではURLを指定してください\n\n")
			printUsage()
			os.Exit(1)
		}
		targetURL = args[1]
		log.Printf("モード: recording, 対象URL: %s, ポート: %d, inventory: %s", targetURL, *port, *inventoryDir)
		
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