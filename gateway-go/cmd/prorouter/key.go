package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func generateAPIKey() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(token))
	return "pr-" + token, hex.EncodeToString(hash[:]), nil
}

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage API keys",
}

var keyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		budget, _ := cmd.Flags().GetFloat64("budget")

		key, hash, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		fmt.Println("New API Key Generated:")
		fmt.Println("─────────────────────")
		fmt.Printf("Key:   %s\n", key)
		fmt.Printf("Hash:  %s\n", hash)
		fmt.Printf("Name:  %s\n", name)
		if budget > 0 {
			fmt.Printf("Budget: $%.2f\n", budget)
		}
		fmt.Printf("Created: %s\n", time.Now().Format(time.RFC3339))
		fmt.Println()
		fmt.Println("Store this key securely - it will not be shown again!")

		return nil
	},
}

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("API Keys:")
		fmt.Println("─────────────────────")
		fmt.Println("(Database not yet connected - no keys to show)")
		return nil
	},
}

var keyRevokeCmd = &cobra.Command{
	Use:   "revoke <key-id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Revoking key: %s\n", args[0])
		fmt.Println("(Database not yet connected - key not revoked)")
		return nil
	},
}

func init() {
	keyGenerateCmd.Flags().StringP("name", "n", "default", "Key name")
	keyGenerateCmd.Flags().Float64P("budget", "b", 0, "Monthly budget limit (0 = unlimited)")

	keyCmd.AddCommand(keyGenerateCmd)
	keyCmd.AddCommand(keyListCmd)
	keyCmd.AddCommand(keyRevokeCmd)
	rootCmd.AddCommand(keyCmd)
}
