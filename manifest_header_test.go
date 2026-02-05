package manifest

import (
	"bytes"
	"testing"
)

func TestParseHeader(t *testing.T) {
	data := buildMinimalManifest()
	m, err := ParseManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if m.Header.HeaderSize != 41 {
		t.Errorf("HeaderSize = %d, want 41", m.Header.HeaderSize)
	}
	if m.Header.Version != EFeatureLevelLatest {
		t.Errorf("Version = %d, want %d", m.Header.Version, EFeatureLevelLatest)
	}
}

func TestFManifestHeader_String_NoFlags(t *testing.T) {
	h := FManifestHeader{
		HeaderSize:           41,
		DataSizeUncompressed: 100,
		DataSizeCompressed:   100,
		StoredAs:             0, // no flags set
		Version:              EFeatureLevelLatest,
	}
	s := h.String()
	if s == "" {
		t.Error("String() returned empty")
	}
	// Should contain "None" for empty flags
	if !bytes.Contains([]byte(s), []byte("None")) {
		t.Errorf("expected 'None' in output, got: %s", s)
	}
}

func TestFManifestHeader_String_Compressed(t *testing.T) {
	h := FManifestHeader{
		StoredAs: StoredCompressed,
		Version:  EFeatureLevelLatest,
	}
	s := h.String()
	if !bytes.Contains([]byte(s), []byte("Compressed")) {
		t.Errorf("expected 'Compressed' in output, got: %s", s)
	}
}

func TestFManifestHeader_String_Both(t *testing.T) {
	h := FManifestHeader{
		StoredAs: StoredCompressed | StoredEncrypted,
		Version:  EFeatureLevelLatest,
	}
	s := h.String()
	if !bytes.Contains([]byte(s), []byte("Compressed")) {
		t.Errorf("expected 'Compressed' in output, got: %s", s)
	}
	if !bytes.Contains([]byte(s), []byte("Encrypted")) {
		t.Errorf("expected 'Encrypted' in output, got: %s", s)
	}
}
