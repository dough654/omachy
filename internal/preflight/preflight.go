package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/dough654/Omachy/internal/shell"
)

// Check represents the result of a single preflight check.
type Check struct {
	Name    string
	Passed  bool
	Detail  string
	Warning bool // true if non-fatal
}

// MinAeroSpaceVersion is the minimum AeroSpace version required for our configs.
// AeroSpace v0.20.0 introduced persistent-workspaces (used by the named-workspaces config).
const MinAeroSpaceVersion = "0.20.0"

// MinNeovimVersion is the minimum Neovim version required by Kickstart.nvim,
// which uses the built-in vim.pack plugin manager added in Neovim 0.12.
const MinNeovimVersion = "0.12.0"

// MinTmuxVersion is the minimum tmux version required by the tmux-enhance
// plugin in the bundled tmux.conf.
const MinTmuxVersion = "3.7"

// RunAll runs all preflight checks and returns results.
func RunAll() []Check {
	return []Check{
		checkArch(),
		checkMacOSVersion(),
		checkHomebrew(),
		checkXcodeCLI(),
		checkSeparateSpaces(),
		checkAeroSpaceVersion(),
		checkNeovimVersion(),
		checkTmuxVersion(),
	}
}

// AllPassed returns true if all non-warning checks passed.
func AllPassed(checks []Check) bool {
	for _, c := range checks {
		if !c.Passed && !c.Warning {
			return false
		}
	}
	return true
}

func checkArch() Check {
	if runtime.GOARCH == "arm64" {
		return Check{Name: "Architecture", Passed: true, Detail: "Apple Silicon (arm64)"}
	}
	return Check{Name: "Architecture", Passed: true, Detail: fmt.Sprintf("%s (supported)", runtime.GOARCH)}
}

func checkMacOSVersion() Check {
	result, err := shell.Run("sw_vers", "-productVersion")
	if err != nil {
		return Check{Name: "macOS version", Passed: false, Detail: "Could not determine macOS version"}
	}

	version := strings.TrimSpace(result.Stdout)
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return Check{Name: "macOS version", Passed: false, Detail: "Could not parse macOS version"}
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Check{Name: "macOS version", Passed: false, Detail: fmt.Sprintf("Could not parse version: %s", version)}
	}

	if major < 13 {
		return Check{Name: "macOS version", Passed: false, Detail: fmt.Sprintf("%s (requires >= 13.0 Ventura)", version)}
	}

	name := macOSName(major)
	return Check{Name: "macOS version", Passed: true, Detail: fmt.Sprintf("%s (%s)", version, name)}
}

func macOSName(major int) string {
	names := map[int]string{
		13: "Ventura",
		14: "Sonoma",
		15: "Sequoia",
		26: "Tahoe", // Apple jumped from 15 to 26 (year-based versioning)
	}
	if name, ok := names[major]; ok {
		return name
	}
	return "macOS"
}

func checkHomebrew() Check {
	path, found := shell.Which("brew")
	if !found {
		return Check{
			Name:   "Homebrew",
			Passed: false,
			Detail: "Not found. Install from https://brew.sh",
		}
	}
	return Check{Name: "Homebrew", Passed: true, Detail: path}
}

func checkXcodeCLI() Check {
	cmd := exec.Command("xcode-select", "-p")
	output, err := cmd.Output()
	if err != nil {
		return Check{
			Name:   "Xcode CLI tools",
			Passed: false,
			Detail: "Not installed. Run: xcode-select --install",
		}
	}
	return Check{Name: "Xcode CLI tools", Passed: true, Detail: strings.TrimSpace(string(output)) + " (ensure up to date via Software Update)"}
}

