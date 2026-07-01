package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage LLM providers",
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Configured Providers:")
		fmt.Println("────────────────────")
		fmt.Println("  OpenAI        - key: ${OPENAI_API_KEY}")
		fmt.Println("  Anthropic     - key: ${ANTHROPIC_API_KEY}")
		fmt.Println("  Google Gemini - key: ${GEMINI_API_KEY}")
		fmt.Println("  DeepSeek      - key: ${DEEPSEEK_API_KEY}")
		fmt.Println("  Ollama        - local: http://localhost:11434")
		fmt.Println("  vLLM          - local: http://localhost:8000")
		fmt.Println()
		fmt.Println("Run 'prorouter doctor' to test connectivity")
	},
}

var providerAddCmd = &cobra.Command{
	Use:   "add <provider>",
	Short: "Add provider credentials",
	Long: `Add or update provider API credentials.
Supported providers: openai, anthropic, gemini, deepseek, ollama`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		fmt.Printf("Configuring provider: %s\n", provider)
		fmt.Println("(Feature pending - use environment variables for now)")
	},
}

var providerTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Test provider connectivity",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Testing %s...\n", args[0])
		fmt.Println("(Provider test pending implementation)")
	},
}

func init() {
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerAddCmd)
	providerCmd.AddCommand(providerTestCmd)
	rootCmd.AddCommand(providerCmd)
}
