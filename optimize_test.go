package main

import (
	"strings"
	"testing"
)

func TestNewContentOptimizer(t *testing.T) {
	optimizer := NewContentOptimizer()
	if optimizer == nil {
		t.Fatal("NewContentOptimizer() returned nil")
	}
	if optimizer.minifier == nil {
		t.Fatal("ContentOptimizer.minifier is nil")
	}
}

func TestDefaultOptimizationOptions(t *testing.T) {
	options := DefaultOptimizationOptions()
	if options == nil {
		t.Fatal("DefaultOptimizationOptions() returned nil")
	}
	
	if options.Type != OptimizerTypeMinify {
		t.Errorf("Expected Type to be %s, got %s", OptimizerTypeMinify, options.Type)
	}
	
	if options.ContentType != ContentTypeHTML {
		t.Errorf("Expected ContentType to be %s, got %s", ContentTypeHTML, options.ContentType)
	}
	
	if options.IndentSize != 2 {
		t.Errorf("Expected IndentSize to be 2, got %d", options.IndentSize)
	}
}

func TestHTMLMinification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testHTML := `<!DOCTYPE html>
<html>
    <head>
        <title>Test Page</title>
    </head>
    <body>
        <h1>Hello World</h1>
        <p>This is a test paragraph with    extra spaces.</p>
    </body>
</html>`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeMinify,
		ContentType: ContentTypeHTML,
	}
	
	minified, err := optimizer.OptimizeContent(testHTML, options)
	if err != nil {
		t.Fatalf("HTML minification failed: %v", err)
	}
	
	if len(minified) >= len(testHTML) {
		t.Errorf("Minified HTML should be smaller than original")
	}
	
	// Check that minification removed unnecessary whitespace
	if strings.Contains(minified, "    ") {
		t.Errorf("Minified HTML still contains multiple spaces")
	}
}

func TestHTMLBeautification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testHTML := `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello</h1><p>Test</p></body></html>`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeBeautify,
		ContentType: ContentTypeHTML,
	}
	
	beautified, err := optimizer.OptimizeContent(testHTML, options)
	if err != nil {
		t.Fatalf("HTML beautification failed: %v", err)
	}
	
	// Check that beautification added newlines and indentation
	if !strings.Contains(beautified, "\n") {
		t.Errorf("Beautified HTML should contain newlines")
	}
	
	if len(beautified) <= len(testHTML) {
		t.Errorf("Beautified HTML should be larger than original")
	}
}

func TestHTMLBeautificationWithLineNumbers(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testHTML := `<html><body><h1>Test</h1></body></html>`
	
	options := &OptimizationOptions{
		Type:           OptimizerTypeBeautify,
		ContentType:    ContentTypeHTML,
		AddLineNumbers: true,
	}
	
	beautified, err := optimizer.OptimizeContent(testHTML, options)
	if err != nil {
		t.Fatalf("HTML beautification with line numbers failed: %v", err)
	}
	
	// Check that line numbers are present (gohtml uses format like "1  ")
	if !strings.Contains(beautified, "1  ") {
		t.Errorf("Beautified HTML should contain line numbers, got: %q", beautified)
	}
}

func TestCSSMinification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCSS := `body {
    margin: 0;
    padding: 0;
    font-family: Arial, sans-serif;
}

.header {
    background-color: #f0f0f0;
    padding: 20px;
}

/* This is a comment */
.content {
    margin: 10px;
}`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeMinify,
		ContentType: ContentTypeCSS,
	}
	
	minified, err := optimizer.OptimizeContent(testCSS, options)
	if err != nil {
		t.Fatalf("CSS minification failed: %v", err)
	}
	
	if len(minified) >= len(testCSS) {
		t.Errorf("Minified CSS should be smaller than original")
	}
	
	// Check that comments are removed
	if strings.Contains(minified, "/* This is a comment */") {
		t.Errorf("Minified CSS should not contain comments")
	}
}

func TestCSSBeautification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCSS := `body{margin:0;padding:0;}div{color:red;background:#fff;}`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeBeautify,
		ContentType: ContentTypeCSS,
		IndentSize:  2,
		IndentChar:  " ",
	}
	
	beautified, err := optimizer.OptimizeContent(testCSS, options)
	if err != nil {
		t.Fatalf("CSS beautification failed: %v", err)
	}
	
	// Check that beautification added formatting
	if !strings.Contains(beautified, "\n") {
		t.Errorf("Beautified CSS should contain newlines")
	}
	
	if !strings.Contains(beautified, " {") {
		t.Errorf("Beautified CSS should have proper spacing")
	}
}

func TestJavaScriptMinification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testJS := `function hello() {
    var message = "Hello, World!";
    console.log(message);
    
    if (true) {
        console.log("This is true");
    }
    
    return message;
}`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeMinify,
		ContentType: ContentTypeJavaScript,
	}
	
	minified, err := optimizer.OptimizeContent(testJS, options)
	if err != nil {
		t.Fatalf("JavaScript minification failed: %v", err)
	}
	
	if len(minified) >= len(testJS) {
		t.Errorf("Minified JavaScript should be smaller than original")
	}
	
	// Check that unnecessary whitespace is removed
	if strings.Contains(minified, "    ") {
		t.Errorf("Minified JavaScript should not contain multiple spaces")
	}
}

