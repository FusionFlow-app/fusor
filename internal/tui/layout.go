package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type rect struct {
	x int
	y int
	w int
	h int
}

func inside(x, y int, r rect) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (m *model) reflow() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	footerY := max(m.height-footerHeight, 0)
	m.footerRect = rect{x: 0, y: footerY, w: m.width, h: footerHeight}

	cardWidth := min(connectCardWidth, max(m.width-4, 24))
	cardHeight := min(connectCardHeight, max(m.height-footerHeight-2, 7))
	m.connectCardRect = rect{
		x: max((m.width-cardWidth)/2, 0),
		y: max((m.height-footerHeight-cardHeight)/2, 0),
		w: cardWidth,
		h: cardHeight,
	}

	m.connectInputRect = rect{
		x: m.connectCardRect.x + max((m.connectCardRect.w-connectInputWidth)/2, 0),
		y: m.connectCardRect.y + 5,
		w: min(connectInputWidth, max(m.connectCardRect.w-4, 1)),
		h: 1,
	}
	m.connectButtonRect = rect{
		x: m.connectCardRect.x + max((m.connectCardRect.w-connectButtonWidth)/2, 0),
		y: m.connectCardRect.y + 7,
		w: min(connectButtonWidth, max(m.connectCardRect.w-4, 1)),
		h: 1,
	}

	bodyHeight := max(m.height-composerHeight-footerHeight-sectionGapSize, 1)
	sidebarWidth := clamp(max(m.width/4, minSidebarWidth), minSidebarWidth, maxSidebarWidth)
	if sidebarWidth > m.width-24-sectionGapSize {
		sidebarWidth = max(m.width/3, minSidebarWidth)
	}

	m.sidebarRect = rect{x: 0, y: 0, w: sidebarWidth, h: bodyHeight}
	m.mainRect = rect{x: sidebarWidth + sectionGapSize, y: 0, w: max(m.width-sidebarWidth-sectionGapSize, 1), h: bodyHeight}
	m.commandRect = rect{x: 0, y: bodyHeight + sectionGapSize, w: m.width, h: composerHeight}

	editorBottomHeight := 7
	editorCanvasHeight := max(m.height-editorBottomHeight-footerHeight-sectionGapSize, 1)
	m.canvasRect = rect{x: 0, y: 0, w: m.width, h: editorCanvasHeight}
	m.paletteRect = rect{x: 0, y: editorCanvasHeight + sectionGapSize, w: min(24, max(m.width/3, 18)), h: editorBottomHeight}
	m.inspectorRect = rect{
		x: m.paletteRect.w + sectionGapSize,
		y: editorCanvasHeight + sectionGapSize,
		w: max(m.width-m.paletteRect.w-sectionGapSize, 1),
		h: editorBottomHeight,
	}
	m.modalRect = rect{
		x: max((m.width-44)/2, 0),
		y: max((m.height-18)/2, 0),
		w: min(44, max(m.width-4, 24)),
		h: min(18, max(m.height-4, 8)),
	}

	sidebarContentX := m.sidebarRect.x + 2
	sidebarContentY := m.sidebarRect.y + 2
	sidebarContentW := max(m.sidebarRect.w-4, 1)
	contentHeight := max(m.sidebarRect.h-3, 1)

	m.navRect = rect{x: sidebarContentX, y: sidebarContentY + 2, w: sidebarContentW, h: 1}
	m.workflowListRect = rect{x: sidebarContentX, y: sidebarContentY + 4, w: sidebarContentW, h: max(contentHeight-4, 1)}
	m.commandInputRect = rect{
		x: m.commandRect.x + 2,
		y: m.commandRect.y + 2,
		w: max(m.commandRect.w-4, 1),
		h: 1,
	}

	m.hostInput.SetWidth(max(m.connectInputRect.w-2, 0))
	m.hostInput.CharLimit = max(m.connectInputRect.w*4, 32)
	m.composerInput.SetWidth(max(m.commandInputRect.w-2, 0))
	m.composerInput.CharLimit = max(m.commandInputRect.w*4, 64)

	m.workflowScroll = clampScroll(m.workflowScroll, len(m.workflows)+1, max(m.workflowListRect.h, 1))
	m.clampEditorNodes()
	m.rebuildNodeRects()
}

func renderSectionPanel(title string, lines []string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	innerWidth := max(width-4, 1)
	contentHeight := max(height-3, 0)
	border := lipgloss.NewStyle().Foreground(panelBorder).Render
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(textColor)

	contentLines := append([]string{titleStyle.Render(title)}, lines...)
	fitted := fitLines(contentLines, innerWidth, contentHeight)

	var b strings.Builder
	b.WriteString(border("╭" + strings.Repeat("─", max(width-2, 0)) + "╮"))
	for _, line := range fitted {
		b.WriteByte('\n')
		b.WriteString(border("│ ") + padRightBlank(line, innerWidth) + border(" │"))
	}
	b.WriteByte('\n')
	b.WriteString(border("╰" + strings.Repeat("─", max(width-2, 0)) + "╯"))
	return b.String()
}

func renderBox(lines []string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	innerWidth := max(width-4, 1)
	contentHeight := max(height-2, 0)
	border := lipgloss.NewStyle().Foreground(panelBorder).Render
	fitted := fitLines(lines, innerWidth, contentHeight)

	var b strings.Builder
	b.WriteString(border("╭" + strings.Repeat("─", max(width-2, 0)) + "╮"))
	for _, line := range fitted {
		b.WriteByte('\n')
		b.WriteString(border("│ ") + padRightBlank(line, innerWidth) + border(" │"))
	}
	b.WriteByte('\n')
	b.WriteString(border("╰" + strings.Repeat("─", max(width-2, 0)) + "╯"))
	return b.String()
}
