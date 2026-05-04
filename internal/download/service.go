package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ObjectInfo struct {
	Size int64  `json:"size"`
	ETag string `json:"etag"`
}

type Client interface {
	HeadObject(ctx context.Context, bucket, key string) (ObjectInfo, error)
	DownloadRange(ctx context.Context, bucket, key string, start, end int64, w io.Writer) error
}

type Service struct {
	client Client
}

type Options struct {
	SessionName string
	TrackerDir  string
	ChunkSize   int64
	ForceRepair bool
}

type chunkState struct {
	Index    int    `json:"index"`
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
	Path     string `json:"path"`
	Expected int64  `json:"expected_size"`
	Status   string `json:"status"`
}

type tracker struct {
	SessionName  string       `json:"session_name"`
	Status       string       `json:"status"`
	Source       string       `json:"source"`
	Bucket       string       `json:"bucket"`
	Key          string       `json:"key"`
	Destination  string       `json:"destination"`
	ObjectSize   int64        `json:"object_size"`
	ETag         string       `json:"etag"`
	ChunkSize    int64        `json:"chunk_size"`
	TrackerPath  string       `json:"tracker_path"`
	ChunkDirPath string       `json:"chunk_dir_path"`
	Chunks       []chunkState `json:"chunks"`
}

func NewService(client Client) Service { return Service{client: client} }

func ParseS3URI(uri string) (string, string, error) { /* unchanged */
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
	return s.DownloadToFileWithOptions(ctx, source, destination, Options{ChunkSize: 64 * 1024 * 1024, TrackerDir: ".alld"})
}

func (s Service) DownloadToFileWithOptions(ctx context.Context, source, destination string, opts Options) error {
	bucket, key, err := ParseS3URI(source)
	if err != nil {
		return err
	}
	if destination == "" {
		return fmt.Errorf("destination is required")
	}
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 64 * 1024 * 1024
	}
	if opts.TrackerDir == "" {
		opts.TrackerDir = ".alld"
	}
	if opts.SessionName == "" {
		opts.SessionName = strings.ReplaceAll(strings.TrimPrefix(source, "s3://"), "/", "_")
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	if err := os.WriteFile(destination+".lock", []byte("locked"), 0o600); err != nil {
		return fmt.Errorf("create lock file: %w", err)
	}
	defer os.Remove(destination + ".lock")
	out, err := os.OpenFile(destination, os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("create destination placeholder: %w", err)
	}
	_ = out.Close()

	obj, err := s.client.HeadObject(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("head s3 object: %w", err)
	}

	chunkDir := filepath.Join(opts.TrackerDir, opts.SessionName)
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		return fmt.Errorf("create chunk dir: %w", err)
	}
	trackerPath := filepath.Join(opts.TrackerDir, opts.SessionName+".alld")
	tr := buildTracker(source, bucket, key, destination, chunkDir, trackerPath, obj, opts)
	if err := loadOrValidateTracker(&tr, trackerPath, opts.ForceRepair); err != nil {
		return err
	}
	if err := saveTracker(trackerPath, tr); err != nil {
		return err
	}

	for i := range tr.Chunks {
		c := &tr.Chunks[i]
		if c.Status == "done" {
			continue
		}
		if err := downloadChunk(ctx, s.client, bucket, key, *c); err != nil {
			return fmt.Errorf("download chunk %d: %w", c.Index, err)
		}
		c.Status = "done"
		if err := saveTracker(trackerPath, tr); err != nil {
			return err
		}
	}

	if err := consolidate(destination, tr.Chunks); err != nil {
		return err
	}
	tr.Status = "completed"
	return saveTracker(trackerPath, tr)
}

func buildTracker(source, bucket, key, dest, chunkDir, trackerPath string, obj ObjectInfo, opts Options) tracker {
	chunks := []chunkState{}
	var idx int
	for start := int64(0); start < obj.Size; start += opts.ChunkSize {
		end := start + opts.ChunkSize - 1
		if end >= obj.Size {
			end = obj.Size - 1
		}
		expected := end - start + 1
		chunks = append(chunks, chunkState{Index: idx, Start: start, End: end, Expected: expected, Path: filepath.Join(chunkDir, fmt.Sprintf("chunk-%06d.part", idx)), Status: "pending"})
		idx++
	}
	return tracker{SessionName: opts.SessionName, Status: "in_progress", Source: source, Bucket: bucket, Key: key, Destination: dest, ObjectSize: obj.Size, ETag: obj.ETag, ChunkSize: opts.ChunkSize, TrackerPath: trackerPath, ChunkDirPath: chunkDir, Chunks: chunks}
}
func loadOrValidateTracker(expected *tracker, path string, forceRepair bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read tracker: %w", err)
	}
	var existing tracker
	if err := json.Unmarshal(b, &existing); err != nil {
		return fmt.Errorf("parse tracker: %w", err)
	}
	if existing.ETag != expected.ETag || existing.ObjectSize != expected.ObjectSize {
		return fmt.Errorf("tracker does not match source object metadata")
	}
	for i := range expected.Chunks {
		fi, err := os.Stat(expected.Chunks[i].Path)
		if err == nil && fi.Size() == expected.Chunks[i].Expected {
			expected.Chunks[i].Status = "done"
			continue
		}
		if err == nil && !forceRepair {
			return fmt.Errorf("chunk validation failed for %s; rerun with --force-repair", expected.Chunks[i].Path)
		}
		_ = os.Remove(expected.Chunks[i].Path)
	}
	return nil
}
func downloadChunk(ctx context.Context, c Client, bucket, key string, ch chunkState) error {
	f, err := os.Create(ch.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.DownloadRange(ctx, bucket, key, ch.Start, ch.End, f)
}
func consolidate(destination string, chunks []chunkState) error {
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".merge-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { tmp.Close(); os.Remove(tmpName) }()
	for _, ch := range chunks {
		f, err := os.Open(ch.Path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tmp, f); err != nil {
			_ = f.Close()
			return err
		}
		_ = f.Close()
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, destination)
}
func saveTracker(path string, tr tracker) error {
	b, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
