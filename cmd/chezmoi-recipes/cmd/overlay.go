package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fgrehm/chezmoi-recipes/internal/ignore"
	"github.com/fgrehm/chezmoi-recipes/internal/overlay"
	"github.com/fgrehm/chezmoi-recipes/internal/paths"
	"github.com/fgrehm/chezmoi-recipes/internal/recipe"
	"github.com/fgrehm/chezmoi-recipes/internal/setup"
	"github.com/fgrehm/chezmoi-recipes/internal/state"
	"github.com/spf13/cobra"
)

var overlayCmd = &cobra.Command{
	Use:   "overlay [recipe...]",
	Short: "Overlay recipe files into chezmoi source directory",
	Long: `Overlay recipe files into the chezmoi source directory. With no
arguments, overlays all recipes (filtered by .recipeignore). Named
recipes override the ignore list.

This command is typically run automatically via chezmoi's
read-source-state.pre hook. Run it manually to preview changes
or debug overlay behavior.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		quiet, _ := cmd.Flags().GetBool("quiet")
		stateFile, err := paths.DefaultStateFile()
		if err != nil {
			return fmt.Errorf("resolving state file: %w", err)
		}
		chezmoiConfig, err := paths.ChezmoiConfigFile()
		if err != nil {
			return fmt.Errorf("resolving chezmoi config: %w", err)
		}
		return runOverlay(cmd.Context(), args, dryRun, quiet, recipesDir(), stateFile, chezmoiConfig, os.Stdout)
	},
}

func init() {
	overlayCmd.Flags().Bool("dry-run", false, "show what would be overlaid without writing")
	overlayCmd.Flags().Bool("quiet", false, "suppress output (for hook use)")
	rootCmd.AddCommand(overlayCmd)
}

func runOverlay(ctx context.Context, names []string, dryRun, quiet bool, recipesDir, stateFile, chezmoiConfigFile string, w io.Writer) error {
	absRecipesDir, err := filepath.Abs(recipesDir)
	if err != nil {
		return fmt.Errorf("resolving recipes dir: %w", err)
	}
	repoRoot := filepath.Dir(absRecipesDir)
	compiledHome := paths.CompiledHomeDir(repoRoot)
	homeDir := paths.HomeDir(repoRoot)

	// Resolve recipe list.
	recipes, allFiltered, err := resolveRecipes(names, absRecipesDir, chezmoiConfigFile)
	if err != nil {
		return err
	}

	// Detect home/recipe conflicts before any copying.
	if err := overlay.DetectHomeRecipeConflicts(homeDir, recipes); err != nil {
		return err
	}

	// Clear and rebuild compiled-home/ from scratch.
	if !dryRun {
		if err := overlay.ClearDir(compiledHome); err != nil {
			return fmt.Errorf("clearing compiled-home: %w", err)
		}
		if err := os.MkdirAll(compiledHome, 0o755); err != nil {
			return fmt.Errorf("creating compiled-home: %w", err)
		}

		// Copy home/ files first.
		if _, err := overlay.CopyTree(homeDir, compiledHome); err != nil {
			return fmt.Errorf("copying home: %w", err)
		}

		// Deploy shared scripts.
		if err := setup.DeploySharedScripts(compiledHome); err != nil {
			return fmt.Errorf("deploying shared scripts: %w", err)
		}
	}

	// Load state (used for recipe-vs-recipe conflict detection and status tracking).
	store, err := state.Load(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Start with a fresh store for recipe tracking since compiled-home/ is rebuilt.
	store.Recipes = make(map[string]*state.RecipeState)

	// Early exit if no recipes to overlay.
	if len(recipes) == 0 {
		if !quiet {
			if allFiltered {
				fmt.Fprintln(w, "No recipes to overlay (all filtered by .recipeignore).")
			} else {
				fmt.Fprintf(w, "No recipes found in %s\n", recipesDir)
			}
		}
		if !dryRun {
			if err := store.Save(stateFile); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
		}
		return nil
	}

	total := len(recipes)
	multi := total > 1
	var totalAdded, totalUpdated int

	if !quiet && multi {
		if dryRun {
			fmt.Fprintf(w, "Overlaying %d recipes (dry-run)...\n", total)
		} else {
			fmt.Fprintf(w, "Overlaying %d recipes...\n", total)
		}
	}

	for i, r := range recipes {
		if !r.HasChezmoi || r.EmptyChezmoi {
			if !quiet {
				printRecipeResult(w, r.Name, nil, i+1, total, multi, dryRun)
			}
			continue
		}

		var result *overlay.Result
		if dryRun {
			result, err = overlay.Plan(ctx, r, compiledHome, store)
		} else {
			result, err = overlay.Execute(ctx, r, compiledHome, store)
		}
		if err != nil {
			return err
		}

		// Record in memory so next recipe sees ownership.
		store.RecordRecipe(r.Name, result.AllFiles())

		totalAdded += len(result.Added)
		totalUpdated += len(result.Updated)

		if !quiet {
			printRecipeResult(w, r.Name, result, i+1, total, multi, dryRun)
		}
	}

	// Merge per-recipe .chezmoiignore (only in all-recipes mode).
	if len(names) == 0 {
		ignoreEntries := make(map[string]string)
		for _, r := range recipes {
			content, err := overlay.ReadIgnoreEntries(r)
			if err != nil {
				return err
			}
			if content != "" {
				ignoreEntries[r.Name] = content
			}
		}
		if dryRun {
			merged := setup.BuildChezmoiIgnore([]string{"scripts/"}, ignoreEntries)
			existing, _ := os.ReadFile(filepath.Join(compiledHome, ".chezmoiignore"))
			if merged != string(existing) && !quiet {
				fmt.Fprintln(w, "\n.chezmoiignore would be updated")
			}
		} else {
			if err := setup.MergeChezmoiIgnore(compiledHome, []string{"scripts/"}, ignoreEntries); err != nil {
				return err
			}
		}
	}

	// Save state (skip if dry-run).
	if !dryRun {
		if err := store.Save(stateFile); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}
	}

	// Summary line (not for dry-run, not for quiet).
	if !quiet && !dryRun {
		printSummary(w, total, totalAdded, totalUpdated, 0)
	}

	return nil
}

// resolveRecipes determines which recipes to overlay.
// If names is empty, loads all and filters by .recipeignore.
// If names are given, loads each by name (no ignore filtering).
// allFiltered is true when recipes were found but all excluded by .recipeignore.
func resolveRecipes(names []string, recipesDir, chezmoiConfigFile string) (recipes []*recipe.Recipe, allFiltered bool, err error) {
	if len(names) == 0 {
		all, err := recipe.LoadAll(recipesDir)
		if err != nil {
			return nil, false, err
		}
		if len(all) == 0 {
			return nil, false, nil
		}

		ignored, err := ignore.Load(recipesDir, chezmoiConfigFile)
		if err != nil {
			return nil, false, fmt.Errorf("loading .recipeignore: %w", err)
		}

		var filtered []*recipe.Recipe
		for _, r := range all {
			if !ignored[r.Name] {
				filtered = append(filtered, r)
			}
		}
		return filtered, len(filtered) == 0, nil
	}

	// Named recipes: sort alphabetically.
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	var result []*recipe.Recipe
	for _, name := range sorted {
		dir := filepath.Join(recipesDir, name)
		r, err := recipe.Load(dir)
		if err != nil {
			return nil, false, fmt.Errorf("recipe %q: %w", name, err)
		}
		if r == nil {
			return nil, false, fmt.Errorf("recipe %q not found (no README.md in %s)", name, dir)
		}
		if !r.HasChezmoi {
			return nil, false, fmt.Errorf("recipe %q has no chezmoi directory", name)
		}
		result = append(result, r)
	}
	return result, false, nil
}

// printRecipeResult prints per-recipe overlay output.
func printRecipeResult(w io.Writer, name string, result *overlay.Result, index, total int, multi, dryRun bool) {
	noChanges := result == nil || (len(result.Added) == 0 && len(result.Updated) == 0)

	if multi {
		if noChanges {
			fmt.Fprintf(w, "\n[%d/%d] %s (no changes)\n", index, total, name)
			return
		}
		if dryRun {
			fmt.Fprintf(w, "\n[%d/%d] %s (dry-run)\n", index, total, name)
		} else {
			fmt.Fprintf(w, "\n[%d/%d] %s\n", index, total, name)
		}
	} else {
		if noChanges {
			fmt.Fprintf(w, "Overlaying %s... (no changes)\n", name)
			return
		}
		if dryRun {
			fmt.Fprintf(w, "Overlaying %s (dry-run)...\n", name)
		} else {
			fmt.Fprintf(w, "Overlaying %s...\n", name)
		}
	}

	if result != nil {
		for _, f := range result.Added {
			fmt.Fprintf(w, "  + %s\n", f)
		}
		for _, f := range result.Updated {
			fmt.Fprintf(w, "  ~ %s\n", f)
		}
	}
}


// printSummary prints the final summary line after all overlays.
func printSummary(w io.Writer, recipeCount, added, updated, removed int) {
	parts := []string{
		fmt.Sprintf("%d added", added),
		fmt.Sprintf("%d updated", updated),
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removed))
	}
	fmt.Fprintf(w, "\nOverlaid %d recipes (%s).\n", recipeCount, strings.Join(parts, ", "))
}
