package manifest

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"

	"github.com/google/uuid"
)

// writeFString writes a length-prefixed null-terminated string.
func writeFString(buf *bytes.Buffer, s string) {
	if s == "" {
		binary.Write(buf, binary.LittleEndian, uint32(0))
		return
	}
	binary.Write(buf, binary.LittleEndian, uint32(len(s)+1))
	buf.WriteString(s)
	buf.WriteByte(0)
}

// writeFStringArray writes a uint32 count followed by that many FStrings.
func writeFStringArray(buf *bytes.Buffer, arr []string) {
	binary.Write(buf, binary.LittleEndian, uint32(len(arr)))
	for _, s := range arr {
		writeFString(buf, s)
	}
}

// writeGUID writes a UUID as 4 big-endian uint32 segments (matching ReadGUID).
func writeGUID(buf *bytes.Buffer, g uuid.UUID) {
	for i := 0; i < 4; i++ {
		v := binary.LittleEndian.Uint32(g[i*4 : (i+1)*4])
		binary.Write(buf, binary.BigEndian, v)
	}
}

// buildMetaSection builds a metadata section.
func buildMetaSection(appName, buildVersion, buildID string) []byte {
	var body bytes.Buffer

	// DataVersion
	body.WriteByte(1) // version 1 to include BuildId

	// FeatureLevel
	binary.Write(&body, binary.LittleEndian, int32(EFeatureLevelLatest))

	// IsFileData
	body.WriteByte(0) // false

	// AppID
	binary.Write(&body, binary.LittleEndian, int32(1))

	// AppName
	writeFString(&body, appName)

	// BuildVersion
	writeFString(&body, buildVersion)

	// LaunchExe
	writeFString(&body, "")

	// LaunchCommand
	writeFString(&body, "")

	// PrereqIds (empty array)
	writeFStringArray(&body, nil)

	// PrereqName
	writeFString(&body, "")

	// PrereqPath
	writeFString(&body, "")

	// PrereqArgs
	writeFString(&body, "")

	// BuildId
	writeFString(&body, buildID)

	// DataSize = 4 (for DataSize field) + body (which already includes DataVersion)
	dataSize := uint32(4 + body.Len())

	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())
	return section.Bytes()
}

// buildChunkDataSection builds a chunk data list section with given chunks.
func buildChunkDataSection(chunks []Chunk) []byte {
	count := uint32(len(chunks))

	var body bytes.Buffer

	// DataVersion
	body.WriteByte(0)

	// Count
	binary.Write(&body, binary.LittleEndian, count)

	// GUIDs
	for _, c := range chunks {
		writeGUID(&body, c.GUID)
	}
	// Hashes
	for _, c := range chunks {
		binary.Write(&body, binary.LittleEndian, c.Hash)
	}
	// SHAHashes
	for _, c := range chunks {
		body.Write(c.SHAHash[:])
	}
	// Groups
	for _, c := range chunks {
		body.WriteByte(c.Group)
	}
	// WindowSizes
	for _, c := range chunks {
		binary.Write(&body, binary.LittleEndian, c.WindowSize)
	}
	// FileSizes
	for _, c := range chunks {
		binary.Write(&body, binary.LittleEndian, c.FileSize)
	}

	dataSize := uint32(4 + body.Len())

	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())
	return section.Bytes()
}

// buildFileManifestSection builds a file manifest list section.
func buildFileManifestSection(files []testFile, chunkGUIDs []uuid.UUID) []byte {
	count := uint32(len(files))

	var body bytes.Buffer

	// DataVersion
	body.WriteByte(0)

	// Count
	binary.Write(&body, binary.LittleEndian, count)

	// FileNames
	for _, f := range files {
		writeFString(&body, f.name)
	}
	// SymlinkTargets
	for range files {
		writeFString(&body, "")
	}
	// SHAHashes
	for range files {
		body.Write(make([]byte, 20))
	}
	// FileMetaFlags
	for range files {
		body.WriteByte(0)
	}
	// InstallTags
	for _, f := range files {
		writeFStringArray(&body, f.tags)
	}
	// ChunkParts
	for _, f := range files {
		binary.Write(&body, binary.LittleEndian, uint32(len(f.chunkParts)))
		for _, cp := range f.chunkParts {
			// DataSize (always 28 for a chunk part: 4 + 16 + 4 + 4)
			binary.Write(&body, binary.LittleEndian, uint32(28))
			writeGUID(&body, chunkGUIDs[cp.chunkIdx])
			binary.Write(&body, binary.LittleEndian, cp.offset)
			binary.Write(&body, binary.LittleEndian, cp.size)
		}
	}

	dataSize := uint32(4 + body.Len())

	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())
	return section.Bytes()
}

type testChunkPart struct {
	chunkIdx int
	offset   uint32
	size     uint32
}

