package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/KilimcininKorOglu/kantar/internal/config"
	"github.com/KilimcininKorOglu/kantar/internal/server"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kantar",
		Short: "Kantar — Unified Local Package Registry Platform",
		Long:  "Kantar provides a unified platform for managing Docker, npm, PyPI, Go Modules, Cargo, Maven, NuGet, and Helm packages.",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	rootCmd.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newVersionCmd(),
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
			fmt.Printf("kantar %s (commit: %s, built: %s)\n", version, commit, buildDate)
		},
	}
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the Kantar server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if err := config.Validate(cfg); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: parseLogLevel(cfg.Logging.Level),
			}))

			srv := server.New(cfg.Server, logger)

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			logger.Info("kantar starting",
				"version", version,
				"addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			)

			return srv.Start(ctx)
		},
	}

	cmd.Flags().String("config", "", "Path to config file")
	return cmd
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration files",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			force, _ := cmd.Flags().GetBool("force")

			var files []string
			var err error

			if force {
				files, err = config.InitConfigForce(dir)
			} else {
				files, err = config.InitConfig(dir)
			}

			if err != nil {
				return err
			}

			fmt.Println("Configuration files created:")
			for _, f := range files {
				fmt.Printf("  %s\n", f)
			}
			return nil
		},
	}

	cmd.Flags().String("dir", ".", "Directory to create config files in")
	cmd.Flags().Bool("force", false, "Overwrite existing files")
	return cmd
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
