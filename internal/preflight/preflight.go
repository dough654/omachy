package preflight

import (
	"fmt"
	"os/exec"
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
// AeroSpace v0.15.0 introduced config-version and persistent-workspaces.
const MinAeroSpaceVersion = "0.15.0"

// RunAll runs all preflight checks and returns results.
func RunAll() []Check {
	return []Check{
		checkArch(),
		checkMacOSVersion(),
		checkHomebrew(),
		checkXcodeCLI(),
		checkSeparateSpaces(),
		checkAeroSpaceVersion(),
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
		16: "Tahoe",
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
