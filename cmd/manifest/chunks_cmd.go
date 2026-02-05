package main

import (
	"fmt"
	"sort"

	"github.com/meszmate/manifest"
	"github.com/spf13/cobra"
)

var chunksCmd = &cobra.Command{
	Use:   "chunks <file|url>",
	Short: "List chunks in a manifest",
	Long:  "List all chunks referenced by an Epic Games manifest.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		sortBy, _ := cmd.Flags().GetString("sort")

		chunks := make([]*manifest.Chunk, len(m.ChunkDataList.Chunks))
		copy(chunks, m.ChunkDataList.Chunks)

		switch sortBy {
		case "size":
			sort.Slice(chunks, func(i, j int) bool { return chunks[i].FileSize > chunks[j].FileSize })
		case "group":
			sort.Slice(chunks, func(i, j int) bool { return chunks[i].Group < chunks[j].Group })
		case "guid":
			sort.Slice(chunks, func(i, j int) bool { return chunks[i].GUID.String() < chunks[j].GUID.String() })
		}

		for _, c := range chunks {
			fmt.Printf("%s  group:%02d  size:%8s  hash:%016X  sha:%x\n",
				c.GUID, c.Group, humanBytes(c.FileSize), c.Hash, c.SHAHash)
		}

		var totalSize uint64
		for _, c := range chunks {
			totalSize += c.FileSize
		}

		fmt.Printf("\n%d chunks, %s total\n", len(chunks), humanBytes(totalSize))
		return nil
	},
}

func init() {
	chunksCmd.Flags().String("sort", "", "sort order: size, group, guid")
	rootCmd.AddCommand(chunksCmd)
}
