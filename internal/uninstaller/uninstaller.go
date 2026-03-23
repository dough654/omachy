package uninstaller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dough654/Omachy/internal/brew"
	"github.com/dough654/Omachy/internal/installer"
	"github.com/dough654/Omachy/internal/shell"
	"github.com/dough654/Omachy/internal/tui"
)

// PhaseNames returns the uninstall phase names.
func PhaseNames() []string {
	return []string{
		"Services",
		"Configs",
		"Packages",
		"Defaults",
	}
}

// Options holds uninstall configuration.
type Options struct {
	DryRun       bool
	KeepConfigs  bool
	KeepPackages bool
}

// Run executes the full uninstall flow.
func Run(p *tea.Program, opts Options) {
	// Check if Omachy is actually installed
	state, err := installer.LoadState()
	if err == nil && len(state.InstalledPackages) == 0 &&
		len(state.DeployedConfigs) == 0 &&
		len(state.OriginalDefaults) == 0 &&
		len(state.Services) == 0 &&
		state.BackupPath == "" {
		p.Send(tui.LogLine{Text: "Omachy is not installed — nothing to uninstall."})
		p.Send(tui.InstallFinished{})
		return
	}

	phases := []struct {
		name string
		fn   func(p *tea.Program, opts Options) error
	}{
		{"Services", stopServices},
		{"Configs", removeConfigs},
		{"Packages", removePackages},
		{"Defaults", restoreDefaults},
	}

	for i, phase := range phases {
		p.Send(tui.PhaseStarted{Name: phase.name})

		err := phase.fn(p, opts)
		if err != nil {
			p.Send(tui.PhaseFailed{Name: phase.name, Error: err})
			p.Send(tui.InstallFinished{Err: fmt.Errorf("phase %q failed: %w", phase.name, err)})
			return
		}

		p.Send(tui.PhaseCompleted{Name: phase.name})
		pct := ((i + 1) * 100) / len(phases)
		p.Send(tui.ProgressUpdate{Percent: pct})
	}

	p.Send(tui.InstallFinished{})
}

func stopServices(p *tea.Program, opts Options) error {
	log := func(text string) { p.Send(tui.LogLine{Text: text}) }

	state, err := installer.LoadState()
	if err != nil {
		return err
	}

	for _, svc := range state.Services {
		if opts.DryRun {
			log(fmt.Sprintf("==> Would stop service: %s", svc))
			continue
		}
		if err := brew.StopService(svc, log); err != nil {
			log(fmt.Sprintf("    Warning: %v", err))
		}
	}

	// Kill processes that Omachy started directly (not managed as brew services).
	// AeroSpace launches sketchybar and borders via after-startup-command.
	// Only kill processes that were not already running before install.
	preExisting := make(map[string]bool, len(state.RunningProcesses))
	for _, p := range state.RunningProcesses {
		preExisting[p] = true
	}
	for _, proc := range []string{"AeroSpace", "sketchybar", "borders"} {
		if preExisting[proc] {
			log(fmt.Sprintf("    Skipping %s (was running before install)", proc))
			continue
		}
		if opts.DryRun {
			log(fmt.Sprintf("==> Would kill process: %s", proc))
			continue
		}
		log(fmt.Sprintf("==> Killing %s", proc))
		if _, err := shell.Run("pkill", "-x", proc); err != nil {
			log(fmt.Sprintf("    %s was not running", proc))
		}
	}

	return nil
}

func removeConfigs(p *tea.Program, opts Options) error {
	log := func(text string) { p.Send(tui.LogLine{Text: text}) }

	if opts.KeepConfigs {
		log("==> Keeping configs (--keep-configs)")
		return nil
	}

	state, err := installer.LoadState()
	if err != nil {
		return err
	}

	for dest := range state.DeployedConfigs {
		if opts.DryRun {
			log(fmt.Sprintf("==> Would remove %s", shortPath(dest)))
			continue
		}

		log(fmt.Sprintf("==> Removing %s", shortPath(dest)))
		os.RemoveAll(dest)
	}

	// Restore from backup if available
	if state.BackupPath != "" {
		if _, err := os.Stat(state.BackupPath); os.IsNotExist(err) {
			log(fmt.Sprintf("==> Backup directory missing: %s", state.BackupPath))
			log("    Original configs cannot be restored (backup was deleted)")
		} else {
			log(fmt.Sprintf("==> Restoring backup from %s", state.BackupPath))
			if !opts.DryRun {
				if err := restoreBackup(state.BackupPath, log); err != nil {
					log(fmt.Sprintf("    Warning: backup restore failed: %v", err))
				}
			}
		}
	}

	// Remove Omachy managed block from .zshrc
	home, _ := os.UserHomeDir()
	zshrcPath := filepath.Join(home, ".zshrc")
	if data, err := os.ReadFile(zshrcPath); err == nil {
		content := string(data)
		if strings.Contains(content, "Omachy managed") {
			if opts.DryRun {
				log("==> Would remove Omachy managed block from ~/.zshrc")
			} else {
				cleaned := removeManagedBlock(content)
				os.WriteFile(zshrcPath, []byte(cleaned), 0644)
				log("==> Removed Omachy managed block from ~/.zshrc")
			}
		}
	}

	return nil
}

