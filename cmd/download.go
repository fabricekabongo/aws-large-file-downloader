package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/example/aws-large-file-downloader/internal/download"
	"github.com/example/aws-large-file-downloader/internal/s3client"
)

type downloadService interface {
	DownloadToFile(ctx context.Context, source, destination string) error
}

func newDownloadCommand(ctx context.Context, service downloadService) *cobra.Command {
	var source string
	var destination string

	c := &cobra.Command{
		Use:          "download",
		Short:        "Download a file from S3 to local filesystem",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || destination == "" {
				return fmt.Errorf("both --source and --dest are required")
			}
			return service.DownloadToFile(ctx, source, destination)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	return c
}

func defaultDownloadCommand(ctx context.Context) *cobra.Command {
	var source string
	var destination string

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
			return svc.DownloadToFile(cmd.Context(), source, destination)
		},
	}
	c.Flags().StringVar(&source, "source", "", "S3 URI of file to download (s3://bucket/key)")
	c.Flags().StringVar(&destination, "dest", "", "Local destination path")
	return c
}