func checkAeroSpaceVersion() Check {
	_, found := shell.Which("aerospace")
	if !found {
		// Not installed yet — that's fine, Homebrew will install it.
		return Check{Name: "AeroSpace version", Passed: true, Detail: "Not yet installed (will be installed)"}
	}

	result, err := shell.Run("aerospace", "--version")
	if err != nil {
		return Check{
			Name:    "AeroSpace version",
			Passed:  false,
			Warning: false,
			Detail:  "Could not determine version. Ensure AeroSpace >= " + MinAeroSpaceVersion,
		}
	}

	// Output format: "aerospace CLI client version: 0.20.3-Beta <hash>\n..."
	version := parseAeroSpaceVersion(result.Stdout)
	if version == "" {
		return Check{
			Name:    "AeroSpace version",
			Passed:  false,
			Warning: true,
			Detail:  "Could not parse version. Ensure AeroSpace >= " + MinAeroSpaceVersion,
		}
	}

	if !versionAtLeast(version, MinAeroSpaceVersion) {
		return Check{
			Name:    "AeroSpace version",
			Passed:  false,
			Warning: false,
			Detail:  fmt.Sprintf("%s is too old (requires >= %s). Run: brew upgrade --cask nikitabobko/tap/aerospace", version, MinAeroSpaceVersion),
		}
	}

	return Check{Name: "AeroSpace version", Passed: true, Detail: version}
}

// parseAeroSpaceVersion extracts the version string from aerospace --version output.
func parseAeroSpaceVersion(output string) string {
	// First line: "aerospace CLI client version: 0.20.3-Beta <hash>"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "CLI client version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			fields := strings.Fields(strings.TrimSpace(parts[1]))
			if len(fields) >= 1 {
				return fields[0]
			}
		}
	}
	return ""
}

// checkNeovimVersion warns when an existing Neovim is too old for Kickstart.nvim.
// Non-fatal: already-installed packages are not upgraded. Relevant when
// Kickstart will be cloned (no ~/.config/nvim yet, see runConfigs) or
// ~/.config/nvim already is a Kickstart clone.
func checkNeovimVersion() Check {
	_, found := shell.Which("nvim")
	if !found {
		return Check{Name: "Neovim version", Passed: true, Detail: "Not yet installed (will be installed)"}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Check{Name: "Neovim version", Passed: false, Warning: true, Detail: "Could not determine home directory"}
	}
	nvimDir := filepath.Join(home, ".config", "nvim")
	customConfig := false
	if _, err := os.Stat(nvimDir); err == nil {
		gitConfig, _ := os.ReadFile(filepath.Join(nvimDir, ".git", "config"))
		customConfig = !isKickstartGitConfig(string(gitConfig))
	}

	version := ""
	if result, err := shell.Run("nvim", "--version"); err == nil {
		version = parseNeovimVersion(result.Stdout)
	}
	return neovimVersionCheck(version, customConfig)
}

// isKickstartGitConfig reports whether a .git/config has a kickstart.nvim
// remote (Omachy's clone, or a fork that keeps the name).
func isKickstartGitConfig(gitConfig string) bool {
	for _, line := range strings.Split(gitConfig, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "url") && strings.Contains(line, "kickstart.nvim") {
			return true
		}
	}
	return false
}

// neovimVersionCheck decides the Neovim check result. A custom (non-Kickstart)
// config is kept as-is, so Kickstart's version floor doesn't apply.
func neovimVersionCheck(version string, customConfig bool) Check {
	if customConfig {
		detail := "custom config, Kickstart.nvim not used"
		if version != "" {
			detail = version + " (" + detail + ")"
		}
		return Check{Name: "Neovim version", Passed: true, Detail: detail}
	}

	if version == "" {
		return Check{
			Name:    "Neovim version",
			Passed:  false,
			Warning: true,
			Detail:  "Could not determine version. Kickstart.nvim requires Neovim >= " + MinNeovimVersion,
		}
	}

	if !versionAtLeast(version, MinNeovimVersion) {
		return Check{
			Name:    "Neovim version",
			Passed:  false,
			Warning: true,
			Detail:  fmt.Sprintf("%s is too old for Kickstart.nvim (requires >= %s). Run: brew upgrade neovim", version, MinNeovimVersion),
		}
	}

	return Check{Name: "Neovim version", Passed: true, Detail: version}
}

// parseNeovimVersion extracts the version string from nvim --version output.
func parseNeovimVersion(output string) string {
	// First line: "NVIM v0.12.6" (dev builds: "NVIM v0.13.0-dev-123+gabc123")
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "NVIM ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}
	return ""
}

