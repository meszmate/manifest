package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/meszmate/manifest/downloader"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <file|url>",
	Short: "Download game files from a manifest",
	Long:  "Download and assemble game files using an Epic Games manifest and CDN base URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := loadManifest(args[0])
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		baseURL, _ := cmd.Flags().GetString("base-url")
		output, _ := cmd.Flags().GetString("output")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		retries, _ := cmd.Flags().GetInt("retries")
		noVerify, _ := cmd.Flags().GetBool("no-verify")
		noResume, _ := cmd.Flags().GetBool("no-resume")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		excludeTags, _ := cmd.Flags().GetStringSlice("exclude-tag")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if output == "" {
			output = "."
		}

		cfg := downloader.Config{
			OutputDir:      output,
			BaseURL:        baseURL,
			Concurrency:    concurrency,
			MaxRetries:     retries,
			RetryBaseDelay: time.Second,
			VerifyChunks:   !noVerify,
			Resume:         !noResume,
			Tags:           tags,
			ExcludeTags:    excludeTags,
			DryRun:         dryRun,
		}

		if !dryRun {
			cfg.Progress = &cliProgress{}
		}

		// Handle Ctrl+C gracefully
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Fprintln(os.Stderr, "\nInterrupted, finishing current downloads...")
			cancel()
		}()

		d := downloader.New(m, cfg)
		result, err := d.Download(ctx)

		if dryRun && result != nil {
			fmt.Printf("Dry run summary:\n")
			fmt.Printf("  Files:  %d\n", result.TotalFiles)
			fmt.Printf("  Chunks: %d\n", result.TotalChunks)
			fmt.Printf("  Size:   %s\n", humanBytes(result.TotalBytes))
			return nil
		}

		if result != nil {
			fmt.Printf("\nDownload complete in %s\n", result.Duration.Round(time.Millisecond))
			fmt.Printf("  Files:     %d\n", result.TotalFiles)
			fmt.Printf("  Chunks:    %d (skipped: %d, failed: %d)\n",
				result.TotalChunks, result.SkippedChunks, result.FailedChunks)
			fmt.Printf("  Downloaded: %s\n", humanBytes(result.DownloadedBytes))
		}

		return err
	},
}

// cliProgress implements downloader.ProgressReporter using progressbar.
type cliProgress struct {
	bar  *progressbar.ProgressBar
	once sync.Once
}

func (p *cliProgress) Start(totalChunks int, totalBytes uint64) {
	p.bar = progressbar.NewOptions64(
		int64(totalBytes),
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
	)
}

func (p *cliProgress) ChunkCompleted(bytes uint64) {
	if p.bar != nil {
		p.bar.Add64(int64(bytes))
	}
}

func (p *cliProgress) ChunkFailed(guid string, err error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "\nChunk %s failed: %v\n", guid, err)
	}
}

func (p *cliProgress) FileCompleted(path string, size uint64) {
	if verbose {
		fmt.Fprintf(os.Stderr, "  Assembled: %s\n", path)
	}
}

func (p *cliProgress) Finish() {
	p.once.Do(func() {
		if p.bar != nil {
			p.bar.Finish()
		}
	})
}

func init() {
	downloadCmd.Flags().String("base-url", "", "CDN base URL (required)")
	downloadCmd.MarkFlagRequired("base-url")
	downloadCmd.Flags().StringP("output", "o", "", "output directory (default: current directory)")
	downloadCmd.Flags().IntP("concurrency", "c", 16, "number of parallel downloads")
	downloadCmd.Flags().Int("retries", 5, "max retries per chunk")
	downloadCmd.Flags().Bool("no-verify", false, "skip SHA1 verification")
	downloadCmd.Flags().Bool("no-resume", false, "disable resume (re-download all chunks)")
	downloadCmd.Flags().StringSlice("tag", nil, "include only files with these tags (repeatable)")
	downloadCmd.Flags().StringSlice("exclude-tag", nil, "exclude files with these tags (repeatable)")
	downloadCmd.Flags().Bool("dry-run", false, "show what would be downloaded without downloading")
	rootCmd.AddCommand(downloadCmd)
}
