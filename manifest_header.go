package manifest

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/meszmate/manifest/binreader"
)

// FManifestHeader represents the header of a binary manifest file.
type FManifestHeader struct {
	HeaderSize           int32
	DataSizeUncompressed int32
	DataSizeCompressed   int32
	SHAHash              [20]byte
	StoredAs             uint8
	Version              EFeatureLevel
}

// String returns a human-readable representation of the manifest header.
func (h FManifestHeader) String() string {
	var flags []string
	if (h.StoredAs & StoredCompressed) != 0 {
		flags = append(flags, "Compressed")
	}
	if (h.StoredAs & StoredEncrypted) != 0 {
		flags = append(flags, "Encrypted")
	}
	storedAs := "None"
	if len(flags) > 0 {
		storedAs = strings.Join(flags, " ")
	}

	return fmt.Sprintf(`Header Size: %d bytes
Compressed Data Size: %d bytes
Uncompressed Data Size: %d bytes
SHA hash: %x
Stored As: %s
Version: %s`, h.HeaderSize, h.DataSizeCompressed, h.DataSizeUncompressed, h.SHAHash,
		storedAs, h.Version.String(),
	)
}

// ParseHeader reads and parses a manifest header from f.
// The caller should have already consumed the 4-byte magic number.
func ParseHeader(f io.ReadSeeker) (*FManifestHeader, error) {
	reader := binreader.NewReader(f, binary.LittleEndian)
	var header FManifestHeader
	var err error

	header.HeaderSize, err = reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	header.DataSizeUncompressed, err = reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	header.DataSizeCompressed, err = reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	_, shaHash, err := reader.ReadBytes(20)
	if err != nil {
		return nil, err
	}
	copy(header.SHAHash[:], shaHash)

	header.StoredAs, err = reader.ReadUint8()
	if err != nil {
		return nil, err
	}
	version, err := reader.ReadInt32()
	if err != nil {
		return nil, err
	}

	header.Version = EFeatureLevel(version)

	return &header, nil
}
