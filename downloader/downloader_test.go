package downloader

import (
	"compress/zlib"
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/meszmate/manifest"
)

// makeChunkPayload creates a fake zlib-compressed chunk payload
// that chunks.Decompress can handle (header + zlib data).
func makeChunkPayload(data []byte) []byte {
	// Build a minimal chunk: 8 header bytes + offset byte + zlib data
	// offset byte at [8] tells Decompress where zlib starts
	var buf []byte
	buf = append(buf, make([]byte, 8)...) // 8 header bytes

	// Compress data with zlib
	var zbuf []byte
	{
		var b = new(zlibBuf)
		w := mustZlibWriter(b)
		w.Write(data)
		w.Close()
		zbuf = b.Bytes()
	}

	// offset points to start of zlib data (position 9 = byte value 9)
	buf = append(buf, 9) // offset byte at position [8]
	buf = append(buf, zbuf...)
	return buf
}

type zlibBuf struct {
	data []byte
}

func (z *zlibBuf) Write(p []byte) (n int, err error) {
	z.data = append(z.data, p...)
	return len(p), nil
}

func (z *zlibBuf) Bytes() []byte {
	return z.data
}

func mustZlibWriter(w *zlibBuf) *zlib.Writer {
	zw, err := zlib.NewWriterLevel(w, zlib.DefaultCompression)
	if err != nil {
		panic(err)
	}
	return zw
}

func buildTestManifest(chunkData []byte, chunkGUID uuid.UUID) *manifest.BinaryManifest {
	shaHash := sha1.Sum(chunkData)

	chunk := &manifest.Chunk{
		GUID:       chunkGUID,
		Hash:       0xABCD,
		SHAHash:    shaHash,
		Group:      1,
		WindowSize: 1048576,
		FileSize:   uint64(len(chunkData)),
	}

	return &manifest.BinaryManifest{
		Metadata: &manifest.FManifestMeta{
			FeatureLevel: manifest.EFeatureLevelLatest,
			AppName:      "TestApp",
			BuildVersion: "1.0.0",
		},
		ChunkDataList: &manifest.FChunkDataList{
			Chunks:      []*manifest.Chunk{chunk},
			ChunkLookup: map[uuid.UUID]uint32{chunkGUID: 0},
		},
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{
					FileName: "test/file.dat",
					FileSize: uint64(len(chunkData)),
					ChunkParts: []manifest.ChunkPart{
						{
							ParentGUID: chunkGUID,
							Offset:     0,
							Size:       uint32(len(chunkData)),
							Chunk:      chunk,
						},
					},
				},
			},
		},
	}
}

func TestDownload_DryRun(t *testing.T) {
	chunkData := []byte("hello world test data")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir: dir,
		BaseURL:   "http://example.com",
		DryRun:    true,
	})

	result, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("DryRun download failed: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", result.TotalFiles)
	}
	if result.TotalChunks != 1 {
		t.Errorf("expected 1 chunk, got %d", result.TotalChunks)
	}

	// Verify no files were created
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir in dry run, got %d entries", len(entries))
	}
}

func TestDownload_Success(t *testing.T) {
	chunkData := []byte("hello world test data for download")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	payload := makeChunkPayload(chunkData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:    dir,
		BaseURL:      server.URL,
		Concurrency:  2,
		VerifyChunks: true,
	})

	result, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if result.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", result.TotalFiles)
	}
	if result.FailedChunks != 0 {
		t.Errorf("expected 0 failed chunks, got %d", result.FailedChunks)
	}

	// Verify file was written
	outPath := filepath.Join(dir, "test", "file.dat")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != string(chunkData) {
		t.Errorf("output data mismatch: got %q", string(data))
	}
}

