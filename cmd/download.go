package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/example/aws-large-file-downloader/internal/download"
	"github.com/example/aws-large-file-downloader/internal/s3client"
)

type downloadService interface {
	DownloadToFileWithOptions(ctx context.Context, source, destination string, opts download.Options) error
}

func newDownloadCommand(ctx context.Context, service downloadService) *cobra.Command {
	var source string
	var destination string
	var forceRepair bool

	c := &cobra.Command{
		Use:          "download",
		Short:        "Download a file from S3 to local filesystem",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || destination == "" {
				return fmt.Errorf("both --source and --dest are required")
			}
			opts := download.Options{ForceRepair: forceRepair}
			return service.DownloadToFileWithOptions(ctx, source, destination, opts)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	c.Flags().BoolVar(&forceRepair, "force-repair", false, "Delete invalid chunk files and redownload them")
	return c
}

func defaultDownloadCommand(ctx context.Context) *cobra.Command {
	var source string
	var destination string
	var forceRepair bool

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
			opts := download.Options{ForceRepair: forceRepair}
			return svc.DownloadToFileWithOptions(cmd.Context(), source, destination, opts)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	c.Flags().BoolVar(&forceRepair, "force-repair", false, "Delete invalid chunk files and redownload them")
	return c
}
