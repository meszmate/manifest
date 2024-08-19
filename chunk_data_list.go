package manifest

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/meszmate/manifest/binreader"
	"github.com/google/uuid"
)

type FChunkDataList struct {
	DataSize    uint32
	DataVersion uint8
	Count       uint32

	Chunks      []*Chunk
	ChunkLookup map[uuid.UUID]uint32
}

type Chunk struct {
	GUID       uuid.UUID
	Hash       uint64
	SHAHash    [20]byte
	Group      uint8
	WindowSize uint32
	FileSize   uint64
}

// gets the URL for a chunk.
// example for chunksDir: http://epicgames-download1.akamaized.net/Builds/Fortnite/CloudDir/ChunksV4
func (c *Chunk) GetURL(chunksDir string) string {
	return fmt.Sprintf("%s/%02d/%016X_%X.chunk", chunksDir, c.Group, c.Hash, c.GUID[:])
}

func ReadChunkDataList(f io.ReadSeeker) (*FChunkDataList, error) {
	reader := binreader.NewReader(f, binary.LittleEndian)
	var list FChunkDataList
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

	list.Chunks = make([]*Chunk, list.Count)
	// initialize all chunks
	for i := uint32(0); i < list.Count; i++ {
		list.Chunks[i] = &Chunk{}
	}
	list.ChunkLookup = map[uuid.UUID]uint32{}

	for i, chunk := range list.Chunks {
		chunk.GUID, err = reader.ReadGUID()
		if err != nil {
			return nil, err
		}
		list.ChunkLookup[chunk.GUID] = uint32(i)
	}

	for _, chunk := range list.Chunks {
		chunk.Hash, err = reader.ReadUint64()
		if err != nil {
			return nil, err
		}
	}

	for _, chunk := range list.Chunks {
		_, shaHash, err := reader.ReadBytes(20)
		if err != nil {
			return nil, err
		}
		copy(chunk.SHAHash[:], shaHash)
	}

	for _, chunk := range list.Chunks {
		chunk.Group, err = reader.ReadUint8()
		if err != nil {
			return nil, err
		}
	}

	for _, chunk := range list.Chunks {
		chunk.WindowSize, err = reader.ReadUint32()
		if err != nil {
			return nil, err
		}
	}

	for _, chunk := range list.Chunks {
		chunk.FileSize, err = reader.ReadUint64()
		if err != nil {
			return nil, err
		}
	}

	return &list, nil
}
