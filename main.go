package egmanifest

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/meszmate/manifest/binreader"
)

var (
	ErrBadMagic = errors.New("bad magic found, must be 0x44BEC00C")
)

const BinaryManifestMagic = 0x44BEC00C

type BinaryManifest struct {
	Header           *FManifestHeader
	Metadata         *FManifestMeta
	ChunkDataList    *FChunkDataList
	FileManifestList *FFileManifestList
	CustomFields     *FCustomFields
}

func ParseManifest(f io.ReadSeeker) (*BinaryManifest, error) {
	magic, err := binreader.NewReader(f, binary.LittleEndian).ReadUint32()
	if err != nil {
		return nil, err
	} else if magic != BinaryManifestMagic {
		return nil, ErrBadMagic
	}

	var manifest BinaryManifest
	manifest.Header, err = ParseHeader(f)
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(int64(manifest.Header.HeaderSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	reader := f
	if (manifest.Header.StoredAs & StoredCompressed) != 0 {
		zreader, err := zlib.NewReader(reader)
		if err != nil {
			return nil, err
		}

		// TODO: avoid buffering the entire file
		data, err := ioutil.ReadAll(zreader)
		if err != nil {
			return nil, err
		}
		if len(data) != int(manifest.Header.DataSizeUncompressed) {
			return nil, fmt.Errorf("decompressed data size mismatch, expected: %d and got: %d", len(data), manifest.Header.DataSizeUncompressed)
		}

		reader = bytes.NewReader(data)
	}
	if (manifest.Header.StoredAs & StoredEncrypted) != 0 {
		return nil, errors.New("manifest file is encrypted")
	}

	currentPos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	manifest.Metadata, err = ReadFManifestMeta(reader)
	if err != nil {
		return nil, err
	}

	currentPos, err = reader.Seek(currentPos+int64(manifest.Metadata.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.ChunkDataList, err = ReadChunkDataList(reader)
	if err != nil {
		return nil, err
	}

	currentPos, err = reader.Seek(currentPos+int64(manifest.ChunkDataList.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.FileManifestList, err = ReadFileManifestList(reader, manifest.ChunkDataList)
	if err != nil {
		return nil, err
	}

	_, err = reader.Seek(currentPos+int64(manifest.FileManifestList.DataSize), io.SeekStart)
	if err != nil {
		return nil, err
	}

	manifest.CustomFields, err = ReadCustomFields(reader)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}
