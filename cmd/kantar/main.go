package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/KilimcininKorOglu/kantar/internal/audit"
	"github.com/KilimcininKorOglu/kantar/internal/auth"
	"github.com/KilimcininKorOglu/kantar/internal/cache"
	"github.com/KilimcininKorOglu/kantar/internal/config"
	"github.com/KilimcininKorOglu/kantar/internal/database"
	"github.com/KilimcininKorOglu/kantar/internal/database/sqlc"
	"github.com/KilimcininKorOglu/kantar/internal/manager"
	"github.com/KilimcininKorOglu/kantar/internal/plugin"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/cargo"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/docker"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/gomod"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/helm"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/maven"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/npm"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/nuget"
	"github.com/KilimcininKorOglu/kantar/internal/plugins/pypi"
	"github.com/KilimcininKorOglu/kantar/internal/server"
	"github.com/KilimcininKorOglu/kantar/internal/storage"
	syncp "github.com/KilimcininKorOglu/kantar/internal/sync"
	"github.com/KilimcininKorOglu/kantar/internal/util"
	"github.com/KilimcininKorOglu/kantar/migrations"
	"github.com/KilimcininKorOglu/kantar/pkg/registry"
	"github.com/KilimcininKorOglu/kantar/web"
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

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			srv, cleanup, err := buildApp(ctx, cfg, logger)
			if err != nil {
				return err
			}
			defer cleanup()

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

func buildApp(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*server.Server, func(), error) {
	// 1. Database
	db, err := database.New(cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("database: %w", err)
	}
	cleanup := func() { db.Close() }

	if err := db.MigrateWithFS(ctx, migrations.FS); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("migrations: %w", err)
	}
	logger.Info("database ready", "type", cfg.Database.Type)

	rawDB := db.Conn()
	queries := sqlc.New(rawDB)

	// 2. Default admin user (first run)
	ensureDefaultAdmin(ctx, queries, logger)

	// 2b. Seed runtime settings from config
	if err := seedSettings(ctx, queries, cfg, logger); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("seed settings: %w", err)
	}
	if err := seedRegistries(ctx, queries, cfg, logger); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("seed registries: %w", err)
	}
	if err := seedPolicies(ctx, queries, logger); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("seed policies: %w", err)
	}

	// 3. JWT Manager
	secret := cfg.Auth.JWTSecret
	if secret == "" {
		secret = generateRandomSecret()
		logger.Warn("jwt_secret not set; using transient secret — sessions will not survive restarts")
	}
	jwtMgr, err := auth.NewJWTManager(secret, cfg.Auth.SessionTTL.Duration)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("jwt manager: %w", err)
	}

	// 4. Storage
	store, err := storage.NewFilesystem(cfg.Storage.Path)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("storage: %w", err)
	}
	logger.Info("storage ready", "path", cfg.Storage.Path)

	// 5. Cache
	var appCache cache.Cache
	if cfg.Cache.Enabled && cfg.Cache.Type == "memory" {
		maxBytes, parseErr := util.ParseSize(cfg.Cache.MaxSize)
		if parseErr != nil {
			logger.Warn("invalid cache max_size, using 1GB default", "error", parseErr)
			maxBytes = 1 << 30
		}
		appCache = cache.NewMemory(maxBytes, cfg.Cache.TTL.Duration)
		logger.Info("cache ready", "type", "memory", "maxSize", cfg.Cache.MaxSize)
	}

	// 6. Manager
	mgr := manager.New(rawDB)

	// 7. Audit Logger
	auditLog := audit.NewLogger(rawDB)

	// 8. Plugin Registry
	pluginReg := plugin.NewRegistry(logger)
	npmPlugin := npm.New(store, logger)
	pypiPlugin := pypi.New(store, logger)
	gomodPlugin := gomod.New(store, logger)
	cargoPlugin := cargo.New(store, logger)
	mavenPlugin := maven.New(store, logger)
	nugetPlugin := nuget.New(store, logger)
	helmPlugin := helm.New(store, logger)

	// Wire cache into plugins
	dockerPlugin := docker.New(store, logger)
	if appCache != nil {
		dockerPlugin.WithCache(appCache)
		npmPlugin.WithCache(appCache)
		pypiPlugin.WithCache(appCache)
		gomodPlugin.WithCache(appCache)
		cargoPlugin.WithCache(appCache)
		mavenPlugin.WithCache(appCache)
		nugetPlugin.WithCache(appCache)
		helmPlugin.WithCache(appCache)
	}

	_ = pluginReg.Register(dockerPlugin)
	_ = pluginReg.Register(npmPlugin)
	_ = pluginReg.Register(pypiPlugin)
	_ = pluginReg.Register(gomodPlugin)
	_ = pluginReg.Register(cargoPlugin)
	_ = pluginReg.Register(mavenPlugin)
	_ = pluginReg.Register(nugetPlugin)
	_ = pluginReg.Register(helmPlugin)

	pluginConfigs := buildPluginConfigs(cfg.Registries)
	if err := pluginReg.ConfigureAll(pluginConfigs); err != nil {
		logger.Warn("plugin configuration error", "error", err)
	}

	// 9. Sync Engine — register dependency resolvers for all supported ecosystems
	syncEngine := syncp.NewEngine(rawDB, auditLog, logger)
	syncEngine.RegisterResolver(registry.EcosystemNPM, npmPlugin)
	syncEngine.RegisterResolver(registry.EcosystemPyPI, pypiPlugin)
	syncEngine.RegisterResolver(registry.EcosystemGoMod, gomodPlugin)
	syncEngine.RegisterResolver(registry.EcosystemCargo, cargoPlugin)
	syncEngine.RegisterResolver(registry.EcosystemMaven, mavenPlugin)
	syncEngine.RegisterResolver(registry.EcosystemNuGet, nugetPlugin)
	syncEngine.RegisterResolver(registry.EcosystemHelm, helmPlugin)
	syncEngine.Start(ctx, 3)
	logger.Info("sync engine started", "workers", 3)

	// 10. Server
	deps := server.Dependencies{
		Queries:     queries,
		JWTManager:  jwtMgr,
		Manager:     mgr,
		AuditLogger: auditLog,
		SyncEngine:  syncEngine,
		Cache:       appCache,
	}
	srv := server.New(cfg.Server, logger, deps)

	// 11. Mount plugin routes
	pluginReg.MountRoutes(srv.Router())

	// 12. Mount Web UI (LAST — catch-all route)
	webFS, webErr := web.FS()
	if webErr != nil {
		logger.Warn("web UI not available", "error", webErr)
	} else {
		srv.MountWebUI(webFS)
		logger.Info("web UI mounted")
	}

	return srv, cleanup, nil
}

