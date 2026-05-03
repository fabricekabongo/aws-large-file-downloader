package cmd

import "testing"

func TestNewRootCommand_HasRunSubcommand(t *testing.T) {
	root := NewRootCommand(nil)
	found := false
	for _, c := range root.Commands() {
		if c.Name() == "run" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected run subcommand")
	}
}
