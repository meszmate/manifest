package manifest

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"net/http"
	"strings"

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

func GetDeltaManifest(baseURL string, newBuildID string, oldBuildID string) []byte{
	req, err := http.Get(baseURL + "/Deltas/" + newBuildID + "/" + oldBuildID + ".delta")
	if err != nil{
		return nil
	}
	defer req.Body.Close()

	if req.StatusCode != 200{
		return nil
	}
	
	deltabytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil
	}
	return deltabytes
}

func (m *BinaryManifest) ApplyDelta(deltaManifest *BinaryManifest) {
	var added []string = make([]string, 0)
	for i, f := range m.FileManifestList.FileManifestList{
		deltaFile := deltaManifest.FileManifestList.GetFileByPath(f.FileName)
		if deltaFile == nil{
			continue
		}
		m.FileManifestList.FileManifestList[i] = *deltaFile
		added = append(added, deltaFile.FileName)
	}
	for _, deltaFile := range deltaManifest.FileManifestList.FileManifestList{
		if !StringContains(added, deltaFile.FileName){
			m.FileManifestList.FileManifestList = append(m.FileManifestList.FileManifestList, deltaFile)
		}
	}
	m.FileManifestList.Count = uint32(len(m.FileManifestList.FileManifestList))

	for _, chunk := range deltaManifest.ChunkDataList.Chunks{
		_, ok := m.ChunkDataList.ChunkLookup[chunk.GUID]
		if !ok{
			m.ChunkDataList.Chunks = append(m.ChunkDataList.Chunks, chunk)
		}
	}
	m.ChunkDataList.Count = uint32(len(m.ChunkDataList.Chunks))
}
func StringContains (array []string, s string) bool{
	for _, i := range array{
		if i == s{
			return true
		}
	}
	return false
}
func StringContains2 (array []string, array2 []string) bool{
	for _, i := range array{
		for _, x := range array2{
			if i == x{
				return true
			}
		}
	}
	return false
}
func StringContains3 (array []string, array2 []string) bool{
	for _, i := range array{
		for _, x := range array2{
			if strings.Contains(i, x){
				return true
			}
		}
	}
	return false
}
func LoadFileBytes(path string) []byte{
	filebytes, err := os.ReadFile(path)
	if err != nil{
		return nil
	}
	return filebytes
}
func LoadURLBytes(url string) []byte{
	req, err := http.Get(url)
	if err != nil{
		return nil
	}
	defer req.Body.Close()

	if req.StatusCode != 200{
		return nil
	}

	urlbytes, err := io.ReadAll(req.Body)
    	if err != nil {
        	return nil
	}
	return urlbytes
}

func (m *BinaryManifest) DelInstallTagFiles(tags []string){
	for i, f := range m.FileManifestList.FileManifestList{
		if StringContains2(f.InstallTags, tags){
			m.FileManifestList.FileManifestList = append(m.FileManifestList.FileManifestList[:i], m.FileManifestList.FileManifestList[i+1:]...)
		}
	}
}
func (m *BinaryManifest) DelInstallTagContainFiles(tags []string){
	for i, f := range m.FileManifestList.FileManifestList{
		if StringContains3(f.InstallTags, tags){
			m.FileManifestList.FileManifestList = append(m.FileManifestList.FileManifestList[:i], m.FileManifestList.FileManifestList[i+1:]...)
		}
	}
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
