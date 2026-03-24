package overlay

import (
	"path/filepath"
	"strings"
)

// https://www.chezmoi.io/reference/source-state-attributes/
// If chezmoi adds new prefixes, update these lists.
var dirPrefixes = []string{
	"remove_",
	"external_",
	"exact_",
	"private_",
	"readonly_",
}

// Ordered longest-first so "run_once_before_" matches before "run_once_".
var fileTypePrefixes = []string{
	"run_onchange_before_",
	"run_onchange_after_",
	"run_once_before_",
	"run_once_after_",
	"run_onchange_",
	"run_once_",
	"run_before_",
	"run_after_",
	"run_",
	"create_",
	"modify_",
	"symlink_",
}

var fileAttrPrefixes = []string{
	"encrypted_",
	"private_",
	"readonly_",
	"empty_",
	"executable_",
}

var fileSuffixes = []string{
	".tmpl",
	".literal",
	".age",
	".asc",
}

// ParseDirTargetName strips chezmoi attribute prefixes from a directory name
// and returns the target name. The dot_ prefix is converted to a leading dot.
//
// Examples:
//
//	"private_dot_config"       -> ".config"
//	"exact_private_dot_config" -> ".config"
//	"readonly_dot_ssh"         -> ".ssh"
//	"dot_config"               -> ".config"
//	"nvim"                     -> "nvim"
func ParseDirTargetName(name string) string {
	for _, prefix := range dirPrefixes {
		name = strings.TrimPrefix(name, prefix)
	}
	return resolveLiteralOrDot(name)
}

// ParseFileTargetName strips chezmoi attribute prefixes and suffixes from a
// file name and returns the target name.
//
// Examples:
//
//	"dot_bashrc"                    -> ".bashrc"
//	"private_dot_bashrc"            -> ".bashrc"
//	"executable_dot_tool"           -> ".tool"
//	"run_once_install-gh.sh"        -> "install-gh.sh"
//	"encrypted_private_dot_secret"  -> ".secret"
//	"dot_gitconfig.tmpl"            -> ".gitconfig"
func ParseFileTargetName(name string) string {
	// Strip type prefix first (mutually exclusive).
	for _, prefix := range fileTypePrefixes {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	// Strip attribute prefixes.
	for _, prefix := range fileAttrPrefixes {
		name = strings.TrimPrefix(name, prefix)
	}
	name = resolveLiteralOrDot(name)
	// Strip suffixes.
	for _, suffix := range fileSuffixes {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

// ParseTargetPath resolves a chezmoi source-relative path to its target path
// by parsing each directory component and the final filename.
// If isDir is true, the last component is parsed as a directory name.
//
// Examples:
//
//	ParseTargetPath("private_dot_config/nvim/init.lua", false)  -> ".config/nvim/init.lua"
//	ParseTargetPath("exact_dot_config/dot_gitconfig", false)    -> ".config/.gitconfig"
//	ParseTargetPath("exact_private_dot_config", true)           -> ".config"
func ParseTargetPath(sourceRelPath string, isDir bool) string {
	dir, file := filepath.Split(sourceRelPath)

	// Parse each directory component.
	var targetDirs []string
	if dir != "" {
		dir = strings.TrimSuffix(dir, string(filepath.Separator))
		for _, component := range strings.Split(dir, string(filepath.Separator)) {
			targetDirs = append(targetDirs, ParseDirTargetName(component))
		}
	}

	var targetFile string
	if isDir {
		targetFile = ParseDirTargetName(file)
	} else {
		targetFile = ParseFileTargetName(file)
	}
	if len(targetDirs) > 0 {
		return filepath.Join(filepath.Join(targetDirs...), targetFile)
	}
	return targetFile
}

// resolveLiteralOrDot handles the final name resolution after attribute
// prefixes are stripped. If "literal_" remains, it is removed and the rest
// is returned verbatim (no dot_ conversion). Otherwise, "dot_" is converted
// to a leading ".".
func resolveLiteralOrDot(name string) string {
	if strings.HasPrefix(name, "literal_") {
		return strings.TrimPrefix(name, "literal_")
	}
	if strings.HasPrefix(name, "dot_") {
		return "." + strings.TrimPrefix(name, "dot_")
	}
	return name
}
