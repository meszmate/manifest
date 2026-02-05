package manifest

import (
	"bytes"
	"testing"
)

func TestParseManifest_ValidMinimal(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if m.Metadata.AppName != "TestApp" {
		t.Errorf("AppName = %q, want %q", m.Metadata.AppName, "TestApp")
	}
	if m.Metadata.BuildVersion != "1.0.0" {
		t.Errorf("BuildVersion = %q, want %q", m.Metadata.BuildVersion, "1.0.0")
	}
	if m.Metadata.BuildId != "build-001" {
		t.Errorf("BuildId = %q, want %q", m.Metadata.BuildId, "build-001")
	}
	if len(m.ChunkDataList.Chunks) != 1 {
		t.Errorf("chunk count = %d, want 1", len(m.ChunkDataList.Chunks))
	}
	if len(m.FileManifestList.FileManifestList) != 1 {
		t.Errorf("file count = %d, want 1", len(m.FileManifestList.FileManifestList))
	}
	if m.FileManifestList.FileManifestList[0].FileName != "test/file.dat" {
		t.Errorf("filename = %q, want %q", m.FileManifestList.FileManifestList[0].FileName, "test/file.dat")
	}
	if m.FileManifestList.FileManifestList[0].FileSize != 512 {
		t.Errorf("FileSize = %d, want 512", m.FileManifestList.FileManifestList[0].FileSize)
	}
	if m.CustomFields.Fields["key1"] != "val1" {
		t.Errorf("custom field key1 = %q, want %q", m.CustomFields.Fields["key1"], "val1")
	}
}

func TestParseManifest_BadMagic(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00}
	_, err := ParseManifest(bytes.NewReader(data))
	if err != ErrBadMagic {
		t.Errorf("expected ErrBadMagic, got %v", err)
	}
}

func TestParseManifest_Compressed(t *testing.T) {
	data := buildCompressedManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseManifest compressed failed: %v", err)
	}
	if m.Metadata.AppName != "TestApp" {
		t.Errorf("AppName = %q, want %q", m.Metadata.AppName, "TestApp")
	}
	if len(m.FileManifestList.FileManifestList) != 1 {
		t.Errorf("file count = %d, want 1", len(m.FileManifestList.FileManifestList))
	}
}

func TestParseManifest_Encrypted(t *testing.T) {
	data := buildMinimalManifest()
	// Set the StoredAs byte to encrypted (offset: magic(4) + headerSize(4) + uncompressed(4) + compressed(4) + sha(20) = 36)
	data[36] = StoredEncrypted
	_, err := ParseManifest(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for encrypted manifest")
	}
}

func TestParseManifest_TooShort(t *testing.T) {
	_, err := ParseManifest(bytes.NewReader([]byte{0x0C}))
	if err == nil {
		t.Fatal("expected error for truncated data")
	}
}

func TestBinaryManifest_String(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	s := m.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestBinaryManifest_TotalDownloadSize(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if m.TotalDownloadSize() != 512 {
		t.Errorf("TotalDownloadSize() = %d, want 512", m.TotalDownloadSize())
	}
}

func TestBinaryManifest_TotalInstallSize(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if m.TotalInstallSize() != 512 {
		t.Errorf("TotalInstallSize() = %d, want 512", m.TotalInstallSize())
	}
}

func TestBinaryManifest_ChunkURL(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	url := m.ChunkURL("http://example.com/CloudDir", m.ChunkDataList.Chunks[0])
	if url == "" {
		t.Error("ChunkURL returned empty string")
	}
}

func TestParseManifestFile_NotFound(t *testing.T) {
	_, err := ParseManifestFile("/nonexistent/path/manifest.bin")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadFileBytes_NotFound(t *testing.T) {
	_, err := LoadFileBytes("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error")
	}
}
