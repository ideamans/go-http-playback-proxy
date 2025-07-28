package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// テスト用のサンプルデータ
var testData = []byte("Hello, World! This is a test string for compression and decompression. " +
	"It contains repeated words to make compression more effective. Hello, World! Hello, World! " +
	"This is a test string for compression and decompression. This is a test string for compression and decompression.")

func TestGzipEncodeDecode(t *testing.T) {
	encoder := NewGzipEncoder(6)
	decoder := NewGzipDecoder()

	// エンコード
	compressed, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Gzip encoding failed: %v", err)
	}

	// エンコード結果の確認
	if len(compressed) >= len(testData) {
		t.Logf("Warning: Compressed data (%d bytes) is not smaller than original (%d bytes)", len(compressed), len(testData))
	}

	// デコード
	decompressed, err := decoder.Decode(compressed)
	if err != nil {
		t.Fatalf("Gzip decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, decompressed) {
		t.Errorf("Decompressed data does not match original")
	}

	t.Logf("Gzip: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
		len(testData), len(compressed), float64(len(compressed))/float64(len(testData))*100)
}

func TestDeflateEncodeDecode(t *testing.T) {
	encoder := NewDeflateEncoder(6)
	decoder := NewDeflateDecoder()

	// エンコード
	compressed, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Deflate encoding failed: %v", err)
	}

	// デコード
	decompressed, err := decoder.Decode(compressed)
	if err != nil {
		t.Fatalf("Deflate decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, decompressed) {
		t.Errorf("Decompressed data does not match original")
	}

	t.Logf("Deflate: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
		len(testData), len(compressed), float64(len(compressed))/float64(len(testData))*100)
}

func TestBrotliEncodeDecode(t *testing.T) {
	encoder := NewBrotliEncoder(6)
	decoder := NewBrotliDecoder()

	// エンコード
	compressed, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Brotli encoding failed: %v", err)
	}

	// デコード
	decompressed, err := decoder.Decode(compressed)
	if err != nil {
		t.Fatalf("Brotli decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, decompressed) {
		t.Errorf("Decompressed data does not match original")
	}

	t.Logf("Brotli: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
		len(testData), len(compressed), float64(len(compressed))/float64(len(testData))*100)
}

func TestZstdEncodeDecode(t *testing.T) {
	encoder := NewZstdEncoder(3)
	decoder := NewZstdDecoder()

	// エンコード
	compressed, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Zstd encoding failed: %v", err)
	}

	// デコード
	decompressed, err := decoder.Decode(compressed)
	if err != nil {
		t.Fatalf("Zstd decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, decompressed) {
		t.Errorf("Decompressed data does not match original")
	}

	t.Logf("Zstd: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
		len(testData), len(compressed), float64(len(compressed))/float64(len(testData))*100)
}

func TestCompressEncodeDecode(t *testing.T) {
	encoder := NewCompressEncoder()
	decoder := NewCompressDecoder()

	// エンコード
	compressed, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Compress encoding failed: %v", err)
	}

	// デコード
	decompressed, err := decoder.Decode(compressed)
	if err != nil {
		t.Fatalf("Compress decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, decompressed) {
		t.Errorf("Decompressed data does not match original")
	}

	t.Logf("Compress: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
		len(testData), len(compressed), float64(len(compressed))/float64(len(testData))*100)
}

func TestIdentityEncodeDecode(t *testing.T) {
	encoder := NewIdentityEncoder()
	decoder := NewIdentityDecoder()

	// エンコード（何もしない）
	encoded, err := encoder.Encode(testData)
	if err != nil {
		t.Fatalf("Identity encoding failed: %v", err)
	}

	// デコード（何もしない）
	decoded, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Identity decoding failed: %v", err)
	}

	// 結果の確認
	if !bytes.Equal(testData, encoded) || !bytes.Equal(testData, decoded) {
		t.Errorf("Identity encoding/decoding should not modify data")
	}

	t.Logf("Identity: All data sizes remain %d bytes", len(testData))
}