func TestDownload_RetryOnFailure(t *testing.T) {
	chunkData := []byte("retry test data")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	payload := makeChunkPayload(chunkData)
	var attempts int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:      dir,
		BaseURL:        server.URL,
		Concurrency:    1,
		MaxRetries:     5,
		RetryBaseDelay: time.Millisecond, // fast retries for tests
		VerifyChunks:   true,
	})

	result, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("download with retries failed: %v", err)
	}
	if result.FailedChunks != 0 {
		t.Errorf("expected 0 failed chunks, got %d", result.FailedChunks)
	}
	if atomic.LoadInt64(&attempts) < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestDownload_AllRetriesFail(t *testing.T) {
	chunkData := []byte("fail test data")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:      dir,
		BaseURL:        server.URL,
		Concurrency:    1,
		MaxRetries:     2,
		RetryBaseDelay: time.Millisecond,
	})

	result, err := d.Download(context.Background())
	if err == nil {
		t.Fatal("expected error when all retries fail")
	}
	if result.FailedChunks != 1 {
		t.Errorf("expected 1 failed chunk, got %d", result.FailedChunks)
	}
}

func TestDownload_ContextCancellation(t *testing.T) {
	chunkData := []byte("cancel test data")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until client cancels
	}))
	defer server.Close()

	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	d := New(m, Config{
		OutputDir:      dir,
		BaseURL:        server.URL,
		Concurrency:    1,
		MaxRetries:     1,
		RetryBaseDelay: time.Millisecond,
	})

	_, err := d.Download(ctx)
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
}

func TestDownload_SHA1Mismatch(t *testing.T) {
	chunkData := []byte("sha1 test data")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	// Serve different data than what the manifest expects
	wrongData := []byte("wrong data!!!!!!!")
	payload := makeChunkPayload(wrongData)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:      dir,
		BaseURL:        server.URL,
		Concurrency:    1,
		MaxRetries:     1,
		RetryBaseDelay: time.Millisecond,
		VerifyChunks:   true,
	})

	result, err := d.Download(context.Background())
	if err == nil {
		t.Fatal("expected error on SHA1 mismatch")
	}
	if result.FailedChunks != 1 {
		t.Errorf("expected 1 failed chunk, got %d", result.FailedChunks)
	}
}

func TestDownload_Resume(t *testing.T) {
	chunkData := []byte("resume test data here")
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	m := buildTestManifest(chunkData, chunkGUID)

	dir := t.TempDir()

	// Pre-populate cache
	cacheDir := filepath.Join(dir, ".manifest-cache", "chunks")
	os.MkdirAll(cacheDir, 0o755)
	cachePath := filepath.Join(cacheDir, chunkGUID.String()+".chunk")
	os.WriteFile(cachePath, chunkData, 0o644)

	var serverCalled int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&serverCalled, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := New(m, Config{
		OutputDir:    dir,
		BaseURL:      server.URL,
		Concurrency:  1,
		Resume:       true,
		VerifyChunks: false,
	})

	result, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("resume download failed: %v", err)
	}
	if result.SkippedChunks != 1 {
		t.Errorf("expected 1 skipped chunk, got %d", result.SkippedChunks)
	}
	if atomic.LoadInt64(&serverCalled) != 0 {
		t.Errorf("expected no server calls with resume, got %d", serverCalled)
	}
}

func TestDownload_TagFilter(t *testing.T) {
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	chunk := &manifest.Chunk{
		GUID:     chunkGUID,
		Hash:     0xABCD,
		Group:    1,
		FileSize: 100,
	}

	m := &manifest.BinaryManifest{
		Metadata: &manifest.FManifestMeta{
			FeatureLevel: manifest.EFeatureLevelLatest,
		},
		ChunkDataList: &manifest.FChunkDataList{
			Chunks:      []*manifest.Chunk{chunk},
			ChunkLookup: map[uuid.UUID]uint32{chunkGUID: 0},
		},
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{FileName: "base.dat", InstallTags: []string{}, ChunkParts: []manifest.ChunkPart{{Chunk: chunk}}},
				{FileName: "hires.dat", InstallTags: []string{"highres"}, ChunkParts: []manifest.ChunkPart{{Chunk: chunk}}},
				{FileName: "extra.dat", InstallTags: []string{"extra"}, ChunkParts: []manifest.ChunkPart{{Chunk: chunk}}},
			},
		},
	}

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:   dir,
		BaseURL:     "http://example.com",
		DryRun:      true,
		ExcludeTags: []string{"highres"},
	})

	result, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("tag filter download failed: %v", err)
	}
	if result.TotalFiles != 2 {
		t.Errorf("expected 2 files after exclude filter, got %d", result.TotalFiles)
	}
}

