package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// DownloadConfig represents the YAML configuration for downloads
type DownloadConfig struct {
	URL string `yaml:"url"`
}

// ChunkInfo represents information about a file chunk to download
type ChunkInfo struct {
	Start int64
	End   int64
	Index int
}

// DownloadStats tracks download performance metrics
type DownloadStats struct {
	BytesDownloaded int64
	StartTime       time.Time
	ChunkTimes      []time.Duration
	mu              sync.Mutex
}

// AdaptiveDownloader manages concurrent downloads with adaptive connection management
type AdaptiveDownloader struct {
	URL                string
	Filename           string
	MaxConnections     int
	MinConnections     int
	CurrentConnections int
	ChunkSize          int64
	FileSize           int64
	Stats              *DownloadStats
	mu                 sync.Mutex
}

// NewAdaptiveDownloader creates a new adaptive downloader
func NewAdaptiveDownloader(url, filename string) *AdaptiveDownloader {
	return &AdaptiveDownloader{
		URL:                url,
		Filename:           filename,
		MaxConnections:     16,
		MinConnections:     2,
		CurrentConnections: 4,
		ChunkSize:          1024 * 1024, // 1MB chunks
		Stats: &DownloadStats{
			StartTime:  time.Now(),
			ChunkTimes: make([]time.Duration, 0),
		},
	}
}

// getFileSize gets the file size from the server and checks range support
func (d *AdaptiveDownloader) getFileSize() (bool, error) {
	resp, err := http.Head(d.URL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("server returned status: %s", resp.Status)
	}

	contentLength := resp.Header.Get("Content-Length")

	// If HEAD request doesn't provide content length, we'll handle it in download
	if contentLength == "" {
		fmt.Printf("Server didn't provide content length in HEAD request. Will determine during download.\n")
		d.FileSize = -1   // Mark as unknown
		return false, nil // Can't do range requests without knowing size
	}

	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return false, err
	}

	d.FileSize = size

	// Check if server supports range requests
	supportsRanges := resp.Header.Get("Accept-Ranges") == "bytes"
	return supportsRanges, nil
}

// downloadChunk downloads a specific chunk of the file
func (d *AdaptiveDownloader) downloadChunk(chunk ChunkInfo, file *os.File) error {
	start := time.Now()
	defer func() {
		d.Stats.mu.Lock()
		d.Stats.ChunkTimes = append(d.Stats.ChunkTimes, time.Since(start))
		d.Stats.mu.Unlock()
	}()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", d.URL, nil)
	if err != nil {
		return err
	}

	// Set range header for partial content
	rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End)
	req.Header.Set("Range", rangeHeader)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// Create a buffer to read the chunk
	buffer := make([]byte, 32*1024) // 32KB buffer
	offset := chunk.Start

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Write to file at the correct offset
			_, writeErr := file.WriteAt(buffer[:n], offset)
			if writeErr != nil {
				return writeErr
			}
			offset += int64(n)

			// Update stats
			d.Stats.mu.Lock()
			d.Stats.BytesDownloaded += int64(n)
			d.Stats.mu.Unlock()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// downloadSingleConnection downloads the file in a single connection (fallback for servers without range support)
