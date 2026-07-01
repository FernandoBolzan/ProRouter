package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ProRouter configuration",
	Long: `Creates the default configuration file and directory structure
at ~/.prorouter/config.yaml. Also generates a JWT secret
and initializes the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot find home directory: %w", err)
		}

		configDir := filepath.Join(home, ".prorouter")
		dataDir := filepath.Join(configDir, "data")
		recipesDir := filepath.Join(configDir, "recipes")

		// Create directories
		for _, dir := range []string{configDir, dataDir, recipesDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("cannot create %s: %w", dir, err)
			}
		}

		// Create default config
		configPath := filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists at %s (skipping)\n", configPath)
		} else {
			config := `# ProRouter Configuration
server:
  host: "0.0.0.0"
  port: 8080
  tls_enabled: false

database:
  engine: "sqlite"
  path: "~/.prorouter/data/prorouter.db"
  wal_mode: true

dashboard:
  enabled: true
  port: 3000
  theme: "system"

auth:
  jwt_secret: "auto-generated-on-first-run"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
  google:
    api_key: "${GEMINI_API_KEY}"
  deepseek:
    api_key: "${DEEPSEEK_API_KEY}"
  local:
    scan_ports: true
    ports: [11434, 8000, 1234]

recipes_file: "~/.prorouter/recipes/*.yaml"
log_level: "info"
update_channel: "stable"
`
			if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
				return fmt.Errorf("cannot write config: %w", err)
			}
			fmt.Printf("Created config: %s\n", configPath)
		}

		fmt.Println("\nProRouter initialized successfully!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Set your API keys in environment variables")
		fmt.Println("  2. Run 'prorouter doctor' to verify connectivity")
		fmt.Println("  3. Run 'prorouter serve' to start the gateway")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
