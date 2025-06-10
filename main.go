package main

import (
	"fmt"
	"os"

	"github.com/davidroman0O/steam-workshop-downloader/cmd"
)

// Build information. Will be set by ldflags during build.
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	// Set version info in root command
	cmd.SetVersionInfo(version, commit, buildTime)

	// Execute the CLI
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
