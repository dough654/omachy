package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 3).
			Width(52)

	confirmTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	confirmTextStyle = lipgloss.NewStyle().
				Foreground(colorText)

	confirmHintStyle = lipgloss.NewStyle().
				Foreground(colorMuted)
)

func renderConfirmDialog(title, message string) string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		confirmTitleStyle.Render(title),
		confirmTextStyle.Render(message),
		confirmHintStyle.Render("[y] yes  [n] no"),
	)
	return confirmBoxStyle.Render(content)
}
