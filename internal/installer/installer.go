package installer

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dough654/Omachy/internal/tui"
)

// Options holds install configuration from CLI flags.
type Options struct {
	DryRun          bool
	Force           bool
	Verbose         bool
	SkipBackup      bool
	NamedWorkspaces bool
}

// PhaseNames returns the ordered list of installation phase names.
func PhaseNames() []string {
	return []string{
		"Preflight",
		"Backup",
		"Packages",
		"Configs",
		"System",
	}
}

// Run executes the full installation flow, sending events to the TUI.
func Run(p *tea.Program, opts Options) {
	phases := []struct {
		name string
		fn   func(p *tea.Program, opts Options) error
	}{
		{"Preflight", runPreflight},
		{"Backup", runBackup},
		{"Packages", runPackages},
		{"Configs", runConfigs},
		{"System", runSystem},
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
