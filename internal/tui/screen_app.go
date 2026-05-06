package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) updateApp(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.composerInput.Focused() {
			switch msg.String() {
			case "enter":
				m.submitCommand()
				return m, nil
			case "esc":
				m.composerInput.Blur()
				return m, nil
			}

			var cmd tea.Cmd
			m.composerInput, cmd = m.composerInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "/":
			return m, m.composerInput.Focus()
		case "up":
			m.moveWorkflow(-1)
			return m, nil
		case "down":
			m.moveWorkflow(1)
			return m, nil
		case "n":
			m.createWorkflow()
			return m, nil
		case "enter":
			m.openSelectedWorkflow()
			return m, nil
		}
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		if mouse.Button == tea.MouseLeft {
			return m.handleAppClick(mouse.X, mouse.Y)
		}
	case tea.MouseWheelMsg:
		m.handleAppScroll(msg.Mouse())
		return m, nil
	}

	var cmd tea.Cmd
	m.composerInput, cmd = m.composerInput.Update(msg)
	return m, cmd
}

func (m model) handleWorkflowsLoaded(msg workflowsLoadedMsg) (tea.Model, tea.Cmd) {
	m.loadingWorkflows = false
	if msg.err != nil {
		m.workflows = nil
		m.selectedWorkflow = 0
		m.workflowScroll = 0
		m.workflowsError = msg.err.Error()
		m.appendActivity("Failed to load workflows: " + truncatePlain(msg.err.Error(), 56))
		return m, nil
	}

	m.workflowsError = ""
	m.workflows = msg.workflows
	if len(m.workflows) == 0 {
		m.selectedWorkflow = 0
		m.workflowScroll = 0
		m.appendActivity("No workflows returned by server.")
		return m, nil
	}

	m.selectedWorkflow = 0
	m.workflowScroll = clampScroll(0, len(m.workflows)+1, max(m.workflowListRect.h, 1))
	m.appendActivity(fmt.Sprintf("Loaded %d workflows.", len(m.workflows)))
	return m, nil
}

func (m model) renderAppScreen() string {
	bodyHeight := max(m.height-composerHeight-footerHeight-sectionGapSize, 1)
	sidebar := renderSectionPanel("FusionFlow", m.renderSidebarLines(), m.sidebarRect.w, bodyHeight)
	main := renderSectionPanel("FusionFlow", m.renderMainLines(), m.mainRect.w, bodyHeight)
	command := m.renderCommandBar()
	footer := renderFooter("↑↓ workflows  ⏎ open  n new  / command  esc unfocus  q quit", m.footerRect.w)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, blank(sectionGapSize), main)
	ui := lipgloss.JoinVertical(lipgloss.Left, body, blank(m.width), command, footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, ui)
}

func (m model) renderSidebarLines() []string {
	contentHeight := max(m.sidebarRect.h-3, 1)
	lines := []string{
		"",
		lipgloss.NewStyle().Foreground(mutedTextColor).Render("Navigation"),
		lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("› Workflows"),
		"",
		lipgloss.NewStyle().Foreground(mutedTextColor).Render("Workflows"),
	}

	visibleHeight := max(contentHeight-4, 1)
	offset := clampScroll(m.workflowScroll, len(m.workflows)+1, visibleHeight)

	for row := 0; row < visibleHeight; row++ {
		index := offset + row
		switch {
		case index < len(m.workflows):
			wf := m.workflows[index]
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(textColor)
			if index == m.selectedWorkflow {
				prefix = "› "
				style = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
			}
			lines = append(lines, style.Render(truncatePlain(prefix+wf.name, max(m.sidebarRect.w-6, 8))))
		case index == len(m.workflows):
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(successColor)
			if index == m.selectedWorkflow {
				prefix = "› "
				style = style.Bold(true)
			}
			lines = append(lines, style.Render(prefix+"+ New workflow"))
		default:
			lines = append(lines, "")
		}
	}

	return fitLines(lines, max(m.sidebarRect.w-4, 1), contentHeight)
}

