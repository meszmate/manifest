package downloader

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/meszmate/manifest"
	"github.com/meszmate/manifest/chunks"
)

// Config controls download behavior.
type Config struct {
	// OutputDir is the directory where files will be written.
	OutputDir string
	// BaseURL is the CDN base URL (required).
	BaseURL string
	// Concurrency is the number of parallel chunk downloads (default 16).
	Concurrency int
	// MaxRetries is the maximum number of retries per chunk (default 5).
	MaxRetries int
	// RetryBaseDelay is the base delay for exponential backoff (default 1s).
	RetryBaseDelay time.Duration
	// VerifyChunks enables SHA1 verification of downloaded chunks (default true).
	VerifyChunks bool
	// Resume enables skipping chunks already in the cache (default true).
	Resume bool
	// HTTPClient is the HTTP client to use (default http.DefaultClient).
	HTTPClient *http.Client
	// Progress receives download progress updates.
	Progress ProgressReporter
	// Tags filters files to only those matching any of the given tags.
	// An empty slice means all files.
	Tags []string
	// ExcludeTags filters out files matching any of the given tags.
	ExcludeTags []string
	// DryRun skips actual downloading and file assembly.
	DryRun bool
}

func (c *Config) defaults() {
	if c.Concurrency <= 0 {
		c.Concurrency = 16
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 5
	}
	if c.RetryBaseDelay <= 0 {
		c.RetryBaseDelay = time.Second
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Progress == nil {
		c.Progress = NoopProgress{}
	}
}

// Result holds download statistics.
type Result struct {
	TotalFiles      int
	TotalChunks     int
	TotalBytes      uint64
	DownloadedBytes uint64
	SkippedChunks   int
	FailedChunks    int
	Errors          []error
	Duration        time.Duration
}

// Downloader orchestrates downloading a manifest's files from a CDN.
type Downloader struct {
	manifest *manifest.BinaryManifest
	cfg      Config
}

// New creates a new Downloader for the given manifest and config.
func New(m *manifest.BinaryManifest, cfg Config) *Downloader {
	cfg.defaults()
	return &Downloader{manifest: m, cfg: cfg}
}

// chunkTask represents a single chunk to download.
type chunkTask struct {
	chunk *manifest.Chunk
	url   string
}

// Download executes the download pipeline: plan, download chunks, assemble files.
func (d *Downloader) Download(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{}

	// Phase 1: Plan — filter files, deduplicate chunks
	files := d.filterFiles()
	result.TotalFiles = len(files)

	neededChunks := make(map[uuid.UUID]*manifest.Chunk)
	for _, f := range files {
		for _, cp := range f.ChunkParts {
			neededChunks[cp.Chunk.GUID] = cp.Chunk
		}
	}
	result.TotalChunks = len(neededChunks)

	var totalBytes uint64
	for _, c := range neededChunks {
		totalBytes += c.FileSize
	}
	result.TotalBytes = totalBytes

	d.cfg.Progress.Start(len(neededChunks), totalBytes)

	if d.cfg.DryRun {
		d.cfg.Progress.Finish()
		result.Duration = time.Since(start)
		return result, nil
	}

	// Ensure cache directory exists
	cacheDir := filepath.Join(d.cfg.OutputDir, ".manifest-cache", "chunks")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	// Phase 2: Download chunks with worker pool
	tasks := make(chan chunkTask, len(neededChunks))
	for _, c := range neededChunks {
		tasks <- chunkTask{
			chunk: c,
			url:   d.manifest.ChunkURL(d.cfg.BaseURL, c),
		}
	}
	close(tasks)

	var (
		mu           sync.Mutex
		downloadErr  []error
		skipped      int64
		downloaded   uint64
		failedChunks int64
	)

	var wg sync.WaitGroup
	for i := 0; i < d.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				if ctx.Err() != nil {
					return
				}

				cachePath := filepath.Join(cacheDir, task.chunk.GUID.String()+".chunk")

				// Resume: check if chunk already exists in cache
				if d.cfg.Resume {
					if info, err := os.Stat(cachePath); err == nil && info.Size() > 0 {
						atomic.AddInt64(&skipped, 1)
						d.cfg.Progress.ChunkCompleted(task.chunk.FileSize)
						atomic.AddUint64(&downloaded, task.chunk.FileSize)
						continue
					}
				}

				data, err := d.downloadChunk(ctx, task)
				if err != nil {
					atomic.AddInt64(&failedChunks, 1)
					d.cfg.Progress.ChunkFailed(task.chunk.GUID.String(), err)
					mu.Lock()
					downloadErr = append(downloadErr, fmt.Errorf("chunk %s: %w", task.chunk.GUID, err))
					mu.Unlock()
					continue
				}

				// Write to temp file, then rename for atomicity
				tmpPath := cachePath + ".tmp"
				if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
					mu.Lock()
					downloadErr = append(downloadErr, fmt.Errorf("write chunk %s: %w", task.chunk.GUID, err))
					mu.Unlock()
					continue
				}
				if err := os.Rename(tmpPath, cachePath); err != nil {
					mu.Lock()
					downloadErr = append(downloadErr, fmt.Errorf("rename chunk %s: %w", task.chunk.GUID, err))
					mu.Unlock()
					continue
				}

				d.cfg.Progress.ChunkCompleted(task.chunk.FileSize)
				atomic.AddUint64(&downloaded, task.chunk.FileSize)
			}
		}()
	}
	wg.Wait()

	result.SkippedChunks = int(skipped)
	result.FailedChunks = int(failedChunks)
	result.DownloadedBytes = downloaded
	result.Errors = downloadErr

	if ctx.Err() != nil {
		d.cfg.Progress.Finish()
		result.Duration = time.Since(start)
		return result, ctx.Err()
	}

	if result.FailedChunks > 0 {
		d.cfg.Progress.Finish()
		result.Duration = time.Since(start)
		return result, fmt.Errorf("%d chunks failed to download", result.FailedChunks)
	}

	// Phase 3: Assemble files from cached chunks
	for _, f := range files {
		if ctx.Err() != nil {
			break
		}

		outPath := filepath.Join(d.cfg.OutputDir, filepath.FromSlash(f.FileName))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", f.FileName, err)
		}

		tmpPath := outPath + ".tmp"
		outFile, err := os.Create(tmpPath)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", f.FileName, err)
		}

		var assembleErr error
		for _, cp := range f.ChunkParts {
			cachePath := filepath.Join(cacheDir, cp.Chunk.GUID.String()+".chunk")
			chunkData, err := os.ReadFile(cachePath)
			if err != nil {
				assembleErr = fmt.Errorf("read cached chunk %s: %w", cp.Chunk.GUID, err)
				break
			}

			end := uint32(cp.Offset) + uint32(cp.Size)
			if int(end) > len(chunkData) {
				assembleErr = fmt.Errorf("chunk %s: offset %d + size %d exceeds data length %d",
					cp.Chunk.GUID, cp.Offset, cp.Size, len(chunkData))
				break
			}

			if _, err := outFile.Write(chunkData[cp.Offset:end]); err != nil {
				assembleErr = fmt.Errorf("write %s: %w", f.FileName, err)
				break
			}
		}
		outFile.Close()

		if assembleErr != nil {
			os.Remove(tmpPath)
			return nil, assembleErr
		}

		if err := os.Rename(tmpPath, outPath); err != nil {
			return nil, fmt.Errorf("rename %s: %w", f.FileName, err)
		}

		d.cfg.Progress.FileCompleted(f.FileName, f.FileSize)
	}

	// Clean up chunk cache after successful assembly
	os.RemoveAll(filepath.Join(d.cfg.OutputDir, ".manifest-cache"))

	d.cfg.Progress.Finish()
	result.Duration = time.Since(start)
	return result, nil
}

