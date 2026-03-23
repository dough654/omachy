package installer

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dough654/Omachy/internal/shell"
	"github.com/dough654/Omachy/internal/tui"
)

// MacOSDefault represents a defaults write operation.
type MacOSDefault struct {
	Domain string
	Key    string
	Type   string // -bool, -int, -float, -string
	Value  string
	Label  string // human-readable description
}

// MacOSDefaults is the list of system defaults that Omachy sets.
var MacOSDefaults = []MacOSDefault{
	{"com.apple.dock", "autohide", "-bool", "true", "Auto-hide Dock"},
	{"com.apple.dock", "autohide-delay", "-float", "0", "Remove Dock auto-hide delay"},
	{"com.apple.dock", "autohide-time-modifier", "-float", "0.25", "Fast Dock hide animation"},
	{"com.apple.dock", "mru-spaces", "-bool", "false", "Disable MRU Spaces reordering"},
	{"com.apple.dock", "tilesize", "-int", "48", "Set Dock icon size"},
	{"com.apple.dock", "mineffect", "-string", "scale", "Use scale minimize effect"},
	{"com.apple.dock", "show-recents", "-bool", "false", "Hide recent apps in Dock"},
	{"NSGlobalDomain", "NSAutomaticWindowAnimationsEnabled", "-bool", "false", "Disable window open/close animations"},
	{"NSGlobalDomain", "AppleShowAllExtensions", "-bool", "true", "Show all file extensions"},
	{"NSGlobalDomain", "KeyRepeat", "-int", "1", "Fastest key repeat rate"},
	{"NSGlobalDomain", "InitialKeyRepeat", "-int", "10", "Shortest key repeat delay"},
	{"-g", "ApplePressAndHoldEnabled", "-bool", "false", "Disable press-and-hold for key repeat"},
	{"NSGlobalDomain", "_HIHideMenuBar", "-bool", "true", "Auto-hide menu bar"},
	{"com.apple.WindowManager", "StandardHideWidgets", "-bool", "true", "Hide desktop widgets"},
}

func runSystem(p *tea.Program, opts Options) error {
	log := func(text string) {
		p.Send(tui.LogLine{Text: text})
	}

	state, err := LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	// Apply macOS defaults
	log("==> Applying macOS defaults")
	for _, d := range MacOSDefaults {
		if opts.DryRun {
			log(fmt.Sprintf("    Would set: %s", d.Label))
			continue
		}

		// Read current value and type first for undo
		stateKey := fmt.Sprintf("%s:%s", d.Domain, d.Key)
		if _, exists := state.OriginalDefaults[stateKey]; !exists {
			current, err := readDefault(d.Domain, d.Key)
			if err == nil {
				typ, typErr := readDefaultType(d.Domain, d.Key)
				if typErr == nil {
					state.OriginalDefaults[stateKey] = typ + ":" + current
				} else {
					state.OriginalDefaults[stateKey] = d.Type + ":" + current
				}
			}
		}

		// Write new value
		if err := writeDefault(d.Domain, d.Key, d.Type, d.Value); err != nil {
			return fmt.Errorf("defaults write %s %s: %w", d.Domain, d.Key, err)
		}
		log(fmt.Sprintf("    %s", d.Label))
	}

	// Restart Dock and SystemUIServer to apply changes
	if !opts.DryRun {
		log("==> Restarting Dock")
		shell.Run("killall", "Dock")
		log("==> Restarting SystemUIServer (menu bar)")
		shell.Run("killall", "SystemUIServer")
	}

	p.Send(tui.ProgressUpdate{Percent: 90})

	// Note: sketchybar and borders are started by AeroSpace via after-startup-command,
	// not as brew services, to avoid duplicate instances.

	// Record which managed processes are already running before we start them,
	// so uninstall only kills processes that Omachy started.
	if !opts.DryRun {
		for _, proc := range []string{"AeroSpace", "sketchybar", "borders"} {
			if isProcessRunning(proc) {
				state.RunningProcesses = appendUnique(state.RunningProcesses, proc)
			}
		}
	}

	// Start AeroSpace and ensure it has accessibility permissions
	if !opts.DryRun {
		log("==> Starting AeroSpace")
		shell.Run("open", "-a", "AeroSpace")
		time.Sleep(3 * time.Second)

		if isAerospaceRunning() {
			log("==> AeroSpace is running (Accessibility permissions granted)")
		} else {
			log("==> AeroSpace needs Accessibility permissions")
			log("    1. A dialog should have appeared — click 'Open System Settings'")
			log("    2. Enable the toggle for AeroSpace in Privacy → Accessibility")

			done := make(chan struct{})
			p.Send(tui.WaitForUser{
				Prompt: "    When you've granted permissions, confirm below.",
				Done:   done,
			})
			<-done

			log("==> Relaunching AeroSpace...")
			shell.Run("open", "-a", "AeroSpace")
			time.Sleep(3 * time.Second)

			if isAerospaceRunning() {
				log("==> AeroSpace is running!")
			} else {
				log("==> AeroSpace still not running — you may need to open it manually")
			}
		}
	} else {
		log("==> Would start AeroSpace and check Accessibility permissions")
	}

	// Save state
	if !opts.DryRun {
		if err := SaveState(state); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
	}

	return nil
}

func readDefault(domain, key string) (string, error) {
	result, err := shell.Run("defaults", "read", domain, key)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func readDefaultType(domain, key string) (string, error) {
	result, err := shell.Run("defaults", "read-type", domain, key)
	if err != nil {
		return "", err
	}
	// Output is like "Type is boolean", "Type is integer", "Type is float", "Type is string"
	out := strings.TrimSpace(result.Stdout)
	switch {
	case strings.Contains(out, "boolean"):
		return "-bool", nil
	case strings.Contains(out, "integer"):
		return "-int", nil
	case strings.Contains(out, "float"):
		return "-float", nil
	default:
		return "-string", nil
	}
}

func writeDefault(domain, key, typ, value string) error {
	_, err := shell.Run("defaults", "write", domain, key, typ, value)
	return err
}

func isProcessRunning(name string) bool {
	result, err := shell.Run("pgrep", "-x", name)
	return err == nil && strings.TrimSpace(result.Stdout) != ""
}

func isAerospaceRunning() bool {
	return isProcessRunning("AeroSpace")
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
