package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/example/aws-large-file-downloader/internal/download"
)

type fakeDownloadService struct {
	source      string
	destination string
	options     download.Options
	err         error
}

func (f *fakeDownloadService) DownloadToFileWithOptions(_ context.Context, source, destination string, opts download.Options) error {
	f.source = source
	f.destination = destination
	f.options = opts
	return f.err
}

func TestNewDownloadCommand_RequiresFlags(t *testing.T) {
	svc := &fakeDownloadService{}
	cmd := newDownloadCommand(context.Background(), svc)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNewDownloadCommand_DownloadsRequestedFile(t *testing.T) {
	svc := &fakeDownloadService{}
	cmd := newDownloadCommand(context.Background(), svc)
	cmd.SetArgs([]string{"--source", "s3://docs/report.csv", "--dest", "./report.csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if svc.source != "s3://docs/report.csv" || svc.destination != "./report.csv" {
		t.Fatalf("unexpected call values: %s %s", svc.source, svc.destination)
	}
}

func TestNewDownloadCommand_PropagatesServiceErrors(t *testing.T) {
	svc := &fakeDownloadService{err: errors.New("boom")}
	cmd := newDownloadCommand(context.Background(), svc)
	cmd.SetArgs([]string{"--source", "s3://docs/report.csv", "--dest", "./report.csv"})

	if err := cmd.Execute(); !errors.Is(err, svc.err) {
		t.Fatalf("expected wrapped service error, got %v", err)
	}
}

func TestNewDownloadCommand_EnablesForceRepairOption(t *testing.T) {
	svc := &fakeDownloadService{}
	cmd := newDownloadCommand(context.Background(), svc)
	cmd.SetArgs([]string{"--source", "s3://docs/report.csv", "--dest", "./report.csv", "--force-repair"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !svc.options.ForceRepair {
		t.Fatal("expected force-repair option to be enabled")
	}
}
