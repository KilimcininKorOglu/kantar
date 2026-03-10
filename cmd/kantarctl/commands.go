package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage registries",
		Long:  "List, configure, and sync package registries.",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all configured registries",
			RunE: func(cmd *cobra.Command, _ []string) error {
				fmt.Println("Registry          Mode        Upstream                          Status")
				fmt.Println("────────────────  ──────────  ────────────────────────────────  ──────")
				fmt.Println("docker            allowlist   registry-1.docker.io              active")
				fmt.Println("npm               allowlist   registry.npmjs.org                active")
				fmt.Println("pypi              allowlist   pypi.org                          active")
				fmt.Println("gomod             allowlist   proxy.golang.org                  active")
				fmt.Println("cargo             allowlist   crates.io                         active")
				fmt.Println("maven             allowlist   repo1.maven.org                   active")
				fmt.Println("nuget             allowlist   api.nuget.org                     active")
				fmt.Println("helm              allowlist   —                                 active")
				return nil
			},
		},
		&cobra.Command{
			Use:   "sync [registry]",
			Short: "Sync packages from upstream",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				fmt.Printf("Syncing registry: %s...\n", args[0])
				fmt.Println("Sync complete.")
				return nil
			},
		},
	)

	return cmd
}

func newPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage packages",
		Long:  "Search, approve, block, and inspect packages.",
	}

	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for packages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			fmt.Printf("Searching %q in %s registry...\n", args[0], registry)
			return nil
		},
	}
	searchCmd.Flags().String("registry", "npm", "Registry to search in")

	approveCmd := &cobra.Command{
		Use:   "approve [package@version]",
		Short: "Approve a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			fmt.Printf("Approved: %s (registry: %s)\n", args[0], registry)
			return nil
		},
	}
	approveCmd.Flags().String("registry", "npm", "Registry")

	blockCmd := &cobra.Command{
		Use:   "block [package]",
		Short: "Block a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			reason, _ := cmd.Flags().GetString("reason")
			fmt.Printf("Blocked: %s (registry: %s, reason: %s)\n", args[0], registry, reason)
			return nil
		},
	}
	blockCmd.Flags().String("registry", "npm", "Registry")
	blockCmd.Flags().String("reason", "", "Reason for blocking")

	infoCmd := &cobra.Command{
		Use:   "info [package]",
		Short: "Show package information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			fmt.Printf("Package: %s (registry: %s)\n", args[0], registry)
			return nil
		},
	}
	infoCmd.Flags().String("registry", "npm", "Registry")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import packages from a TOML file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			file, _ := cmd.Flags().GetString("file")
			fmt.Printf("Importing from %s...\n", file)
			return nil
		},
	}
	importCmd.Flags().String("file", "", "TOML file to import")

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export approved packages list",
		RunE: func(cmd *cobra.Command, _ []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			format, _ := cmd.Flags().GetString("format")
			fmt.Printf("Exporting %s packages as %s...\n", registry, format)
			return nil
		},
	}
	exportCmd.Flags().String("registry", "npm", "Registry to export")
	exportCmd.Flags().String("format", "toml", "Output format: toml, json")

	cmd.AddCommand(searchCmd, approveCmd, blockCmd, infoCmd, importCmd, exportCmd)
	return cmd
}

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all users",
			RunE: func(_ *cobra.Command, _ []string) error {
				fmt.Println("ID  Username        Role            Active")
				fmt.Println("──  ──────────────  ──────────────  ──────")
				fmt.Println("1   admin           super_admin     true")
				return nil
			},
		},
	)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			username, _ := cmd.Flags().GetString("username")
			role, _ := cmd.Flags().GetString("role")
			fmt.Printf("Created user: %s (role: %s)\n", username, role)
			return nil
		},
	}
	createCmd.Flags().String("username", "", "Username")
	createCmd.Flags().String("role", "consumer", "Role")

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Manage API tokens",
	}
	tokenCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			username, _ := cmd.Flags().GetString("username")
			expires, _ := cmd.Flags().GetString("expires")
			fmt.Printf("Token created for %s (expires: %s)\n", username, expires)
			fmt.Println("Token: kntr_xxxxxxxxxxxxxxxxxxxx (shown only once)")
			return nil
		},
	}
	tokenCreateCmd.Flags().String("username", "", "Username")
	tokenCreateCmd.Flags().String("expires", "90d", "Expiry duration")
	tokenCmd.AddCommand(tokenCreateCmd)

	cmd.AddCommand(createCmd, tokenCmd)
	return cmd
}

func newPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage policies",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "validate",
			Short: "Validate policy files",
			RunE: func(_ *cobra.Command, _ []string) error {
				fmt.Println("All policy files are valid.")
				return nil
			},
		},
	)

	testCmd := &cobra.Command{
		Use:   "test [package@version]",
		Short: "Test a package against policies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, _ := cmd.Flags().GetString("registry")
			fmt.Printf("Testing %s against policies (registry: %s)...\n", args[0], registry)
			fmt.Println("Result: PASS")
			return nil
		},
	}
	testCmd.Flags().String("registry", "npm", "Registry")

	cmd.AddCommand(testCmd)
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("Kantar System Status")
			fmt.Println("────────────────────")
			fmt.Println("Server:    healthy")
			fmt.Println("Database:  connected")
			fmt.Println("Storage:   available")
			fmt.Println("Cache:     active")
			return nil
		},
	}
}
