package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ProRouter to the latest version",
	Long: `Checks for the latest release and updates the binary.
Supports stable, beta, and nightly channels.
Use --rollback to revert to the previous version.`,
	Run: func(cmd *cobra.Command, args []string) {
		rollback, _ := cmd.Flags().GetBool("rollback")
		channel, _ := cmd.Flags().GetString("channel")

		if rollback {
			fmt.Println("Rolling back to previous version...")
			fmt.Println("(Rollback pending implementation)")
			return
		}

		fmt.Printf("Checking for updates (%s channel)...\n", channel)
		fmt.Printf("Current version: 0.1.0\n")
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()
		fmt.Println("(Update mechanism pending implementation)")
		fmt.Println("For now, download the latest binary from:")
		fmt.Println("  https://github.com/FernandoBolzan/ProRouter/releases")
	},
}

func init() {
	updateCmd.Flags().Bool("rollback", false, "Rollback to previous version")
	updateCmd.Flags().StringP("channel", "c", "stable", "Update channel (stable|beta|nightly)")
	rootCmd.AddCommand(updateCmd)
}
