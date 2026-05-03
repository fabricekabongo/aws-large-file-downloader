package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestNewLogger_WritesJSON(t *testing.T) {
	var b bytes.Buffer
	l := NewLogger(&b, slog.LevelInfo)
	l.InfoContext(context.Background(), "hello", slog.String("k", "v"))
	if b.Len() == 0 {
		t.Fatal("expected output")
	}
}
