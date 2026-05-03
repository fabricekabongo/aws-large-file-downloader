package cmd

import "testing"

func TestNewRootCommand_HasRunAndDownloadSubcommands(t *testing.T) {
	root := NewRootCommand(nil)
	found := map[string]bool{"run": false, "download": false}
	for _, c := range root.Commands() {
		if _, ok := found[c.Name()]; ok {
			found[c.Name()] = true
		}
	}
	for name, ok := range found {
		if !ok {
			t.Fatalf("expected %s subcommand", name)
		}
	}
}
