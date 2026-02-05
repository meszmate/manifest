package manifest

import (
	"testing"

	"github.com/google/uuid"
)

func TestApplyDelta_UpdateExistingFile(t *testing.T) {
	chunk1 := &Chunk{GUID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), FileSize: 100}
	chunk2 := &Chunk{GUID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), FileSize: 200}

	base := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			Count: 1,
			FileManifestList: []File{
				{FileName: "file1.dat", FileSize: 100},
			},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk1},
			ChunkLookup: map[uuid.UUID]uint32{chunk1.GUID: 0},
		},
	}

	delta := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			Count: 1,
			FileManifestList: []File{
				{FileName: "file1.dat", FileSize: 200},
			},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk2},
			ChunkLookup: map[uuid.UUID]uint32{chunk2.GUID: 0},
		},
	}

	base.ApplyDelta(delta)

	if base.FileManifestList.FileManifestList[0].FileSize != 200 {
		t.Errorf("file1 FileSize = %d, want 200", base.FileManifestList.FileManifestList[0].FileSize)
	}
	if len(base.ChunkDataList.Chunks) != 2 {
		t.Errorf("chunk count = %d, want 2", len(base.ChunkDataList.Chunks))
	}
}

func TestApplyDelta_AddNewFile(t *testing.T) {
	chunk1 := &Chunk{GUID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), FileSize: 100}
	chunk2 := &Chunk{GUID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), FileSize: 200}

	base := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			Count: 1,
			FileManifestList: []File{
				{FileName: "file1.dat", FileSize: 100},
			},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk1},
			ChunkLookup: map[uuid.UUID]uint32{chunk1.GUID: 0},
		},
	}

	delta := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			Count: 1,
			FileManifestList: []File{
				{FileName: "file2.dat", FileSize: 300},
			},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk2},
			ChunkLookup: map[uuid.UUID]uint32{chunk2.GUID: 0},
		},
	}

	base.ApplyDelta(delta)

	if len(base.FileManifestList.FileManifestList) != 2 {
		t.Fatalf("file count = %d, want 2", len(base.FileManifestList.FileManifestList))
	}
	if base.FileManifestList.FileManifestList[1].FileName != "file2.dat" {
		t.Errorf("second file = %q, want %q", base.FileManifestList.FileManifestList[1].FileName, "file2.dat")
	}
	if base.FileManifestList.Count != 2 {
		t.Errorf("Count = %d, want 2", base.FileManifestList.Count)
	}
}

func TestApplyDelta_ChunkLookupUpdated(t *testing.T) {
	chunk1 := &Chunk{GUID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), FileSize: 100}
	chunk2 := &Chunk{GUID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), FileSize: 200}

	base := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			FileManifestList: []File{},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk1},
			ChunkLookup: map[uuid.UUID]uint32{chunk1.GUID: 0},
		},
	}

	delta := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			FileManifestList: []File{},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk2},
			ChunkLookup: map[uuid.UUID]uint32{chunk2.GUID: 0},
		},
	}

	base.ApplyDelta(delta)

	// Verify ChunkLookup was updated (this was the bug)
	idx, ok := base.ChunkDataList.ChunkLookup[chunk2.GUID]
	if !ok {
		t.Fatal("new chunk GUID not found in ChunkLookup — ApplyDelta bug")
	}
	if idx != 1 {
		t.Errorf("new chunk index = %d, want 1", idx)
	}
	if base.ChunkDataList.Count != 2 {
		t.Errorf("Count = %d, want 2", base.ChunkDataList.Count)
	}
}

func TestApplyDelta_DuplicateChunkNotAdded(t *testing.T) {
	chunk1 := &Chunk{GUID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), FileSize: 100}

	base := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			FileManifestList: []File{},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk1},
			ChunkLookup: map[uuid.UUID]uint32{chunk1.GUID: 0},
		},
	}

	delta := &BinaryManifest{
		Metadata: &FManifestMeta{AppName: "Test"},
		FileManifestList: &FFileManifestList{
			FileManifestList: []File{},
		},
		ChunkDataList: &FChunkDataList{
			Count:       1,
			Chunks:      []*Chunk{chunk1},
			ChunkLookup: map[uuid.UUID]uint32{chunk1.GUID: 0},
		},
	}

	base.ApplyDelta(delta)

	if len(base.ChunkDataList.Chunks) != 1 {
		t.Errorf("chunk count = %d, want 1 (duplicate should not be added)", len(base.ChunkDataList.Chunks))
	}
}