// downloadChunk downloads and decompresses a single chunk with retries.
func (d *Downloader) downloadChunk(ctx context.Context, task chunkTask) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= d.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := d.cfg.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			delay = min(delay, 30*time.Second)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		data, err := d.fetchChunk(ctx, task.url)
		if err != nil {
			lastErr = err
			continue
		}

		// Decompress
		decompressed, err := chunks.Decompress(data)
		if err != nil {
			lastErr = fmt.Errorf("decompress: %w", err)
			continue
		}

		// SHA1 verification
		if d.cfg.VerifyChunks {
			h := sha1.Sum(decompressed)
			if h != task.chunk.SHAHash {
				lastErr = fmt.Errorf("SHA1 mismatch: expected %x, got %x", task.chunk.SHAHash, h)
				continue
			}
		}

		return decompressed, nil
	}
	return nil, fmt.Errorf("after %d retries: %w", d.cfg.MaxRetries, lastErr)
}

// fetchChunk performs a single HTTP GET for a chunk.
func (d *Downloader) fetchChunk(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// filterFiles returns files matching the configured tag filters.
func (d *Downloader) filterFiles() []manifest.File {
	allFiles := d.manifest.FileManifestList.FileManifestList

	if len(d.cfg.Tags) == 0 && len(d.cfg.ExcludeTags) == 0 {
		return allFiles
	}

	tagSet := make(map[string]struct{}, len(d.cfg.Tags))
	for _, t := range d.cfg.Tags {
		tagSet[t] = struct{}{}
	}

	excludeSet := make(map[string]struct{}, len(d.cfg.ExcludeTags))
	for _, t := range d.cfg.ExcludeTags {
		excludeSet[t] = struct{}{}
	}

	var filtered []manifest.File
	for _, f := range allFiles {
		if len(excludeSet) > 0 && hasAnyTag(f.InstallTags, excludeSet) {
			continue
		}
		if len(tagSet) > 0 && !hasAnyTag(f.InstallTags, tagSet) {
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

func hasAnyTag(tags []string, set map[string]struct{}) bool {
	for _, t := range tags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}
