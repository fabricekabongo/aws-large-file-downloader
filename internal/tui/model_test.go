package tui

import (
	"strings"
	"testing"
)

func TestModelView_ProgressBarIsAtTop(t *testing.T) {
	m := NewModel()
	m.Percent = 0.5

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 4 {
		t.Fatalf("unexpected view: %q", view)
	}
	if lines[0] != "Progress:" {
		t.Fatalf("expected progress heading on first line, got %q", lines[0])
	}
	if !strings.Contains(view, "Download target:") {
		t.Fatalf("expected download target prompt in view: %q", view)
	}
}
