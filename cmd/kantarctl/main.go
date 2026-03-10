package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kantarctl",
		Short: "Kantar CLI — Unified Package Registry Management",
		Long:  "kantarctl is the command-line tool for managing the Kantar package registry platform.",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	// Global flags
	rootCmd.PersistentFlags().String("server", "http://localhost:8080", "Kantar server URL")
	rootCmd.PersistentFlags().String("token", "", "API authentication token")
	rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format: table, json")

	// Add subcommands
	rootCmd.AddCommand(
		newVersionCmd(),
		newRegistryCmd(),
		newPackageCmd(),
		newUserCmd(),
		newPolicyCmd(),
		newStatusCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("kantarctl %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}
}
