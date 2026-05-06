package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) updateConnect(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" && !m.connecting {
			return m.startConnection()
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Button == tea.MouseLeft {
			return m.handleConnectClick(mouse.X, mouse.Y)
		}
	}

	var cmd tea.Cmd
	m.hostInput, cmd = m.hostInput.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return baseView("")
	}

	if m.width < minWidth || m.height < minHeight {
		body := lipgloss.NewStyle().
			Foreground(mutedTextColor).
			Render("Increase terminal size to use FusionFlow.")
		frame := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
		return baseView(frame)
	}

	switch m.screen {
	case screenWorkflowEditor:
		return baseView(m.renderWorkflowEditor())
	case screenApp:
		return baseView(m.renderAppScreen())
	default:
		return baseView(m.renderConnectScreen())
	}
}

func (m model) handleConnectionResult(msg connectionResultMsg) (tea.Model, tea.Cmd) {
	m.connecting = false

	if msg.err != nil {
		m.connected = false
		m.connectError = msg.err.Error()
		m.connectStatus = "Error: " + truncatePlain(msg.err.Error(), 48)
		return m, m.hostInput.Focus()
	}

	m.connected = true
	m.activeHost = msg.host
	m.connectError = ""
	m.connectStatus = "Connected."
	m.screen = screenApp
	m.loadingWorkflows = true
	m.workflowsError = ""
	m.hostInput.Blur()
	m.composerInput.Blur()
	m.appendActivity(fmt.Sprintf("Connected to %s.", msg.host))

	return m, tea.Batch(m.spinner.Tick, loadWorkflowsCmd(msg.host))
}

func (m model) handleConnectClick(x, y int) (model, tea.Cmd) {
	switch {
	case inside(x, y, m.connectInputRect):
		cmd := m.hostInput.Focus()
		return m, cmd
	case inside(x, y, m.connectButtonRect):
		if m.connecting {
			return m, nil
		}
		return m.startConnection()
	default:
		return m, nil
	}
}

func (m model) renderConnectScreen() string {
	cardWidth := m.connectCardRect.w
	innerWidth := max(cardWidth-4, 1)
	formWidth := min(connectInputWidth, max(innerWidth-4, 18))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor).
		Render("FusionFlow")

	subtitle := lipgloss.NewStyle().
		Foreground(mutedTextColor).
		Render("Connect to your FusionFlow server")

	inputView := padRightBlank(m.hostInput.View(), max(formWidth-2, 1))
	inputLine := inputShell(formWidth, m.hostInput.Focused()).Render(inputView)

	buttonLabel := "Connect"
	if m.connecting {
		buttonLabel = m.spinner.View() + " Connecting"
	}
	button := buttonStyle(m.connecting).Width(connectButtonWidth).Render(buttonLabel)

	statusText := m.connectStatus
	statusStyle := lipgloss.NewStyle().Foreground(mutedTextColor)
	switch {
	case m.connecting:
		statusText = "Connecting..."
		statusStyle = statusStyle.Foreground(accentColor)
	case m.connectError != "":
		statusStyle = statusStyle.Foreground(errorColor)
	case m.connected:
		statusText = "Connected."
		statusStyle = statusStyle.Foreground(successColor)
	}
	status := statusStyle.Render(statusText)

	lines := []string{
		"",
		centeredLine(title, innerWidth),
		"",
		centeredLine(subtitle, innerWidth),
		"",
		centeredLine(inputLine, innerWidth),
		"",
		centeredLine(button, innerWidth),
		"",
		centeredLine(status, innerWidth),
		"",
	}

	card := renderBox(lines, cardWidth, m.connectCardRect.h)
	body := lipgloss.Place(m.width, m.height-footerHeight, lipgloss.Center, lipgloss.Center, card)
	footer := renderFooter("⏎ connect  esc/q quit", m.footerRect.w)

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}
