package overlay

import "testing"

func TestParseDirTargetName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"plain", "nvim", "nvim"},
		{"dot only", "dot_config", ".config"},
		{"private dot", "private_dot_config", ".config"},
		{"exact dot", "exact_dot_config", ".config"},
		{"exact private dot", "exact_private_dot_config", ".config"},
		{"readonly dot", "readonly_dot_ssh", ".ssh"},
		{"private readonly dot", "private_readonly_dot_local", ".local"},
		{"remove dot", "remove_dot_old", ".old"},
		{"external dot", "external_dot_fonts", ".fonts"},
		{"all prefixes", "remove_external_exact_private_readonly_dot_all", ".all"},
		{"no dot", "exact_private_data", "data"},
		{"literal", "literal_dot_file", "dot_file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDirTargetName(tt.input)
			if got != tt.expect {
				t.Errorf("ParseDirTargetName(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseFileTargetName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"plain", "init.lua", "init.lua"},
		{"dot only", "dot_bashrc", ".bashrc"},
		{"private dot", "private_dot_bashrc", ".bashrc"},
		{"executable dot", "executable_dot_tool", ".tool"},
		{"encrypted private", "encrypted_private_dot_secret", ".secret"},
		{"readonly", "readonly_dot_profile", ".profile"},
		{"empty dot", "empty_dot_placeholder", ".placeholder"},
		{"tmpl suffix", "dot_gitconfig.tmpl", ".gitconfig"},
		{"literal suffix", "dot_file.literal", ".file"},
		{"age suffix", "encrypted_dot_secret.age", ".secret"},
		{"run once", "run_once_install-gh.sh", "install-gh.sh"},
		{"run onchange", "run_onchange_reload.sh", "reload.sh"},
		{"run once before", "run_once_before_setup.sh", "setup.sh"},
		{"run once after", "run_once_after_cleanup.sh", "cleanup.sh"},
		{"run onchange before", "run_onchange_before_rebuild.sh", "rebuild.sh"},
		{"run onchange after", "run_onchange_after_notify.sh", "notify.sh"},
		{"run before", "run_before_check.sh", "check.sh"},
		{"run after", "run_after_done.sh", "done.sh"},
		{"create", "create_dot_env", ".env"},
		{"modify", "modify_dot_bashrc", ".bashrc"},
		{"symlink", "symlink_dot_link", ".link"},
		{"literal prefix", "literal_dot_file", "dot_file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFileTargetName(tt.input)
			if got != tt.expect {
				t.Errorf("ParseFileTargetName(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestParseTargetPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isDir  bool
		expect string
	}{
		{"file at root", "dot_bashrc", false, ".bashrc"},
		{"nested file", "private_dot_config/nvim/init.lua", false, ".config/nvim/init.lua"},
		{"attr mismatch dirs", "exact_private_dot_config/dot_gitconfig", false, ".config/.gitconfig"},
		{"deep nesting", "dot_config/nvim/lua/plugins/init.lua", false, ".config/nvim/lua/plugins/init.lua"},
		{"script in chezmoiscripts", ".chezmoiscripts/run_once_install-gh.sh", false, ".chezmoiscripts/install-gh.sh"},
		{"template file", "private_dot_config/starship.toml.tmpl", false, ".config/starship.toml"},
		{"dir at root", "private_dot_config", true, ".config"},
		{"exact dir", "exact_private_dot_config", true, ".config"},
		{"nested dir", "private_dot_config/nvim", true, ".config/nvim"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTargetPath(tt.input, tt.isDir)
			if got != tt.expect {
				t.Errorf("ParseTargetPath(%q, %v) = %q, want %q", tt.input, tt.isDir, got, tt.expect)
			}
		})
	}
}
