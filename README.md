# FAS-Download: Fast Adaptive Streaming Downloader

A high-performance concurrent file downloader written in Go that uses adaptive concurrent connection management to download files efficiently.

## Features

- **Concurrent Downloads**: Downloads files in parallel chunks for maximum speed
- **Adaptive Connection Management**: Automatically adjusts the number of concurrent connections based on performance
- **Range Request Support**: Uses HTTP range requests to download file parts simultaneously
- **Fallback Mode**: Gracefully handles servers that don't support range requests
- **Real-time Progress**: Shows download progress, speed, and statistics
- **Flexible File Size Handling**: Works with both known and unknown file sizes

## Usage

```bash
# Download using YAML configuration
go run main.go <config.yaml> [output_filename]
```

### YAML Configuration Format

Create a YAML file with the following structure:

```yaml
url: https://example.com/file.zip
```

Where:
- `url`: The URL to download from

The file size is automatically detected from the server using HTTP HEAD requests and Content-Length headers.

### Examples

```bash
# Download with automatic filename detection
go run main.go config.yaml

# Download with custom filename
go run main.go config.yaml my_file.zip
```

**Sample config.yaml:**
```yaml
url: https://releases.ubuntu.com/20.04/ubuntu-20.04.6-desktop-amd64.iso.torrent
```

## How It Works

### Concurrent Download Mode
When the server supports range requests:
1. **File Analysis**: Checks server capabilities and file size
2. **Chunk Creation**: Divides file into 1MB chunks
3. **Concurrent Download**: Downloads multiple chunks simultaneously
4. **Adaptive Management**: Adjusts connection count based on performance
5. **Progress Tracking**: Real-time progress and speed reporting

### Fallback Mode
When the server doesn't support range requests:
1. **Single Connection**: Downloads entire file in one request
2. **Progress Tracking**: Shows download progress and speed
3. **Efficient Buffering**: Uses optimized buffer sizes for best performance

## Technical Details

### Adaptive Algorithm
- **Start**: Begins with 4 concurrent connections
- **Increase**: Adds connections when chunks complete quickly (< 2 seconds)
- **Decrease**: Reduces connections when chunks are slow (> 5 seconds)
- **Limits**: Min 2, Max 16 concurrent connections

### Performance Optimizations
- **32KB Buffer**: Efficient memory usage during download
- **Pre-allocated Files**: Reduces file system overhead
- **Goroutine Pool**: Manages concurrent downloads efficiently
- **Memory-safe Statistics**: Thread-safe progress tracking

## Requirements

- Go 1.21 or later
- Internet connection
- Write permissions in the target directory
- YAML configuration file with URL

## Error Handling

The downloader handles various error conditions:
- Network timeouts (30-60 second timeouts)
- Server errors (non-200 status codes)
- File system errors (permissions, disk space)
- Invalid URLs or unreachable hosts

## Performance

Typical performance improvements:
- **2-5x faster** than single-threaded downloads
- **Adaptive scaling** based on network conditions
- **Efficient memory usage** with streaming buffers
- **Minimal CPU overhead** with optimized goroutines

## Example Output

```
Downloading https://example.com/file.zip to file.zip
File size: 104857600 bytes
Starting download with 4 connections
Created 100 chunks
Progress: 45.2% (47370240/104857600 bytes) Speed: 8.45 MB/s
Increasing connections to 5 (avg chunk time: 1.8s)
Progress: 78.1% (81788928/104857600 bytes) Speed: 9.23 MB/s

Download completed!
Total time: 11.2s
Average speed: 8.9 MB/s
Final connections: 5
```

## Development

### Building from Source

#### Quick Build (Current Platform)
```bash
# Build for your current platform
./build.sh build

# Or manually with Go
go build -o fas-download .
```

#### Cross-Platform Build
```bash
# Build for all platforms
./build.sh all

# This creates binaries for:
# - Linux (AMD64, ARM64)
# - macOS (AMD64, ARM64)
# - Windows (AMD64, ARM64)
```

#### Build Commands
```bash
./build.sh build    # Build for current platform
./build.sh all      # Build for all platforms
./build.sh test     # Run tests
./build.sh lint     # Run linting
./build.sh clean    # Clean build directory
./build.sh help     # Show help
```

### CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment:

#### Workflow Triggers
- **Pull Requests**: Runs tests and linting
- **Push to main**: Runs tests, builds, and creates artifacts
- **Version Tags**: Runs full pipeline + creates releases

#### Build Matrix
- **Platforms**: Linux, macOS, Windows
- **Architectures**: AMD64, ARM64
- **Go Version**: 1.21

#### Release Process
1. **Automated Testing**: All PRs and pushes run comprehensive tests
2. **Cross-Platform Builds**: Builds for 6 platform combinations
3. **Artifact Creation**: Uploads binaries as GitHub artifacts
4. **Release Creation**: Tags trigger automated releases with:
   - Pre-built binaries for all platforms
   - SHA256 checksums
   - Detailed release notes
   - Docker images (multi-arch)

#### Docker Support
```bash
# Build Docker image
docker build -t fas-download .

# Run with Docker
docker run -v $(pwd):/app -w /app fas-download config.yaml
```

### Testing

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test
go test -v -run TestNewAdaptiveDownloader
```
