package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/meszmate/manifest"
	"github.com/spf13/cobra"
)

var deltaCmd = &cobra.Command{
	Use:   "delta <old> <new>",
	Short: "Compare two manifests",
	Long:  "Show differences between two Epic Games manifest files.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldM, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading old manifest: %w", err)
		}

		newM, err := loadManifest(args[1])
		if err != nil {
			return fmt.Errorf("loading new manifest: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")

		oldFiles := make(map[string]*manifest.File)
		for i := range oldM.FileManifestList.FileManifestList {
			f := &oldM.FileManifestList.FileManifestList[i]
			oldFiles[f.FileName] = f
		}

		newFiles := make(map[string]*manifest.File)
		for i := range newM.FileManifestList.FileManifestList {
			f := &newM.FileManifestList.FileManifestList[i]
			newFiles[f.FileName] = f
		}

		type deltaEntry struct {
			Path    string `json:"path"`
			Status  string `json:"status"`
			OldSize uint64 `json:"old_size,omitempty"`
			NewSize uint64 `json:"new_size,omitempty"`
			SizeDiff int64 `json:"size_diff,omitempty"`
		}

		var entries []deltaEntry
		var added, removed, changed, unchanged int

		// Check new files
		for name, nf := range newFiles {
			of, exists := oldFiles[name]
			if !exists {
				added++
				entries = append(entries, deltaEntry{
					Path:    name,
					Status:  "added",
					NewSize: nf.FileSize,
				})
				continue
			}

			if fmt.Sprintf("%x", of.SHAHash) != fmt.Sprintf("%x", nf.SHAHash) {
				changed++
				entries = append(entries, deltaEntry{
					Path:     name,
					Status:   "changed",
					OldSize:  of.FileSize,
					NewSize:  nf.FileSize,
					SizeDiff: int64(nf.FileSize) - int64(of.FileSize),
				})
			} else {
				unchanged++
			}
		}

		// Check for removed files
		for name, of := range oldFiles {
			if _, exists := newFiles[name]; !exists {
				removed++
				entries = append(entries, deltaEntry{
					Path:    name,
					Status:  "removed",
					OldSize: of.FileSize,
				})
			}
		}

		if format == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"added":     added,
				"removed":   removed,
				"changed":   changed,
				"unchanged": unchanged,
				"details":   entries,
			})
		}

		// Text output
		for _, e := range entries {
			switch e.Status {
			case "added":
				fmt.Printf("+ %s (%s)\n", e.Path, humanBytes(e.NewSize))
			case "removed":
				fmt.Printf("- %s (%s)\n", e.Path, humanBytes(e.OldSize))
			case "changed":
				sign := "+"
				diff := e.SizeDiff
				if diff < 0 {
					sign = "-"
					diff = -diff
				}
				fmt.Printf("~ %s (%s -> %s, %s%s)\n", e.Path,
					humanBytes(e.OldSize), humanBytes(e.NewSize),
					sign, humanBytes(uint64(diff)))
			}
		}

		fmt.Printf("\nSummary: %d added, %d removed, %d changed, %d unchanged\n",
			added, removed, changed, unchanged)

		return nil
	},
}

func init() {
	deltaCmd.Flags().String("format", "", "output format: json (default: text)")
	rootCmd.AddCommand(deltaCmd)
}