func (m model) renderMainLines() []string {
	lines := []string{
		lipgloss.NewStyle().Foreground(mutedTextColor).Render("Connected to " + m.activeHost),
		"",
	}

	if m.loadingWorkflows {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(accentColor).Render(m.spinner.View()+" Loading workflows..."),
			"",
		)
	}
	if m.workflowsError != "" {
		lines = append(lines, renderOptionalError(m.workflowsError), "")
	}

	wf, ok := m.selectedWorkflowItem()
	if !ok {
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(textColor).Render("No workflow selected."),
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Create a new workflow to get started."),
		)
	} else {
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(wf.name),
			"",
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Status"),
			renderStatusValue(wf.status),
			"",
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Last run"),
			lipgloss.NewStyle().Foreground(textColor).Render(wf.lastRun),
			"",
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Nodes"),
			lipgloss.NewStyle().Foreground(textColor).Render(fmt.Sprintf("%d nodes", wf.nodes)),
			"",
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Recent runs"),
		)
		for _, run := range wf.recentRuns {
			lines = append(lines, truncatePlain("• "+run, max(m.mainRect.w-6, 12)))
		}
	}

	if len(m.commandOutput) > 0 {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(mutedTextColor).Render("Help"))
		for _, line := range m.commandOutput {
			lines = append(lines, truncatePlain(line, max(m.mainRect.w-6, 12)))
		}
	}

	return fitLines(lines, max(m.mainRect.w-4, 1), max(m.mainRect.h-3, 1))
}

func (m model) renderCommandLines() []string {
	return nil
}

func (m model) renderCommandBar() string {
	title := "Command"
	if m.composerInput.Focused() {
		title = "Command  focused"
	}

	width := m.commandRect.w
	innerWidth := max(width-4, 1)
	inputWidth := max(innerWidth, 8)
	value := m.composerInput.View()
	if strings.TrimSpace(m.composerInput.Value()) == "" && !m.composerInput.Focused() {
		value = lipgloss.NewStyle().Foreground(mutedTextColor).Render("Type a command, search workflow, or ask the agent...")
	}

	titleLine := lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(title)
	inputLine := inputShell(inputWidth, m.composerInput.Focused()).Render(padRightBlank(value, max(inputWidth-2, 1)))

	return renderSectionPanel(titleLine, []string{inputLine}, width, m.commandRect.h)
}

func (m *model) moveWorkflow(delta int) {
	next := m.selectedWorkflow + delta
	if next < 0 {
		next = 0
	}
	maxIndex := len(m.workflows)
	if next > maxIndex {
		next = maxIndex
	}
	m.selectedWorkflow = next
	m.ensureSelectedWorkflowVisible()
}

func (m *model) createWorkflow() {
	m.beginNewWorkflowEditor()
}

func (m *model) openSelectedWorkflow() {
	if m.selectedWorkflow == len(m.workflows) {
		m.createWorkflow()
		return
	}

	wf, ok := m.selectedWorkflowItem()
	if !ok {
		return
	}
	m.beginWorkflowEditor(wf.name)
	m.appendActivity("Opened workflow: " + wf.name)
}

func (m *model) submitCommand() {
	value := strings.TrimSpace(m.composerInput.Value())
	if value == "" {
		return
	}

	switch strings.ToLower(value) {
	case "help":
		m.commandOutput = []string{
			"Available commands:",
			"• help",
			"• create workflow",
			"• open selected workflow",
			"• run selected workflow",
			"• search workflow <name>",
		}
	case "create workflow":
		m.composerInput.SetValue("")
		m.beginNewWorkflowEditor()
		return
	case "open selected workflow":
		m.composerInput.SetValue("")
		m.openSelectedWorkflow()
		return
	default:
		m.commandOutput = []string{
			"Unknown command: " + value,
			"Try: help",
		}
	}

	m.appendActivity("Command submitted: " + value)
	m.composerInput.SetValue("")
}

func (m model) selectedWorkflowItem() (workflow, bool) {
	if len(m.workflows) == 0 || m.selectedWorkflow < 0 || m.selectedWorkflow >= len(m.workflows) {
		return workflow{}, false
	}
	return m.workflows[m.selectedWorkflow], true
}

func (m *model) ensureSelectedWorkflowVisible() {
	visibleHeight := max(m.workflowListRect.h, 1)
	if m.selectedWorkflow < m.workflowScroll {
		m.workflowScroll = m.selectedWorkflow
	}
	if m.selectedWorkflow >= m.workflowScroll+visibleHeight {
		m.workflowScroll = clampScroll(m.selectedWorkflow-visibleHeight+1, len(m.workflows)+1, visibleHeight)
	}
}
