package preflight

import "testing"

func TestAllPassedAllGreen(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: true},
		{Name: "C", Passed: true},
	}
	if !AllPassed(checks) {
		t.Error("all checks passed, AllPassed should return true")
	}
}

func TestAllPassedWithWarning(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: false, Warning: true},
		{Name: "C", Passed: true},
	}
	if !AllPassed(checks) {
		t.Error("warning-only failure should not cause AllPassed to return false")
	}
}

func TestAllPassedWithFailure(t *testing.T) {
	checks := []Check{
		{Name: "A", Passed: true},
		{Name: "B", Passed: false, Warning: false},
		{Name: "C", Passed: true},
	}
	if AllPassed(checks) {
		t.Error("non-warning failure should cause AllPassed to return false")
	}
}

func TestParseAeroSpaceVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"aerospace CLI client version: 0.20.3-Beta 6dde91ba\nAeroSpace.app server version: 0.20.3-Beta 6dde91ba\n",
			"0.20.3-Beta",
		},
		{
			"aerospace CLI client version: 0.15.0 abc123\n",
			"0.15.0",
		},
		{"", ""},
		{"some random output\n", ""},
	}
	for _, tt := range tests {
		got := parseAeroSpaceVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseAeroSpaceVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		minimum string
		want    bool
	}{
		{"0.20.3-Beta", "0.15.0", true},
		{"0.15.0", "0.15.0", true},
		{"0.14.9", "0.15.0", false},
		{"0.12.0", "0.15.0", false},
		{"1.0.0", "0.15.0", true},
		{"0.15.1", "0.15.0", true},
		{"v0.20.0", "0.15.0", true},
		{"0.19.2", "0.20.0", false},
		{"0.21.3-Beta", "0.20.0", true},
		{"v0.11.4", "0.12.0", false},
		{"v0.12.6", "0.12.0", true},
		{"v0.13.0-dev-123+gabc123", "0.12.0", true},
	}
	for _, tt := range tests {
		got := versionAtLeast(tt.version, tt.minimum)
		if got != tt.want {
			t.Errorf("versionAtLeast(%q, %q) = %v, want %v", tt.version, tt.minimum, got, tt.want)
		}
	}
}

func TestParseNeovimVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"NVIM v0.12.6\nBuild type: Release\nLuaJIT 2.1.1741730670\n", "v0.12.6"},
		{"NVIM v0.13.0-dev-123+gabc123\nBuild type: RelWithDebInfo\n", "v0.13.0-dev-123+gabc123"},
		{"", ""},
		{"some random output\n", ""},
	}
	for _, tt := range tests {
		got := parseNeovimVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseNeovimVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNeovimVersionCheck(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		custom      bool
		wantPassed  bool
		wantWarning bool
		wantDetail  string
	}{
		{"old nvim, custom config", "v0.11.6", true, true, false, "v0.11.6 (custom config, Kickstart.nvim not used)"},
		{"unknown version, custom config", "", true, true, false, "custom config, Kickstart.nvim not used"},
		{"old nvim, no config", "v0.11.6", false, false, true, "v0.11.6 is too old for Kickstart.nvim (requires >= 0.12.0). Run: brew upgrade neovim"},
		{"new nvim, no config", "v0.12.6", false, true, false, "v0.12.6"},
		{"unknown version, no config", "", false, false, true, "Could not determine version. Kickstart.nvim requires Neovim >= 0.12.0"},
	}
	for _, tt := range tests {
		got := neovimVersionCheck(tt.version, tt.custom)
		if got.Passed != tt.wantPassed || got.Warning != tt.wantWarning || got.Detail != tt.wantDetail {
			t.Errorf("%s: got {Passed:%v Warning:%v Detail:%q}, want {Passed:%v Warning:%v Detail:%q}",
				tt.name, got.Passed, got.Warning, got.Detail, tt.wantPassed, tt.wantWarning, tt.wantDetail)
		}
	}
}

func TestIsKickstartGitConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"omachy clone", "[remote \"origin\"]\n\turl = https://github.com/nvim-lua/kickstart.nvim\n", true},
		{"ssh fork", "[remote \"origin\"]\n\turl = git@github.com:me/kickstart.nvim.git\n", true},
		{"own dotfiles", "[remote \"origin\"]\n\turl = https://github.com/me/nvim-config\n", false},
		{"branch named kickstart.nvim", "[branch \"kickstart.nvim\"]\n\tremote = origin\n", false},
		{"not a git repo", "", false},
	}
	for _, tt := range tests {
		if got := isKickstartGitConfig(tt.input); got != tt.want {
			t.Errorf("%s: isKickstartGitConfig = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestUsesTmuxEnhance(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"omachy config", "set -g @plugin 'tmux-plugins/tpm'\nset -g @plugin 'dough654/tmux-enhance#v0.1.5'\n", true},
		{"unpinned", "  set -g @plugin \"dough654/tmux-enhance\"\n", true},
		{"commented out", "# set -g @plugin 'dough654/tmux-enhance'\n", false},
		{"other plugins only", "set -g @plugin 'tmux-plugins/tpm'\nset -g @plugin 'dracula/tmux'\n", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		if got := usesTmuxEnhance(tt.input); got != tt.want {
			t.Errorf("%s: usesTmuxEnhance = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseTmuxVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tmux 3.8\n", "3.8"},
		{"tmux 3.6a\n", "3.6"},
		{"tmux next-3.7\n", "3.7"},
		{"tmux 3.10\n", "3.10"},
		{"", ""},
		{"some random output\n", ""},
		{"tmux master\n", ""},
	}
	for _, tt := range tests {
		got := parseTmuxVersion(tt.input)
		if got != tt.want {
			t.Errorf("parseTmuxVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTmuxVersionCheck(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		custom      bool
		wantPassed  bool
		wantWarning bool
		wantDetail  string
	}{
		{"old tmux, custom config", "3.6", true, true, false, "3.6 (custom ~/.tmux.conf, tmux-enhance not used)"},
		{"unknown version, custom config", "", true, true, false, "custom ~/.tmux.conf, tmux-enhance not used"},
		{"old tmux, no config", "3.6", false, false, true, "3.6 is too old for tmux-enhance (requires >= 3.7). Run: brew upgrade tmux"},
		{"new tmux, no config", "3.8", false, true, false, "3.8"},
		{"minimum tmux, no config", "3.7", false, true, false, "3.7"},
		{"unknown version, no config", "", false, false, true, "Could not determine version. tmux-enhance requires tmux >= 3.7"},
	}
	for _, tt := range tests {
		got := tmuxVersionCheck(tt.version, tt.custom)
		if got.Passed != tt.wantPassed || got.Warning != tt.wantWarning || got.Detail != tt.wantDetail {
			t.Errorf("%s: got {Passed:%v Warning:%v Detail:%q}, want {Passed:%v Warning:%v Detail:%q}",
				tt.name, got.Passed, got.Warning, got.Detail, tt.wantPassed, tt.wantWarning, tt.wantDetail)
		}
	}
}

func TestMacOSName(t *testing.T) {
	tests := []struct {
		major int
		want  string
	}{
		{13, "Ventura"},
		{14, "Sonoma"},
		{15, "Sequoia"},
		{26, "Tahoe"},
		{16, "macOS"},
		{99, "macOS"},
	}
	for _, tt := range tests {
		got := macOSName(tt.major)
		if got != tt.want {
			t.Errorf("macOSName(%d) = %q, want %q", tt.major, got, tt.want)
		}
	}
}
