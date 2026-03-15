package installer

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dough654/Omachy/internal/brew"
	"github.com/dough654/Omachy/internal/manifest"
	"github.com/dough654/Omachy/internal/tui"
)

func runPackages(p *tea.Program, opts Options) error {
	log := func(text string) {
		p.Send(tui.LogLine{Text: text})
	}

	state, err := LoadState()
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	// Add taps — track which ones Omachy added
	for _, tap := range manifest.Taps() {
		if opts.DryRun {
			log(fmt.Sprintf("==> Would tap %s", tap))
			continue
		}
		alreadyTapped := brew.IsTapped(tap)
		if err := brew.Tap(tap, log); err != nil {
			return fmt.Errorf("tap %s: %w", tap, err)
		}
		if !alreadyTapped {
			state.InstalledTaps = appendUnique(state.InstalledTaps, tap)
		}
	}

	// Install packages — only record ones Omachy actually installed
	pkgs := manifest.Packages()
	for i, pkg := range pkgs {
		if opts.DryRun {
			log(fmt.Sprintf("==> Would install %s", pkg.Name))
			continue
		}

		alreadyInstalled := brew.IsInstalled(pkg.Name, pkg.Cask)
		if err := brew.Install(pkg.Name, pkg.Cask, log); err != nil {
			return fmt.Errorf("install %s: %w", pkg.Name, err)
		}
		if !alreadyInstalled {
			state.InstalledPackages = append(state.InstalledPackages, InstalledPackage{
				Name: pkg.Name,
				Cask: pkg.Cask,
			})
		}

		pct := 40 + ((i+1)*20)/len(pkgs) // packages phase covers 40-60%
		p.Send(tui.ProgressUpdate{Percent: pct})
	}

	if !opts.DryRun {
		if err := SaveState(state); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
	}

	return nil
}