func (d *AdaptiveDownloader) downloadSingleConnection() error {
	fmt.Printf("Downloading file in single connection...\n")

	// Create output file
	file, err := os.Create(d.Filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create HTTP client and request
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(d.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// Start progress reporter
	go d.reportProgress()

	// Copy the entire file
	buffer := make([]byte, 32*1024) // 32KB buffer
	start := time.Now()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}

			// Update stats
			d.Stats.mu.Lock()
			d.Stats.BytesDownloaded += int64(n)
			d.Stats.mu.Unlock()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	duration := time.Since(start)
	actualFileSize := d.Stats.BytesDownloaded
	speed := float64(actualFileSize) / duration.Seconds() / 1024 / 1024 // MB/s

	fmt.Printf("\nDownload completed!\n")
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("File size: %d bytes\n", actualFileSize)
	fmt.Printf("Average speed: %.2f MB/s\n", speed)

	return nil
}

// calculateOptimalConnections adapts the number of connections based on performance
func (d *AdaptiveDownloader) calculateOptimalConnections() {
	d.Stats.mu.Lock()
	defer d.Stats.mu.Unlock()

	if len(d.Stats.ChunkTimes) < 3 {
		return // Not enough data yet
	}

	// Calculate average time for recent chunks
	recent := d.Stats.ChunkTimes[len(d.Stats.ChunkTimes)-3:]
	var totalTime time.Duration
	for _, t := range recent {
		totalTime += t
	}
	avgTime := totalTime / time.Duration(len(recent))

	d.mu.Lock()
	defer d.mu.Unlock()

	// Adaptive logic: if chunks are completing quickly, increase connections
	if avgTime < 2*time.Second && d.CurrentConnections < d.MaxConnections {
		d.CurrentConnections++
		fmt.Printf("Increasing connections to %d (avg chunk time: %v)\n", d.CurrentConnections, avgTime)
	} else if avgTime > 5*time.Second && d.CurrentConnections > d.MinConnections {
		d.CurrentConnections--
		fmt.Printf("Decreasing connections to %d (avg chunk time: %v)\n", d.CurrentConnections, avgTime)
	}
}

// Download performs the concurrent download
func (d *AdaptiveDownloader) Download() error {
	// Get file size and check if server supports range requests
	supportsRanges, err := d.getFileSize()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	if d.FileSize > 0 {
		fmt.Printf("File size: %d bytes\n", d.FileSize)
	} else {
		fmt.Printf("File size: unknown\n")
	}

	if !supportsRanges {
		fmt.Printf("Server doesn't support range requests. Downloading in single connection.\n")
		return d.downloadSingleConnection()
	}

	fmt.Printf("Starting download with %d connections\n", d.CurrentConnections)

	// Create output file
	file, err := os.Create(d.Filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Pre-allocate file space
	err = file.Truncate(d.FileSize)
	if err != nil {
		return err
	}

	// Create chunks
	chunks := make([]ChunkInfo, 0)
	for i := int64(0); i < d.FileSize; i += d.ChunkSize {
		end := i + d.ChunkSize - 1
		if end >= d.FileSize {
			end = d.FileSize - 1
		}
		chunks = append(chunks, ChunkInfo{
			Start: i,
			End:   end,
			Index: len(chunks),
		})
	}

	fmt.Printf("Created %d chunks\n", len(chunks))

	// Download chunks concurrently
	chunkChan := make(chan ChunkInfo, len(chunks))
	for _, chunk := range chunks {
		chunkChan <- chunk
	}
	close(chunkChan)

	var wg sync.WaitGroup
	errChan := make(chan error, d.CurrentConnections)

	// Start progress reporter
	go d.reportProgress()

	// Dynamic worker management
	for i := 0; i < d.CurrentConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range chunkChan {
				if err := d.downloadChunk(chunk, file); err != nil {
					errChan <- fmt.Errorf("chunk %d failed: %v", chunk.Index, err)
					return
				}

				// Periodically adapt connections
				if chunk.Index%5 == 0 {
					d.calculateOptimalConnections()
				}
			}
		}()
	}

	// Wait for all chunks to complete
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
	}

	duration := time.Since(d.Stats.StartTime)
	speed := float64(d.FileSize) / duration.Seconds() / 1024 / 1024 // MB/s

	fmt.Printf("\nDownload completed!\n")
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Average speed: %.2f MB/s\n", speed)
	fmt.Printf("Final connections: %d\n", d.CurrentConnections)

	return nil
}

// reportProgress shows download progress
func (d *AdaptiveDownloader) reportProgress() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.Stats.mu.Lock()
			downloaded := d.Stats.BytesDownloaded
			d.Stats.mu.Unlock()

			if d.FileSize > 0 && downloaded >= d.FileSize {
				return
			}

			elapsed := time.Since(d.Stats.StartTime)
			speed := float64(downloaded) / elapsed.Seconds() / 1024 / 1024 // MB/s

			if d.FileSize > 0 {
				progress := float64(downloaded) / float64(d.FileSize) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes) Speed: %.2f MB/s",
					progress, downloaded, d.FileSize, speed)
			} else {
				fmt.Printf("\rDownloaded: %d bytes Speed: %.2f MB/s", downloaded, speed)
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <config.yaml> [output_filename]")
		fmt.Println("Example: go run main.go config.yaml")
		fmt.Println("\nConfig YAML format:")
		fmt.Println("url: https://example.com/file.zip")
		os.Exit(1)
	}

	configFile := os.Args[1]

	// Read YAML configuration
	configData, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	var config DownloadConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		fmt.Printf("Error parsing YAML config: %v\n", err)
		os.Exit(1)
	}

	if config.URL == "" {
		fmt.Println("Error: URL is required in config")
		os.Exit(1)
	}

	filename := "downloaded_file"

	if len(os.Args) > 2 {
		filename = os.Args[2]
	} else {
		// Try to extract filename from URL
		if name := filepath.Base(config.URL); name != "/" && name != "." {
			filename = name
		}
	}

	fmt.Printf("Downloading %s to %s\n", config.URL, filename)

	downloader := NewAdaptiveDownloader(config.URL, filename)

	if err := downloader.Download(); err != nil {
		fmt.Printf("Download failed: %v\n", err)
		os.Exit(1)
	}
}
