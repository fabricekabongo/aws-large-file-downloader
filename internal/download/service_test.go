package download

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type fakeClient struct {
	bucket string
	key    string
	body   string
	err    error
}

func (f *fakeClient) Download(_ context.Context, bucket, key string, w io.Writer) error {
	f.bucket = bucket
	f.key = key
	if f.err != nil {
		return f.err
	}
	_, err := w.Write([]byte(f.body))
	return err
}

func TestParseS3URI(t *testing.T) {
	bucket, key, err := ParseS3URI("s3://docs/report.csv")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bucket != "docs" || key != "report.csv" {
		t.Fatalf("unexpected bucket/key: %s/%s", bucket, key)
	}
}

func TestDownloadToFile_WritesDownloadedBytes(t *testing.T) {
	tempDir := t.TempDir()
	destination := filepath.Join(tempDir, "nested", "report.csv")
	client := &fakeClient{body: "hello-world"}
	svc := NewService(client)

	if err := svc.DownloadToFile(context.Background(), "s3://docs/report.csv", destination); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if client.bucket != "docs" || client.key != "report.csv" {
		t.Fatalf("unexpected bucket/key: %s/%s", client.bucket, client.key)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != "hello-world" {
		t.Fatalf("unexpected file body: %q", string(got))
	}
}

func TestDownloadToFile_PropagatesDownloaderErrors(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(&fakeClient{err: expected})

	err := svc.DownloadToFile(context.Background(), "s3://docs/report.csv", filepath.Join(t.TempDir(), "report.csv"))
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped error %v, got %v", expected, err)
	}
}
