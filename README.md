# Epic Games Manifest Parser
Epic games is using manifest files to download or update games 

## Installation
```go
go get github.com/meszmate/manifest
```

## Example
If you want to use this libary, you need to understand how epic games is downloading and updating games. You can check Epic Games alternative libaries on github, but here's a fast example of installing the files.
```go
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	manifest "github.com/meszmate/manifest"
	chunks "github.com/meszmate/manifest/chunks"
)

const downloadpath string = "/Users/meszmate/manifestparse/" // Leave it empty if you want to install in the current directory
const max_retries int = 7

func main(){
	stime := time.Now().Unix()
	filebytes := manifest.LoadFileBytes("/Users/meszmate/Downloads/889Cfv4W7UAZ6Jn0dUyIuV0kX7gTog.manifest") // manifest.LoadURLBytes("url") if you want to get the manifest from url
	manifestreader := bytes.NewReader(filebytes)
	binary, err := manifest.ParseManifest(manifestreader)
	if err != nil{
		fmt.Println(err)
	}
	for _, i := range binary.FileManifestList.FileManifestList{
		if !manifest.StringContains3(i.InstallTags, []string{"br_highres", "stw_highres", "core_highres", "ondemand", "sm6"}){ 
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
				newbytes := getChunkByURL(x.Chunk.GetURL("http://epicgames-download1.akamaized.net/Builds/Fortnite/CloudDir/" + binary.Metadata.FeatureLevel.ChunkSubDir()), max_retries) 
				if newbytes != nil{
					newdata, err := chunks.Decompress(newbytes)
					if err != nil{
						log.Fatal("Failed to decompress: " + err.Error())
					}
					file.Write(newdata[x.Offset:x.Offset+x.Size])
				}else{
					fmt.Println("Failed to get a chunk, next...")
				}
			}
			fmt.Println(i.FileName + " Successfully installed")
		}
	}
	etime := time.Now().Unix()
	fmt.Printf("Installed in %d seconds\n", etime-stime)
}

func getChunkByURL(url string, retries int) []byte{
	retry := 0
	for retry < retries+1{
		newbytes := manifest.LoadURLBytes(url)
		if newbytes != nil{
			return newbytes
		}
		retry++
	}
	return nil
}
```
## Chunk Download BaseURLs Performance
```
http://download.epicgames.com/Builds/Fortnite/CloudDir/       		20-21 ms
http://cloudflare.epicgamescdn.com/Builds/Fortnite/CloudDir/  		34-36 ms
http://fastly-download.epicgames.com/Builds/Fortnite/CloudDir/ 		19-20 ms
http://epicgames-download1.akamaized.net/Builds/Fortnite/CloudDir/      27-28 ms
```

## Manifest ApplyDelta Usage
When you get the manifest from epic games api, you will get "elements" for manifest, you have to choose one "uri", and that will be the delta manifest baseURL. Example:
```go
newManifest, _ := manifest.ParseManifest(...)
oldManifest, _ := manifest.ParseManifest(...)
deltaManifestBytes := manifest.GetDeltaManifest("the base url", new_manifest.Metadata.BuildId, old_manifest.Metadata.BuildId)
deltaManifestReader := bytes.NewReader(deltaManifestBytes)
deltaManifest, _ := manifest.ParseManifest(deltaManifestReader)
newManifest.ApplyDelta(deltaManifest)
```

## How updating is working?
for updating a game, you will need the old manifest + you can use applyDelta function for optimizing the NEW manifest. Changed chunks will have different guid.

