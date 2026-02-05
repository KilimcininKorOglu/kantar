package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("kantar %s (commit: %s, built: %s)\n", version, commit, buildDate)
		return
	}

	_ = ctx
	fmt.Println("Kantar - Unified Local Package Registry Platform")
	fmt.Printf("Version: %s\n", version)
}
