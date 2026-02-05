package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meszmate/manifest"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list <file|url>",
	Short: "List files in a manifest",
	Long:  "List all files contained in an Epic Games manifest.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		tag, _ := cmd.Flags().GetString("tag")
		sortBy, _ := cmd.Flags().GetString("sort")
		filter, _ := cmd.Flags().GetString("filter")
		long, _ := cmd.Flags().GetBool("long")

		var files []manifest.File
		if tag != "" {
			for _, f := range m.FileManifestList.GetFilesByTag(tag) {
				files = append(files, *f)
			}
		} else {
			files = append(files, m.FileManifestList.FileManifestList...)
		}

		if filter != "" {
			var filtered []manifest.File
			for _, f := range files {
				matched, err := filepath.Match(filter, filepath.Base(f.FileName))
				if err != nil {
					return fmt.Errorf("invalid filter pattern: %w", err)
				}
				if matched {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}

		switch sortBy {
		case "size":
			sort.Slice(files, func(i, j int) bool { return files[i].FileSize > files[j].FileSize })
		case "name":
			sort.Slice(files, func(i, j int) bool { return files[i].FileName < files[j].FileName })
		default:
			// keep manifest order
		}

		for _, f := range files {
			if long {
				tags := "-"
				if len(f.InstallTags) > 0 {
					tags = strings.Join(f.InstallTags, ",")
				}
				fmt.Printf("%12s  %3d chunks  %-20s  %x  %s\n",
					humanBytes(f.FileSize),
					len(f.ChunkParts),
					tags,
					f.SHAHash,
					f.FileName,
				)
			} else {
				fmt.Printf("%12s  %s\n", humanBytes(f.FileSize), f.FileName)
			}
		}

		fmt.Printf("\n%d files, %s total\n", len(files), humanBytes(totalSize(files)))
		return nil
	},
}

func totalSize(files []manifest.File) uint64 {
	var total uint64
	for _, f := range files {
		total += f.FileSize
	}
	return total
}

func init() {
	listCmd.Flags().String("tag", "", "filter files by install tag")
	listCmd.Flags().String("sort", "", "sort order: name, size")
	listCmd.Flags().String("filter", "", "filter files by glob pattern (matched against filename)")
	listCmd.Flags().BoolP("long", "l", false, "show SHA hash, chunk count, and tags")
	rootCmd.AddCommand(listCmd)
}