func TestFactoryFunctions(t *testing.T) {
	testCases := []struct {
		encodingType ContentEncodingType
		level        int
	}{
		{ContentEncodingGzip, 6},
		{ContentEncodingDeflate, 6},
		{ContentEncodingBr, 6},
		{ContentEncodingZstd, 3},
		{ContentEncodingCompress, 0},
		{ContentEncodingIdentity, 0},
	}

	for _, tc := range testCases {
		t.Run(string(tc.encodingType), func(t *testing.T) {
			// ファクトリー関数でエンコーダー・デコーダーを作成
			encoder, err := CreateEncoder(tc.encodingType, tc.level)
			if err != nil {
				t.Fatalf("Failed to create encoder for %s: %v", tc.encodingType, err)
			}

			decoder, err := CreateDecoder(tc.encodingType)
			if err != nil {
				t.Fatalf("Failed to create decoder for %s: %v", tc.encodingType, err)
			}

			// エンコード
			compressed, err := encoder.Encode(testData)
			if err != nil {
				t.Fatalf("Encoding failed for %s: %v", tc.encodingType, err)
			}

			// デコード
			decompressed, err := decoder.Decode(compressed)
			if err != nil {
				t.Fatalf("Decoding failed for %s: %v", tc.encodingType, err)
			}

			// 結果の確認
			if !bytes.Equal(testData, decompressed) {
				t.Errorf("Round-trip failed for %s", tc.encodingType)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	testCases := []ContentEncodingType{
		ContentEncodingGzip,
		ContentEncodingDeflate,
		ContentEncodingBr,
		ContentEncodingZstd,
		ContentEncodingCompress,
		ContentEncodingIdentity,
	}

	for _, encodingType := range testCases {
		t.Run(string(encodingType), func(t *testing.T) {
			// ヘルパー関数でエンコード
			compressed, err := EncodeData(testData, encodingType, 6)
			if err != nil {
				t.Fatalf("EncodeData failed for %s: %v", encodingType, err)
			}

			// ヘルパー関数でデコード
			decompressed, err := DecodeData(compressed, encodingType)
			if err != nil {
				t.Fatalf("DecodeData failed for %s: %v", encodingType, err)
			}

			// 結果の確認
			if !bytes.Equal(testData, decompressed) {
				t.Errorf("Helper function round-trip failed for %s", encodingType)
			}
		})
	}
}

func TestUnsupportedEncoding(t *testing.T) {
	unsupportedType := ContentEncodingType("unsupported")

	// エンコーダー作成でエラーになることを確認
	_, err := CreateEncoder(unsupportedType, 6)
	if err == nil {
		t.Errorf("Expected error for unsupported encoder type")
	}

	// デコーダー作成でエラーになることを確認
	_, err = CreateDecoder(unsupportedType)
	if err == nil {
		t.Errorf("Expected error for unsupported decoder type")
	}
}

func TestCompressionLevels(t *testing.T) {
	// 大きなテストデータを作成
	largeData := []byte(strings.Repeat("Hello, World! This is a test for compression levels. ", 1000))

	levels := []int{1, 6, 9}
	
	for _, level := range levels {
		t.Run(fmt.Sprintf("Gzip_Level_%d", level), func(t *testing.T) {
			encoder := NewGzipEncoder(level)
			compressed, err := encoder.Encode(largeData)
			if err != nil {
				t.Fatalf("Gzip encoding failed at level %d: %v", level, err)
			}

			decoder := NewGzipDecoder()
			decompressed, err := decoder.Decode(compressed)
			if err != nil {
				t.Fatalf("Gzip decoding failed: %v", err)
			}

			if !bytes.Equal(largeData, decompressed) {
				t.Errorf("Round-trip failed for gzip level %d", level)
			}

			t.Logf("Gzip Level %d: Original: %d bytes, Compressed: %d bytes (ratio: %.2f%%)",
				level, len(largeData), len(compressed), float64(len(compressed))/float64(len(largeData))*100)
		})
	}
}