type testFile struct {
	name       string
	tags       []string
	chunkParts []testChunkPart
}

// buildCustomFieldsSection builds a custom fields section.
func buildCustomFieldsSection(fields map[string]string) []byte {
	var body bytes.Buffer
	body.WriteByte(0) // DataVersion

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}

	binary.Write(&body, binary.LittleEndian, uint32(len(keys)))

	// Keys
	for _, k := range keys {
		writeFString(&body, k)
	}
	// Values
	for _, k := range keys {
		writeFString(&body, fields[k])
	}

	dataSize := uint32(4 + body.Len())
	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, dataSize)
	section.Write(body.Bytes())
	return section.Bytes()
}

// buildMinimalManifest constructs a valid minimal binary manifest.
func buildMinimalManifest() []byte {
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	chunks := []Chunk{
		{
			GUID:       chunkGUID,
			Hash:       0xABCD,
			Group:      1,
			WindowSize: 1048576,
			FileSize:   512,
		},
	}

	files := []testFile{
		{
			name: "test/file.dat",
			tags: []string{},
			chunkParts: []testChunkPart{
				{chunkIdx: 0, offset: 0, size: 256},
				{chunkIdx: 0, offset: 256, size: 256},
			},
		},
	}

	metaBytes := buildMetaSection("TestApp", "1.0.0", "build-001")
	chunkBytes := buildChunkDataSection(chunks)
	fileBytes := buildFileManifestSection(files, []uuid.UUID{chunkGUID})
	customBytes := buildCustomFieldsSection(map[string]string{"key1": "val1"})

	// Assemble the uncompressed body
	var body bytes.Buffer
	body.Write(metaBytes)
	body.Write(chunkBytes)
	body.Write(fileBytes)
	body.Write(customBytes)

	bodyData := body.Bytes()

	// Build header
	var header bytes.Buffer

	// HeaderSize: magic(4) + headerSize(4) + uncompressedSize(4) + compressedSize(4) + sha(20) + storedAs(1) + version(4) = 41
	headerSize := int32(41)
	binary.Write(&header, binary.LittleEndian, headerSize)

	// DataSizeUncompressed
	binary.Write(&header, binary.LittleEndian, int32(len(bodyData)))

	// DataSizeCompressed (same since not compressed)
	binary.Write(&header, binary.LittleEndian, int32(len(bodyData)))

	// SHAHash (20 zero bytes)
	header.Write(make([]byte, 20))

	// StoredAs (0 = uncompressed)
	header.WriteByte(0)

	// Version
	binary.Write(&header, binary.LittleEndian, int32(EFeatureLevelLatest))

	// Assemble full manifest
	var manifest bytes.Buffer
	binary.Write(&manifest, binary.LittleEndian, uint32(BinaryManifestMagic))
	manifest.Write(header.Bytes())
	manifest.Write(bodyData)

	return manifest.Bytes()
}

// buildCompressedManifest constructs a valid compressed binary manifest.
func buildCompressedManifest() []byte {
	chunkGUID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	chunks := []Chunk{
		{
			GUID:       chunkGUID,
			Hash:       0xABCD,
			Group:      1,
			WindowSize: 1048576,
			FileSize:   512,
		},
	}

	files := []testFile{
		{
			name: "compressed/file.dat",
			tags: []string{},
			chunkParts: []testChunkPart{
				{chunkIdx: 0, offset: 0, size: 512},
			},
		},
	}

	metaBytes := buildMetaSection("TestApp", "1.0.0", "build-001")
	chunkBytes := buildChunkDataSection(chunks)
	fileBytes := buildFileManifestSection(files, []uuid.UUID{chunkGUID})
	customBytes := buildCustomFieldsSection(map[string]string{})

	var body bytes.Buffer
	body.Write(metaBytes)
	body.Write(chunkBytes)
	body.Write(fileBytes)
	body.Write(customBytes)

	bodyData := body.Bytes()

	// Compress the body
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write(bodyData)
	w.Close()

	compressedData := compressed.Bytes()

	// Build header
	var header bytes.Buffer
	headerSize := int32(41)
	binary.Write(&header, binary.LittleEndian, headerSize)
	binary.Write(&header, binary.LittleEndian, int32(len(bodyData)))
	binary.Write(&header, binary.LittleEndian, int32(len(compressedData)))
	header.Write(make([]byte, 20))
	header.WriteByte(StoredCompressed) // compressed flag
	binary.Write(&header, binary.LittleEndian, int32(EFeatureLevelLatest))

	var manifest bytes.Buffer
	binary.Write(&manifest, binary.LittleEndian, uint32(BinaryManifestMagic))
	manifest.Write(header.Bytes())
	manifest.Write(compressedData)

	return manifest.Bytes()
}
