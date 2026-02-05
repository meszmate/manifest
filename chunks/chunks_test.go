package chunks

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"testing"
)

func TestDecompress_TooShort(t *testing.T) {
	_, err := Decompress([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for data too short")
	}
}

func TestDecompress_Valid(t *testing.T) {
	// Build data: 8 bytes header + zlib data
	// offset byte at [8] = 0x78 means zlib starts at offset 8
	original := []byte("hello world, this is test data for zlib compression")
	var zlibBuf bytes.Buffer
	w := zlib.NewWriter(&zlibBuf)
	w.Write(original)
	w.Close()

	zlibData := zlibBuf.Bytes()

	// Build: 8 header bytes (first 8 don't matter except [8] needs to be the zlib magic)
	// Actually we need at least 9 bytes total. [8] should be 0x78 (zlib header)
	data := make([]byte, 8)
	data = append(data, zlibData...)

	result, err := Decompress(data)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(result, original) {
		t.Errorf("decompressed data mismatch")
	}
}

func TestDecompress_InvalidZlib(t *testing.T) {
	data := make([]byte, 20)
	data[8] = 0x78 // pretend it's zlib offset, but rest is garbage
	data[9] = 0x01
	data[10] = 0xFF
	_, err := Decompress(data)
	if err == nil {
		t.Fatal("expected error for invalid zlib data")
	}
}

func buildChunkHeaderBytes(magic, version, headerSize, dataSize uint32, storedAs byte) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, magic)
	binary.Write(&buf, binary.LittleEndian, version)
	binary.Write(&buf, binary.LittleEndian, headerSize)
	binary.Write(&buf, binary.LittleEndian, dataSize)

	// GUID: 16 bytes
	buf.Write(make([]byte, 16))
	// RollingHash: 8 bytes
	binary.Write(&buf, binary.LittleEndian, uint64(0))
	// StoredAs
	buf.WriteByte(storedAs)
	// SHAHash: 20 bytes
	buf.Write(make([]byte, 20))
	// HashType
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	return buf.Bytes()
}

func TestParseChunkHeader_Valid(t *testing.T) {
	data := buildChunkHeaderBytes(ChunkHeaderMagic, 3, 67, 1000, byte(ChunkStoredAsPlaintext))
	header, err := ParseChunkHeader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseChunkHeader failed: %v", err)
	}
	if header.Magic != ChunkHeaderMagic {
		t.Errorf("Magic = 0x%X, want 0x%X", header.Magic, ChunkHeaderMagic)
	}
	if header.Version != 3 {
		t.Errorf("Version = %d, want 3", header.Version)
	}
	if header.StoredAs != ChunkStoredAsPlaintext {
		t.Errorf("StoredAs = %d, want %d", header.StoredAs, ChunkStoredAsPlaintext)
	}
}

func TestParseChunkHeader_BadMagic(t *testing.T) {
	data := buildChunkHeaderBytes(0xDEADBEEF, 3, 67, 1000, 0)
	_, err := ParseChunkHeader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestParseChunk_Plaintext(t *testing.T) {
	headerSize := uint32(67)
	headerData := buildChunkHeaderBytes(ChunkHeaderMagic, 3, headerSize, 5, byte(ChunkStoredAsPlaintext))

	// Pad to headerSize and add body
	padded := make([]byte, headerSize)
	copy(padded, headerData)

	body := []byte("hello")
	full := append(padded, body...)

	result, err := ParseChunk(bytes.NewReader(full))
	if err != nil {
		t.Fatalf("ParseChunk failed: %v", err)
	}
	out, err := io.ReadAll(result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, body) {
		t.Errorf("got %q, want %q", out, body)
	}
}

func TestParseChunk_Compressed(t *testing.T) {
	headerSize := uint32(67)
	headerData := buildChunkHeaderBytes(ChunkHeaderMagic, 3, headerSize, 0, byte(ChunkStoredAsCompressed))

	padded := make([]byte, headerSize)
	copy(padded, headerData)

	// Compress body
	original := []byte("compressed chunk data")
	var zlibBuf bytes.Buffer
	w := zlib.NewWriter(&zlibBuf)
	w.Write(original)
	w.Close()

	full := append(padded, zlibBuf.Bytes()...)

	result, err := ParseChunk(bytes.NewReader(full))
	if err != nil {
		t.Fatalf("ParseChunk compressed failed: %v", err)
	}
	out, err := io.ReadAll(result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, original) {
		t.Errorf("decompressed mismatch")
	}
}

func TestParseChunk_UnsupportedVersion(t *testing.T) {
	headerData := buildChunkHeaderBytes(ChunkHeaderMagic, 2, 67, 0, 0)
	_, err := ParseChunk(bytes.NewReader(headerData))
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}