func buildPluginConfigs(registries map[string]config.RegistryConfig) map[string]map[string]any {
	configs := make(map[string]map[string]any, len(registries))
	for eco, rc := range registries {
		if !rc.Enabled {
			continue
		}
		configs[eco] = map[string]any{
			"upstream": rc.Upstream,
			"mode":     rc.Mode,
		}
	}
	return configs
}

func generateRandomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func ensureDefaultAdmin(ctx context.Context, queries *sqlc.Queries, logger *slog.Logger) {
	count, err := queries.CountUsers(ctx)
	if err != nil {
		logger.Warn("failed to count users", "error", err)
		return
	}
	if count > 0 {
		return
	}

	password := generateRandomPassword(16)
	hash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error("failed to hash default admin password", "error", err)
		return
	}

	_, err = queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     "admin",
		Email:        sql.NullString{Valid: false},
		PasswordHash: hash,
		Role:         string(auth.RoleSuperAdmin),
	})
	if err != nil {
		logger.Error("failed to create default admin", "error", err)
		return
	}

	logger.Warn("default admin user created",
		"username", "admin",
		"password", password,
		"action", "change this password immediately",
	)
}

func generateRandomPassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)[:n]
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

func seedSettings(ctx context.Context, queries *sqlc.Queries, cfg *config.Config, logger *slog.Logger) error {
	defaults := []struct {
		key, value, category, description string
	}{
		{"log.level", cfg.Logging.Level, "logging", "Log level (debug, info, warn, error)"},
		{"log.format", cfg.Logging.Format, "logging", "Log format (json, text)"},
		{"log.audit_enabled", fmt.Sprintf("%t", cfg.Logging.AuditEnabled), "logging", "Enable audit logging"},
		{"log.audit_retention", cfg.Logging.AuditRetention.String(), "logging", "Audit log retention period"},
		{"cache.enabled", fmt.Sprintf("%t", cfg.Cache.Enabled), "cache", "Enable caching"},
		{"cache.max_size", cfg.Cache.MaxSize, "cache", "Maximum cache size"},
		{"cache.ttl", cfg.Cache.TTL.String(), "cache", "Cache TTL"},
		{"auth.session_ttl", cfg.Auth.SessionTTL.String(), "auth", "Session token TTL"},
		{"auth.token_ttl", cfg.Auth.TokenTTL.String(), "auth", "API token TTL"},
		{"notifications.enabled", fmt.Sprintf("%t", cfg.Notifications.Enabled), "notifications", "Enable notifications"},
	}

	for _, d := range defaults {
		// Only seed if key doesn't exist yet (don't overwrite admin changes)
		_, err := queries.GetSetting(ctx, d.key)
		if err != nil {
			if err := queries.UpsertSetting(ctx, sqlc.UpsertSettingParams{
				Key:         d.key,
				Value:       d.value,
				Category:    d.category,
				Description: d.description,
			}); err != nil {
				return fmt.Errorf("seeding setting %s: %w", d.key, err)
			}
		}
	}
	logger.Info("settings seeded")
	return nil
}

