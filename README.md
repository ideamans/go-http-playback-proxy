# Go HTTP Playback Proxy

A Go-based MITM proxy with HTTP/HTTPS traffic recording and playback capabilities. Functions as a browser proxy for performance testing and analysis.

## Overview

- **MITM Proxy**: Complete HTTP/HTTPS traffic monitoring and recording
- **Compression Preservation**: Maintains response compression state for performance optimization
- **DNS Monitoring**: Detailed DNS resolution process logging
- **TypeScript Compatible**: Fully compatible type definitions with TypeScript
- **URL-to-Filepath Conversion**: Converts HTTP requests to appropriate file paths
- **Content Optimization**: HTML/CSS/JavaScript beautification and minification

## Quick Start

```bash
# Build
make build

# Recording mode - Record traffic to specified URL
./http-playback-proxy recording https://www.example.com/

# Playback mode - Replay recorded traffic
./http-playback-proxy playback

# With options
./http-playback-proxy --port 8080 --inventory-dir ./inventory recording https://www.example.com/
```

## Installation

### Prerequisites

- Go 1.22 or higher
- Make (optional, for build automation)

### Build from Source

```bash
git clone https://github.com/ideamans/go-http-playback-proxy.git
cd go-http-playback-proxy
make build
```

### Download Binaries

Pre-built binaries are available from the [Releases](https://github.com/ideamans/go-http-playback-proxy/releases) page for:
- Linux (amd64, arm64)
- macOS (amd64, arm64) 
- Windows (amd64)
- FreeBSD (amd64)

## Usage

### Command Line Options

```bash
./http-playback-proxy [options] <command>

Commands:
  recording <url>  Record traffic to specified URL
  playback        Replay recorded traffic

Options:
  --port, -p          Proxy server port (default: 8080)
  --inventory-dir, -i Inventory directory path (default: ./inventory)
  --log-level, -l     Log level (debug, info, warn, error) (default: info)

Recording Options:
  --no-beautify       Disable HTML/CSS/JavaScript beautification
```

### Browser Configuration

Configure your browser to use `localhost:8080` as HTTP/HTTPS proxy.

#### Chrome Example

```bash
google-chrome --proxy-server=localhost:8080 --ignore-certificate-errors --ignore-ssl-errors
```

### Recording Mode

Records all HTTP/HTTPS traffic to the specified URL pattern:

```bash
./http-playback-proxy recording https://www.example.com/
```

Recorded data is saved to:
```
./inventory/
├── inventory.json     # Resource metadata and domain info
└── contents/          # Decoded response bodies
    └── get/https/example.com/index.html
```

### Playback Mode

Replays recorded traffic with accurate timing:

```bash
./http-playback-proxy playback
```

Features:
- Preserves original TTFB (Time To First Byte)
- Maintains transfer speeds (Mbps)
- Adds `x-playback-proxy: 1` header to responses
- Falls back to upstream proxy for unrecorded requests

## Features

### Content Encoding Support

Supports multiple compression formats:
- **Gzip**: RFC 1952 compliant
- **Deflate**: RFC 1951 compliant
- **Brotli**: Google's compression algorithm
- **Zstd**: Facebook's Zstandard compression
- **Identity**: Uncompressed passthrough

### Character Encoding Support

Automatic character encoding detection and conversion:
- Detects charset from HTTP headers and HTML meta tags
- Converts to UTF-8 for storage
- Restores original encoding during playback
- Supports: Shift_JIS, EUC-JP, ISO-8859-1, UTF-8

### Content Optimization

Optional beautification and minification:
- **HTML**: Formatting with gohtml
- **CSS**: Manual indentation formatting
- **JavaScript**: Beautification with jsbeautifier-go
- **Minification**: Using tdewolff/minify for all formats

### URL-to-Filepath Conversion

Intelligent conversion between URLs and file paths:

```
GET https://example.com/api?user=123&action=view
→ get/https/example.com/api/index~user=123&action=view.html

GET https://example.com/image.jpg?param=value
→ get/https/example.com/image~param=value.jpg
```

Features:
- Directory paths automatically append `/index.html`
- Query parameters preserved with `~` separator
- Long parameters (>32 chars) hashed with SHA1
- Full Unicode support for international characters

## Performance

### Connection Optimization

- TLS session cache (256 entries)
- Connection pooling (10 per host)
- TCP_NODELAY enabled
- Keep-Alive optimization

### Compression Handling

- Preserves original compression
- `DisableCompression=true` prevents automatic decompression
- Reduces CPU overhead by maintaining compressed state

### Accurate Playback Timing

- Records and replays TTFB accurately
- Maintains original transfer speeds (Mbps)
- Chunk-based timing for realistic network behavior

## Development

### Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run all tests
make test-all

# Performance testing with Lighthouse
make lighthouse
```

### API Usage

```go
// Convert URL to filepath
filePath, err := MethodURLToFilePath("GET", "https://example.com/api?user=123")
// → "get/https/example.com/api/index~user=123.html"

// Convert filepath back to URL
method, url, err := FilePathToMethodURL("get/https/example.com/api/index~user=123.html")
// → "GET", "https://example.com/api?user=123"
```

## CI/CD

GitHub Actions workflows:
- **CI**: Tests on push to main/develop branches
- **Release**: Automated releases with GoReleaser on version tags

## Limitations

- Designed for development and testing use
- Uses self-signed certificates (not for production)
- HTTP/2 disabled for compatibility
- No WebSocket support (yet)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built on [go-mitmproxy](https://github.com/lqqyt2423/go-mitmproxy) for MITM proxy functionality
- Uses various excellent Go libraries for compression, encoding, and optimization