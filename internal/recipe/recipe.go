// Package recipe implements recipe discovery and loading. A recipe is a
// directory containing a README.md; an optional chezmoi/ subdirectory holds
// the chezmoi source state fragment to be overlaid.
package recipe

// Recipe represents a discovered recipe.
type Recipe struct {
	// Name is the recipe's directory name.
	Name string

	// Dir is the absolute path to the recipe directory on disk.
	Dir string

	// HasChezmoi indicates whether the recipe has a chezmoi/ subdirectory.
	HasChezmoi bool

	// EmptyChezmoi indicates the chezmoi/ directory exists but contains no files.
	EmptyChezmoi bool
}
