package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// ResourcePathOptions contains options for resource path conversion
type ResourcePathOptions struct {
	MaxParamLength int // Maximum length for URL parameters before hashing (default: 32)
	HashLength     int // Length of hash suffix to use (default: 8)
}

// DefaultResourcePathOptions returns default options
func DefaultResourcePathOptions() ResourcePathOptions {
	return ResourcePathOptions{
		MaxParamLength: 32,
		HashLength:     8,
	}
}

// MethodURLToFilePath converts HTTP method and URL to a file path
func MethodURLToFilePath(method, rawURL string) (string, error) {
	return MethodURLToFilePathWithOptions(method, rawURL, DefaultResourcePathOptions())
}

// MethodURLToFilePathWithOptions converts HTTP method and URL to a file path with custom options
func MethodURLToFilePathWithOptions(method, rawURL string, options ResourcePathOptions) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %s: %w", rawURL, err)
	}

	// Convert method to lowercase
	methodLower := strings.ToLower(method)

	// Get protocol (scheme)
	protocol := strings.ToLower(parsedURL.Scheme)
	if protocol == "" {
		protocol = "http" // default to http if no scheme
	}

	// Get hostname (convert to lowercase)
	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "" {
		return "", fmt.Errorf("hostname is required in URL: %s", rawURL)
	}

	// Get path
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Handle directory paths and missing extensions
	path = normalizeResourcePath(path)

	// Handle URL parameters
	if parsedURL.RawQuery != "" {
		path = handleURLParameters(path, parsedURL.RawQuery, options)
	}

	// Combine all parts
	filePath := filepath.Join(methodLower, protocol, hostname, strings.TrimPrefix(path, "/"))

	return filePath, nil
}

// normalizeResourcePath handles directory paths and missing extensions
func normalizeResourcePath(path string) string {
	// If path ends with / or has no extension, append index.html
	if strings.HasSuffix(path, "/") {
		return path + "index.html"
	}

	// Check if path has an extension
	if filepath.Ext(path) == "" {
		// No extension, assume it's a directory and add index.html
		return path + "/index.html"
	}

	return path
}

// customEncodeQuery properly encodes query parameters by parsing them
func customEncodeQuery(query string) string {
	if query == "" {
		return ""
	}
	
	// First decode the entire query to handle mixed encoded/unencoded content
	decoded, err := url.QueryUnescape(query)
	if err != nil {
		// If decoding fails, use the original
		decoded = query
	}
	
	// Split by & to get individual parameters
	params := strings.Split(decoded, "&")
	var encodedParams []string
	
	for _, param := range params {
		// Split by = to get name and value (max 2 parts)
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 1 {
			// No value, just encode the name
			encodedParams = append(encodedParams, url.PathEscape(parts[0]))
		} else {
			// Encode both name and value
			name := url.PathEscape(parts[0])
			value := url.PathEscape(parts[1])
			encodedParams = append(encodedParams, name+"="+value)
		}
	}
	
	return strings.Join(encodedParams, "&")
}


// customDecodeQuery decodes query parameters properly
func customDecodeQuery(query string) string {
	if query == "" {
		return ""
	}
	
	// Split by & to get individual parameters
	params := strings.Split(query, "&")
	var decodedParams []string
	
	for _, param := range params {
		// Split by = to get name and value (max 2 parts)
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 1 {
			// No value, just decode the name
			if decoded, err := url.QueryUnescape(parts[0]); err == nil {
				decodedParams = append(decodedParams, decoded)
			} else {
				decodedParams = append(decodedParams, parts[0])
			}
		} else {
			// Decode both name and value
			var name, value string
			if decoded, err := url.QueryUnescape(parts[0]); err == nil {
				name = decoded
			} else {
				name = parts[0]
			}
			if decoded, err := url.QueryUnescape(parts[1]); err == nil {
				value = decoded
			} else {
				value = parts[1]
			}
			decodedParams = append(decodedParams, name+"="+value)
		}
	}
	
	return strings.Join(decodedParams, "&")
}

