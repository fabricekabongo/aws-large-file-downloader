package download

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClient struct {
	bucket string
	key    string
	body   string
	err    error
	size   int64
	etag   string
	delay  time.Duration

	mu            sync.Mutex
	inflight      int
	maxConcurrent int
}

type captureReporter struct {
	mu        sync.Mutex
	snapshots []ProgressSnapshot
}

func (c *captureReporter) Report(s ProgressSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshots = append(c.snapshots, s)
}

func (f *fakeClient) HeadObject(_ context.Context, bucket, key string) (ObjectInfo, error) {
	f.bucket = bucket
	f.key = key
	if f.err != nil {
		return ObjectInfo{}, f.err
	}
	return ObjectInfo{Size: f.size, ETag: f.etag}, nil
}

func (f *fakeClient) DownloadRange(_ context.Context, bucket, key string, start, end int64, w io.Writer) error {
	f.bucket = bucket
	f.key = key
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	f.inflight++
	if f.inflight > f.maxConcurrent {
		f.maxConcurrent = f.inflight
	}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.inflight--
		f.mu.Unlock()
	}()

	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if start < 0 || end >= int64(len(f.body)) || start > end {
		return errors.New("invalid range")
	}
	_, err := w.Write([]byte(f.body[start : end+1]))
	return err
}

func TestDownloadToFile_ParallelRespectsWorkerLimit(t *testing.T) {
	tempDir := t.TempDir()
	destination := filepath.Join(tempDir, "parallel.csv")
	body := strings.Repeat("p", 1024)
	client := &fakeClient{body: body, size: int64(len(body)), etag: "etag-p", delay: 20 * time.Millisecond}
	svc := NewService(client)

	err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, Options{ChunkSize: 64, Workers: 4, TrackerDir: filepath.Join(tempDir, ".alld")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.maxConcurrent < 2 {
		t.Fatalf("expected parallelism, got max concurrency %d", client.maxConcurrent)
	}
	if client.maxConcurrent > 4 {
		t.Fatalf("worker limit exceeded, max concurrency %d", client.maxConcurrent)
	}
}

func TestParseS3URI(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		bucket, key, err := ParseS3URI("s3://docs/report.csv")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if bucket != "docs" || key != "report.csv" {
			t.Fatalf("unexpected bucket/key: %s/%s", bucket, key)
		}
	})
	t.Run("invalid format", func(t *testing.T) {
		_, _, err := ParseS3URI("docs/report.csv")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDownloadToFile_WritesDownloadedBytes(t *testing.T) { /* existing */
	tempDir := t.TempDir()
	destination := filepath.Join(tempDir, "nested", "report.csv")
	body := strings.Repeat("a", 200)
	client := &fakeClient{body: body, size: int64(len(body)), etag: "etag-1"}
	svc := NewService(client)
	if err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, Options{ChunkSize: 64, TrackerDir: filepath.Join(tempDir, ".alld")}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != body {
		t.Fatalf("unexpected file body")
	}
}

func TestDownloadToFile_ResumeValidatesAndRepairsWhenForced(t *testing.T) {
	tempDir := t.TempDir()
	destination := filepath.Join(tempDir, "report.csv")
	trackerDir := filepath.Join(tempDir, ".alld")
	body := strings.Repeat("b", 150)
	client := &fakeClient{body: body, size: int64(len(body)), etag: "etag-2"}
	svc := NewService(client)
	opts := Options{ChunkSize: 50, TrackerDir: trackerDir}
	if err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, opts); err != nil {
		t.Fatal(err)
	}
	chunk := filepath.Join(trackerDir, "docs_report.csv", "chunk-000001.part")
	if err := os.WriteFile(chunk, []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, opts)
	if err == nil || !strings.Contains(err.Error(), "--force-repair") {
		t.Fatalf("expected force-repair error, got %v", err)
	}
	opts.ForceRepair = true
	if err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, opts); err != nil {
		t.Fatalf("force repair should succeed: %v", err)
	}
}

func TestDownloadToFile_ReportsProgress(t *testing.T) {
	tempDir := t.TempDir()
	destination := filepath.Join(tempDir, "report.csv")
	body := strings.Repeat("z", 128)
	client := &fakeClient{body: body, size: int64(len(body)), etag: "etag-z"}
	reporter := &captureReporter{}
	svc := NewService(client)

	err := svc.DownloadToFileWithOptions(context.Background(), "s3://docs/report.csv", destination, Options{ChunkSize: 32, Workers: 2, TrackerDir: filepath.Join(tempDir, ".alld"), Reporter: reporter})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(reporter.snapshots) == 0 {
		t.Fatal("expected progress snapshots")
	}
	last := reporter.snapshots[len(reporter.snapshots)-1]
	if last.Status != "completed" || last.DoneChunks != last.TotalChunks {
		t.Fatalf("expected completed snapshot, got %+v", last)
	}
}
