package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available recipes",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runList(recipesDir(), jsonOutput, cmd.OutOrStdout())
	},
}

func init() {
	listCmd.Flags().Bool("json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}

func runList(recipesDir string, jsonOutput bool, w io.Writer) error {
	recipes, err := recipe.LoadAll(recipesDir)
	if err != nil {
		return err
	}

	if jsonOutput {
		type entry struct {
			Name string `json:"name"`
		}
		entries := make([]entry, 0, len(recipes))
		for _, r := range recipes {
			entries = append(entries, entry{Name: r.Name})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	if len(recipes) == 0 {
		fmt.Fprintln(w, "No recipes found.")
		return nil
	}

	for _, r := range recipes {
		fmt.Fprintln(w, r.Name)
	}
	return nil
}
