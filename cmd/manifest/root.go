package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables injected via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Epic Games manifest parser and downloader",
	Long:  "A CLI tool for inspecting, downloading, and verifying Epic Games manifest files.",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}
