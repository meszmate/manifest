package chunks

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/meszmate/manifest/binreader"
	"github.com/google/uuid"
)

const (
	ChunkHeaderMagic = 0xB1FE3AA2
)

type ChunkStoredAs uint8

const (
	ChunkStoredAsPlaintext  ChunkStoredAs = 0x00
	ChunkStoredAsCompressed ChunkStoredAs = 0x01
	ChunkStoredAsEncrypted  ChunkStoredAs = 0x02
)

// ChunkHeader defines the binary chunk header
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
	return &header, err
}

func ParseChunk(reader io.ReadSeeker) (io.ReadSeeker, error) {
	header, err := ParseChunkHeader(reader)
	if err != nil {
		return nil, err
	}
	if header.Version != 3 {
		return nil, fmt.Errorf("unsupported verion %d", header.Version)
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
		chunkData, err := ioutil.ReadAll(inflatedReader) // we need a ReadSeeker
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(chunkData), err
	case ChunkStoredAsEncrypted:
		return nil, errors.New("chunk is encrypted")
	default:
		return nil, fmt.Errorf("unknown storage mode %d", header.StoredAs)
	}
}