// handleURLParameters processes URL parameters and handles long parameter strings
func handleURLParameters(path, rawQuery string, options ResourcePathOptions) string {
	// Custom encode the query parameters - preserve = and & but encode spaces and other special chars
	encodedQuery := customEncodeQuery(rawQuery)
	
	// If the encoded query is longer than maxLength, hash the excess
	if len(encodedQuery) > options.MaxParamLength {
		// Take first maxParamLength characters
		prefix := encodedQuery[:options.MaxParamLength]
		
		// Hash the remaining part
		remaining := encodedQuery[options.MaxParamLength:]
		hash := sha1.Sum([]byte(remaining))
		hashB64 := base64.StdEncoding.EncodeToString(hash[:])
		
		// Take first hashLength characters of the hash
		hashSuffix := hashB64[:options.HashLength]
		
		encodedQuery = prefix + hashSuffix
	}

	// Special handling for paths ending with /index.html
	if strings.HasSuffix(path, "/index.html") {
		// Insert parameters into the index.html filename
		basePath := strings.TrimSuffix(path, "/index.html")
		return basePath + "/index~" + encodedQuery + ".html"
	}

	// Get the file extension from the original path
	ext := filepath.Ext(path)
	basePath := strings.TrimSuffix(path, ext)

	// Insert parameters before the extension
	return basePath + "~" + encodedQuery + ext
}

// FilePathToMethodURL converts a file path back to method and URL (reverse operation)
func FilePathToMethodURL(filePath string) (method, urlString string, err error) {
	// Split the path components
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid file path format: %s", filePath)
	}

	method = strings.ToUpper(parts[0])
	protocol := parts[1]
	hostname := parts[2]

	// Reconstruct the path
	var pathParts []string
	if len(parts) > 3 {
		pathParts = parts[3:]
	}

	path := "/" + strings.Join(pathParts, "/")

	// Handle index.html suffix
	if strings.HasSuffix(path, "/index.html") {
		path = strings.TrimSuffix(path, "index.html")
		// Keep trailing slash for directory paths (don't remove for root)
	}

	// Handle URL parameters (extract from ~ notation)
	var query string
	if strings.Contains(path, "~") {
		// Find the last occurrence of ~ to handle cases where ~ might be in the path
		lastTilde := strings.LastIndex(path, "~")
		if lastTilde != -1 {
			ext := filepath.Ext(path)
			if ext != "" {
				// Extract query from between ~ and file extension
				queryWithExt := path[lastTilde+1:]
				query = strings.TrimSuffix(queryWithExt, ext)
				
				// Check if this is an index~params.html pattern
				if strings.Contains(path[:lastTilde], "/index") && ext == ".html" {
					// Remove the index filename - this represents a directory access
					pathBeforeIndex := strings.TrimSuffix(path[:lastTilde], "/index")
					if pathBeforeIndex == "" {
						path = "/"
					} else {
						path = pathBeforeIndex
					}
				} else {
					path = path[:lastTilde] + ext
				}
			} else {
				// No extension, everything after ~ is query
				query = path[lastTilde+1:]
				path = path[:lastTilde]
			}

			// Custom decode the query - only decode %20 back to spaces
			query = customDecodeQuery(query)
		}
	}

	// Construct the URL
	reconstructedURL := protocol + "://" + hostname + path
	if query != "" {
		reconstructedURL += "?" + query
	}

	return method, reconstructedURL, nil
}

// SanitizeFilePath sanitizes a file path for safe filesystem usage
func SanitizeFilePath(path string) string {
	// Replace unsafe characters
	unsafe := []string{"<", ">", ":", "\"", "|", "?", "*"}
	result := path
	
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Handle Windows reserved names
	windowsReserved := []string{"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	
	parts := strings.Split(result, "/")
	for i, part := range parts {
		nameWithoutExt := strings.TrimSuffix(part, filepath.Ext(part))
		for _, reserved := range windowsReserved {
			if strings.EqualFold(nameWithoutExt, reserved) {
				parts[i] = "_" + part
				break
			}
		}
	}
	
	return strings.Join(parts, "/")
}

// GetResourceFilePath is a convenience function that combines path conversion and sanitization
func GetResourceFilePath(method, rawURL string) (string, error) {
	path, err := MethodURLToFilePath(method, rawURL)
	if err != nil {
		return "", err
	}
	return SanitizeFilePath(path), nil
}