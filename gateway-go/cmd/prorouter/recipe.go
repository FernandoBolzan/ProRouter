package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage routing recipes",
}

var recipeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recipes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Routing Recipes:")
		fmt.Println("────────────────")
		fmt.Println("(No recipes configured yet)")
		fmt.Println()
		fmt.Println("Create a recipe file in ~/.prorouter/recipes/")
		fmt.Println("Or run 'prorouter recipe create'")
	},
}

var recipeApplyCmd = &cobra.Command{
	Use:   "apply <file>",
	Short: "Apply a recipe from file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Applying recipe from: %s\n", args[0])
		fmt.Println("(Recipe engine pending implementation)")
	},
}

func init() {
	recipeCmd.AddCommand(recipeListCmd)
	recipeCmd.AddCommand(recipeApplyCmd)
	rootCmd.AddCommand(recipeCmd)
}
