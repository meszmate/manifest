// Package downloader provides a download engine for Epic Games manifests.
// It handles chunk downloading, decompression, verification, and file assembly
// with support for concurrency, resume, and progress reporting.
package downloader

// ProgressReporter receives download progress updates.
// Implementations must be safe for concurrent use.
type ProgressReporter interface {
	// Start is called once at the beginning of a download with totals.
	Start(totalChunks int, totalBytes uint64)
	// ChunkCompleted is called after a chunk is successfully downloaded.
	ChunkCompleted(bytes uint64)
	// ChunkFailed is called when a chunk download fails after all retries.
	ChunkFailed(guid string, err error)
	// FileCompleted is called after a file is fully assembled on disk.
	FileCompleted(path string, size uint64)
	// Finish is called once when the download is complete.
	Finish()
}

// NoopProgress is a no-op implementation of ProgressReporter for library users
// who don't need progress reporting.
type NoopProgress struct{}

func (NoopProgress) Start(int, uint64)           {}
func (NoopProgress) ChunkCompleted(uint64)        {}
func (NoopProgress) ChunkFailed(string, error)    {}
func (NoopProgress) FileCompleted(string, uint64) {}
func (NoopProgress) Finish()                      {}
