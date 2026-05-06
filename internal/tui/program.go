package tui

import tea "charm.land/bubbletea/v2"

func NewProgram() *tea.Program {
	return tea.NewProgram(initialModel())
}