// checkTmuxVersion warns when an existing tmux is too old for tmux-enhance.
// Non-fatal: already-installed packages are not upgraded. Relevant when the
// bundled tmux.conf will be deployed (no ~/.tmux.conf yet; it's NeverOverwrite)
// or the existing ~/.tmux.conf already loads tmux-enhance.
func checkTmuxVersion() Check {
	_, found := shell.Which("tmux")
	if !found {
		return Check{Name: "tmux version", Passed: true, Detail: "Not yet installed (will be installed)"}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Check{Name: "tmux version", Passed: false, Warning: true, Detail: "Could not determine home directory"}
	}
	customConfig := false
	if conf, err := os.ReadFile(filepath.Join(home, ".tmux.conf")); err == nil {
		customConfig = !usesTmuxEnhance(string(conf))
	}

	version := ""
	if result, err := shell.Run("tmux", "-V"); err == nil {
		version = parseTmuxVersion(result.Stdout)
	}
	return tmuxVersionCheck(version, customConfig)
}

// usesTmuxEnhance reports whether a tmux.conf declares the tmux-enhance plugin
// on an uncommented line.
func usesTmuxEnhance(conf string) bool {
	for _, line := range strings.Split(conf, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "@plugin") && strings.Contains(line, "tmux-enhance") {
			return true
		}
	}
	return false
}

// tmuxVersionCheck decides the tmux check result. A custom ~/.tmux.conf
// without tmux-enhance is never replaced, so the floor doesn't apply.
func tmuxVersionCheck(version string, customConfig bool) Check {
	if customConfig {
		detail := "custom ~/.tmux.conf, tmux-enhance not used"
		if version != "" {
			detail = version + " (" + detail + ")"
		}
		return Check{Name: "tmux version", Passed: true, Detail: detail}
	}

	if version == "" {
		return Check{
			Name:    "tmux version",
			Passed:  false,
			Warning: true,
			Detail:  "Could not determine version. tmux-enhance requires tmux >= " + MinTmuxVersion,
		}
	}

	if !versionAtLeast(version, MinTmuxVersion) {
		return Check{
			Name:    "tmux version",
			Passed:  false,
			Warning: true,
			Detail:  fmt.Sprintf("%s is too old for tmux-enhance (requires >= %s). Run: brew upgrade tmux", version, MinTmuxVersion),
		}
	}

	return Check{Name: "tmux version", Passed: true, Detail: version}
}

// parseTmuxVersion extracts a numeric version from tmux -V output.
// Handles "tmux 3.8", "tmux 3.6a" (letter suffix), and "tmux next-3.7".
func parseTmuxVersion(output string) string {
	fields := strings.Fields(output)
	if len(fields) < 2 || fields[0] != "tmux" {
		return ""
	}
	v := strings.TrimPrefix(fields[1], "next-")
	end := 0
	for end < len(v) && (v[end] == '.' || (v[end] >= '0' && v[end] <= '9')) {
		end++
	}
	return strings.TrimSuffix(v[:end], ".")
}

// versionAtLeast returns true if version >= minimum, comparing numeric segments.
// Strips pre-release suffixes (e.g. "0.20.3-Beta" → [0, 20, 3]).
func versionAtLeast(version, minimum string) bool {
	parse := func(v string) []int {
		// Strip leading 'v' and any pre-release suffix after '-'
		v = strings.TrimPrefix(v, "v")
		if idx := strings.Index(v, "-"); idx != -1 {
			v = v[:idx]
		}
		var nums []int
		for _, seg := range strings.Split(v, ".") {
			n, err := strconv.Atoi(seg)
			if err != nil {
				return nil
			}
			nums = append(nums, n)
		}
		return nums
	}

	ver := parse(version)
	min := parse(minimum)
	if ver == nil || min == nil {
		return false
	}

	for i := 0; i < len(min); i++ {
		if i >= len(ver) {
			return false
		}
		if ver[i] > min[i] {
			return true
		}
		if ver[i] < min[i] {
			return false
		}
	}
	return true
}

func checkSeparateSpaces() Check {
	result, err := shell.Run("defaults", "read", "com.apple.spaces", "spans-displays")
	if err != nil {
		// This key may not exist, which is fine (default is separate spaces)
		return Check{Name: "Separate Spaces", Passed: true, Detail: "Enabled (default)", Warning: false}
	}

	val := strings.TrimSpace(result.Stdout)
	if val == "1" {
		return Check{
			Name:    "Separate Spaces",
			Passed:  false,
			Warning: true,
			Detail:  "\"Displays have separate Spaces\" may be disabled. Enable in System Settings → Desktop & Dock.",
		}
	}

	return Check{Name: "Separate Spaces", Passed: true, Detail: "Enabled"}
}
