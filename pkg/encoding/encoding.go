package encoding

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"go-http-playback-proxy/pkg/types"
)

// Encoder interface for content encoding
type Encoder interface {
	Encode(data []byte) ([]byte, error)
}

// Decoder interface for content decoding
type Decoder interface {
	Decode(data []byte) ([]byte, error)
}

// GzipEncoder implements gzip compression
type GzipEncoder struct {
	Level int // compression level (1-9, default 6)
}

func NewGzipEncoder(level int) *GzipEncoder {
	if level < 1 || level > 9 {
		level = gzip.DefaultCompression
	}
	return &GzipEncoder{Level: level}
}

func (e *GzipEncoder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, e.Level)
	if err != nil {
		return nil, fmt.Errorf("gzip writer creation failed: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("gzip encoding failed: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("gzip writer close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// GzipDecoder implements gzip decompression
type GzipDecoder struct{}

func NewGzipDecoder() *GzipDecoder {
	return &GzipDecoder{}
}

func (d *GzipDecoder) Decode(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader creation failed: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("gzip decoding failed: %w", err)
	}

	return decompressed, nil
}

// DeflateEncoder implements deflate compression
type DeflateEncoder struct {
	Level int // compression level
}

func NewDeflateEncoder(level int) *DeflateEncoder {
	if level < -2 || level > 9 {
		level = flate.DefaultCompression
	}
	return &DeflateEncoder{Level: level}
}

func (e *DeflateEncoder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, e.Level)
	if err != nil {
		return nil, fmt.Errorf("deflate writer creation failed: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("deflate encoding failed: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("deflate writer close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// DeflateDecoder implements deflate decompression
type DeflateDecoder struct{}

func NewDeflateDecoder() *DeflateDecoder {
	return &DeflateDecoder{}
}

func (d *DeflateDecoder) Decode(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("deflate decoding failed: %w", err)
	}

	return decompressed, nil
}

// CompressEncoder implements LZW compression (Unix compress format)
type CompressEncoder struct{}

func NewCompressEncoder() *CompressEncoder {
	return &CompressEncoder{}
}

func (e *CompressEncoder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lzw.NewWriter(&buf, lzw.MSB, 8)
	defer writer.Close()

	_, err := writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("compress encoding failed: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("compress writer close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// CompressDecoder implements LZW decompression (Unix compress format)
type CompressDecoder struct{}

func NewCompressDecoder() *CompressDecoder {
	return &CompressDecoder{}
}

func (d *CompressDecoder) Decode(data []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("compress decoding failed: %w", err)
	}

	return decompressed, nil
}

// BrotliEncoder implements Brotli compression
type BrotliEncoder struct {
	Level int // compression level (0-11, default 6)
}

func NewBrotliEncoder(level int) *BrotliEncoder {
	if level < 0 || level > 11 {
		level = 6 // default level
	}
	return &BrotliEncoder{Level: level}
}

func (e *BrotliEncoder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, e.Level)
	defer writer.Close()

	_, err := writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("brotli encoding failed: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("brotli writer close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// BrotliDecoder implements Brotli decompression
type BrotliDecoder struct{}

func NewBrotliDecoder() *BrotliDecoder {
	return &BrotliDecoder{}
}

func (d *BrotliDecoder) Decode(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("brotli decoding failed: %w", err)
	}

	return decompressed, nil
}

// ZstdEncoder implements Zstandard compression
type ZstdEncoder struct {
	Level int // compression level
}

func NewZstdEncoder(level int) *ZstdEncoder {
	return &ZstdEncoder{Level: level}
}

func (e *ZstdEncoder) Encode(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(e.Level)))
	if err != nil {
		return nil, fmt.Errorf("zstd encoder creation failed: %w", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(data, make([]byte, 0, len(data)))
	return compressed, nil
}

// ZstdDecoder implements Zstandard decompression
type ZstdDecoder struct{}

func NewZstdDecoder() *ZstdDecoder {
	return &ZstdDecoder{}
}

func (d *ZstdDecoder) Decode(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decoder creation failed: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decoding failed: %w", err)
	}

	return decompressed, nil
}

// IdentityEncoder implements no compression (passthrough)
type IdentityEncoder struct{}

func NewIdentityEncoder() *IdentityEncoder {
	return &IdentityEncoder{}
}

func (e *IdentityEncoder) Encode(data []byte) ([]byte, error) {
	// No compression, just return a copy
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// IdentityDecoder implements no decompression (passthrough)
type IdentityDecoder struct{}

func NewIdentityDecoder() *IdentityDecoder {
	return &IdentityDecoder{}
}

func (d *IdentityDecoder) Decode(data []byte) ([]byte, error) {
	// No decompression, just return a copy
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// CreateEncoder creates encoders based on ContentEncodingType
func CreateEncoder(encodingType types.ContentEncodingType, level int) (Encoder, error) {
	switch encodingType {
	case types.ContentEncodingGzip:
		return NewGzipEncoder(level), nil
	case types.ContentEncodingDeflate:
		return NewDeflateEncoder(level), nil
	case types.ContentEncodingCompress:
		return NewCompressEncoder(), nil
	case types.ContentEncodingBr:
		return NewBrotliEncoder(level), nil
	case types.ContentEncodingZstd:
		return NewZstdEncoder(level), nil
	case types.ContentEncodingIdentity:
		return NewIdentityEncoder(), nil
	default:
		return nil, fmt.Errorf("unsupported encoding type: %s", encodingType)
	}
}

// CreateDecoder creates decoders based on ContentEncodingType
func CreateDecoder(encodingType types.ContentEncodingType) (Decoder, error) {
	switch encodingType {
	case types.ContentEncodingGzip:
		return NewGzipDecoder(), nil
	case types.ContentEncodingDeflate:
		return NewDeflateDecoder(), nil
	case types.ContentEncodingCompress:
		return NewCompressDecoder(), nil
	case types.ContentEncodingBr:
		return NewBrotliDecoder(), nil
	case types.ContentEncodingZstd:
		return NewZstdDecoder(), nil
	case types.ContentEncodingIdentity:
		return NewIdentityDecoder(), nil
	default:
		return nil, fmt.Errorf("unsupported encoding type: %s", encodingType)
	}
}

// EncodeData encodes data using the specified encoding type
func EncodeData(data []byte, encodingType types.ContentEncodingType, level int) ([]byte, error) {
	encoder, err := CreateEncoder(encodingType, level)
	if err != nil {
		return nil, err
	}
	return encoder.Encode(data)
}

// DecodeData decodes data using the specified encoding type
func DecodeData(data []byte, encodingType types.ContentEncodingType) ([]byte, error) {
	decoder, err := CreateDecoder(encodingType)
	if err != nil {
		return nil, err
	}
	return decoder.Decode(data)
}