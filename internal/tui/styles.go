package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // purple
	colorSecondary = lipgloss.Color("#A78BFA") // light purple
	colorSuccess   = lipgloss.Color("#10B981") // green
	colorError     = lipgloss.Color("#EF4444") // red
	colorWarning   = lipgloss.Color("#F59E0B") // amber
	colorMuted     = lipgloss.Color("#6B7280") // gray
	colorText      = lipgloss.Color("#E5E7EB") // light gray
	colorBg        = lipgloss.Color("#1F2937") // dark bg

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	progressStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary).
			PaddingLeft(1)

	phaseIconPending = lipgloss.NewStyle().Foreground(colorMuted).Render("○")
	phaseIconActive  = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("▸")
	phaseIconDone    = lipgloss.NewStyle().Foreground(colorSuccess).Render("✓")
	phaseIconFailed  = lipgloss.NewStyle().Foreground(colorError).Render("✗")

	phaseNameActive = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	phaseNameDone = lipgloss.NewStyle().
			Foreground(colorSuccess)

	phaseNameFailed = lipgloss.NewStyle().
			Foreground(colorError)

	phaseNamePending = lipgloss.NewStyle().
				Foreground(colorMuted)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	logPrefixStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)
)