const (
	zshrcMarkerStart = "# ── Omachy managed (do not edit between these markers) ──"
	zshrcMarkerEnd   = "# ── End Omachy managed ──"
)

func removeManagedBlock(content string) string {
	startIdx := strings.Index(content, zshrcMarkerStart)
	endIdx := strings.Index(content, zshrcMarkerEnd)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return content
	}
	before := content[:startIdx]
	after := content[endIdx+len(zshrcMarkerEnd):]
	if len(after) > 0 && after[0] == '\n' {
		after = after[1:]
	}
	return before + after
}

func removePackages(p *tea.Program, opts Options) error {
	log := func(text string) { p.Send(tui.LogLine{Text: text}) }

	if opts.KeepPackages {
		log("==> Keeping packages (--keep-packages)")
		return nil
	}

	state, err := installer.LoadState()
	if err != nil {
		return err
	}

	if len(state.InstalledPackages) == 0 {
		log("    No packages were installed by Omachy")
		return nil
	}

	// Remove in reverse order — only packages Omachy installed
	for i := len(state.InstalledPackages) - 1; i >= 0; i-- {
		pkg := state.InstalledPackages[i]
		if opts.DryRun {
			log(fmt.Sprintf("==> Would uninstall %s", pkg.Name))
			continue
		}
		if err := brew.Uninstall(pkg.Name, pkg.Cask, log); err != nil {
			log(fmt.Sprintf("    Warning: %v", err))
		}
	}

	// Remove taps that Omachy added
	for _, tap := range state.InstalledTaps {
		if opts.DryRun {
			log(fmt.Sprintf("==> Would untap %s", tap))
			continue
		}
		log(fmt.Sprintf("==> Untapping %s", tap))
		shell.Run("brew", "untap", tap)
	}

	return nil
}

func restoreDefaults(p *tea.Program, opts Options) error {
	log := func(text string) { p.Send(tui.LogLine{Text: text}) }

	state, err := installer.LoadState()
	if err != nil {
		return err
	}

	log("==> Restoring macOS defaults")
	for key, stored := range state.OriginalDefaults {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		domain, defKey := parts[0], parts[1]

		// Stored format is "type:value" (e.g. "-bool:1") or legacy plain value
		typ, value := parseStoredDefault(stored)

		if opts.DryRun {
			log(fmt.Sprintf("    Would restore %s %s → %s %s", domain, defKey, typ, value))
			continue
		}

		_, err := shell.Run("defaults", "write", domain, defKey, typ, value)
		if err != nil {
			log(fmt.Sprintf("    Warning: could not restore %s: %v", defKey, err))
		} else {
			log(fmt.Sprintf("    Restored %s", defKey))
		}
	}

	// Delete any defaults Omachy sets that have no saved original —
	// these didn't exist before Omachy and should be removed, not left behind.
	for _, d := range installer.MacOSDefaults {
		stateKey := fmt.Sprintf("%s:%s", d.Domain, d.Key)
		if _, hasSaved := state.OriginalDefaults[stateKey]; !hasSaved {
			if opts.DryRun {
				log(fmt.Sprintf("    Would delete %s %s (no original value)", d.Domain, d.Key))
				continue
			}
			shell.Run("defaults", "delete", d.Domain, d.Key)
			log(fmt.Sprintf("    Deleted %s (no original value)", d.Key))
		}
	}

	if !opts.DryRun {
		log("==> Restarting Dock")
		shell.Run("killall", "Dock")

		// Clean up state file
		log("==> Cleaning up state file")
		home, _ := os.UserHomeDir()
		os.Remove(filepath.Join(home, ".omachy", "state.json"))
	}

	return nil
}

func restoreBackup(backupDir string, onLine func(string)) error {
	home, _ := os.UserHomeDir()

	return filepath.WalkDir(backupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel, _ := filepath.Rel(backupDir, path)
		dest := filepath.Join(home, rel)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		info, _ := d.Info()
		mode := info.Mode()

		onLine(fmt.Sprintf("    Restoring %s", rel))
		return os.WriteFile(dest, data, mode)
	})
}

// parseStoredDefault splits a stored default into type and value.
// Format is "type:value" (e.g. "-bool:1"). Falls back to "-string" for legacy values.
func parseStoredDefault(stored string) (typ, value string) {
	// Check for known type prefixes
	for _, prefix := range []string{"-bool:", "-int:", "-float:", "-string:"} {
		if strings.HasPrefix(stored, prefix) {
			return stored[:len(prefix)-1], stored[len(prefix):]
		}
	}
	// Legacy format: no type stored, treat as string
	return "-string", stored
}

func shortPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
