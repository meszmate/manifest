package manifest

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/meszmate/manifest/binreader"
)

// FFileManifestList holds all file entries in a manifest.
type FFileManifestList struct {
	DataSize    uint32
	DataVersion uint8
	Count       uint32

	FileManifestList []File
}

// ChunkPart represents a region within a chunk that contributes to a file.
type ChunkPart struct {
	DataSize   uint32
	ParentGUID uuid.UUID
	Offset     uint32
	Size       uint32

	Chunk *Chunk
}

// File represents a single file entry in the manifest.
type File struct {
	FileName      string
	SymlinkTarget string
	SHAHash       [20]byte
	FileMetaFlags uint8
	InstallTags   []string
	FileSize      uint64

	ChunkParts []ChunkPart
}

// String returns a human-readable summary of the file.
func (f File) String() string {
	return fmt.Sprintf("%s (%d bytes, %d chunks)", f.FileName, f.FileSize, len(f.ChunkParts))
}

// GetFileByPath returns a pointer to the file with the given path, or nil if not found.
func (f *FFileManifestList) GetFileByPath(p string) *File {
	for idx := range f.FileManifestList {
		if f.FileManifestList[idx].FileName == p {
			return &f.FileManifestList[idx]
		}
	}
	return nil
}

// GetFilesByTag returns all files that have the given install tag.
func (f *FFileManifestList) GetFilesByTag(tag string) []*File {
	var result []*File
	for idx := range f.FileManifestList {
		for _, t := range f.FileManifestList[idx].InstallTags {
			if t == tag {
				result = append(result, &f.FileManifestList[idx])
				break
			}
		}
	}
	return result
}

// ReadFileManifestList reads the file manifest list section from f.
func ReadFileManifestList(f io.ReadSeeker, dataList *FChunkDataList) (*FFileManifestList, error) {
	reader := binreader.NewReader(f, binary.LittleEndian)
	var list FFileManifestList
	var err error

	list.DataSize, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	list.DataVersion, err = reader.ReadUint8()
	if err != nil {
		return nil, err
	}

	list.Count, err = reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	list.FileManifestList = make([]File, list.Count)

	for idx := range list.FileManifestList {
		list.FileManifestList[idx].FileName, err = reader.ReadFString()
		if err != nil {
			return nil, err
		}
	}

	for idx := range list.FileManifestList {
		list.FileManifestList[idx].SymlinkTarget, err = reader.ReadFString()
		if err != nil {
			return nil, err
		}
	}

	for idx := range list.FileManifestList {
		_, shaHash, err := reader.ReadBytes(20)
		if err != nil {
			return nil, err
		}
		copy(list.FileManifestList[idx].SHAHash[:], shaHash)
	}

	for idx := range list.FileManifestList {
		list.FileManifestList[idx].FileMetaFlags, err = reader.ReadUint8()
		if err != nil {
			return nil, err
		}
	}

	for idx := range list.FileManifestList {
		list.FileManifestList[idx].InstallTags, err = reader.ReadFStringArray()
		if err != nil {
			return nil, err
		}
	}

	for idx := range list.FileManifestList {
		chunkPartsSize, err := reader.ReadUint32()
		if err != nil {
			return nil, err
		}

		list.FileManifestList[idx].ChunkParts = make([]ChunkPart, chunkPartsSize)

		for cpIdx := range list.FileManifestList[idx].ChunkParts {
			list.FileManifestList[idx].ChunkParts[cpIdx].DataSize, err = reader.ReadUint32()
			if err != nil {
				return nil, err
			}
			list.FileManifestList[idx].ChunkParts[cpIdx].ParentGUID, err = reader.ReadGUID()
			if err != nil {
				return nil, err
			}
			chunkID, ok := dataList.ChunkLookup[list.FileManifestList[idx].ChunkParts[cpIdx].ParentGUID]
			if !ok {
				return nil, fmt.Errorf("in chunkPart %d for file %d: parent GUID (%s) not found", cpIdx, idx, list.FileManifestList[idx].ChunkParts[cpIdx].ParentGUID.String())
			}
			list.FileManifestList[idx].ChunkParts[cpIdx].Chunk = dataList.Chunks[chunkID]

			list.FileManifestList[idx].ChunkParts[cpIdx].Offset, err = reader.ReadUint32()
			if err != nil {
				return nil, err
			}
			list.FileManifestList[idx].ChunkParts[cpIdx].Size, err = reader.ReadUint32()
			if err != nil {
				return nil, err
			}
		}
	}

	for idx := range list.FileManifestList {
		var dataSize uint64
		for cpIdx := range list.FileManifestList[idx].ChunkParts {
			dataSize += uint64(list.FileManifestList[idx].ChunkParts[cpIdx].Size)
		}
		list.FileManifestList[idx].FileSize = dataSize
	}

	return &list, nil
}
