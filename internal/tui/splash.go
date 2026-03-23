package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SplashOptions describes what will be shown on the splash screen.
type SplashOptions struct {
	DryRun          bool
	Force           bool
	SkipBackup      bool
	KeepConfigs     bool
	KeepPackages    bool
	Uninstall       bool
	NamedWorkspaces bool
}

var (
	splashLogo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	splashSubtitle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	splashSection = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	splashItem = lipgloss.NewStyle().
			Foreground(colorMuted)

	splashFlag = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	splashPrompt = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)
)

func renderSplash(width, height int, opts SplashOptions, version string) string {
	var b strings.Builder

	// Logo
	logo := `
   ___                       _
  / _ \ _ __ ___   __ _  ___| |__  _   _
 | | | | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ __| '_ \| | | |
 | |_| | | | | | | (_| | (__| | | | |_| |
  \___/|_| |_| |_|\__,_|\___|_| |_|\__, |
                                    |___/`

	b.WriteString(splashLogo.Render(logo))
	b.WriteString("\n\n")
	b.WriteString(splashSubtitle.Render("  Tiling window manager setup for macOS"))
	b.WriteString("\n")
	b.WriteString(splashItem.Render(fmt.Sprintf("  Version %s", version)))
	b.WriteString("\n\n")

	bullet := lipgloss.NewStyle().Foreground(colorPrimary).Render("*")

	if opts.Uninstall {
		renderUninstallSplash(&b, opts, bullet)
	} else {
		renderInstallSplash(&b, opts, bullet)
	}

	b.WriteString("\n")

	return b.String()
}

func renderInstallSplash(b *strings.Builder, opts SplashOptions, bullet string) {
	b.WriteString(splashSection.Render("  This will install and configure:"))
	b.WriteString("\n")
	tools := []struct{ name, desc string }{
		{"AeroSpace", "tiling window manager"},
		{"SketchyBar", "custom menu bar"},
		{"JankyBorders", "window border highlights"},
		{"Ghostty", "terminal emulator"},
		{"Neovim + Kickstart", "text editor"},
		{"Tmux + TPM", "terminal multiplexer"},
		{"Starship", "cross-shell prompt"},
		{"fzf", "fuzzy finder"},
		{"Lazygit", "git TUI"},
		{"opencode", "AI coding agent CLI"},
		{"Lazydocker", "docker TUI"},
		{"Atuin", "shell history search"},
		{"Nerd Fonts", "Hack + JetBrains Mono"},
		{"Node, Python, Go", "language runtimes"},
	}
	for _, t := range tools {
		b.WriteString(fmt.Sprintf("    %s %s  %s\n",
			bullet,
			lipgloss.NewStyle().Foreground(colorText).Render(t.name),
			splashItem.Render(t.desc),
		))
	}

	b.WriteString("\n")
	b.WriteString(splashSection.Render("  Additionally:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("    %s Existing configs will be backed up\n", bullet))
	b.WriteString(fmt.Sprintf("    %s macOS system defaults will be adjusted (dock, animations, key repeat)\n", bullet))

	var flags []string
	if opts.DryRun {
		flags = append(flags, "--dry-run (no changes will be made)")
	}
	if opts.Force {
		flags = append(flags, "--force (overwrite existing configs)")
	}
	if opts.SkipBackup {
		flags = append(flags, "--skip-backup (no backup of existing configs)")
	}
	if opts.NamedWorkspaces {
		flags = append(flags, "--named-workspaces (Dev, Web, Messaging, Email, Scratch)")
	}
	renderFlags(b, flags)

	b.WriteString("\n")
	b.WriteString(splashPrompt.Render("  Press [Enter] to begin installation, [q] to quit."))
	b.WriteString("\n")
}

func renderUninstallSplash(b *strings.Builder, opts SplashOptions, bullet string) {
	b.WriteString(splashSection.Render("  This will remove Omachy and restore your system:"))
	b.WriteString("\n\n")
	steps := []string{
		"Stop running brew services (SketchyBar, JankyBorders)",
		"Remove deployed config files",
		"Uninstall packages that Omachy installed",
		"Restore original macOS system defaults",
		"Restore config backups (if available)",
	}
	for _, s := range steps {
		b.WriteString(fmt.Sprintf("    %s %s\n", bullet, lipgloss.NewStyle().Foreground(colorText).Render(s)))
	}

	b.WriteString("\n")
	b.WriteString(splashSection.Render("  Your system will be returned to its pre-Omachy state."))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("    %s Packages you had before Omachy will not be removed\n", bullet))
	b.WriteString(fmt.Sprintf("    %s Original settings will be restored from saved state\n", bullet))

	var flags []string
	if opts.DryRun {
		flags = append(flags, "--dry-run (no changes will be made)")
	}
	if opts.KeepConfigs {
		flags = append(flags, "--keep-configs (config files will not be removed)")
	}
	if opts.KeepPackages {
		flags = append(flags, "--keep-packages (packages will not be uninstalled)")
	}
	renderFlags(b, flags)

	b.WriteString("\n")
	b.WriteString(splashPrompt.Render("  Press [Enter] to begin uninstall, [q] to quit."))
	b.WriteString("\n")
}

func renderFlags(b *strings.Builder, flags []string) {
	if len(flags) > 0 {
		b.WriteString("\n")
		b.WriteString(splashSection.Render("  Active flags:"))
		b.WriteString("\n")
		for _, f := range flags {
			b.WriteString(fmt.Sprintf("    %s\n", splashFlag.Render(f)))
		}
	}
}
