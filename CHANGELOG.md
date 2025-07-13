# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CI/CD pipeline with GitHub Actions
- Docker support with multi-stage builds
- Cross-platform builds for Linux, macOS, and Windows
- Automated releases with checksums

## [1.0.0] - 2024-01-01

### Added
- Initial release of FAS-Download
- Concurrent file downloading with adaptive connection management
- YAML configuration support
- Automatic file size detection via HTTP headers
- Range request support with fallback for non-supporting servers
- Real-time progress tracking and speed reporting
- Cross-platform compatibility
- Comprehensive error handling

### Features
- **Adaptive Connection Management**: Automatically adjusts concurrent connections (2-16) based on performance
- **Range Request Support**: Uses HTTP range requests for parallel chunk downloads
- **Fallback Mode**: Gracefully handles servers without range request support
- **Progress Tracking**: Real-time download progress with speed and ETA
- **YAML Configuration**: Simple configuration with just URL required
- **Cross-Platform**: Works on Linux, macOS, and Windows

### Technical Details
- Written in Go 1.21
- Uses goroutines for concurrent downloads
- Memory-efficient streaming with optimized buffers
- Thread-safe statistics tracking
- Automatic file size detection via Content-Length headers
