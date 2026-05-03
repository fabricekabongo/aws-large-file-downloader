package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/example/aws-large-file-downloader/internal/app"
	"github.com/example/aws-large-file-downloader/internal/logging"
	"github.com/example/aws-large-file-downloader/internal/telemetry"
)

func NewRootCommand(ctx context.Context) *cobra.Command {
	var logLevel string
	root := &cobra.Command{Use: "aws-large-file-downloader", Short: "Download large files from AWS with a guided TUI."}
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug|info|warn|error")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run interactive download workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			level := parseLevel(logLevel)
			logger := logging.NewLogger(os.Stdout, level)
			shutdown, err := telemetry.Setup(ctx, "aws-large-file-downloader")
			if err != nil {
				return err
			}
			defer func() { _ = shutdown(context.Background()) }()
			return app.Run(ctx, logger)
		},
	}
	root.AddCommand(runCmd)

	root.AddCommand(defaultDownloadCommand(ctx))

	return root
}

func parseLevel(v string) slog.Level {
	switch v {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		if v != "info" {
			fmt.Fprintf(os.Stderr, "unknown log level %q, defaulting to info\n", v)
		}
		return slog.LevelInfo
	}
}
