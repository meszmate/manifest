# Epic Games Manifest Parser
Epic games is using manifest files to download or update games 

# Example
If you want to use this libary, you need to understand how epic games is downloading and updating games. You can check Epic Games alternative libaries on github, but here's a fast example of installing all pakchunk.
```go
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	manifest "github.com/meszmate/manifest"
)

const downloadpath string = "/Users/meszmate/manifestparse/"
const max_retries int = 7

func main(){
	
	filebytes := manifest.LoadFileBytes("/Users/meszmate/Downloads/889Cfv4W7UAZ6Jn0dUyIuV0kX7gTog.manifest")
	manifestreader := bytes.NewReader(filebytes)
	binary, err := manifest.ParseManifest(manifestreader)
	if err != nil{
		fmt.Println(err)
	}
	for _, i := range binary.FileManifestList.FileManifestList{
		if strings.HasPrefix(i.FileName, "FortniteGame/Content/Paks"){
			fpath := downloadpath + i.FileName
			err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
			if err != nil {
				log.Fatalf("Failed to create directories: %v", err)
			}
			file, err := os.Create(fpath)
			if err != nil {
				log.Fatalf("Failed to create file: %v", err)
			}
			defer file.Close()
			for _, x := range i.ChunkParts{
				newbytes := getChunkByURL(x.Chunk.GetURL("http://epicgames-download1.akamaized.net/Builds/Fortnite/CloudDir/ChunksV4"), max_retries)
				if newbytes != nil{
					file.Seek(int64(x.Offset), 0)
					file.Write(newbytes)
				}else{
					log.Fatal("Chunk url is not working")
				}
			}
			fmt.Println(i.FileName + " Successfully installed")
		}
	}
}

func getChunkByURL(url string, retries int) []byte{
	retry := 0
	for retry < retries+1{
		newbytes := manifest.LoadURLBytes(url)
		if newbytes != nil{
			return newbytes
		}
	}
	return nil
}
```
# How updating is working?
for updating a game, you will need the old manifest + you can use applyDelta function for optimizing the NEW manifest. Changed chunks will have different guid.

