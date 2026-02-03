package main

import (
	"fmt"
	"os"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("kantarctl %s (commit: %s, built: %s)\n", version, commit, buildDate)
		return
	}

	fmt.Println("kantarctl - Kantar CLI Tool")
	fmt.Printf("Version: %s\n", version)
}
