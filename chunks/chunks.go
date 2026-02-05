// Package chunks provides parsing and decompression for Epic Games chunk files.
package chunks

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/meszmate/manifest/binreader"
)

// ChunkHeaderMagic is the expected magic number at the start of every chunk file.
const ChunkHeaderMagic = 0xB1FE3AA2

// ChunkStoredAs describes how a chunk's data is stored on disk.
type ChunkStoredAs uint8

const (
	// ChunkStoredAsPlaintext indicates uncompressed chunk data.
	ChunkStoredAsPlaintext ChunkStoredAs = 0x00
	// ChunkStoredAsCompressed indicates zlib-compressed chunk data.
	ChunkStoredAsCompressed ChunkStoredAs = 0x01
	// ChunkStoredAsEncrypted indicates encrypted chunk data.
	ChunkStoredAsEncrypted ChunkStoredAs = 0x02
)

// ChunkHeader defines the binary chunk header.
type ChunkHeader struct {
	Magic              uint32 // 0xB1FE3AA2
	Version            uint32
	HeaderSize         uint32
	DataSizeCompressed uint32
	GUID               uuid.UUID
	RollingHash        uint64
	StoredAs           ChunkStoredAs
	SHAHash            [20]byte
	HashType           uint32
}

// Decompress decompresses a raw chunk downloaded from the CDN.
// The data layout is: header bytes followed by zlib-compressed payload.
// The byte at offset 8 indicates the start of the zlib stream unless it equals 120 (0x78),
// in which case the zlib data starts at offset 8 itself.
func Decompress(data []byte) ([]byte, error) {
	if len(data) < 9 {
		return nil, errors.New("chunk data too short: need at least 9 bytes")
	}

	offset := data[8]

	var slicedData []byte
	if offset == 120 {
		slicedData = data[8:]
	} else {
		if int(offset) > len(data) {
			return nil, fmt.Errorf("chunk offset %d exceeds data length %d", offset, len(data))
		}
		slicedData = data[offset:]
	}

	reader, err := zlib.NewReader(bytes.NewReader(slicedData))
	if err != nil {
		return nil, fmt.Errorf("zlib init: %w", err)
	}
	defer reader.Close()

	var outBuffer bytes.Buffer
	_, err = io.Copy(&outBuffer, reader)
	if err != nil {
		return nil, fmt.Errorf("zlib decompress: %w", err)
	}

	return outBuffer.Bytes(), nil
}

// ParseChunkHeader reads and validates a chunk header from the given reader.
func ParseChunkHeader(r io.ReadSeeker) (*ChunkHeader, error) {
	header := ChunkHeader{}
	reader := binreader.NewReader(r, binary.LittleEndian)
	var err error
	header.Magic, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}
	if header.Magic != ChunkHeaderMagic {
		return nil, fmt.Errorf("invalid chunk header magic: expected 0x%04X, have 0x%04X", ChunkHeaderMagic, header.Magic)
	}
	header.Version, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}
	header.HeaderSize, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}
	header.DataSizeCompressed, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}
	header.GUID, err = reader.ReadGUID()
	if err != nil {
		return nil, err
	}
	header.RollingHash, err = reader.ReadUint64()
	if err != nil {
		return nil, err
	}
	storedAs, err := reader.ReadUint8()
	if err != nil {
		return nil, err
	}
	header.StoredAs = ChunkStoredAs(storedAs)
	_, err = io.ReadFull(reader, header.SHAHash[:])
	if err != nil {
		return nil, err
	}
	header.HashType, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}
	return &header, nil
}

// ParseChunk reads a chunk file (header + body) and returns a ReadSeeker
// over the decompressed chunk data.
func ParseChunk(reader io.ReadSeeker) (io.ReadSeeker, error) {
	header, err := ParseChunkHeader(reader)
	if err != nil {
		return nil, err
	}
	if header.Version != 3 {
		return nil, fmt.Errorf("unsupported version %d", header.Version)
	}
	_, err = reader.Seek(int64(header.HeaderSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	switch header.StoredAs {
	case ChunkStoredAsPlaintext:
		return reader, nil
	case ChunkStoredAsCompressed:
		inflatedReader, err := zlib.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer inflatedReader.Close()
		chunkData, err := io.ReadAll(inflatedReader)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(chunkData), nil
	case ChunkStoredAsEncrypted:
		return nil, errors.New("chunk is encrypted")
	default:
		return nil, fmt.Errorf("unknown storage mode %d", header.StoredAs)
	}
}
