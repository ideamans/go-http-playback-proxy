package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestDetectCharset(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		body          []byte
		expectedHTTP  string
		expectedBody  string
	}{
		{
			name:          "HTML with meta charset",
			contentType:   "text/html",
			body:          []byte(`<html><meta charset="shift_jis"><body>日本語</body></html>`),
			expectedHTTP:  "",
			expectedBody:  "shift_jis",
		},
		{
			name:          "HTML with HTTP charset",
			contentType:   "text/html; charset=utf-8",
			body:          []byte(`<html><body>test</body></html>`),
			expectedHTTP:  "utf-8",
			expectedBody:  "",
		},
		{
			name:          "CSS with @charset",
			contentType:   "text/css",
			body:          []byte(`@charset "euc-jp"; body { font-family: "日本語"; }`),
			expectedHTTP:  "",
			expectedBody:  "euc-jp",
		},
		{
			name:          "No charset specified",
			contentType:   "text/html",
			body:          []byte(`<html><body>test</body></html>`),
			expectedHTTP:  "",
			expectedBody:  "",
		},
		{
			name:          "Non-HTML/CSS content",
			contentType:   "application/json",
			body:          []byte(`{"test": "value"}`),
			expectedHTTP:  "",
			expectedBody:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCharset, bodyCharset := detectCharset(tt.contentType, tt.body)
			if httpCharset != tt.expectedHTTP {
				t.Errorf("detectCharset() httpCharset = %v, want %v", httpCharset, tt.expectedHTTP)
			}
			if bodyCharset != tt.expectedBody {
				t.Errorf("detectCharset() bodyCharset = %v, want %v", bodyCharset, tt.expectedBody)
			}
		})
	}
}

func TestConvertToUTF8(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		fromCharset string
		wantErr     bool
	}{
		{
			name:        "UTF-8 to UTF-8 (no conversion)",
			content:     []byte("Hello, 世界"),
			fromCharset: "utf-8",
			wantErr:     false,
		},
		{
			name:        "Empty charset (no conversion)",
			content:     []byte("Hello, World"),
			fromCharset: "",
			wantErr:     false,
		},
		{
			name:        "Unsupported charset",
			content:     []byte("Hello, World"),
			fromCharset: "unsupported-charset",
			wantErr:     true,
		},
		{
			name:        "Windows-1252 to UTF-8",
			content:     []byte("Hello, World"),
			fromCharset: "windows-1252",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToUTF8(tt.content, tt.fromCharset)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToUTF8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Errorf("convertToUTF8() returned nil result when no error expected")
			}
		})
	}
}

func TestConvertFromUTF8(t *testing.T) {
	tests := []struct {
		name      string
		content   []byte
		toCharset string
		wantErr   bool
	}{
		{
			name:      "UTF-8 to UTF-8 (no conversion)",
			content:   []byte("Hello, 世界"),
			toCharset: "utf-8",
			wantErr:   false,
		},
		{
			name:      "Empty charset (no conversion)",
			content:   []byte("Hello, World"),
			toCharset: "",
			wantErr:   false,
		},
		{
			name:      "UTF-8 to Windows-1252",
			content:   []byte("Hello, World"),
			toCharset: "windows-1252",
			wantErr:   false,
		},
		{
			name:      "Unsupported charset",
			content:   []byte("Hello, World"),
			toCharset: "unsupported-charset",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertFromUTF8(tt.content, tt.toCharset)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertFromUTF8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Errorf("convertFromUTF8() returned nil result when no error expected")
			}
		})
	}
}

func TestProcessCharsetForRecording(t *testing.T) {
	tests := []struct {
		name                string
		contentType         string
		body                []byte
		expectedHTTPCharset string
		expectedCharset     string
		expectConversion    bool
	}{
		{
			name:                "HTML with Shift_JIS",
			contentType:         "text/html; charset=shift_jis",
			body:                []byte(`<html><meta charset="shift_jis"><body>test</body></html>`),
			expectedHTTPCharset: "shift_jis",
			expectedCharset:     "shift_jis",
			expectConversion:    false, // Won't actually convert in test due to encoding complexity
		},
		{
			name:                "HTML with UTF-8",
			contentType:         "text/html; charset=utf-8",
			body:                []byte(`<html><body>test</body></html>`),
			expectedHTTPCharset: "utf-8",
			expectedCharset:     "utf-8",
			expectConversion:    false,
		},
		{
			name:                "CSS with EUC-JP",
			contentType:         "text/css",
			body:                []byte(`@charset "euc-jp"; body { font-size: 12px; }`),
			expectedHTTPCharset: "",
			expectedCharset:     "euc-jp",
			expectConversion:    false,
		},
		{
			name:                "No charset specified",
			contentType:         "text/html",
			body:                []byte(`<html><body>test</body></html>`),
			expectedHTTPCharset: "",
			expectedCharset:     "",
			expectConversion:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processedBody, httpCharset, contentCharset, err := processCharsetForRecording(tt.contentType, tt.body)
			if err != nil {
				t.Errorf("processCharsetForRecording() error = %v", err)
				return
			}
			if httpCharset != tt.expectedHTTPCharset {
				t.Errorf("processCharsetForRecording() httpCharset = %v, want %v", httpCharset, tt.expectedHTTPCharset)
			}
			if contentCharset != tt.expectedCharset {
				t.Errorf("processCharsetForRecording() contentCharset = %v, want %v", contentCharset, tt.expectedCharset)
			}
			if processedBody == nil {
				t.Errorf("processCharsetForRecording() returned nil body")
			}
		})
	}
}

