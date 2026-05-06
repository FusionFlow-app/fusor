package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	backgroundColor = lipgloss.Color("#222222")
	inputBackground = lipgloss.Color("#1F1F1F")
	panelBorder     = lipgloss.Color("#4B5563")
	shadowColor     = lipgloss.Color("#222222")
	shadowEdgeColor = lipgloss.Color("#222222")
	textColor       = lipgloss.Color("#F3F4F6")
	mutedTextColor  = lipgloss.Color("#9CA3AF")
	accentColor     = lipgloss.Color("#60A5FA")
	successColor    = lipgloss.Color("#34D399")
	errorColor      = lipgloss.Color("#F87171")
)

func textInputStyles() textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	base := lipgloss.NewStyle().Foreground(textColor)
	muted := lipgloss.NewStyle().Foreground(mutedTextColor)

	styles.Focused.Text = base
	styles.Focused.Prompt = base
	styles.Focused.Placeholder = muted
	styles.Focused.Suggestion = muted
	styles.Blurred.Text = base.Foreground(mutedTextColor)
	styles.Blurred.Prompt = muted
	styles.Blurred.Placeholder = muted
	styles.Blurred.Suggestion = muted
	styles.Cursor.Color = accentColor
	styles.Cursor.Blink = true

	return styles
}

func baseView(content string) tea.View {
	v := tea.NewView(content)
	v.BackgroundColor = backgroundColor
	v.ForegroundColor = textColor
	v.WindowTitle = "Fusor"
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	return v
}

func renderOptionalError(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(errorColor).Render("Error: " + truncatePlain(message, 72))
}

func renderFooter(text string, width int) string {
	style := lipgloss.NewStyle().Foreground(mutedTextColor)
	return padRightBlank(style.Render(truncatePlain(text, max(width, 1))), max(width, 1))
}

func buttonStyle(disabled bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Foreground(textColor).
		Background(panelBorder).
		Align(lipgloss.Center)
	if disabled {
		style = style.Background(panelBorder).Foreground(mutedTextColor)
	}
	return style
}

func renderStatusValue(value string) string {
	style := lipgloss.NewStyle().Foreground(textColor)
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		style = style.Foreground(successColor)
	case "running":
		style = style.Foreground(accentColor)
	case "failed":
		style = style.Foreground(errorColor)
	}
	return style.Render(value)
}

func renderShadowColumn(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	primaryWidth := max(width-1, 1)
	primary := lipgloss.NewStyle().
		Background(shadowColor).
		Render(strings.Repeat(" ", primaryWidth))
	edge := lipgloss.NewStyle().
		Background(shadowEdgeColor).
		Render(strings.Repeat(" ", width-primaryWidth))
	line := primary + edge

	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func renderShadowRow(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	primary := lipgloss.NewStyle().
		Background(shadowColor).
		Render(strings.Repeat(" ", width))
	edge := lipgloss.NewStyle().
		Background(shadowEdgeColor).
		Render(strings.Repeat(" ", width))

	lines := make([]string, height)
	for i := range lines {
		if i == height-1 {
			lines[i] = edge
			continue
		}
		lines[i] = primary
	}
	return strings.Join(lines, "\n")
}

func inputShell(width int, focused bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(max(width-2, 1)).
		Padding(0, 1).
		Background(inputBackground).
		Foreground(textColor)

	if focused {
		style = style.Foreground(textColor)
	}

	return style
}

func panelLine(width int, content string, align lipgloss.Position) string {
	if width <= 0 {
		return ""
	}

	contentWidth := lipgloss.Width(content)
	if contentWidth > width {
		content = truncatePlain(content, width)
		contentWidth = lipgloss.Width(content)
	}

	left := 0
	switch align {
	case lipgloss.Center:
		left = max((width-contentWidth)/2, 0)
	case lipgloss.Right:
		left = max(width-contentWidth, 0)
	}

	right := max(width-left-contentWidth, 0)
	line := blank(left) + content + blank(right)
	return lipgloss.NewStyle().Foreground(textColor).Render(line)
}
