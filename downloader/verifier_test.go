package downloader

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/meszmate/manifest"
)

func TestVerifyInstallation_AllValid(t *testing.T) {
	dir := t.TempDir()

	fileContent := []byte("valid file content here")
	shaHash := sha1.Sum(fileContent)

	fpath := filepath.Join(dir, "game", "data.bin")
	os.MkdirAll(filepath.Dir(fpath), 0o755)
	os.WriteFile(fpath, fileContent, 0o644)

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{
					FileName: "game/data.bin",
					FileSize: uint64(len(fileContent)),
					SHAHash:  shaHash,
				},
			},
		},
	}

	result, err := VerifyInstallation(m, dir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.ValidFiles != 1 {
		t.Errorf("expected 1 valid file, got %d", result.ValidFiles)
	}
	if result.MissingFiles != 0 {
		t.Errorf("expected 0 missing, got %d", result.MissingFiles)
	}
	if result.InvalidFiles != 0 {
		t.Errorf("expected 0 invalid, got %d", result.InvalidFiles)
	}
	if result.Details[0].Status != StatusOK {
		t.Errorf("expected status ok, got %s", result.Details[0].Status)
	}
}

func TestVerifyInstallation_MissingFile(t *testing.T) {
	dir := t.TempDir()

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{
					FileName: "missing/file.dat",
					FileSize: 100,
				},
			},
		},
	}

	result, err := VerifyInstallation(m, dir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.MissingFiles != 1 {
		t.Errorf("expected 1 missing file, got %d", result.MissingFiles)
	}
	if result.Details[0].Status != StatusMissing {
		t.Errorf("expected status missing, got %s", result.Details[0].Status)
	}
}

func TestVerifyInstallation_SizeMismatch(t *testing.T) {
	dir := t.TempDir()

	fileContent := []byte("short")
	fpath := filepath.Join(dir, "data.bin")
	os.WriteFile(fpath, fileContent, 0o644)

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{
					FileName: "data.bin",
					FileSize: 999, // different from actual
				},
			},
		},
	}

	result, err := VerifyInstallation(m, dir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.InvalidFiles != 1 {
		t.Errorf("expected 1 invalid file, got %d", result.InvalidFiles)
	}
	if result.Details[0].Status != StatusSizeMismatch {
		t.Errorf("expected status size_mismatch, got %s", result.Details[0].Status)
	}
	if result.Details[0].Expected != "999" {
		t.Errorf("expected size 999, got %s", result.Details[0].Expected)
	}
	if result.Details[0].Actual != fmt.Sprintf("%d", len(fileContent)) {
		t.Errorf("expected actual size %d, got %s", len(fileContent), result.Details[0].Actual)
	}
}

func TestVerifyInstallation_HashMismatch(t *testing.T) {
	dir := t.TempDir()

	fileContent := []byte("actual content")
	fpath := filepath.Join(dir, "data.bin")
	os.WriteFile(fpath, fileContent, 0o644)

	// Set a wrong SHA hash
	wrongHash := sha1.Sum([]byte("different content"))

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{
					FileName: "data.bin",
					FileSize: uint64(len(fileContent)),
					SHAHash:  wrongHash,
				},
			},
		},
	}

	result, err := VerifyInstallation(m, dir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.InvalidFiles != 1 {
		t.Errorf("expected 1 invalid file, got %d", result.InvalidFiles)
	}
	if result.Details[0].Status != StatusHashMismatch {
		t.Errorf("expected status hash_mismatch, got %s", result.Details[0].Status)
	}
}

func TestVerifyInstallation_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	// File 1: valid
	content1 := []byte("file one content")
	sha1hash := sha1.Sum(content1)
	os.MkdirAll(filepath.Join(dir, "a"), 0o755)
	os.WriteFile(filepath.Join(dir, "a", "valid.dat"), content1, 0o644)

	// File 2: missing (don't create it)

	// File 3: wrong size
	os.WriteFile(filepath.Join(dir, "wrong_size.dat"), []byte("x"), 0o644)

	m := &manifest.BinaryManifest{
		FileManifestList: &manifest.FFileManifestList{
			FileManifestList: []manifest.File{
				{FileName: "a/valid.dat", FileSize: uint64(len(content1)), SHAHash: sha1hash},
				{FileName: "b/missing.dat", FileSize: 50},
				{FileName: "wrong_size.dat", FileSize: 999},
			},
		},
	}

	result, err := VerifyInstallation(m, dir)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if result.TotalFiles != 3 {
		t.Errorf("expected 3 total files, got %d", result.TotalFiles)
	}
	if result.ValidFiles != 1 {
		t.Errorf("expected 1 valid, got %d", result.ValidFiles)
	}
	if result.MissingFiles != 1 {
		t.Errorf("expected 1 missing, got %d", result.MissingFiles)
	}
	if result.InvalidFiles != 1 {
		t.Errorf("expected 1 invalid, got %d", result.InvalidFiles)
	}
}
