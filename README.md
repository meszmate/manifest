# Epic Games Manifest Parser

[![CI](https://github.com/meszmate/manifest/actions/workflows/ci.yml/badge.svg)](https://github.com/meszmate/manifest/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/meszmate/manifest.svg)](https://pkg.go.dev/github.com/meszmate/manifest)
[![Go Report Card](https://goreportcard.com/badge/github.com/meszmate/manifest)](https://goreportcard.com/report/github.com/meszmate/manifest)

A Go library and CLI tool for parsing, inspecting, downloading, and verifying Epic Games binary manifest files.

## Features

- Parse binary manifest files from disk or URL
- Inspect metadata, file lists, chunks, and custom fields
- Download game files with parallel chunk downloading, resume, and SHA1 verification
- Verify installed files against a manifest
- Compare two manifests to see what changed (delta)
- Apply delta manifests for efficient patching
- Export manifest data to JSON or CSV
- Cross-platform CLI with progress bars

## Installation

### CLI (Binary)

Download a pre-built binary from [GitHub Releases](https://github.com/meszmate/manifest/releases).

Or install with Go:

```bash
go install github.com/meszmate/manifest/cmd/manifest@latest
```

### Library

```bash
go get github.com/meszmate/manifest
```

## CLI Usage

### Inspect a manifest

```bash
manifest info path/to/file.manifest
manifest info https://example.com/file.manifest
```

### List files

```bash
manifest list file.manifest
manifest list file.manifest --sort size --long
manifest list file.manifest --tag core --filter "*.pak"
```

### Download game files

```bash
manifest download file.manifest \
  --base-url http://download.epicgames.com/Builds/Fortnite/CloudDir/ \
  --output ./game \
  --concurrency 16
```

Use `--dry-run` to see what would be downloaded without actually downloading:

```bash
manifest download file.manifest --base-url http://example.com --dry-run
```

### Verify an installation

```bash
manifest verify file.manifest --path ./game
```

### Compare two manifests

```bash
manifest delta old.manifest new.manifest
manifest delta old.manifest new.manifest --format json
```

### List chunks

```bash
manifest chunks file.manifest --sort size
```

### Export manifest data

```bash
manifest export file.manifest --format json --output manifest.json
manifest export file.manifest --format csv --output files.csv
```

## Library Usage

### Parsing a manifest

Load from a file, URL, or any `io.ReadSeeker`:

```go
// From a file
m, err := manifest.ParseManifestFile("path/to/file.manifest")

// From a URL
m, err := manifest.ParseManifestURL(context.Background(), "https://example.com/file.manifest")

// From raw bytes (e.g. already in memory)
data, _ := os.ReadFile("file.manifest")
m, err := manifest.ParseManifest(bytes.NewReader(data))
```

### Inspecting metadata

```go
m, _ := manifest.ParseManifestFile("file.manifest")

// Basic info
fmt.Println(m.Metadata.AppName)       // e.g. "Fortnite"
fmt.Println(m.Metadata.BuildVersion)   // e.g. "++Fortnite+Release-29.10-CL-33169942"
fmt.Println(m.Metadata.BuildId)        // unique build identifier
fmt.Println(m.Metadata.AppID)          // numeric app ID
fmt.Println(m.Metadata.FeatureLevel)   // e.g. EFeatureLevelStoresUniqueBuildId

// Launch info
fmt.Println(m.Metadata.LaunchExe)      // e.g. "FortniteClient-Win64-Shipping.exe"
fmt.Println(m.Metadata.LaunchCommand)

// Prerequisites
fmt.Println(m.Metadata.PrereqName)
fmt.Println(m.Metadata.PrereqPath)
fmt.Println(m.Metadata.PrereqArgs)
fmt.Println(m.Metadata.PrereqIds)

// Header details
fmt.Println(m.Header.Version)              // feature level
fmt.Println(m.Header.StoredAs)             // 0x01 = compressed, 0x02 = encrypted
fmt.Printf("SHA: %x\n", m.Header.SHAHash) // manifest body hash

// Aggregate sizes
fmt.Printf("Download: %d bytes\n", m.TotalDownloadSize()) // sum of chunk file sizes
fmt.Printf("Install:  %d bytes\n", m.TotalInstallSize())  // sum of file sizes
```

### Iterating files

```go
for _, f := range m.FileManifestList.FileManifestList {
    fmt.Printf("%-60s %10d bytes  %d chunks",
        f.FileName, f.FileSize, len(f.ChunkParts))

    if len(f.InstallTags) > 0 {
        fmt.Printf("  tags: %v", f.InstallTags)
    }
    fmt.Println()
}

// Look up a specific file by path
f := m.FileManifestList.GetFileByPath("FortniteGame/Content/Paks/pakchunk0-WindowsClient.pak")
if f != nil {
    fmt.Printf("%s: %d bytes, SHA: %x\n", f.FileName, f.FileSize, f.SHAHash)
}

// Get all files matching an install tag
coreFiles := m.FileManifestList.GetFilesByTag("core")
fmt.Printf("%d files with 'core' tag\n", len(coreFiles))
```

### Inspecting chunks

```go
for _, c := range m.ChunkDataList.Chunks {
    fmt.Printf("GUID: %s  group: %02d  size: %d  hash: %016X  sha: %x\n",
        c.GUID, c.Group, c.FileSize, c.Hash, c.SHAHash)
}

// Build a CDN URL for a chunk
baseURL := "http://download.epicgames.com/Builds/Fortnite/CloudDir/"
chunk := m.ChunkDataList.Chunks[0]
url := m.ChunkURL(baseURL, chunk)
fmt.Println(url) // full CDN URL including version subdirectory (ChunksV2/V3/V4)

// Or build the URL directly from the chunk
subDir := m.Metadata.FeatureLevel.ChunkSubDir() // "ChunksV4"
url = chunk.GetURL(baseURL + subDir)
```

### Inspecting chunk parts (file → chunk mapping)

Each file is composed of one or more chunk parts. A chunk part is a byte range within a decompressed chunk:

```go
f := m.FileManifestList.GetFileByPath("game/data.pak")
for i, cp := range f.ChunkParts {
    fmt.Printf("  part %d: chunk %s  offset: %d  size: %d  (chunk file size: %d)\n",
        i, cp.Chunk.GUID, cp.Offset, cp.Size, cp.Chunk.FileSize)
}
```

### Reading custom fields

```go
for key, value := range m.CustomFields.Fields {
    fmt.Printf("%s = %s\n", key, value)
}
```

### Downloading and decompressing a single chunk

```go
import "github.com/meszmate/manifest/chunks"

// Download raw chunk bytes from CDN
url := m.ChunkURL(baseURL, chunk)
rawBytes, err := manifest.LoadURLBytes(context.Background(), url)

// Decompress (handles zlib + offset detection)
decompressed, err := chunks.Decompress(rawBytes)

// Or parse the full chunk (header + body) from a reader
reader, err := chunks.ParseChunk(bytes.NewReader(rawBytes))
```

### Manual file assembly from chunks

For full control over the download process:

```go
import "github.com/meszmate/manifest/chunks"

baseURL := "http://download.epicgames.com/Builds/Fortnite/CloudDir/"

for _, f := range m.FileManifestList.FileManifestList {
    outFile, _ := os.Create(f.FileName)
    for _, cp := range f.ChunkParts {
        // Download and decompress the chunk
        url := m.ChunkURL(baseURL, cp.Chunk)
        raw, _ := manifest.LoadURLBytes(context.Background(), url)
        data, _ := chunks.Decompress(raw)

        // Write only the relevant byte range from this chunk
        outFile.Write(data[cp.Offset : cp.Offset+cp.Size])
    }
    outFile.Close()
}
```

### Downloading files (high-level)

The `downloader` package handles parallelism, retries, resume, deduplication, and verification:

```go
import "github.com/meszmate/manifest/downloader"

d := downloader.New(m, downloader.Config{
    OutputDir:      "./game",
    BaseURL:        "http://download.epicgames.com/Builds/Fortnite/CloudDir/",
    Concurrency:    16,             // parallel chunk downloads
    MaxRetries:     5,              // retries per chunk with exponential backoff
    RetryBaseDelay: time.Second,
    VerifyChunks:   true,           // SHA1 verify each chunk
    Resume:         true,           // skip already-downloaded chunks
    Tags:           []string{},     // empty = all files
    ExcludeTags:    []string{"highres", "ondemand"}, // skip optional content
})

result, err := d.Download(context.Background())
fmt.Printf("Downloaded %d files (%d chunks, %d skipped) in %s\n",
    result.TotalFiles, result.TotalChunks, result.SkippedChunks, result.Duration)
```

### Delta patching

```go
// Load the new and old manifests
newManifest, _ := manifest.ParseManifestFile("new.manifest")
oldManifest, _ := manifest.ParseManifestFile("old.manifest")

// Download the delta manifest
deltaBytes, err := manifest.GetDeltaManifest(
    context.Background(),
    "https://cdn.example.com/base",
    newManifest.Metadata.BuildId,
    oldManifest.Metadata.BuildId,
)

// Parse and apply the delta
deltaManifest, _ := manifest.ParseManifest(bytes.NewReader(deltaBytes))
newManifest.ApplyDelta(deltaManifest)
// newManifest now contains updated file entries and new chunks
```

### Verifying an installation

```go
result, err := downloader.VerifyInstallation(m, "./game")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Valid: %d, Missing: %d, Invalid: %d\n",
    result.ValidFiles, result.MissingFiles, result.InvalidFiles)

for _, d := range result.Details {
    if d.Status != downloader.StatusOK {
        fmt.Printf("  %s: %s\n", d.Status, d.Path)
    }
}
```

## CDN Base URLs

Performance benchmarks for common CDN endpoints:

```
http://download.epicgames.com/Builds/Fortnite/CloudDir/                 20-21 ms
http://fastly-download.epicgames.com/Builds/Fortnite/CloudDir/          19-20 ms
http://epicgames-download1.akamaized.net/Builds/Fortnite/CloudDir/      27-28 ms
http://cloudflare.epicgamescdn.com/Builds/Fortnite/CloudDir/            34-36 ms
```

The `--base-url` flag is required because the CDN URL is game-specific and comes from the Epic Games API (which requires authentication). Auto-discovery is out of scope for this tool.

## License

MIT
