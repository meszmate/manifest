package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <file|url>",
	Short: "Display manifest information",
	Long:  "Show detailed information about an Epic Games manifest file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		meta := m.Metadata
		header := m.Header

		fmt.Println("=== Manifest Info ===")
		fmt.Printf("App Name:        %s\n", meta.AppName)
		fmt.Printf("Build Version:   %s\n", meta.BuildVersion)
		if meta.DataVersion >= 1 {
			fmt.Printf("Build ID:        %s\n", meta.BuildId)
		}
		fmt.Printf("App ID:          %d\n", meta.AppID)
		fmt.Printf("Feature Level:   %s\n", meta.FeatureLevel)
		fmt.Printf("Is File Data:    %v\n", meta.IsFileData)
		fmt.Println()

		fmt.Println("=== Sizes ===")
		fmt.Printf("Files:           %d\n", len(m.FileManifestList.FileManifestList))
		fmt.Printf("Chunks:          %d\n", len(m.ChunkDataList.Chunks))
		fmt.Printf("Download Size:   %s\n", humanBytes(m.TotalDownloadSize()))
		fmt.Printf("Install Size:    %s\n", humanBytes(m.TotalInstallSize()))
		fmt.Println()

		fmt.Println("=== Header ===")
		var flags []string
		if header.StoredAs&0x01 != 0 {
			flags = append(flags, "Compressed")
		}
		if header.StoredAs&0x02 != 0 {
			flags = append(flags, "Encrypted")
		}
		storedAs := "None"
		if len(flags) > 0 {
			storedAs = strings.Join(flags, ", ")
		}
		fmt.Printf("Stored As:       %s\n", storedAs)
		fmt.Printf("Header Size:     %d bytes\n", header.HeaderSize)
		fmt.Printf("Version:         %s\n", header.Version)

		if meta.LaunchExe != "" {
			fmt.Println()
			fmt.Println("=== Launch ===")
			fmt.Printf("Executable:      %s\n", meta.LaunchExe)
			if meta.LaunchCommand != "" {
				fmt.Printf("Command:         %s\n", meta.LaunchCommand)
			}
		}

		if meta.PrereqName != "" {
			fmt.Println()
			fmt.Println("=== Prerequisites ===")
			fmt.Printf("Name:            %s\n", meta.PrereqName)
			if meta.PrereqPath != "" {
				fmt.Printf("Path:            %s\n", meta.PrereqPath)
			}
			if meta.PrereqArgs != "" {
				fmt.Printf("Args:            %s\n", meta.PrereqArgs)
			}
			if len(meta.PrereqIds) > 0 {
				fmt.Printf("IDs:             %s\n", strings.Join(meta.PrereqIds, ", "))
			}
		}

		if len(m.CustomFields.Fields) > 0 {
			fmt.Println()
			fmt.Println("=== Custom Fields ===")
			keys := make([]string, 0, len(m.CustomFields.Fields))
			for k := range m.CustomFields.Fields {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%-16s %s\n", k+":", m.CustomFields.Fields[k])
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
