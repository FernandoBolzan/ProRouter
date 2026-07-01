package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run system diagnostics",
	Long: `Checks configuration validity, database health,
provider connectivity, and port availability.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ProRouter Diagnostics")
		fmt.Println("─────────────────────")
		fmt.Println()

		// Version info
		fmt.Printf("Version: 0.1.0\n")
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()

		// Config file
		home, _ := os.UserHomeDir()
		configPath := home + "/.prorouter/config.yaml"
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("Config file: ~/.prorouter/config.yaml (found)")
		} else {
			fmt.Println("Config file: ~/.prorouter/config.yaml (not found - run 'prorouter init')")
		}

		// Database
		dbPath := home + "/.prorouter/data/prorouter.db"
		if _, err := os.Stat(dbPath); err == nil {
			fmt.Println("Database: SQLite (WAL mode)")
		} else {
			fmt.Println("Database: not initialized")
		}

		// Port check
		fmt.Println()
		fmt.Println("Port Availability:")
		for _, port := range []int{8080, 3000} {
			addr := fmt.Sprintf("localhost:%d", port)
			conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
			if err != nil {
				fmt.Printf("  Port %d: available\n", port)
			} else {
				conn.Close()
				fmt.Printf("  Port %d: in use\n", port)
			}
		}

		// TODO: Provider connectivity checks
		fmt.Println()
		fmt.Println("Provider Connectivity:")
		providers := []struct {
			name string
			env  string
			url  string
		}{
			{"OpenAI", "OPENAI_API_KEY", "https://api.openai.com/v1/models"},
			{"Anthropic", "ANTHROPIC_API_KEY", "https://api.anthropic.com/v1/messages"},
			{"Ollama (local)", "", "http://localhost:11434/api/tags"},
		}

		for _, p := range providers {
			key := os.Getenv(p.env)
			prefix := "✘"
			if p.env == "" || key != "" {
				// Try basic connectivity
				resp, err := http.Get(p.url)
				if err == nil {
					resp.Body.Close()
					prefix = "✔"
				}
			}
			fmt.Printf("  %s %s\n", prefix, p.name)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
