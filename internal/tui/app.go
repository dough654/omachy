package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const phasePanelWidth = 20

// InstallerFunc is the function signature for the installer goroutine.
// It receives the tea.Program to send messages back to the TUI.
type InstallerFunc func(p *tea.Program)

// App is the root Bubbletea model.
type App struct {
	header          HeaderModel
	phases          PhasesModel
	output          OutputModel
	help            HelpModel
	width           int
	height          int
	started         bool // false = showing splash, true = installer running
	finished        bool
	err             error
	installer       InstallerFunc
	splashOpts      SplashOptions
	version         string
	program         *tea.Program  // set after Run() creates the program
	showConfirm     bool          // true = showing logout confirmation dialog
	logoutRequested bool          // true = user chose to log out
	waitingForUser  bool          // true = waiting for user to press Enter
	waitDone        chan struct{} // signal to unblock the installer goroutine
}

func NewApp(phaseNames []string, installer InstallerFunc, splashOpts SplashOptions, version string) App {
	title := "Omachy Installer"
	if splashOpts.Uninstall {
		title = "Omachy Uninstaller"
	}
	return App{
		header:     NewHeaderModel(title, 80),
		phases:     NewPhasesModel(phaseNames),
		output:     NewOutputModel(56, 15),
		help:       NewHelpModel(80),
		installer:  installer,
		splashOpts: splashOpts,
		version:    version,
	}
}

func (a App) Init() tea.Cmd {
	return a.phases.spinner.Tick
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layout()
		// Don't return early — let the viewport also process the resize

	case tea.KeyMsg:
		// Confirmation dialog intercepts all keys
		if a.showConfirm {
			switch msg.String() {
			case "y", "Y":
				a.logoutRequested = true
				return a, tea.Quit
			case "n", "N", "esc", "q":
				a.showConfirm = false
				return a, tea.Quit
			}
			return a, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "enter":
			if a.waitingForUser {
				a.waitingForUser = false
				close(a.waitDone)
				return a, nil
			}
			if !a.started {
				a.started = true
				// Launch the installer goroutine now
				return a, func() tea.Msg {
					go a.installer(a.program)
					return nil
				}
			}
			if a.finished {
				if a.err == nil && !a.splashOpts.DryRun {
					a.showConfirm = true
					return a, nil
				}
				return a, tea.Quit
			}
		}

	case spinner.TickMsg:
		if a.started {
			var cmd tea.Cmd
			a.phases, cmd = a.phases.Update(msg)
			cmds = append(cmds, cmd)
		}

	case PhaseStarted:
		a.phases.SetStatus(msg.Name, StatusActive)

	case PhaseCompleted:
		a.phases.SetStatus(msg.Name, StatusDone)

	case PhaseFailed:
		a.phases.SetStatus(msg.Name, StatusFailed)

	case LogLine:
		a.output.AppendLine(msg.Text)

	case WaitForUser:
		a.output.AppendLine(msg.Prompt)
		a.output.AppendLine("    Press [Enter] to continue...")
		a.waitingForUser = true
		a.waitDone = msg.Done

	case ProgressUpdate:
		a.header.Percent = msg.Percent

	case InstallFinished:
		a.finished = true
		a.help.Finished = true
		a.err = msg.Err
		action := "Installation"
		if a.splashOpts.Uninstall {
			action = "Uninstall"
		}
		if msg.Err != nil {
			a.output.AppendLine("")
			a.output.AppendLine(lipgloss.NewStyle().Foreground(colorError).Bold(true).Render(action + " failed: " + msg.Err.Error()))
		} else {
			a.header.Percent = 100
			a.output.AppendLine("")
			a.output.AppendLine(lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(action + " complete!"))
		}
	}

	// Pass through to viewport for scroll handling
	if a.started {
		var cmd tea.Cmd
		a.output, cmd = a.output.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Splash screen before installation starts
	if !a.started {
		return renderSplash(a.width, a.height, a.splashOpts, a.version)
	}

	header := a.header.View()

	// Phase panel
	contentHeight := a.height - 4 // header + help + borders
	if contentHeight < 3 {
		contentHeight = 3
	}

	phaseContent := a.phases.View(contentHeight - 2) // panel border
	phasePanel := panelStyle.
		Width(phasePanelWidth).
		Height(contentHeight).
		Render(panelTitleStyle.Render("Phases") + "\n" + phaseContent)

	// Output panel
	outputWidth := a.width - phasePanelWidth - 5
	if outputWidth < 10 {
		outputWidth = 10
	}
	outputPanel := panelStyle.
		Width(outputWidth).
		Height(contentHeight).
		Render(panelTitleStyle.Render("Output") + "\n" + a.output.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, phasePanel, outputPanel)
	help := a.help.View()

	base := lipgloss.JoinVertical(lipgloss.Left, header, body, help)

	if a.showConfirm {
		dialog := renderConfirmDialog(
			"Log out now?",
			"Some changes (like hiding the menu bar)\nrequire a logout to take effect.",
		)
		// Center the dialog over the base view
		overlay := lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
		return overlay
	}

	return base
}

func (a *App) layout() {
	a.header.Width = a.width
	a.help.Width = a.width

	contentHeight := a.height - 4
	if contentHeight < 3 {
		contentHeight = 3
	}
	outputWidth := a.width - phasePanelWidth - 7
	if outputWidth < 10 {
		outputWidth = 10
	}
	outputHeight := contentHeight - 3
	if outputHeight < 1 {
		outputHeight = 1
	}
	a.output.SetSize(outputWidth, outputHeight)
}

// RunResult holds the outcome of a TUI run.
type RunResult struct {
	Finished        bool  // true if installer ran to completion
	Err             error // installer error (nil = success)
	LogoutRequested bool  // true if user chose to log out
}

// Run starts the Bubbletea program with the given installer function.
func Run(phaseNames []string, installer InstallerFunc, splashOpts SplashOptions, version string) (RunResult, error) {
	app := NewApp(phaseNames, installer, splashOpts, version)
	p := tea.NewProgram(&app, tea.WithAltScreen())
	app.program = p

	model, err := p.Run()
	if err != nil {
		return RunResult{}, err
	}

	switch a := model.(type) {
	case *App:
		return RunResult{Finished: a.finished, Err: a.err, LogoutRequested: a.logoutRequested}, nil
	case App:
		return RunResult{Finished: a.finished, Err: a.err, LogoutRequested: a.logoutRequested}, nil
	}
	return RunResult{}, nil
}
