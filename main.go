// Package manifest provides a parser for Epic Games binary manifest files.
// These manifests describe game installations including file lists, chunk data,
// and metadata needed for downloading and patching.
package manifest

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/meszmate/manifest/binreader"
)

// ErrBadMagic is returned when the manifest does not start with the expected magic number.
var ErrBadMagic = errors.New("bad magic found, must be 0x44BEC00C")

// BinaryManifestMagic is the expected magic number at the start of every binary manifest.
const BinaryManifestMagic = 0x44BEC00C

// BinaryManifest is the top-level structure representing a parsed Epic Games manifest.
type BinaryManifest struct {
	Header           *FManifestHeader
	Metadata         *FManifestMeta
	ChunkDataList    *FChunkDataList
	FileManifestList *FFileManifestList
	CustomFields     *FCustomFields
}

// String returns a one-line summary of the manifest.
func (m *BinaryManifest) String() string {
	return fmt.Sprintf("Manifest: %s %s (%d files, %d chunks)",
		m.Metadata.AppName, m.Metadata.BuildVersion,
		len(m.FileManifestList.FileManifestList),
		len(m.ChunkDataList.Chunks))
}

// TotalDownloadSize returns the sum of all chunk file sizes (download bytes).
func (m *BinaryManifest) TotalDownloadSize() uint64 {
	var total uint64
	for _, c := range m.ChunkDataList.Chunks {
		total += c.FileSize
	}
	return total
}

// TotalInstallSize returns the sum of all file sizes (installed bytes).
func (m *BinaryManifest) TotalInstallSize() uint64 {
	var total uint64
	for _, f := range m.FileManifestList.FileManifestList {
		total += f.FileSize
	}
	return total
}

// ChunkURL builds the full CDN URL for the given chunk using the manifest's feature level
// to determine the correct chunk sub-directory (ChunksV2/V3/V4).
func (m *BinaryManifest) ChunkURL(baseURL string, c *Chunk) string {
	baseURL = strings.TrimRight(baseURL, "/")
	subDir := m.Metadata.FeatureLevel.ChunkSubDir()
	return c.GetURL(baseURL + "/" + subDir)
}

// GetDeltaManifest downloads a delta manifest file from the given base URL.
// It constructs the path as baseURL/Deltas/newBuildID/oldBuildID.delta.
func GetDeltaManifest(ctx context.Context, baseURL string, newBuildID string, oldBuildID string) ([]byte, error) {
	url := baseURL + "/Deltas/" + newBuildID + "/" + oldBuildID + ".delta"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching delta manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("delta manifest returned status %d", resp.StatusCode)
	}

	deltaBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading delta manifest body: %w", err)
	}
	return deltaBytes, nil
}

// ApplyDelta merges a delta manifest into this manifest, updating existing files
// and adding new ones. New chunks are also merged into the chunk data list.
func (m *BinaryManifest) ApplyDelta(deltaManifest *BinaryManifest) {
	added := make(map[string]struct{})
	for i, f := range m.FileManifestList.FileManifestList {
		deltaFile := deltaManifest.FileManifestList.GetFileByPath(f.FileName)
		if deltaFile == nil {
			continue
		}
		m.FileManifestList.FileManifestList[i] = *deltaFile
		added[deltaFile.FileName] = struct{}{}
	}
	for _, deltaFile := range deltaManifest.FileManifestList.FileManifestList {
		if _, ok := added[deltaFile.FileName]; !ok {
			m.FileManifestList.FileManifestList = append(m.FileManifestList.FileManifestList, deltaFile)
		}
	}
	m.FileManifestList.Count = uint32(len(m.FileManifestList.FileManifestList))

	for _, chunk := range deltaManifest.ChunkDataList.Chunks {
		_, ok := m.ChunkDataList.ChunkLookup[chunk.GUID]
		if !ok {
			idx := uint32(len(m.ChunkDataList.Chunks))
			m.ChunkDataList.Chunks = append(m.ChunkDataList.Chunks, chunk)
			m.ChunkDataList.ChunkLookup[chunk.GUID] = idx
		}
	}
	m.ChunkDataList.Count = uint32(len(m.ChunkDataList.Chunks))
}

// LoadFileBytes reads a file from disk and returns its contents.
func LoadFileBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}
	return data, nil
}

// LoadURLBytes downloads the contents of a URL and returns the response body.
func LoadURLBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return data, nil
}

// ParseManifestFile is a convenience function that reads a manifest file from disk
// and parses it.
func ParseManifestFile(path string) (*BinaryManifest, error) {
	data, err := LoadFileBytes(path)
	if err != nil {
		return nil, err
	}
	return ParseManifest(bytes.NewReader(data))
}

// ParseManifestURL is a convenience function that downloads a manifest from a URL
// and parses it.
func ParseManifestURL(ctx context.Context, url string) (*BinaryManifest, error) {
	data, err := LoadURLBytes(ctx, url)
	if err != nil {
		return nil, err
	}
	return ParseManifest(bytes.NewReader(data))
}

// ParseManifest parses a binary manifest from the given ReadSeeker.
// It validates the magic number, reads all sections (header, metadata, chunk data,
// file manifest list, custom fields), and returns the fully populated BinaryManifest.
func ParseManifest(f io.ReadSeeker) (*BinaryManifest, error) {
	magic, err := binreader.NewReader(f, binary.LittleEndian).ReadUint32()
	if err != nil {
		return nil, err
	} else if magic != BinaryManifestMagic {
		return nil, ErrBadMagic
	}

	var manifest BinaryManifest
	manifest.Header, err = ParseHeader(f)
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(int64(manifest.Header.HeaderSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	reader := f
	if (manifest.Header.StoredAs & StoredCompressed) != 0 {
		zreader, err := zlib.NewReader(reader)
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(zreader)
		if err != nil {
			return nil, err
		}
		if len(data) != int(manifest.Header.DataSizeUncompressed) {
			return nil, fmt.Errorf("decompressed data size mismatch, expected: %d and got: %d", manifest.Header.DataSizeUncompressed, len(data))
		}

		reader = bytes.NewReader(data)
	}
	if (manifest.Header.StoredAs & StoredEncrypted) != 0 {
		return nil, errors.New("manifest file is encrypted")
	}

	currentPos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	manifest.Metadata, err = ReadFManifestMeta(reader)
	if err != nil {
		return nil, err
	}

	currentPos, err = reader.Seek(currentPos+int64(manifest.Metadata.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.ChunkDataList, err = ReadChunkDataList(reader)
	if err != nil {
		return nil, err
	}

	currentPos, err = reader.Seek(currentPos+int64(manifest.ChunkDataList.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.FileManifestList, err = ReadFileManifestList(reader, manifest.ChunkDataList)
	if err != nil {
		return nil, err
	}

	_, err = reader.Seek(currentPos+int64(manifest.FileManifestList.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.CustomFields, err = ReadCustomFields(reader)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
