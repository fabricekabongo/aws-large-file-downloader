package telemetry

import (
	"context"
	"testing"
)

func TestSetup_ReturnsShutdown(t *testing.T) {
	shutdown, err := Setup(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown func")
	}
	_ = shutdown(context.Background())
}
