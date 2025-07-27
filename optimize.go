package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/yosssi/gohtml"
)

// OptimizerType represents the type of optimization to perform
type OptimizerType string

const (
	OptimizerTypeMinify   OptimizerType = "minify"
	OptimizerTypeBeautify OptimizerType = "beautify"
)

// ContentType represents the type of content to optimize
type ContentType string

const (
	ContentTypeHTML       ContentType = "text/html"
	ContentTypeCSS        ContentType = "text/css"
	ContentTypeJavaScript ContentType = "text/javascript"
)

// OptimizationOptions contains options for content optimization
type OptimizationOptions struct {
	Type        OptimizerType
	ContentType ContentType
	// Beautify options for JavaScript
	IndentSize  int
	IndentChar  string
	BraceStyle  string
	// HTML beautify options
	AddLineNumbers bool
}

// DefaultOptimizationOptions returns default optimization options
func DefaultOptimizationOptions() *OptimizationOptions {
	return &OptimizationOptions{
		Type:           OptimizerTypeMinify,
		ContentType:    ContentTypeHTML,
		IndentSize:     2,
		IndentChar:     " ",
		BraceStyle:     "collapse",
		AddLineNumbers: false,
	}
}

// ContentOptimizer handles content optimization (minify/beautify)
type ContentOptimizer struct {
	minifier *minify.M
}

// NewContentOptimizer creates a new content optimizer
func NewContentOptimizer() *ContentOptimizer {
	m := minify.New()
	
	// Add minifiers for different content types
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/javascript", js.Minify)
	m.AddFunc("application/javascript", js.Minify)
	
	return &ContentOptimizer{
		minifier: m,
	}
}

// OptimizeContent optimizes content based on the provided options
func (co *ContentOptimizer) OptimizeContent(content string, options *OptimizationOptions) (string, error) {
	if options == nil {
		options = DefaultOptimizationOptions()
	}

	switch options.Type {
	case OptimizerTypeMinify:
		return co.minifyContent(content, options.ContentType)
	case OptimizerTypeBeautify:
		return co.beautifyContent(content, options)
	default:
		return "", fmt.Errorf("unsupported optimizer type: %s", options.Type)
	}
}

// minifyContent minifies the content based on content type
func (co *ContentOptimizer) minifyContent(content string, contentType ContentType) (string, error) {
	var buf bytes.Buffer
	
	err := co.minifier.Minify(string(contentType), &buf, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("minification failed: %w", err)
	}
	
	return buf.String(), nil
}

// beautifyContent beautifies the content based on content type and options
func (co *ContentOptimizer) beautifyContent(content string, options *OptimizationOptions) (string, error) {
	switch options.ContentType {
	case ContentTypeHTML:
		return co.beautifyHTML(content, options)
	case ContentTypeCSS:
		return co.beautifyCSS(content, options)
	case ContentTypeJavaScript:
		return co.beautifyJavaScript(content, options)
	default:
		return "", fmt.Errorf("unsupported content type for beautification: %s", options.ContentType)
	}
}

// beautifyHTML beautifies HTML content
func (co *ContentOptimizer) beautifyHTML(content string, options *OptimizationOptions) (string, error) {
	if options.AddLineNumbers {
		return gohtml.FormatWithLineNo(content), nil
	}
	return gohtml.Format(content), nil
}

// beautifyCSS beautifies CSS content (basic implementation)
func (co *ContentOptimizer) beautifyCSS(content string, options *OptimizationOptions) (string, error) {
	// For CSS, we implement a basic beautifier since there's no dedicated Go library
	return co.formatCSS(content, options.IndentChar, options.IndentSize), nil
}

