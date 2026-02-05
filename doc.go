// Package manifest provides a parser for Epic Games binary manifest files.
//
// These manifests describe game installations including file lists, chunk data,
// and metadata needed for downloading and patching games via the Epic Games CDN.
//
// # Parsing
//
// Parse a manifest from a file, URL, or any io.ReadSeeker:
//
//	m, err := manifest.ParseManifestFile("path/to/file.manifest")
//	m, err := manifest.ParseManifestURL(ctx, "https://example.com/file.manifest")
//	m, err := manifest.ParseManifest(reader)
//
// # Inspecting
//
// The returned [BinaryManifest] provides access to all manifest sections:
//
//	fmt.Println(m.Metadata.AppName)
//	fmt.Println(m.TotalInstallSize())
//	for _, f := range m.FileManifestList.FileManifestList {
//	    fmt.Println(f.FileName, f.FileSize)
//	}
//
// # Downloading
//
// Use the [github.com/meszmate/manifest/downloader] sub-package for downloading
// game files from the CDN:
//
//	d := downloader.New(m, downloader.Config{
//	    OutputDir: "./game",
//	    BaseURL:   "http://download.epicgames.com/Builds/Fortnite/CloudDir/",
//	})
//	result, err := d.Download(ctx)
//
// # Delta Patching
//
// Apply delta manifests to update an existing manifest:
//
//	newManifest.ApplyDelta(deltaManifest)
package manifest
