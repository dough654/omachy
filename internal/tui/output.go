package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// OutputModel wraps a viewport for scrollable log output.
type OutputModel struct {
	viewport   viewport.Model
	lines      []string
	autoScroll bool
}

func NewOutputModel(width, height int) OutputModel {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return OutputModel{
		viewport:   vp,
		autoScroll: true,
	}
}

func (m OutputModel) Update(msg tea.Msg) (OutputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	// If user scrolled up, disable auto-scroll
	if m.viewport.AtBottom() {
		m.autoScroll = true
	} else {
		m.autoScroll = false
	}

	return m, cmd
}

func (m *OutputModel) AppendLine(text string) {
	m.lines = append(m.lines, text)
	content := ""
	for i, line := range m.lines {
		if i > 0 {
			content += "\n"
		}
		content += line
	}
	m.viewport.SetContent(content)
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

func (m OutputModel) View() string {
	return m.viewport.View()
}

func (m *OutputModel) SetSize(width, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
}
