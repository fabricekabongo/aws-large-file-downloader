package download

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Client interface {
	Download(ctx context.Context, bucket, key string, w io.Writer) error
}

type Service struct {
	client Client
}

func NewService(client Client) Service {
	return Service{client: client}
}

func ParseS3URI(uri string) (string, string, error) {
	trimmed := strings.TrimSpace(uri)
	if !strings.HasPrefix(trimmed, "s3://") {
		return "", "", fmt.Errorf("source must start with s3://")
	}
	withoutScheme := strings.TrimPrefix(trimmed, "s3://")
	parts := strings.SplitN(withoutScheme, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("source must be in the format s3://bucket/key")
	}
	return parts[0], parts[1], nil
}

func (s Service) DownloadToFile(ctx context.Context, source, destination string) error {
	bucket, key, err := ParseS3URI(source)
	if err != nil {
		return err
	}
	if destination == "" {
		return fmt.Errorf("destination is required")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return fmt.Errorf("create temp destination file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if err := s.client.Download(ctx, bucket, key, tmp); err != nil {
		return fmt.Errorf("download from s3: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp destination file: %w", err)
	}
	if err := os.Rename(tmpName, destination); err != nil {
		return fmt.Errorf("move temp file to destination: %w", err)
	}
	return nil
}
