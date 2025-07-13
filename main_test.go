package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAdaptiveDownloader(t *testing.T) {
	downloader := NewAdaptiveDownloader("https://example.com/file.zip", "test.zip")

	if downloader.URL != "https://example.com/file.zip" {
		t.Errorf("Expected URL to be 'https://example.com/file.zip', got %s", downloader.URL)
	}

	if downloader.Filename != "test.zip" {
		t.Errorf("Expected filename to be 'test.zip', got %s", downloader.Filename)
	}

	if downloader.MaxConnections != 16 {
		t.Errorf("Expected MaxConnections to be 16, got %d", downloader.MaxConnections)
	}

	if downloader.MinConnections != 2 {
		t.Errorf("Expected MinConnections to be 2, got %d", downloader.MinConnections)
	}

	if downloader.CurrentConnections != 4 {
		t.Errorf("Expected CurrentConnections to be 4, got %d", downloader.CurrentConnections)
	}

	if downloader.ChunkSize != 1024*1024 {
		t.Errorf("Expected ChunkSize to be 1MB, got %d", downloader.ChunkSize)
	}
}

func TestGetFileSize(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}

		w.Header().Set("Content-Length", "1024")
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	downloader := NewAdaptiveDownloader(server.URL, "test.file")

	supportsRanges, err := downloader.getFileSize()
	if err != nil {
		t.Fatalf("getFileSize() returned error: %v", err)
	}

	if !supportsRanges {
		t.Error("Expected server to support range requests")
	}

	if downloader.FileSize != 1024 {
		t.Errorf("Expected file size to be 1024, got %d", downloader.FileSize)
	}
}

func TestGetFileSizeNoRangeSupport(t *testing.T) {
	// Create a test server without range support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2048")
		// Don't set Accept-Ranges header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	downloader := NewAdaptiveDownloader(server.URL, "test.file")

	supportsRanges, err := downloader.getFileSize()
	if err != nil {
		t.Fatalf("getFileSize() returned error: %v", err)
	}

	if supportsRanges {
		t.Error("Expected server to not support range requests")
	}

	if downloader.FileSize != 2048 {
		t.Errorf("Expected file size to be 2048, got %d", downloader.FileSize)
	}
}

func TestCalculateOptimalConnections(t *testing.T) {
	downloader := NewAdaptiveDownloader("https://example.com/file.zip", "test.zip")

	// Test with no chunk times (should not change connections)
	originalConnections := downloader.CurrentConnections
	downloader.calculateOptimalConnections()

	if downloader.CurrentConnections != originalConnections {
		t.Errorf("Expected connections to remain unchanged with no chunk times")
	}

	// Test with fast chunk times (should increase connections)
	downloader.Stats.ChunkTimes = []time.Duration{
		1 * time.Second,
		1 * time.Second,
		1 * time.Second,
	}

	downloader.calculateOptimalConnections()

	if downloader.CurrentConnections != originalConnections+1 {
		t.Errorf("Expected connections to increase with fast chunk times")
	}

	// Test with slow chunk times (should decrease connections)
	downloader.Stats.ChunkTimes = []time.Duration{
		6 * time.Second,
		6 * time.Second,
		6 * time.Second,
	}

	downloader.calculateOptimalConnections()

	if downloader.CurrentConnections >= originalConnections+1 {
		t.Errorf("Expected connections to decrease with slow chunk times")
	}
}

func TestDownloadConfig(t *testing.T) {
	config := DownloadConfig{
		URL: "https://example.com/test.zip",
	}

	if config.URL != "https://example.com/test.zip" {
		t.Errorf("Expected URL to be 'https://example.com/test.zip', got %s", config.URL)
	}
}
