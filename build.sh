#!/bin/bash

# Build script for FAS-Download

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="fas-download"
VERSION=${VERSION:-"dev"}
BUILD_DIR="dist"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to build for specific platform
build_for_platform() {
    local goos=$1
    local goarch=$2
    local ext=$3

    local output_name="${APP_NAME}-${goos}-${goarch}${ext}"

    print_status "Building for ${goos}/${goarch}..."

    GOOS=${goos} GOARCH=${goarch} CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${BUILD_DIR}/${output_name}" .

    if [ $? -eq 0 ]; then
        print_status "✓ Built ${output_name}"
    else
        print_error "✗ Failed to build ${output_name}"
        exit 1
    fi
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    go test -v ./...

    if [ $? -eq 0 ]; then
        print_status "✓ All tests passed"
    else
        print_error "✗ Tests failed"
        exit 1
    fi
}

# Function to run linting
run_lint() {
    print_status "Running go vet..."
    go vet ./...

    if [ $? -eq 0 ]; then
        print_status "✓ go vet passed"
    else
        print_error "✗ go vet failed"
        exit 1
    fi

    # Run staticcheck if available
    if command -v staticcheck >/dev/null 2>&1; then
        print_status "Running staticcheck..."
        staticcheck ./...

        if [ $? -eq 0 ]; then
            print_status "✓ staticcheck passed"
        else
            print_error "✗ staticcheck failed"
            exit 1
        fi
    else
        print_warning "staticcheck not found, skipping (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)"
    fi
}

# Function to clean build directory
clean() {
    print_status "Cleaning build directory..."
    rm -rf "${BUILD_DIR}"
    mkdir -p "${BUILD_DIR}"
}

# Function to generate checksums
generate_checksums() {
    print_status "Generating checksums..."
    cd "${BUILD_DIR}"

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum ${APP_NAME}-* > checksums.txt
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 ${APP_NAME}-* > checksums.txt
    else
        print_warning "No checksum utility found, skipping checksum generation"
        return
    fi

    cd ..
    print_status "✓ Checksums generated"
}

# Function to build all platforms
build_all() {
    print_status "Building for all platforms..."

    # Linux
    build_for_platform "linux" "amd64" ""
    build_for_platform "linux" "arm64" ""

    # macOS
    build_for_platform "darwin" "amd64" ""
    build_for_platform "darwin" "arm64" ""

    # Windows
    build_for_platform "windows" "amd64" ".exe"
    build_for_platform "windows" "arm64" ".exe"

    generate_checksums
}

# Function to build for current platform only
build_current() {
    local goos=$(go env GOOS)
    local goarch=$(go env GOARCH)
    local ext=""

    if [ "$goos" = "windows" ]; then
        ext=".exe"
    fi

    build_for_platform "$goos" "$goarch" "$ext"
}

# Function to show help
show_help() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  build       Build for current platform only"
    echo "  all         Build for all platforms"
    echo "  test        Run tests"
    echo "  lint        Run linting"
    echo "  clean       Clean build directory"
    echo "  help        Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  VERSION     Version to embed in the binary (default: dev)"
    echo ""
    echo "Examples:"
    echo "  $0 build           # Build for current platform"
    echo "  $0 all             # Build for all platforms"
    echo "  VERSION=1.0.0 $0 all  # Build with version 1.0.0"
}

# Main script logic
main() {
    local command=${1:-"build"}

    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Check if we're in the right directory
    if [ ! -f "main.go" ]; then
        print_error "main.go not found. Are you in the right directory?"
        exit 1
    fi

    case $command in
        "build")
            clean
            run_tests
            run_lint
            build_current
            print_status "Build complete! Binary available in ${BUILD_DIR}/"
            ;;
        "all")
            clean
            run_tests
            run_lint
            build_all
            print_status "All builds complete! Binaries available in ${BUILD_DIR}/"
            ;;
        "test")
            run_tests
            ;;
        "lint")
            run_lint
            ;;
        "clean")
            clean
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
