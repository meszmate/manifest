package manifest

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/meszmate/manifest/binreader"
	"github.com/google/uuid"
)

type FFileManifestList struct {
	DataSize    		uint32
	DataVersion 		uint8
	Count       		uint32

	FileManifestList 	[]File
}

type ChunkPart struct {
	DataSize   uint32
	ParentGUID uuid.UUID
	Offset     uint32
	Size       uint32

	Chunk *Chunk
}

//TODO: implement io.ReadSeeker on this
type File struct {
	FileName      	string
	SymlinkTarget 	string
	SHAHash       	[20]byte
	FileMetaFlags 	uint8
	InstallTags   	[]string
	FileSize      	uint32

	ChunkParts 	[]ChunkPart
}

func (f *FFileManifestList) GetFileByPath(p string) *File{
	for _, i := range f.FileManifestList{
		if i.FileName == p{
			return &i
		}
	}
	return nil
}

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
		DataSize := 0 
		for cidx := range list.FileManifestList[idx].ChunkParts{
			DataSize += int(list.FileManifestList[idx].ChunkParts[cidx].Size)
		}
		list.FileManifestList[idx].FileSize = uint32(DataSize)
	}
	return &list, nil
}
