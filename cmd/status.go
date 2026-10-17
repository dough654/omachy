package cmd

import (
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/dough654/Omachy/internal/brew"
	"github.com/dough654/Omachy/internal/checksum"
	"github.com/dough654/Omachy/internal/installer"
	"github.com/dough654/Omachy/internal/manifest"
	"github.com/dough654/Omachy/internal/shell"
	"github.com/spf13/cobra"
)

var (
	statusTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	statusName = lipgloss.NewStyle().Width(24)

	statusDetail = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A78BFA"))
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status and detect config drift",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(statusTitle.Render("Omachy Status"))
		fmt.Println()

		state, err := installer.LoadState()
		if err != nil {
			fmt.Printf("  %s Could not load state: %v\n", iconFail, err)
			return
		}

		// Check packages
		fmt.Println(sectionStyle.Render("  Packages"))
		for _, pkg := range manifest.Packages() {
			installed := brew.IsInstalled(pkg.Name, pkg.Cask)
			// Show "sketchybar" rather than "FelixKratz/formulae/sketchybar" so
			// fully qualified tap names don't wrap the fixed-width column.
			name := path.Base(pkg.Name)
			if installed {
				fmt.Printf("    %s %s %s\n", iconPass, statusName.Render(name), statusDetail.Render("installed"))
			} else if _, onPath := shell.Which(pkg.SkipIfBinary); pkg.SkipIfBinary != "" && onPath {
				// The installer skips these when the binary already exists
				// (e.g. node via nvm), so they're present, just not via Homebrew.
				fmt.Printf("    %s %s %s\n", iconPass, statusName.Render(name), statusDetail.Render(pkg.SkipIfBinary+" on PATH (not managed by Omachy)"))
			} else {
				fmt.Printf("    %s %s %s\n", iconFail, statusName.Render(name), statusDetail.Render("not installed"))
			}
		}
		fmt.Println()

		// Check configs
		fmt.Println(sectionStyle.Render("  Configs"))
		if len(state.DeployedConfigs) == 0 {
			fmt.Printf("    %s\n", statusDetail.Render("No configs deployed yet"))
		} else {
			dests := make([]string, 0, len(state.DeployedConfigs))
			for dest := range state.DeployedConfigs {
				dests = append(dests, dest)
			}
			sort.Strings(dests)
			for _, dest := range dests {
				expectedHash := state.DeployedConfigs[dest]
				_, err := os.Stat(dest)
				if err != nil {
					fmt.Printf("    %s %s %s\n", iconFail, statusName.Render(shortPath(dest)), statusDetail.Render("missing"))
					continue
				}

				currentHash, _ := checksum.Path(dest)
				if currentHash == expectedHash {
					fmt.Printf("    %s %s %s\n", iconPass, statusName.Render(shortPath(dest)), statusDetail.Render("unchanged"))
				} else {
					fmt.Printf("    %s %s %s\n", iconWarn, statusName.Render(shortPath(dest)), statusDetail.Render("modified (drift detected)"))
				}
			}
		}
		fmt.Println()

		// Backup info
		if state.BackupPath != "" {
			fmt.Println(sectionStyle.Render("  Backup"))
			fmt.Printf("    %s %s\n", iconPass, statusDetail.Render(state.BackupPath))
			fmt.Println()
		}
	},
}

func shortPath(path string) string {
	home, _ := os.UserHomeDir()
	if len(path) > len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}