func TestDownload_ConcurrencyLimit(t *testing.T) {
	chunkData := []byte("concurrency test")
	payload := makeChunkPayload(chunkData)

	var maxConcurrent int64
	var current int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&current, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if n <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		w.Write(payload)
	}))
	defer server.Close()

	// Build manifest with multiple chunks
	var chunkList []*manifest.Chunk
	lookup := make(map[uuid.UUID]uint32)
	var files []manifest.File

	for i := 0; i < 10; i++ {
		g := uuid.New()
		shaHash := sha1.Sum(chunkData)
		c := &manifest.Chunk{
			GUID:     g,
			Hash:     uint64(i),
			SHAHash:  shaHash,
			Group:    1,
			FileSize: uint64(len(chunkData)),
		}
		chunkList = append(chunkList, c)
		lookup[g] = uint32(i)
		files = append(files, manifest.File{
			FileName: fmt.Sprintf("file_%d.dat", i),
			FileSize: uint64(len(chunkData)),
			ChunkParts: []manifest.ChunkPart{
				{ParentGUID: g, Offset: 0, Size: uint32(len(chunkData)), Chunk: c},
			},
		})
	}

	m := &manifest.BinaryManifest{
		Metadata: &manifest.FManifestMeta{FeatureLevel: manifest.EFeatureLevelLatest},
		ChunkDataList: &manifest.FChunkDataList{
			Chunks:      chunkList,
			ChunkLookup: lookup,
		},
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: files,
		},
	}

	dir := t.TempDir()
	d := New(m, Config{
		OutputDir:      dir,
		BaseURL:        server.URL,
		Concurrency:    3,
		VerifyChunks:   true,
		RetryBaseDelay: time.Millisecond,
	})

	_, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("concurrent download failed: %v", err)
	}

	if atomic.LoadInt64(&maxConcurrent) > 3 {
		t.Errorf("concurrency exceeded limit: max was %d, expected <= 3", maxConcurrent)
	}
}

func TestFilterFiles_Tags(t *testing.T) {
	chunk := &manifest.Chunk{GUID: uuid.New()}

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{FileName: "a.dat", InstallTags: []string{"core"}},
				{FileName: "b.dat", InstallTags: []string{"highres"}},
				{FileName: "c.dat", InstallTags: []string{"core", "extra"}},
			},
		},
		ChunkDataList: &manifest.FChunkDataList{Chunks: []*manifest.Chunk{chunk}},
		Metadata:      &manifest.FManifestMeta{FeatureLevel: manifest.EFeatureLevelLatest},
	}

	// Include only "core" tag
	d := New(m, Config{BaseURL: "http://x", Tags: []string{"core"}})
	files := d.filterFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 files with 'core' tag, got %d", len(files))
	}

	// Exclude "highres" tag
	d = New(m, Config{BaseURL: "http://x", ExcludeTags: []string{"highres"}})
	files = d.filterFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 files after excluding 'highres', got %d", len(files))
	}
}

func TestNoopProgress(t *testing.T) {
	var p NoopProgress
	// Just verify it doesn't panic
	p.Start(10, 1000)
	p.ChunkCompleted(100)
	p.ChunkFailed("guid", fmt.Errorf("err"))
	p.FileCompleted("path", 100)
	p.Finish()
}