func seedRegistries(ctx context.Context, queries *sqlc.Queries, cfg *config.Config, logger *slog.Logger) error {
	ecosystems := map[string]struct {
		upstream string
		mode     string
		enabled  bool
	}{
		"docker": {"https://registry-1.docker.io", "allowlist", true},
		"npm":    {"https://registry.npmjs.org", "allowlist", true},
		"pypi":   {"https://pypi.org", "allowlist", true},
		"gomod":  {"https://proxy.golang.org", "allowlist", true},
		"cargo":  {"https://crates.io", "allowlist", true},
		"maven":  {"https://repo1.maven.org/maven2", "allowlist", true},
		"nuget":  {"https://api.nuget.org/v3", "allowlist", true},
		"helm":   {"", "allowlist", true},
	}

	// Override from config
	for eco, rc := range cfg.Registries {
		if e, ok := ecosystems[eco]; ok {
			if rc.Upstream != "" {
				e.upstream = rc.Upstream
			}
			if rc.Mode != "" {
				e.mode = rc.Mode
			}
			e.enabled = rc.Enabled
			ecosystems[eco] = e
		}
	}

	for eco, e := range ecosystems {
		// Only seed if doesn't exist
		_, err := queries.GetRegistry(ctx, eco)
		if err != nil {
			enabledInt := int64(0)
			if e.enabled {
				enabledInt = 1
			}
			if err := queries.UpsertRegistry(ctx, sqlc.UpsertRegistryParams{
				Ecosystem:        eco,
				Mode:             e.mode,
				Upstream:         e.upstream,
				AutoSync:         0,
				AutoSyncInterval: "6h",
				MaxVersions:      0,
				Enabled:          enabledInt,
				ConfigJson:       "{}",
			}); err != nil {
				return fmt.Errorf("seeding registry %s: %w", eco, err)
			}
		}
	}
	logger.Info("registries seeded")
	return nil
}

func seedPolicies(ctx context.Context, queries *sqlc.Queries, logger *slog.Logger) error {
	defaults := []struct {
		name, policyType, configToml string
	}{
		{"license", "license", `allowed = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]
blocked = ["GPL-3.0", "AGPL-3.0"]
action = "block"`},
		{"vulnerability", "vulnerability", `block_severity = ["critical", "high"]
warn_severity = ["medium"]
allow_severity = ["low"]`},
		{"age", "age", `min_package_age = "7d"`},
		{"size", "size", `max_package_size = "500MB"`},
		{"version", "version", `allow_prerelease = false
allow_deprecated = false`},
		{"naming", "naming", `blocked_scopes = []
blocked_prefixes = []`},
	}

	for _, d := range defaults {
		_, err := queries.GetPolicy(ctx, d.name)
		if err != nil {
			if err := queries.UpsertPolicy(ctx, sqlc.UpsertPolicyParams{
				Name:       d.name,
				PolicyType: d.policyType,
				ConfigToml: d.configToml,
				Enabled:    1,
			}); err != nil {
				return fmt.Errorf("seeding policy %s: %w", d.name, err)
			}
		}
	}
	logger.Info("policies seeded")
	return nil
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
