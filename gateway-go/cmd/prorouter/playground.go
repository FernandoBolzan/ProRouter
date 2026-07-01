package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

var playgroundCmd = &cobra.Command{
	Use:   "playground",
	Short: "Open the Model Arena in browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		url := fmt.Sprintf("http://localhost:%d/playground", port)

		fmt.Printf("Opening Model Arena at %s\n", url)
		if err := openBrowser(url); err != nil {
			return fmt.Errorf("failed to open browser: %w", err)
		}

		return nil
	},
}

func init() {
	playgroundCmd.Flags().IntP("port", "p", 8080, "Dashboard port")
	rootCmd.AddCommand(playgroundCmd)
}
