package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "prorouter",
	Short: "ProRouter - Universal LLM Gateway",
	Long: `ProRouter is an open-source, high-performance LLM gateway and router.
It provides a unified OpenAI-compatible API for routing requests to
cloud providers (OpenAI, Anthropic, Gemini, DeepSeek) and local
instances (Ollama, vLLM, Llama.cpp) with intelligent fallbacks.`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ProRouter v%s\n", Version)
		fmt.Println("Run 'prorouter serve' to start the gateway")
		fmt.Println("Run 'prorouter init' to create configuration")
		fmt.Println("Run 'prorouter doctor' for diagnostics")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ProRouter v%s\n", Version)
	},
}

func main() {
	rootCmd.AddCommand(versionCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
