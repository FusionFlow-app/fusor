package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/zalando/go-keyring"
)

func (m model) updateConnect(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEnter && !m.connecting {
			return m.startConnection()
		}
		if msg.String() == "tab" || msg.String() == "shift+tab" || msg.String() == "up" || msg.String() == "down" {
			if m.hostInput.Focused() {
				m.hostInput.Blur()
				m.apiKeyInput.Focus()
			} else {
				m.apiKeyInput.Blur()
				m.hostInput.Focus()
			}
			return m, nil
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Button == tea.MouseLeft {
			return m.handleConnectClick(mouse.X, mouse.Y)
		}
	}

	var cmd1, cmd2 tea.Cmd
	m.hostInput, cmd1 = m.hostInput.Update(msg)
	m.apiKeyInput, cmd2 = m.apiKeyInput.Update(msg)
	return m, tea.Batch(cmd1, cmd2)
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
	m.activeAPIKey = msg.apiKey

	_ = keyring.Set("fusor", "host", msg.host)
	_ = keyring.Set("fusor", "api_key", msg.apiKey)

	m.connectError = ""
	m.connectStatus = "Connected."
	m.screen = screenApp
	m.loadingWorkflows = true
	m.workflowsError = ""
	m.hostInput.Blur()
	m.apiKeyInput.Blur()
	m.composerInput.Blur()
	m.appendActivity(fmt.Sprintf("Connected to %s.", msg.host))

	return m, tea.Batch(m.spinner.Tick, loadWorkflowsCmd(msg.host, msg.apiKey))
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

	inputLine := renderInputBlock(m.hostInput.View(), formWidth, m.hostInput.Focused())
	apiKeyLine := renderInputBlock(m.apiKeyInput.View(), formWidth, m.apiKeyInput.Focused())

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
	}

	leftPadding := (innerWidth - formWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}
	padStr := strings.Repeat(" ", leftPadding)

	for _, l := range strings.Split(inputLine, "\n") {
		lines = append(lines, padStr+l)
	}
	lines = append(lines, "")

	for _, l := range strings.Split(apiKeyLine, "\n") {
		lines = append(lines, padStr+l)
	}
	lines = append(lines, "")

	lines = append(lines,
		centeredLine(button, innerWidth),
		"",
		centeredLine(status, innerWidth),
		"",
	)

	card := renderBox(lines, cardWidth, m.connectCardRect.h)
	body := lipgloss.Place(m.width, m.height-footerHeight, lipgloss.Center, lipgloss.Center, card)
	footer := renderFooter("tab switch  ⏎ connect  esc/q quit", m.footerRect.w)

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}
