package manifest

import (
	"bytes"
	"testing"
)

func TestGetFileByPath_Found(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f := m.FileManifestList.GetFileByPath("test/file.dat")
	if f == nil {
		t.Fatal("expected to find file")
	}
	if f.FileName != "test/file.dat" {
		t.Errorf("FileName = %q, want %q", f.FileName, "test/file.dat")
	}
}

func TestGetFileByPath_NotFound(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f := m.FileManifestList.GetFileByPath("nonexistent.txt")
	if f != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestGetFileByPath_ReturnedPointerModifiesOriginal(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f := m.FileManifestList.GetFileByPath("test/file.dat")
	if f == nil {
		t.Fatal("expected to find file")
	}
	// Modify through the returned pointer
	f.FileName = "modified.dat"

	// Verify the original slice is modified (no dangling pointer)
	if m.FileManifestList.FileManifestList[0].FileName != "modified.dat" {
		t.Error("modifying returned pointer did not modify original — dangling pointer bug")
	}
}

func TestGetFilesByTag(t *testing.T) {
	list := &FFileManifestList{
		FileManifestList: []File{
			{FileName: "a.txt", InstallTags: []string{"core", "highres"}},
			{FileName: "b.txt", InstallTags: []string{"core"}},
			{FileName: "c.txt", InstallTags: []string{"dlc"}},
		},
	}

	coreFiles := list.GetFilesByTag("core")
	if len(coreFiles) != 2 {
		t.Errorf("expected 2 files with 'core' tag, got %d", len(coreFiles))
	}

	dlcFiles := list.GetFilesByTag("dlc")
	if len(dlcFiles) != 1 {
		t.Errorf("expected 1 file with 'dlc' tag, got %d", len(dlcFiles))
	}

	noFiles := list.GetFilesByTag("nonexistent")
	if len(noFiles) != 0 {
		t.Errorf("expected 0 files with 'nonexistent' tag, got %d", len(noFiles))
	}
}

func TestFileSize_LargeFile(t *testing.T) {
	// Verify uint64 can hold > 4GB
	f := File{
		FileSize: 5 * 1024 * 1024 * 1024, // 5 GB
	}
	if f.FileSize != 5368709120 {
		t.Errorf("FileSize = %d, want 5368709120", f.FileSize)
	}
}

func TestFile_String(t *testing.T) {
	f := File{
		FileName: "game/data.pak",
		FileSize: 1024,
		ChunkParts: []ChunkPart{
			{}, {},
		},
	}
	s := f.String()
	expected := "game/data.pak (1024 bytes, 2 chunks)"
	if s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}
