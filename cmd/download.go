package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/example/aws-large-file-downloader/internal/download"
	"github.com/example/aws-large-file-downloader/internal/logging"
	"github.com/example/aws-large-file-downloader/internal/s3client"
)

type logProgressReporter struct{ logger *slog.Logger }

func (r logProgressReporter) Report(s download.ProgressSnapshot) {
	if r.logger == nil {
		return
	}
	pct := 0.0
	if s.BytesTotal > 0 {
		pct = (float64(s.BytesDone) / float64(s.BytesTotal)) * 100
	}
	r.logger.Info("download status",
		slog.String("status", s.Status),
		slog.Int("chunks_done", s.DoneChunks),
		slog.Int("chunks_total", s.TotalChunks),
		slog.Int("workers", s.Workers),
		slog.Int64("chunk_size_bytes", s.ChunkSize),
		slog.Int64("bytes_done", s.BytesDone),
		slog.Int64("bytes_total", s.BytesTotal),
		slog.Float64("percent", pct),
	)
}

type downloadService interface {
	DownloadToFileWithOptions(ctx context.Context, source, destination string, opts download.Options) error
}

func newDownloadCommand(ctx context.Context, service downloadService) *cobra.Command {
	var source string
	var destination string
	var forceRepair bool
	var workers int
	var chunkSizeMB int64

	c := &cobra.Command{
		Use:          "download",
		Short:        "Download a file from S3 to local filesystem",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || destination == "" {
				return fmt.Errorf("both --source and --dest are required")
			}
			opts := download.Options{ForceRepair: forceRepair, Workers: workers, ChunkSize: chunkSizeMB * 1024 * 1024}
			return service.DownloadToFileWithOptions(ctx, source, destination, opts)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	c.Flags().BoolVar(&forceRepair, "force-repair", false, "Delete invalid chunk files and redownload them")
	c.Flags().IntVar(&workers, "workers", 0, "Number of parallel download workers (default: CPU count)")
	c.Flags().Int64Var(&chunkSizeMB, "chunk-size-mb", 64, "Chunk size in MiB")
	return c
}

func defaultDownloadCommand(ctx context.Context) *cobra.Command {
	var source string
	var destination string
	var forceRepair bool
	var workers int
	var chunkSizeMB int64

	c := &cobra.Command{
		Use:          "download",
		Short:        "Download a file from S3 to local filesystem",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || destination == "" {
				return fmt.Errorf("both --source and --dest are required")
			}
			client, err := s3client.New(cmd.Context())
			if err != nil {
				return err
			}
			svc := download.NewService(client)
			logLevel := "info"
			if cmd.Flag("verbose") != nil && cmd.Flag("verbose").Value.String() == "true" {
				logLevel = "debug"
			}
			logger := logging.NewLogger(os.Stdout, parseLevel(logLevel))
			opts := download.Options{ForceRepair: forceRepair, Workers: workers, ChunkSize: chunkSizeMB * 1024 * 1024, Reporter: logProgressReporter{logger: logger}}
			logger.Info("starting download", slog.String("source", source), slog.String("destination", destination), slog.Int("workers", workers), slog.Int64("chunk_size_mb", chunkSizeMB))
			return svc.DownloadToFileWithOptions(cmd.Context(), source, destination, opts)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	c.Flags().BoolVar(&forceRepair, "force-repair", false, "Delete invalid chunk files and redownload them")
	c.Flags().IntVar(&workers, "workers", 0, "Number of parallel download workers (default: CPU count)")
	c.Flags().Int64Var(&chunkSizeMB, "chunk-size-mb", 64, "Chunk size in MiB")
	return c
}