func TestProcessCharsetForPlayback(t *testing.T) {
	tests := []struct {
		name            string
		body            []byte
		contentCharset  string
		initialHeaders  map[string]string
		expectError     bool
		expectHeaderSet bool
	}{
		{
			name:           "UTF-8 content (no conversion)",
			body:           []byte("Hello, World"),
			contentCharset: "utf-8",
			initialHeaders: map[string]string{"Content-Type": "text/html"},
			expectError:    false,
		},
		{
			name:           "Empty charset (no conversion)",
			body:           []byte("Hello, World"),
			contentCharset: "",
			initialHeaders: map[string]string{"Content-Type": "text/html"},
			expectError:    false,
		},
		{
			name:           "Failed charset (no conversion)",
			body:           []byte("Hello, World"),
			contentCharset: "shift_jis-failed",
			initialHeaders: map[string]string{"Content-Type": "text/html"},
			expectError:    false,
		},
		{
			name:            "Windows-1252 restoration",
			body:            []byte("Hello, World"),
			contentCharset:  "windows-1252",
			initialHeaders:  map[string]string{"Content-Type": "text/html"},
			expectError:     false,
			expectHeaderSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := make(http.Header)
			for k, v := range tt.initialHeaders {
				headers.Set(k, v)
			}

			result, err := processCharsetForPlayback(tt.body, tt.contentCharset, headers)
			if (err != nil) != tt.expectError {
				t.Errorf("processCharsetForPlayback() error = %v, wantErr %v", err, tt.expectError)
				return
			}
			if result == nil {
				t.Errorf("processCharsetForPlayback() returned nil result")
			}

			if tt.expectHeaderSet {
				contentType := headers.Get("Content-Type")
				if !strings.Contains(contentType, "charset=") {
					t.Errorf("processCharsetForPlayback() did not set charset in Content-Type header: %s", contentType)
				}
			}
		})
	}
}

func TestIsHTMLContent(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},
		{"application/json", false},
		{"text/css", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isHTMLContent(tt.contentType)
			if result != tt.expected {
				t.Errorf("isHTMLContent(%s) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestIsCSSContent(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/css", true},
		{"text/css; charset=utf-8", true},
		{"TEXT/CSS", true},
		{"text/html", false},
		{"application/json", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isCSSContent(tt.contentType)
			if result != tt.expected {
				t.Errorf("isCSSContent(%s) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestDetectHTMLCharset(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "Meta charset with double quotes",
			body:     []byte(`<html><meta charset="shift_jis"><body>test</body></html>`),
			expected: "shift_jis",
		},
		{
			name:     "Meta charset with single quotes",
			body:     []byte(`<html><meta charset='euc-jp'><body>test</body></html>`),
			expected: "euc-jp",
		},
		{
			name:     "Meta charset without quotes",
			body:     []byte(`<html><meta charset=utf-8><body>test</body></html>`),
			expected: "utf-8",
		},
		{
			name:     "No meta charset",
			body:     []byte(`<html><body>test</body></html>`),
			expected: "",
		},
		{
			name:     "Multiple meta tags, first with charset",
			body:     []byte(`<html><meta charset="iso-8859-1"><meta name="description" content="test"><body>test</body></html>`),
			expected: "iso-8859-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectHTMLCharset(tt.body)
			if result != tt.expected {
				t.Errorf("detectHTMLCharset() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectCSSCharset(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "@charset with double quotes",
			body:     []byte(`@charset "shift_jis"; body { font-size: 12px; }`),
			expected: "shift_jis",
		},
		{
			name:     "@charset with single quotes",
			body:     []byte(`@charset 'euc-jp'; body { font-size: 12px; }`),
			expected: "euc-jp",
		},
		{
			name:     "No @charset",
			body:     []byte(`body { font-size: 12px; }`),
			expected: "",
		},
		{
			name:     "@charset at start of file",
			body:     []byte(`@charset "utf-8"; /* CSS comment */ body { color: red; }`),
			expected: "utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCSSCharset(tt.body)
			if result != tt.expected {
				t.Errorf("detectCSSCharset() = %v, want %v", result, tt.expected)
			}
		})
	}
}