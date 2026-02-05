package downloader

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/meszmate/manifest"
)

// FileStatus describes the verification status of a single file.
type FileStatus string

const (
	StatusOK           FileStatus = "ok"
	StatusMissing      FileStatus = "missing"
	StatusSizeMismatch FileStatus = "size_mismatch"
	StatusHashMismatch FileStatus = "hash_mismatch"
)

// VerifyResult holds the result of verifying an installation.
type VerifyResult struct {
	TotalFiles   int
	ValidFiles   int
	InvalidFiles int
	MissingFiles int
	Details      []FileVerifyStatus
}

// FileVerifyStatus holds the verification status for a single file.
type FileVerifyStatus struct {
	Path     string
	Status   FileStatus
	Expected string
	Actual   string
}

// VerifyInstallation checks all files in the manifest against the installed directory.
// It verifies file existence, size, and SHA1 hash.
func VerifyInstallation(m *manifest.BinaryManifest, installDir string) (*VerifyResult, error) {
	result := &VerifyResult{
		TotalFiles: len(m.FileManifestList.FileManifestList),
	}

	for _, f := range m.FileManifestList.FileManifestList {
		fpath := filepath.Join(installDir, filepath.FromSlash(f.FileName))

		info, err := os.Stat(fpath)
		if os.IsNotExist(err) {
			result.MissingFiles++
			result.Details = append(result.Details, FileVerifyStatus{
				Path:   f.FileName,
				Status: StatusMissing,
			})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", fpath, err)
		}

		if uint64(info.Size()) != f.FileSize {
			result.InvalidFiles++
			result.Details = append(result.Details, FileVerifyStatus{
				Path:     f.FileName,
				Status:   StatusSizeMismatch,
				Expected: fmt.Sprintf("%d", f.FileSize),
				Actual:   fmt.Sprintf("%d", info.Size()),
			})
			continue
		}

		expectedHash := fmt.Sprintf("%x", f.SHAHash)

		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", fpath, err)
		}

		h := sha1.New()
		if _, err := io.Copy(h, file); err != nil {
			file.Close()
			return nil, fmt.Errorf("hash %s: %w", fpath, err)
		}
		file.Close()

		actualHash := fmt.Sprintf("%x", h.Sum(nil))

		if actualHash != expectedHash {
			result.InvalidFiles++
			result.Details = append(result.Details, FileVerifyStatus{
				Path:     f.FileName,
				Status:   StatusHashMismatch,
				Expected: expectedHash,
				Actual:   actualHash,
			})
			continue
		}

		result.ValidFiles++
		result.Details = append(result.Details, FileVerifyStatus{
			Path:   f.FileName,
			Status: StatusOK,
		})
	}

	return result, nil
}