// beautifyJavaScript beautifies JavaScript content
func (co *ContentOptimizer) beautifyJavaScript(content string, options *OptimizationOptions) (string, error) {
	jsOptions := jsbeautifier.DefaultOptions()
	
	// Configure options - jsbeautifier uses map[string]interface{}
	jsOptions["indent_size"] = options.IndentSize
	jsOptions["indent_char"] = options.IndentChar
	
	// Map brace style
	switch options.BraceStyle {
	case "expand":
		jsOptions["brace_style"] = "expand"
	case "collapse":
		jsOptions["brace_style"] = "collapse"
	case "end-expand":
		jsOptions["brace_style"] = "end-expand"
	default:
		jsOptions["brace_style"] = "collapse"
	}
	
	result, err := jsbeautifier.Beautify(&content, jsOptions)
	if err != nil {
		return "", fmt.Errorf("JavaScript beautification failed: %w", err)
	}
	
	return result, nil
}

// formatCSS provides basic CSS formatting
func (co *ContentOptimizer) formatCSS(content string, indentChar string, indentSize int) string {
	var result strings.Builder
	var indentLevel int
	indent := strings.Repeat(indentChar, indentSize)
	
	// Remove existing whitespace and newlines
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")
	
	// Normalize multiple spaces to single space
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}
	
	inComment := false
	commentStart := false
	
	for i, char := range content {
		switch char {
		case '/':
			if i+1 < len(content) && content[i+1] == '*' {
				commentStart = true
			}
			result.WriteRune(char)
		case '*':
			if commentStart {
				inComment = true
				commentStart = false
			}
			if i+1 < len(content) && content[i+1] == '/' {
				inComment = false
			}
			result.WriteRune(char)
		case '{':
			if !inComment {
				result.WriteString(" {\n")
				indentLevel++
				result.WriteString(strings.Repeat(indent, indentLevel))
			} else {
				result.WriteRune(char)
			}
		case '}':
			if !inComment {
				result.WriteString("\n")
				indentLevel--
				result.WriteString(strings.Repeat(indent, indentLevel))
				result.WriteRune(char)
				result.WriteString("\n")
				if indentLevel > 0 {
					result.WriteString(strings.Repeat(indent, indentLevel))
				}
			} else {
				result.WriteRune(char)
			}
		case ';':
			if !inComment {
				result.WriteString(";\n")
				result.WriteString(strings.Repeat(indent, indentLevel))
			} else {
				result.WriteRune(char)
			}
		case ' ':
			// Skip multiple spaces and spaces after certain characters
			if i > 0 {
				prevChar := rune(content[i-1])
				if prevChar != '{' && prevChar != ';' && prevChar != '}' {
					result.WriteRune(char)
				}
			}
		default:
			result.WriteRune(char)
		}
	}
	
	// Clean up extra newlines and spaces at the end
	formatted := result.String()
	formatted = strings.TrimSpace(formatted)
	
	// Normalize multiple newlines
	for strings.Contains(formatted, "\n\n\n") {
		formatted = strings.ReplaceAll(formatted, "\n\n\n", "\n\n")
	}
	
	return formatted
}

// OptimizeByMimeType is a convenience function to optimize content by MIME type
func (co *ContentOptimizer) OptimizeByMimeType(content string, mimeType string, optimizerType OptimizerType) (string, error) {
	options := DefaultOptimizationOptions()
	options.Type = optimizerType
	
	// Map MIME types to content types
	switch {
	case strings.Contains(mimeType, "html"):
		options.ContentType = ContentTypeHTML
	case strings.Contains(mimeType, "css"):
		options.ContentType = ContentTypeCSS
	case strings.Contains(mimeType, "javascript") || strings.Contains(mimeType, "ecmascript"):
		options.ContentType = ContentTypeJavaScript
	default:
		return content, nil // Return unchanged for unsupported types
	}
	
	return co.OptimizeContent(content, options)
}

// GetOptimizationStats returns statistics about the optimization
func (co *ContentOptimizer) GetOptimizationStats(original, optimized string) map[string]interface{} {
	originalSize := len(original)
	optimizedSize := len(optimized)
	
	var compressionRatio float64
	if originalSize > 0 {
		compressionRatio = float64(optimizedSize) / float64(originalSize)
	}
	
	return map[string]interface{}{
		"originalSize":      originalSize,
		"optimizedSize":     optimizedSize,
		"sizeReduction":     originalSize - optimizedSize,
		"compressionRatio":  compressionRatio,
		"compressionPercent": (1.0 - compressionRatio) * 100.0,
	}
}