package tui

import tea "charm.land/bubbletea/v2"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.reflow()
	case tea.FocusMsg:
		m.terminalFocused = true
	case tea.BlurMsg:
		m.terminalFocused = false
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.screen == screenWorkflowEditor {
				next, cmd := m.updateWorkflowEditor(msg)
				return next, cmd
			}
			if m.screen == screenApp {
				if m.composerInput.Focused() {
					m.composerInput.Blur()
				}
				return m, nil
			}
			return m, tea.Quit
		}
	case connectionResultMsg:
		return m.handleConnectionResult(msg)
	case workflowsLoadedMsg:
		return m.handleWorkflowsLoaded(msg)
	}

	var cmds []tea.Cmd

	switch m.screen {
	case screenConnect:
		next, cmd := m.updateConnect(msg)
		m = next
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case screenApp:
		next, cmd := m.updateApp(msg)
		m = next
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case screenWorkflowEditor:
		next, cmd := m.updateWorkflowEditor(msg)
		m = next
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if m.connecting || m.loadingWorkflows {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
