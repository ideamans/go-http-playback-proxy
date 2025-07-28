package tests

import (
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	TestServerURL = "http://localhost:9999"
)

func TestBasicFunctionality(t *testing.T) {
	// 基本的なテストサーバーの動作確認
	t.Run("TestServerHealth", func(t *testing.T) {
		t.Parallel() // 並行実行を有効化
		testServerHealth(t)
	})

	t.Run("StaticContent", func(t *testing.T) {
		t.Parallel() // 並行実行を有効化
		testStaticContent(t)
	})

	t.Run("CompressionSupport", func(t *testing.T) {
		t.Parallel() // 並行実行を有効化
		testCompressionSupport(t)
	})

	t.Run("CharsetSupport", func(t *testing.T) {
		t.Parallel() // 並行実行を有効化
		testCharsetSupport(t)
	})
}

func testServerHealth(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(TestServerURL + "/")
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if len(body) == 0 {
		t.Error("Response body is empty")
	}

	t.Logf("Test server is healthy, response length: %d bytes", len(body))
}

func testStaticContent(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		contentType    string
	}{
		{
			name:           "JSON API",
			url:            "/api/users.json",
			expectedStatus: 200,
			contentType:    "application/json",
		},
		{
			name:           "HTML content",
			url:            "/html/utf8.html",
			expectedStatus: 200,
			contentType:    "text/html",
		},
		{
			name:           "CSS content",
			url:            "/css/utf8.css",
			expectedStatus: 200,
			contentType:    "text/css",
		},
		{
			name:           "Image content",
			url:            "/images/small.jpg",
			expectedStatus: 200,
			contentType:    "image/jpeg",
		},
		{
			name:           "Not found",
			url:            "/nonexistent",
			expectedStatus: 404,
			contentType:    "",
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(TestServerURL + tc.url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if tc.contentType != "" {
				contentType := resp.Header.Get("Content-Type")
				if contentType == "" || contentType[:len(tc.contentType)] != tc.contentType {
					t.Errorf("Expected Content-Type to start with %s, got %s", tc.contentType, contentType)
				}
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			t.Logf("%s: Status=%d, ContentType=%s, Size=%d bytes", 
				tc.name, resp.StatusCode, resp.Header.Get("Content-Type"), len(body))
		})
	}
}

func testCompressionSupport(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		acceptEncoding string
		expectEncoding string
	}{
		{
			name:           "Gzip compression",
			url:            "/api/users.json?compression=gzip",
			acceptEncoding: "gzip, deflate, br",
			expectEncoding: "gzip",
		},
		{
			name:           "Brotli compression",
			url:            "/api/users.json?compression=br",
			acceptEncoding: "gzip, deflate, br",
			expectEncoding: "br",
		},
		{
			name:           "No compression",
			url:            "/api/users.json?compression=identity",
			acceptEncoding: "identity",
			expectEncoding: "",
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", TestServerURL+tc.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Accept-Encoding", tc.acceptEncoding)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			contentEncoding := resp.Header.Get("Content-Encoding")
			if tc.expectEncoding == "" {
				if contentEncoding != "" {
					t.Errorf("Expected no Content-Encoding, got %s", contentEncoding)
				}
			} else {
				if contentEncoding != tc.expectEncoding {
					t.Errorf("Expected Content-Encoding %s, got %s", tc.expectEncoding, contentEncoding)
				}
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			t.Logf("%s: ContentEncoding=%s, Size=%d bytes", 
				tc.name, contentEncoding, len(body))
		})
	}
}

func testCharsetSupport(t *testing.T) {
	testCases := []struct {
		name            string
		url             string
		expectedCharset string
	}{
		{
			name:            "UTF-8 charset",
			url:             "/charset/utf8",
			expectedCharset: "UTF-8",
		},
		{
			name:            "Shift_JIS charset",
			url:             "/charset/shift_jis",
			expectedCharset: "Shift_JIS",
		},
		{
			name:            "EUC-JP charset",
			url:             "/charset/euc_jp",
			expectedCharset: "EUC-JP",
		},
		{
			name:            "ISO-8859-1 charset",
			url:             "/charset/iso8859",
			expectedCharset: "ISO-8859-1",
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(TestServerURL + tc.url)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if contentType == "" {
				t.Error("No Content-Type header")
				return
			}

			if !containsCharset(contentType, tc.expectedCharset) {
				t.Errorf("Expected charset %s in Content-Type, got %s", tc.expectedCharset, contentType)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			t.Logf("%s: ContentType=%s, Size=%d bytes", 
				tc.name, contentType, len(body))
		})
	}
}

func containsCharset(contentType, expectedCharset string) bool {
	// Content-Type ヘッダーに期待される charset が含まれているかチェック
	return len(contentType) > len(expectedCharset) && 
		   (contentType[len(contentType)-len(expectedCharset):] == expectedCharset ||
		    findInString(contentType, expectedCharset))
}

func findInString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}