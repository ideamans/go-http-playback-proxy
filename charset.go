package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	htmlMetaCharsetRegex = regexp.MustCompile(`<meta[^>]+charset\s*=\s*["']?([^"'\s>]+)["']?[^>]*>`)
	cssCharsetRegex      = regexp.MustCompile(`@charset\s+["']([^"']+)["']`)
)

// detectCharset detects charset from HTTP Content-Type header and content body for HTML/CSS
func detectCharset(contentType string, body []byte) (httpCharset, contentCharset string) {
	// Extract charset from Content-Type header
	if contentType != "" {
		if idx := strings.Index(strings.ToLower(contentType), "charset="); idx != -1 {
			charset := contentType[idx+8:]
			if idx := strings.Index(charset, ";"); idx != -1 {
				charset = charset[:idx]
			}
			charset = strings.Trim(charset, " \"'")
			if charset != "" {
				httpCharset = charset
			}
		}
	}

	// Detect charset from content if it's HTML or CSS
	if isHTMLContent(contentType) {
		contentCharset = detectHTMLCharset(body)
	} else if isCSSContent(contentType) {
		contentCharset = detectCSSCharset(body)
	}

	return httpCharset, contentCharset
}

// isHTMLContent checks if the content type indicates HTML
func isHTMLContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html")
}

// isCSSContent checks if the content type indicates CSS
func isCSSContent(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/css")
}

// detectHTMLCharset detects charset from HTML meta tags
func detectHTMLCharset(body []byte) string {
	// Only check the first 1024 bytes for performance
	searchBody := body
	if len(body) > 1024 {
		searchBody = body[:1024]
	}

	matches := htmlMetaCharsetRegex.FindSubmatch(searchBody)
	if len(matches) > 1 {
		return strings.ToLower(string(matches[1]))
	}
	return ""
}

// detectCSSCharset detects charset from CSS @charset rule
func detectCSSCharset(body []byte) string {
	// Only check the first 512 bytes for performance
	searchBody := body
	if len(body) > 512 {
		searchBody = body[:512]
	}

	matches := cssCharsetRegex.FindSubmatch(searchBody)
	if len(matches) > 1 {
		return strings.ToLower(string(matches[1]))
	}
	return ""
}

// convertToUTF8 converts content from the specified charset to UTF-8
func convertToUTF8(content []byte, fromCharset string) ([]byte, error) {
	if fromCharset == "" || strings.ToLower(fromCharset) == "utf-8" {
		return content, nil
	}

	// Get encoding by name
	enc := getEncodingByName(fromCharset)
	if enc == nil {
		return nil, fmt.Errorf("unsupported charset: %s", fromCharset)
	}

	decoder := enc.NewDecoder()
	result, err := io.ReadAll(transform.NewReader(bytes.NewReader(content), decoder))
	if err != nil {
		return nil, fmt.Errorf("failed to convert from %s to UTF-8: %w", fromCharset, err)
	}

	return result, nil
}

// convertFromUTF8 converts UTF-8 content to the specified charset
func convertFromUTF8(content []byte, toCharset string) ([]byte, error) {
	if toCharset == "" || strings.ToLower(toCharset) == "utf-8" {
		return content, nil
	}

	enc := getEncodingByName(toCharset)
	if enc == nil {
		return nil, fmt.Errorf("unsupported charset: %s", toCharset)
	}

	encoder := enc.NewEncoder()
	result, err := io.ReadAll(transform.NewReader(bytes.NewReader(content), encoder))
	if err != nil {
		return nil, fmt.Errorf("failed to convert from UTF-8 to %s: %w", toCharset, err)
	}

	return result, nil
}

// getEncodingByName returns encoding for the given charset name
func getEncodingByName(name string) encoding.Encoding {
	name = strings.ToLower(name)
	
	switch name {
	// UTF encodings
	case "utf-8", "utf8":
		return unicode.UTF8
	case "utf-16", "utf16":
		return unicode.UTF16(unicode.BigEndian, unicode.UseBOM)
	case "utf-16be", "utf16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "utf-16le", "utf16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)

	// Japanese encodings
	case "shift_jis", "shift-jis", "sjis", "ms_kanji":
		return japanese.ShiftJIS
	case "euc-jp", "eucjp":
		return japanese.EUCJP
	case "iso-2022-jp", "iso2022jp":
		return japanese.ISO2022JP

	// Korean encodings
	case "euc-kr", "euckr":
		return korean.EUCKR

	// Chinese encodings
	case "gb2312", "gb-2312":
		return simplifiedchinese.GB18030 // GB18030 is superset of GB2312
	case "gb18030", "gb-18030":
		return simplifiedchinese.GB18030
	case "gbk":
		return simplifiedchinese.GBK
	case "big5", "big-5":
		return traditionalchinese.Big5

	// ISO encodings
	case "iso-8859-1", "iso8859-1", "latin1":
		return charmap.ISO8859_1
	case "iso-8859-2", "iso8859-2", "latin2":
		return charmap.ISO8859_2
	case "iso-8859-15", "iso8859-15":
		return charmap.ISO8859_15

	// Windows encodings
	case "windows-1252", "cp1252":
		return charmap.Windows1252
	case "windows-1251", "cp1251":
		return charmap.Windows1251

	default:
		return nil
	}
}

// processCharsetForRecording processes charset conversion during recording
func processCharsetForRecording(contentType string, body []byte) (processedBody []byte, httpCharset, contentCharset string, err error) {
	httpCharset, contentCharset = detectCharset(contentType, body)
	
	// Determine the final charset to use
	finalCharset := contentCharset
	if finalCharset == "" {
		finalCharset = httpCharset
	}

	// If no charset specified or already UTF-8, no conversion needed
	if finalCharset == "" || strings.ToLower(finalCharset) == "utf-8" {
		return body, httpCharset, finalCharset, nil
	}

	// Convert to UTF-8
	processedBody, err = convertToUTF8(body, finalCharset)
	if err != nil {
		// If conversion fails, save original content and mark charset as failed
		failedCharset := finalCharset + "-failed"
		return body, httpCharset, failedCharset, nil
	}

	return processedBody, httpCharset, finalCharset, nil
}

// processCharsetForPlayback processes charset restoration during playback
func processCharsetForPlayback(body []byte, contentCharset string, headers http.Header) ([]byte, error) {
	// If no charset or UTF-8, no conversion needed
	if contentCharset == "" || strings.ToLower(contentCharset) == "utf-8" {
		return body, nil
	}

	// If charset has -failed suffix, return content as-is
	if strings.HasSuffix(contentCharset, "-failed") {
		return body, nil
	}

	// Convert from UTF-8 to original charset
	result, err := convertFromUTF8(body, contentCharset)
	if err != nil {
		return nil, fmt.Errorf("failed to restore charset %s: %w", contentCharset, err)
	}

	// Update Content-Type header with charset
	contentType := headers.Get("Content-Type")
	if contentType != "" {
		// Remove existing charset if present
		if idx := strings.Index(strings.ToLower(contentType), "charset="); idx != -1 {
			before := contentType[:idx]
			after := contentType[idx:]
			if semiIdx := strings.Index(after, ";"); semiIdx != -1 {
				after = after[semiIdx:]
			} else {
				after = ""
			}
			contentType = strings.TrimSpace(before) + after
		}
		
		// Add charset
		if !strings.HasSuffix(contentType, ";") && contentType != "" {
			contentType += "; "
		}
		contentType += fmt.Sprintf("charset=%s", contentCharset)
		headers.Set("Content-Type", contentType)
	}

	return result, nil
}