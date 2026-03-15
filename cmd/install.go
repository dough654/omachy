package cmd

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dough654/Omachy/internal/installer"
	"github.com/dough654/Omachy/internal/tui"
	"github.com/spf13/cobra"
)

var (
	flagDryRun          bool
	flagForce           bool
	flagVerbose         bool
	flagSkipBackup      bool
	flagNamedWorkspaces bool
)

func init() {
	installCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be done without making changes")
	installCmd.Flags().BoolVar(&flagForce, "force", false, "Overwrite existing configs without prompting")
	installCmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Show detailed output")
	installCmd.Flags().BoolVar(&flagSkipBackup, "skip-backup", false, "Skip backing up existing configs")
	installCmd.Flags().BoolVar(&flagNamedWorkspaces, "named-workspaces", false, "Use named workspaces (D/W/M/E/S) with app-to-workspace rules instead of numbered 1-9")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the full Omachy desktop environment",
	Long:  "Runs preflight checks, backs up existing configs, installs packages, deploys configs, and configures macOS system defaults.",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := installer.Options{
			DryRun:          flagDryRun,
			Force:           flagForce,
			Verbose:         flagVerbose,
			SkipBackup:      flagSkipBackup,
			NamedWorkspaces: flagNamedWorkspaces,
		}

		splashOpts := tui.SplashOptions{
			DryRun:          flagDryRun,
			Force:           flagForce,
			SkipBackup:      flagSkipBackup,
			NamedWorkspaces: flagNamedWorkspaces,
		}

		result, err := tui.Run(installer.PhaseNames(), func(p *tea.Program) {
			installer.Run(p, opts)
		}, splashOpts, Version)
		if err != nil {
			return err
		}

		if result.LogoutRequested {
			fmt.Println("Logging out...")
			exec.Command("osascript", "-e", `tell application "loginwindow" to «event aevtrlgo»`).Run()
		}

		return result.Err
	},
}