func TestJavaScriptBeautification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testJS := `function test(){var x=1;if(x>0){console.log("positive");}}var global="value";`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeBeautify,
		ContentType: ContentTypeJavaScript,
		IndentSize:  4,
		IndentChar:  " ",
		BraceStyle:  "collapse",
	}
	
	beautified, err := optimizer.OptimizeContent(testJS, options)
	if err != nil {
		t.Fatalf("JavaScript beautification failed: %v", err)
	}
	
	// Check that beautification added formatting
	if !strings.Contains(beautified, "\n") {
		t.Errorf("Beautified JavaScript should contain newlines")
	}
	
	if len(beautified) <= len(testJS) {
		t.Errorf("Beautified JavaScript should be larger than original")
	}
}

func TestJavaScriptBeautificationBraceStyles(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testJS := `function test(){console.log("hello");}`
	
	braceStyles := []string{"collapse", "expand", "end-expand"}
	
	for _, style := range braceStyles {
		options := &OptimizationOptions{
			Type:        OptimizerTypeBeautify,
			ContentType: ContentTypeJavaScript,
			BraceStyle:  style,
		}
		
		beautified, err := optimizer.OptimizeContent(testJS, options)
		if err != nil {
			t.Fatalf("JavaScript beautification with brace style %s failed: %v", style, err)
		}
		
		if len(beautified) <= len(testJS) {
			t.Errorf("Beautified JavaScript with %s brace style should be larger than original", style)
		}
	}
}

func TestOptimizeByMimeType(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCases := []struct {
		content   string
		mimeType  string
		optType   OptimizerType
		shouldErr bool
	}{
		{
			content:   `<html><body><h1>Test</h1></body></html>`,
			mimeType:  "text/html",
			optType:   OptimizerTypeMinify,
			shouldErr: false,
		},
		{
			content:   `body { margin: 0; }`,
			mimeType:  "text/css",
			optType:   OptimizerTypeMinify,
			shouldErr: false,
		},
		{
			content:   `function test() { console.log("hello"); }`,
			mimeType:  "application/javascript",
			optType:   OptimizerTypeMinify,
			shouldErr: false,
		},
		{
			content:   `Some plain text`,
			mimeType:  "text/plain",
			optType:   OptimizerTypeMinify,
			shouldErr: false, // Should return unchanged
		},
	}
	
	for _, tc := range testCases {
		result, err := optimizer.OptimizeByMimeType(tc.content, tc.mimeType, tc.optType)
		
		if tc.shouldErr && err == nil {
			t.Errorf("Expected error for mime type %s, but got none", tc.mimeType)
		}
		
		if !tc.shouldErr && err != nil {
			t.Errorf("Unexpected error for mime type %s: %v", tc.mimeType, err)
		}
		
		if tc.mimeType == "text/plain" && result != tc.content {
			t.Errorf("Plain text should be returned unchanged")
		}
	}
}

func TestGetOptimizationStats(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	original := "This is a test string with some content"
	optimized := "Shorter string"
	
	stats := optimizer.GetOptimizationStats(original, optimized)
	
	expectedOriginalSize := len(original)
	expectedOptimizedSize := len(optimized)
	expectedSizeReduction := expectedOriginalSize - expectedOptimizedSize
	
	if stats["originalSize"] != expectedOriginalSize {
		t.Errorf("Expected originalSize %d, got %v", expectedOriginalSize, stats["originalSize"])
	}
	
	if stats["optimizedSize"] != expectedOptimizedSize {
		t.Errorf("Expected optimizedSize %d, got %v", expectedOptimizedSize, stats["optimizedSize"])
	}
	
	if stats["sizeReduction"] != expectedSizeReduction {
		t.Errorf("Expected sizeReduction %d, got %v", expectedSizeReduction, stats["sizeReduction"])
	}
	
	// Check compression ratio
	expectedRatio := float64(expectedOptimizedSize) / float64(expectedOriginalSize)
	if ratio, ok := stats["compressionRatio"].(float64); !ok || ratio != expectedRatio {
		t.Errorf("Expected compressionRatio %f, got %v", expectedRatio, stats["compressionRatio"])
	}
}

func TestUnsupportedOptimizerType(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	options := &OptimizationOptions{
		Type:        "unsupported",
		ContentType: ContentTypeHTML,
	}
	
	_, err := optimizer.OptimizeContent("<html></html>", options)
	if err == nil {
		t.Errorf("Expected error for unsupported optimizer type")
	}
	
	if !strings.Contains(err.Error(), "unsupported optimizer type") {
		t.Errorf("Error message should mention unsupported optimizer type")
	}
}

func TestUnsupportedContentTypeForBeautification(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeBeautify,
		ContentType: "text/unknown",
	}
	
	_, err := optimizer.OptimizeContent("some content", options)
	if err == nil {
		t.Errorf("Expected error for unsupported content type")
	}
	
	if !strings.Contains(err.Error(), "unsupported content type for beautification") {
		t.Errorf("Error message should mention unsupported content type")
	}
}

func TestCSSFormattingWithComments(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCSS := `/* Header styles */ .header{color:red;} /* Footer styles */ .footer{color:blue;}`
	
	options := &OptimizationOptions{
		Type:        OptimizerTypeBeautify,
		ContentType: ContentTypeCSS,
		IndentSize:  2,
		IndentChar:  " ",
	}
	
	beautified, err := optimizer.OptimizeContent(testCSS, options)
	if err != nil {
		t.Fatalf("CSS beautification with comments failed: %v", err)
	}
	
	// Comments should be preserved in beautification
	if !strings.Contains(beautified, "/* Header styles */") {
		t.Errorf("CSS beautification should preserve comments")
	}
	
	if !strings.Contains(beautified, "/* Footer styles */") {
		t.Errorf("CSS beautification should preserve comments")
	}
}