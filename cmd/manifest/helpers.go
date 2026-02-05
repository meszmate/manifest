package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/meszmate/manifest"
)

// loadManifest auto-detects whether source is a URL or file path and parses the manifest.
func loadManifest(source string) (*manifest.BinaryManifest, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return manifest.ParseManifestURL(context.Background(), source)
	}
	return manifest.ParseManifestFile(source)
}

// humanBytes formats a byte count as a human-readable string.
func humanBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
