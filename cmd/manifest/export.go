package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/meszmate/manifest"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <file|url>",
	Short: "Export manifest data",
	Long:  "Export manifest data to JSON or CSV format.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		var w *os.File
		if output != "" {
			w, err = os.Create(output)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer w.Close()
		} else {
			w = os.Stdout
		}

		switch format {
		case "json":
			return exportJSON(m, w)
		case "csv":
			return exportCSV(m, w)
		default:
			return fmt.Errorf("unsupported format: %s (use json or csv)", format)
		}
	},
}

type jsonManifest struct {
	AppName      string            `json:"app_name"`
	BuildVersion string            `json:"build_version"`
	BuildID      string            `json:"build_id,omitempty"`
	AppID        int32             `json:"app_id"`
	FeatureLevel string            `json:"feature_level"`
	DownloadSize uint64            `json:"download_size"`
	InstallSize  uint64            `json:"install_size"`
	FileCount    int               `json:"file_count"`
	ChunkCount   int               `json:"chunk_count"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
	Files        []jsonFile        `json:"files"`
	Chunks       []jsonChunk       `json:"chunks"`
}

type jsonFile struct {
	Path     string   `json:"path"`
	Size     uint64   `json:"size"`
	SHA      string   `json:"sha"`
	Tags     []string `json:"tags,omitempty"`
	ChunkNum int      `json:"chunk_count"`
}

type jsonChunk struct {
	GUID     string `json:"guid"`
	Hash     string `json:"hash"`
	SHA      string `json:"sha"`
	Group    uint8  `json:"group"`
	FileSize uint64 `json:"file_size"`
}

func exportJSON(m *manifest.BinaryManifest, w *os.File) error {
	out := jsonManifest{
		AppName:      m.Metadata.AppName,
		BuildVersion: m.Metadata.BuildVersion,
		BuildID:      m.Metadata.BuildId,
		AppID:        m.Metadata.AppID,
		FeatureLevel: m.Metadata.FeatureLevel.String(),
		DownloadSize: m.TotalDownloadSize(),
		InstallSize:  m.TotalInstallSize(),
		FileCount:    len(m.FileManifestList.FileManifestList),
		ChunkCount:   len(m.ChunkDataList.Chunks),
		CustomFields: m.CustomFields.Fields,
	}

	for _, f := range m.FileManifestList.FileManifestList {
		out.Files = append(out.Files, jsonFile{
			Path:     f.FileName,
			Size:     f.FileSize,
			SHA:      fmt.Sprintf("%x", f.SHAHash),
			Tags:     f.InstallTags,
			ChunkNum: len(f.ChunkParts),
		})
	}

	for _, c := range m.ChunkDataList.Chunks {
		out.Chunks = append(out.Chunks, jsonChunk{
			GUID:     c.GUID.String(),
			Hash:     fmt.Sprintf("%016X", c.Hash),
			SHA:      fmt.Sprintf("%x", c.SHAHash),
			Group:    c.Group,
			FileSize: c.FileSize,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func exportCSV(m *manifest.BinaryManifest, w *os.File) error {
	enc := csv.NewWriter(w)
	defer enc.Flush()

	if err := enc.Write([]string{"path", "size", "sha", "tags", "chunk_count"}); err != nil {
		return err
	}

	for _, f := range m.FileManifestList.FileManifestList {
		tags := strings.Join(f.InstallTags, ";")
		record := []string{
			f.FileName,
			strconv.FormatUint(f.FileSize, 10),
			fmt.Sprintf("%x", f.SHAHash),
			tags,
			strconv.Itoa(len(f.ChunkParts)),
		}
		if err := enc.Write(record); err != nil {
			return err
		}
	}

	return enc.Error()
}

func init() {
	exportCmd.Flags().String("format", "json", "output format: json, csv")
	exportCmd.Flags().StringP("output", "o", "", "output file (default: stdout)")
	rootCmd.AddCommand(exportCmd)
}
