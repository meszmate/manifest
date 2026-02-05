package main

import (
	"fmt"

	"github.com/meszmate/manifest/downloader"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <file|url>",
	Short: "Verify an installation against a manifest",
	Long:  "Check installed files against a manifest to find missing or corrupted files.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		path, _ := cmd.Flags().GetString("path")

		result, err := downloader.VerifyInstallation(m, path)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		for _, d := range result.Details {
			switch d.Status {
			case downloader.StatusMissing:
				fmt.Printf("MISSING  %s\n", d.Path)
			case downloader.StatusSizeMismatch:
				fmt.Printf("BAD SIZE %s (expected %s, got %s)\n", d.Path, d.Expected, d.Actual)
			case downloader.StatusHashMismatch:
				fmt.Printf("BAD HASH %s\n", d.Path)
				if verbose {
					fmt.Printf("  expected: %s\n", d.Expected)
					fmt.Printf("  actual:   %s\n", d.Actual)
				}
			case downloader.StatusOK:
				if verbose {
					fmt.Printf("OK       %s\n", d.Path)
				}
			}
		}

		fmt.Printf("\nVerification: %d/%d files OK", result.ValidFiles, result.TotalFiles)
		if result.MissingFiles > 0 {
			fmt.Printf(", %d missing", result.MissingFiles)
		}
		if result.InvalidFiles > 0 {
			fmt.Printf(", %d invalid", result.InvalidFiles)
		}
		fmt.Println()

		if result.MissingFiles > 0 || result.InvalidFiles > 0 {
			return fmt.Errorf("verification failed: %d files need repair",
				result.MissingFiles+result.InvalidFiles)
		}

		return nil
	},
}

func init() {
	verifyCmd.Flags().StringP("path", "p", "", "installation directory to verify (required)")
	verifyCmd.MarkFlagRequired("path")
	rootCmd.AddCommand(verifyCmd)
}
