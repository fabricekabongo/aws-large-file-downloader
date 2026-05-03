package main

import (
	"context"
	"os"

	"github.com/example/aws-large-file-downloader/cmd"
)

func main() {
	ctx := context.Background()
	if err := cmd.NewRootCommand(ctx).Execute(); err != nil {
		os.Exit(1)
	}
}
