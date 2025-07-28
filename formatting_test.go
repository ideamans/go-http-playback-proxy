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
	if optimizer.config == nil {
		t.Fatal("ContentOptimizer.config is nil")
	}
}

func TestDefaultOptimizerConfig(t *testing.T) {
	config := DefaultOptimizerConfig()
	if config == nil {
		t.Fatal("DefaultOptimizerConfig() returned nil")
	}
	
	if config.IndentSize != 2 {
		t.Errorf("Expected IndentSize to be 2, got %d", config.IndentSize)
	}
	
	if config.IndentChar != " " {
		t.Errorf("Expected IndentChar to be space, got %q", config.IndentChar)
	}
	
	if config.BraceStyle != "collapse" {
		t.Errorf("Expected BraceStyle to be collapse, got %s", config.BraceStyle)
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
	
	minified, err := optimizer.Minify("text/html", testHTML)
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
	
	beautified, err := optimizer.Beautify("text/html", testHTML)
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
	config := &OptimizerConfig{
		IndentSize:     2,
		IndentChar:     " ",
		BraceStyle:     "collapse",
		AddLineNumbers: true,
	}
	optimizer := NewContentOptimizer(config)
	
	testHTML := `<html><body><h1>Test</h1></body></html>`
	
	beautified, err := optimizer.Beautify("text/html", testHTML)
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
	
	minified, err := optimizer.Minify("text/css", testCSS)
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
	
	beautified, err := optimizer.Beautify("text/css", testCSS)
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
	
	minified, err := optimizer.Minify("text/javascript", testJS)
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
	config := &OptimizerConfig{
		IndentSize:  4,
		IndentChar:  " ",
		BraceStyle:  "collapse",
	}
	optimizer := NewContentOptimizer(config)
	
	testJS := `function test(){var x=1;if(x>0){console.log("positive");}}var global="value";`
	
	beautified, err := optimizer.Beautify("text/javascript", testJS)
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
	testJS := `function test(){console.log("hello");}`
	
	braceStyles := []string{"collapse", "expand", "end-expand"}
	
	for _, style := range braceStyles {
		config := &OptimizerConfig{
			IndentSize:  2,
			IndentChar:  " ",
			BraceStyle:  style,
		}
		optimizer := NewContentOptimizer(config)
		
		beautified, err := optimizer.Beautify("text/javascript", testJS)
		if err != nil {
			t.Fatalf("JavaScript beautification with brace style %s failed: %v", style, err)
		}
		
		if len(beautified) <= len(testJS) {
			t.Errorf("Beautified JavaScript with %s brace style should be larger than original", style)
		}
	}
}

func TestAcceptMethod(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCases := []struct {
		mimeType string
		expected bool
	}{
		{"text/html", true},
		{"text/css", true},
		{"text/javascript", true},
		{"application/javascript", true},
		{"application/ecmascript", true},
		{"text/plain", false},
		{"image/png", false},
		{"application/json", false},
	}
	
	for _, tc := range testCases {
		result := optimizer.Accept(tc.mimeType)
		if result != tc.expected {
			t.Errorf("Accept(%s) = %v, expected %v", tc.mimeType, result, tc.expected)
		}
	}
}

func TestMinifyAndBeautifyMethods(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCases := []struct {
		content  string
		mimeType string
	}{
		{`<html><body><h1>Test</h1></body></html>`, "text/html"},
		{`body { margin: 0; }`, "text/css"},
		{`function test() { console.log("hello"); }`, "application/javascript"},
	}
	
	for _, tc := range testCases {
		// Test Minify
		minified, err := optimizer.Minify(tc.mimeType, tc.content)
		if err != nil {
			t.Errorf("Minify failed for %s: %v", tc.mimeType, err)
		}
		if len(minified) > len(tc.content) {
			t.Errorf("Minified content should not be larger than original for %s", tc.mimeType)
		}
		
		// Test Beautify
		beautified, err := optimizer.Beautify(tc.mimeType, tc.content)
		if err != nil {
			t.Errorf("Beautify failed for %s: %v", tc.mimeType, err)
		}
		if len(beautified) < len(tc.content) {
			t.Errorf("Beautified content should not be smaller than original for %s", tc.mimeType)
		}
	}
	
	// Test unsupported mime type - should return unchanged
	original := "Some plain text"
	minified, err := optimizer.Minify("text/plain", original)
	if err != nil {
		t.Errorf("Minify should not error for unsupported mime type: %v", err)
	}
	if minified != original {
		t.Errorf("Unsupported mime type should return unchanged content")
	}
	
	beautified, err := optimizer.Beautify("text/plain", original)
	if err != nil {
		t.Errorf("Beautify should not error for unsupported mime type: %v", err)
	}
	if beautified != original {
		t.Errorf("Unsupported mime type should return unchanged content")
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

func TestCSSFormattingWithComments(t *testing.T) {
	optimizer := NewContentOptimizer()
	
	testCSS := `/* Header styles */ .header{color:red;} /* Footer styles */ .footer{color:blue;}`
	
	beautified, err := optimizer.Beautify("text/css", testCSS)
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