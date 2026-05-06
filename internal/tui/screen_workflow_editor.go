package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var editorPalette = []string{"Start", "Evaluate Code", "Variable", "Output"}

func (m *model) beginNewWorkflowEditor() {
	m.beginWorkflowEditor(defaultWorkflowName)
}

func (m *model) beginWorkflowEditor(name string) {
	m.screen = screenWorkflowEditor
	m.editorWorkflowName = fallbackString(name, defaultWorkflowName)
	m.editorMode = editorModeNormal
	m.selectedNode = 0
	m.draggingNode = -1
	m.connectSource = -1
	m.paletteIndex = 0
	m.editorOverlay = editorOverlayNone
	m.editorMenuSelected = 0
	m.editorMenuTarget = -1
	m.editorCreateAtMenu = false
	m.modalFocusedControl = 0
	m.modalDraft = nil
	m.editorNodes = []editorNode{
		{
			id:       "start",
			kind:     "Start",
			label:    "Start",
			x:        4,
			y:        3,
			w:        14,
			h:        3,
			controls: defaultControls("Start"),
		},
		{
			id:       "output",
			kind:     "Output",
			label:    "Output",
			x:        34,
			y:        3,
			w:        14,
			h:        3,
			controls: defaultControls("Output"),
		},
	}
	m.editorConnections = []editorConnection{{source: "start", target: "output"}}
	m.rebuildNodeRects()
}

func (m model) updateWorkflowEditor(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.updateWorkflowEditorKey(msg), nil
	case tea.MouseClickMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseLeft:
			m.handleWorkflowEditorClick(mouse.X, mouse.Y)
		case tea.MouseRight:
			m.openWorkflowContextMenu(mouse.X, mouse.Y)
		}
	case tea.MouseMotionMsg:
		m.handleWorkflowEditorMotion(msg.Mouse())
	case tea.MouseReleaseMsg:
		m.draggingNode = -1
	case tea.MouseWheelMsg:
		return m, nil
	}
	return m, nil
}

func (m model) updateWorkflowEditorKey(msg tea.KeyPressMsg) model {
	if m.editorMode == editorModeModal {
		return m.updateNodeModalKey(msg)
	}
	if m.editorOverlay != editorOverlayNone {
		return m.updateEditorOverlayKey(msg)
	}

	switch msg.String() {
	case "esc":
		if m.editorMode == editorModeAdd || m.editorMode == editorModeConnect {
			m.editorMode = editorModeNormal
			m.connectSource = -1
			return m
		}
		m.screen = screenApp
		return m
	case "ctrl+p", "ctrl+shift+p":
		m.openWorkflowCommandPalette()
		return m
	case "tab":
		m.selectNextNode(1)
	case "shift+tab":
		m.selectNextNode(-1)
	case "left":
		m.moveSelectedEditorNode(-1, 0)
	case "right":
		m.moveSelectedEditorNode(1, 0)
	case "up":
		if m.editorMode == editorModeAdd {
			m.paletteIndex = clamp(m.paletteIndex-1, 0, len(editorPalette)-1)
			return m
		}
		m.moveSelectedEditorNode(0, -1)
	case "down":
		if m.editorMode == editorModeAdd {
			m.paletteIndex = clamp(m.paletteIndex+1, 0, len(editorPalette)-1)
			return m
		}
		m.moveSelectedEditorNode(0, 1)
	case "enter":
		switch m.editorMode {
		case editorModeAdd:
			m.addEditorNode(editorPalette[m.paletteIndex])
			m.editorMode = editorModeNormal
		case editorModeConnect:
			m.finishEditorConnection()
		default:
			m.openNodeModal()
		}
	case "a":
		m.editorMode = editorModeAdd
	case "c":
		if m.hasSelectedEditorNode() {
			m.editorMode = editorModeConnect
			m.connectSource = m.selectedNode
		}
	case "backspace", "delete":
		m.deleteSelectedEditorNode()
	}

	return m
}

func (m model) renderWorkflowEditor() string {
	canvas := renderSectionPanel("Workflow: "+m.editorWorkflowName, m.renderCanvasLines(), m.canvasRect.w, m.canvasRect.h)
	palette := renderSectionPanel("Palette", m.renderPaletteLines(), m.paletteRect.w, m.paletteRect.h)
	inspector := renderSectionPanel("Inspector", m.renderInspectorLines(), m.inspectorRect.w, m.inspectorRect.h)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, palette, blank(sectionGapSize), inspector)
	footer := renderFooter("drag move  right click menu  ctrl+shift+p palette  arrows move  enter edit  esc back", m.footerRect.w)
	ui := lipgloss.JoinVertical(lipgloss.Left, canvas, blank(m.width), bottom, footer)

	if m.editorMode == editorModeModal {
		return m.renderNodeModal()
	}
	if m.editorOverlay == editorOverlayCommand || m.editorOverlay == editorOverlayCreate {
		return m.renderWorkflowCommandPalette()
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, ui)
}

func (m model) renderCanvasLines() []string {
	w := max(m.canvasRect.w-4, 1)
	h := max(m.canvasRect.h-3, 1)
	grid := newRuneGrid(w, h)

	for _, conn := range m.editorConnections {
		m.drawConnection(grid, conn)
	}
	for i, node := range m.editorNodes {
		drawEditorNode(grid, node, i == m.selectedNode)
	}
	if m.editorOverlay == editorOverlayContext {
		m.drawContextMenu(grid)
	}

	lines := make([]string, len(grid))
	for y := range grid {
		lines[y] = string(grid[y])
	}
	return lines
}

func (m model) renderPaletteLines() []string {
	lines := make([]string, 0, len(editorPalette)+2)
	mode := "Add node"
	if m.editorMode != editorModeAdd {
		mode = "Press a to add"
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(mutedTextColor).Render(mode), "")
	for i, kind := range editorPalette {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(textColor)
		if i == m.paletteIndex {
			prefix = "› "
			style = style.Foreground(accentColor).Bold(true)
		}
		lines = append(lines, style.Render(prefix+kind))
	}
	return lines
}

func (m model) renderInspectorLines() []string {
	lines := []string{}
	if node, ok := m.selectedEditorNode(); ok {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(mutedTextColor).Render("Selected"),
			lipgloss.NewStyle().Bold(true).Foreground(textColor).Render(node.label),
			fmt.Sprintf("Type: %s", node.kind),
		)
		if m.editorMode == editorModeConnect && m.connectSource >= 0 {
			lines = append(lines, "", lipgloss.NewStyle().Foreground(accentColor).Render("Connect mode: choose target, enter to connect"))
		}
	} else {
		lines = append(lines, "No node selected.")
	}
	lines = append(lines, "", "Drag with mouse to move", "Enter edit  C connect", "A add node  Esc back")
	return lines
}

func (m model) renderNodeModal() string {
	node, ok := m.selectedEditorNode()
	if !ok {
		return m.renderWorkflowEditor()
	}

	width := max(m.modalRect.w, 32)
	lines := []string{
		lipgloss.NewStyle().Foreground(mutedTextColor).Render("Type: " + node.kind),
		"",
	}
	for i, control := range m.modalDraft {
		style := lipgloss.NewStyle().Foreground(textColor)
		prefix := "  "
		if i == m.modalFocusedControl {
			style = style.Foreground(accentColor).Bold(true)
			prefix = "› "
		}
		lines = append(lines, style.Render(prefix+control.label))
		lines = append(lines, "  "+control.value)
		lines = append(lines, "")
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(mutedTextColor).Render("tab field  type edit  enter save  esc cancel"))
	lines = append(lines, "", "          Save        Cancel")

	modal := renderSectionPanel("Edit node", lines, width, m.modalRect.h)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m model) renderWorkflowCommandPalette() string {
	items := m.editorMenuItems()
	title := "Command Palette"
	subtitle := "Workflow actions"
	if m.editorOverlay == editorOverlayCreate {
		title = "Create Node"
		subtitle = "Choose a node type"
	}
	lines := []string{
		lipgloss.NewStyle().Foreground(mutedTextColor).Render(subtitle),
		"",
	}
	for i, item := range items {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(textColor)
		if i == m.editorMenuSelected {
			prefix = "› "
			style = style.Foreground(accentColor).Bold(true)
		}
		lines = append(lines, style.Render(prefix+item.label))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(mutedTextColor).Render("↑↓ select  enter run  esc close"))

	width := min(46, max(m.width-8, 28))
	height := min(max(len(lines)+3, 10), max(m.height-4, 8))
	palette := renderSectionPanel(title, lines, width, height)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, palette)
}

func (m *model) handleWorkflowEditorClick(x, y int) {
	if m.editorOverlay != editorOverlayNone {
		m.handleEditorOverlayClick(x, y)
		return
	}

	if m.editorMode == editorModeModal {
		if !inside(x, y, m.modalRect) {
			return
		}
		if y >= m.modalRect.y+m.modalRect.h-3 {
			if x < m.modalRect.x+m.modalRect.w/2 {
				m.saveNodeModal()
			} else {
				m.editorMode = editorModeNormal
				m.modalDraft = nil
			}
			return
		}
		row := y - m.modalRect.y - 4
		if row >= 0 {
			index := row / 3
			if index >= 0 && index < len(m.modalDraft) {
				m.modalFocusedControl = index
			}
		}
		return
	}

	if inside(x, y, m.paletteRect) {
		row := y - m.paletteRect.y - 4
		if row >= 0 && row < len(editorPalette) {
			m.paletteIndex = row
			m.addEditorNode(editorPalette[row])
			m.editorMode = editorModeNormal
		}
		return
	}

	for i, node := range m.editorNodes {
		if inside(x, y, m.nodeRects[node.id]) {
			wasSelected := m.selectedNode == i
			m.selectedNode = i
			if m.editorMode == editorModeConnect {
				m.finishEditorConnection()
				return
			}
			if wasSelected {
				m.draggingNode = i
			} else {
				m.draggingNode = i
			}
			m.dragOffsetX = x - m.nodeRects[node.id].x
			m.dragOffsetY = y - m.nodeRects[node.id].y
			return
		}
	}
}

func (m *model) openWorkflowContextMenu(x, y int) {
	if m.editorMode == editorModeModal {
		return
	}
	m.draggingNode = -1
	m.editorOverlay = editorOverlayContext
	m.editorMenuSelected = 0
	m.editorMenuTarget = -1
	m.editorCreateAtMenu = true

	for i, node := range m.editorNodes {
		if inside(x, y, m.nodeRects[node.id]) {
			m.selectedNode = i
			m.editorMenuTarget = i
			break
		}
	}
	if m.editorMenuTarget < 0 && !inside(x, y, m.canvasRect) {
		m.closeEditorOverlay()
		return
	}

	contentX := x - m.canvasRect.x - 2
	contentY := y - m.canvasRect.y - 2
	menuWidth, menuHeight := m.contextMenuSize()
	m.editorMenuX = clamp(contentX, 0, max(m.canvasRect.w-4-menuWidth, 0))
	m.editorMenuY = clamp(contentY, 0, max(m.canvasRect.h-3-menuHeight, 0))
}

func (m *model) openWorkflowCommandPalette() {
	m.draggingNode = -1
	m.editorOverlay = editorOverlayCommand
	m.editorMenuSelected = 0
	m.editorMenuTarget = m.selectedNode
	m.editorCreateAtMenu = false
}

func (m model) updateEditorOverlayKey(msg tea.KeyPressMsg) model {
	items := m.editorMenuItems()
	switch msg.String() {
	case "esc":
		m.closeEditorOverlay()
	case "up":
		m.editorMenuSelected = clamp(m.editorMenuSelected-1, 0, len(items)-1)
	case "down":
		m.editorMenuSelected = clamp(m.editorMenuSelected+1, 0, len(items)-1)
	case "enter":
		if len(items) > 0 {
			m.runEditorAction(items[m.editorMenuSelected])
		}
	}
	return m
}

func (m *model) handleEditorOverlayClick(x, y int) {
	menuRect := m.editorMenuRect()
	if !inside(x, y, menuRect) {
		m.closeEditorOverlay()
		return
	}

	row := y - menuRect.y - 1
	if m.editorOverlay == editorOverlayCommand || m.editorOverlay == editorOverlayCreate {
		row = y - menuRect.y - 4
	}
	items := m.editorMenuItems()
	if row < 0 || row >= len(items) {
		return
	}
	m.editorMenuSelected = row
	m.runEditorAction(items[row])
}

func (m *model) closeEditorOverlay() {
	m.editorOverlay = editorOverlayNone
	m.editorMenuSelected = 0
	m.editorMenuTarget = -1
	m.editorCreateAtMenu = false
}

func (m model) editorMenuItems() []editorMenuItem {
	if m.editorOverlay == editorOverlayCreate {
		return editorCreateMenuItems()
	}

	if m.editorOverlay == editorOverlayContext {
		if m.editorMenuTarget >= 0 && m.editorMenuTarget < len(m.editorNodes) {
			items := []editorMenuItem{
				{label: "Edit node", action: editorActionEdit},
			}
			if m.editorNodes[m.editorMenuTarget].kind != "Start" {
				items = append(items, editorMenuItem{label: "Delete node", action: editorActionDelete})
			}
			return items
		}
		return []editorMenuItem{
			{label: "Create node", action: editorActionOpenCreate},
		}
	}

	items := []editorMenuItem{
		{label: "Create node", action: editorActionOpenCreate},
	}
	if m.hasSelectedEditorNode() {
		items = append(items,
			editorMenuItem{label: "Edit selected node", action: editorActionEdit},
			editorMenuItem{label: "Connect from selected node", action: editorActionConnect},
		)
		if m.editorNodes[m.selectedNode].kind != "Start" {
			items = append(items, editorMenuItem{label: "Delete selected node", action: editorActionDelete})
		}
	}
	return append(items, editorMenuItem{label: "Back to workflows", action: editorActionBack})
}

func (m model) editorMenuRect() rect {
	if m.editorOverlay == editorOverlayCommand || m.editorOverlay == editorOverlayCreate {
		items := m.editorMenuItems()
		width := min(46, max(m.width-8, 28))
		height := min(max(len(items)+7, 10), max(m.height-4, 8))
		return rect{x: max((m.width-width)/2, 0), y: max((m.height-height)/2, 0), w: width, h: height}
	}

	width, height := m.contextMenuSize()
	return rect{
		x: m.canvasRect.x + 2 + m.editorMenuX,
		y: m.canvasRect.y + 2 + m.editorMenuY,
		w: width,
		h: height,
	}
}

func (m model) contextMenuSize() (int, int) {
	width := 0
	for _, item := range m.editorMenuItems() {
		width = max(width, len(item.label)+6)
	}
	return max(width, 16), len(m.editorMenuItems()) + 2
}

func editorCreateMenuItems() []editorMenuItem {
	items := make([]editorMenuItem, 0, len(editorPalette))
	for _, kind := range editorPalette {
		items = append(items, editorMenuItem{
			label:  "Add " + kind,
			action: editorActionCreate,
			kind:   kind,
		})
	}
	return items
}

func (m *model) runEditorAction(item editorMenuItem) {
	target := m.editorMenuTarget
	if target >= 0 && target < len(m.editorNodes) {
		m.selectedNode = target
	}

	switch item.action {
	case editorActionOpenCreate:
		m.editorOverlay = editorOverlayCreate
		m.editorMenuSelected = 0
	case editorActionCreate:
		kind := fallbackString(item.kind, "Evaluate Code")
		if m.editorCreateAtMenu {
			m.addEditorNodeAt(kind, m.editorMenuX, m.editorMenuY)
		} else {
			m.addEditorNode(kind)
		}
		m.closeEditorOverlay()
	case editorActionDelete:
		m.closeEditorOverlay()
		m.deleteSelectedEditorNode()
	case editorActionEdit:
		m.closeEditorOverlay()
		m.openNodeModal()
	case editorActionConnect:
		m.closeEditorOverlay()
		if m.hasSelectedEditorNode() {
			m.editorMode = editorModeConnect
			m.connectSource = m.selectedNode
		}
	case editorActionBack:
		m.closeEditorOverlay()
		m.screen = screenApp
	}
}

func (m *model) handleWorkflowEditorMotion(mouse tea.Mouse) {
	if m.draggingNode < 0 || m.draggingNode >= len(m.editorNodes) {
		return
	}
	node := &m.editorNodes[m.draggingNode]
	contentOriginX := m.canvasRect.x + 2
	contentOriginY := m.canvasRect.y + 2
	maxX := max(m.canvasRect.w-4-node.w, 0)
	maxY := max(m.canvasRect.h-3-node.h, 0)
	node.x = clamp(mouse.X-contentOriginX-m.dragOffsetX, 0, maxX)
	node.y = clamp(mouse.Y-contentOriginY-m.dragOffsetY, 0, maxY)
	m.rebuildNodeRects()
}

func (m *model) moveSelectedEditorNode(dx, dy int) {
	if !m.hasSelectedEditorNode() {
		return
	}
	node := &m.editorNodes[m.selectedNode]
	node.x = clamp(node.x+dx, 0, max(m.canvasRect.w-4-node.w, 0))
	node.y = clamp(node.y+dy, 0, max(m.canvasRect.h-3-node.h, 0))
	m.rebuildNodeRects()
}

func (m *model) addEditorNode(kind string) {
	m.addEditorNodeAt(kind, max((m.canvasRect.w-4)/2-8, 0), max((m.canvasRect.h-3)/2-1, 0))
}

func (m *model) addEditorNodeAt(kind string, x, y int) {
	id := fmt.Sprintf("node_%d", len(m.editorNodes)+1)
	node := editorNode{
		id:       id,
		kind:     kind,
		label:    kind,
		w:        16,
		h:        3,
		controls: defaultControls(kind),
	}
	node.x = clamp(x, 0, max(m.canvasRect.w-4-node.w, 0))
	node.y = clamp(y, 0, max(m.canvasRect.h-3-node.h, 0))
	m.editorNodes = append(m.editorNodes, node)
	m.selectedNode = len(m.editorNodes) - 1
	m.rebuildNodeRects()
}

func (m *model) finishEditorConnection() {
	if m.connectSource < 0 || m.connectSource >= len(m.editorNodes) || !m.hasSelectedEditorNode() || m.connectSource == m.selectedNode {
		m.editorMode = editorModeNormal
		m.connectSource = -1
		return
	}
	source := m.editorNodes[m.connectSource].id
	target := m.editorNodes[m.selectedNode].id
	for _, conn := range m.editorConnections {
		if conn.source == source && conn.target == target {
			m.editorMode = editorModeNormal
			m.connectSource = -1
			return
		}
	}
	m.editorConnections = append(m.editorConnections, editorConnection{source: source, target: target})
	m.editorMode = editorModeNormal
	m.connectSource = -1
}

func (m *model) openNodeModal() {
	if !m.hasSelectedEditorNode() {
		return
	}
	m.editorMode = editorModeModal
	m.modalFocusedControl = 0
	m.modalDraft = append([]nodeControl(nil), m.editorNodes[m.selectedNode].controls...)
}

func (m model) updateNodeModalKey(msg tea.KeyPressMsg) model {
	switch msg.String() {
	case "esc":
		m.editorMode = editorModeNormal
		m.modalDraft = nil
	case "tab":
		if len(m.modalDraft) > 0 {
			m.modalFocusedControl = (m.modalFocusedControl + 1) % len(m.modalDraft)
		}
	case "enter":
		m.saveNodeModal()
	case "backspace":
		if len(m.modalDraft) > 0 {
			value := []rune(m.modalDraft[m.modalFocusedControl].value)
			if len(value) > 0 {
				m.modalDraft[m.modalFocusedControl].value = string(value[:len(value)-1])
			}
		}
	default:
		text := msg.Key().Text
		if text != "" && len(m.modalDraft) > 0 {
			m.modalDraft[m.modalFocusedControl].value += text
		}
	}
	return m
}

func (m *model) saveNodeModal() {
	if m.hasSelectedEditorNode() {
		m.editorNodes[m.selectedNode].controls = append([]nodeControl(nil), m.modalDraft...)
		for _, control := range m.modalDraft {
			if control.name == "label" {
				m.editorNodes[m.selectedNode].label = fallbackString(control.value, m.editorNodes[m.selectedNode].kind)
			}
		}
	}
	m.editorMode = editorModeNormal
	m.modalDraft = nil
}

func (m *model) deleteSelectedEditorNode() {
	if !m.hasSelectedEditorNode() || m.editorNodes[m.selectedNode].kind == "Start" {
		return
	}
	deletedID := m.editorNodes[m.selectedNode].id
	m.editorNodes = append(m.editorNodes[:m.selectedNode], m.editorNodes[m.selectedNode+1:]...)
	nextConnections := m.editorConnections[:0]
	for _, conn := range m.editorConnections {
		if conn.source != deletedID && conn.target != deletedID {
			nextConnections = append(nextConnections, conn)
		}
	}
	m.editorConnections = nextConnections
	m.selectedNode = clamp(m.selectedNode, 0, len(m.editorNodes)-1)
	m.rebuildNodeRects()
}

func (m *model) selectNextNode(delta int) {
	if len(m.editorNodes) == 0 {
		m.selectedNode = -1
		return
	}
	m.selectedNode = (m.selectedNode + delta + len(m.editorNodes)) % len(m.editorNodes)
}

func (m model) hasSelectedEditorNode() bool {
	return m.selectedNode >= 0 && m.selectedNode < len(m.editorNodes)
}

func (m model) selectedEditorNode() (editorNode, bool) {
	if !m.hasSelectedEditorNode() {
		return editorNode{}, false
	}
	return m.editorNodes[m.selectedNode], true
}

func (m *model) clampEditorNodes() {
	for i := range m.editorNodes {
		node := &m.editorNodes[i]
		node.x = clamp(node.x, 0, max(m.canvasRect.w-4-node.w, 0))
		node.y = clamp(node.y, 0, max(m.canvasRect.h-3-node.h, 0))
	}
}

func (m *model) rebuildNodeRects() {
	if m.nodeRects == nil {
		m.nodeRects = map[string]rect{}
	}
	for key := range m.nodeRects {
		delete(m.nodeRects, key)
	}
	for _, node := range m.editorNodes {
		m.nodeRects[node.id] = rect{
			x: m.canvasRect.x + 2 + node.x,
			y: m.canvasRect.y + 2 + node.y,
			w: node.w,
			h: node.h,
		}
	}
}

func newRuneGrid(w, h int) [][]rune {
	grid := make([][]rune, h)
	for y := range grid {
		grid[y] = make([]rune, w)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}
	return grid
}

func drawEditorNode(grid [][]rune, node editorNode, selected bool) {
	if node.w < 4 || node.h < 3 {
		return
	}
	leftTop, rightTop, leftBottom, rightBottom := '╭', '╮', '╰', '╯'
	drawRune(grid, node.x, node.y, leftTop)
	drawRune(grid, node.x+node.w-1, node.y, rightTop)
	drawRune(grid, node.x, node.y+node.h-1, leftBottom)
	drawRune(grid, node.x+node.w-1, node.y+node.h-1, rightBottom)
	for x := node.x + 1; x < node.x+node.w-1; x++ {
		drawRune(grid, x, node.y, '─')
		drawRune(grid, x, node.y+node.h-1, '─')
	}
	for y := node.y + 1; y < node.y+node.h-1; y++ {
		drawRune(grid, node.x, y, '│')
		drawRune(grid, node.x+node.w-1, y, '│')
	}
	label := node.label
	if selected {
		label = "> " + label
	}
	drawText(grid, node.x+2, node.y+1, truncatePlain(label, node.w-4))
}

func (m model) drawContextMenu(grid [][]rune) {
	items := m.editorMenuItems()
	if len(items) == 0 {
		return
	}
	width, height := m.contextMenuSize()
	x := m.editorMenuX
	y := m.editorMenuY

	drawRune(grid, x, y, '╭')
	drawRune(grid, x+width-1, y, '╮')
	drawRune(grid, x, y+height-1, '╰')
	drawRune(grid, x+width-1, y+height-1, '╯')
	for col := x + 1; col < x+width-1; col++ {
		drawRune(grid, col, y, '─')
		drawRune(grid, col, y+height-1, '─')
	}
	for row := y + 1; row < y+height-1; row++ {
		drawRune(grid, x, row, '│')
		drawRune(grid, x+width-1, row, '│')
		for col := x + 1; col < x+width-1; col++ {
			drawRune(grid, col, row, ' ')
		}
	}
	for i, item := range items {
		prefix := "  "
		if i == m.editorMenuSelected {
			prefix = "› "
		}
		drawText(grid, x+2, y+1+i, truncatePlain(prefix+item.label, max(width-4, 1)))
	}
}

func (m model) drawConnection(grid [][]rune, conn editorConnection) {
	source, okSource := m.editorNodeByID(conn.source)
	target, okTarget := m.editorNodeByID(conn.target)
	if !okSource || !okTarget {
		return
	}
	x1 := source.x + source.w
	y1 := source.y + source.h/2
	x2 := target.x - 1
	y2 := target.y + target.h/2
	arrow := '▶'
	if x1 > x2 {
		x1 = source.x + source.w/2
		y1 = source.y + source.h
		x2 = target.x + target.w/2
		y2 = target.y - 1
		arrow = '▼'
		if target.y < source.y {
			y1 = source.y - 1
			y2 = target.y + target.h
			arrow = '▲'
		}
	}
	if y1 == y2 {
		drawHorizontalLine(grid, x1, x2, y1)
		drawRune(grid, x2, y2, arrow)
		return
	}
	midX := (x1 + x2) / 2
	drawHorizontalLine(grid, x1, midX, y1)
	drawVerticalLine(grid, midX, y1, y2)
	drawHorizontalLine(grid, midX, x2, y2)
	drawRune(grid, midX, y1, connectionCorner(horizontalDirection(x1, midX), verticalDirection(y1, y2)))
	drawRune(grid, midX, y2, connectionCorner(-horizontalDirection(midX, x2), -verticalDirection(y1, y2)))
	drawRune(grid, x2, y2, arrow)
}

func drawHorizontalLine(grid [][]rune, x1, x2, y int) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		drawRune(grid, x, y, '─')
	}
}

func drawVerticalLine(grid [][]rune, x, y1, y2 int) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		drawRune(grid, x, y, '│')
	}
}

func horizontalDirection(from, to int) int {
	if to < from {
		return -1
	}
	return 1
}

func verticalDirection(from, to int) int {
	if to < from {
		return -1
	}
	return 1
}

func connectionCorner(dx, dy int) rune {
	switch {
	case dx > 0 && dy > 0:
		return '╮'
	case dx > 0 && dy < 0:
		return '╯'
	case dx < 0 && dy > 0:
		return '╭'
	default:
		return '╰'
	}
}

func (m model) editorNodeByID(id string) (editorNode, bool) {
	for _, node := range m.editorNodes {
		if node.id == id {
			return node, true
		}
	}
	return editorNode{}, false
}

func drawRune(grid [][]rune, x, y int, r rune) {
	if y < 0 || y >= len(grid) || x < 0 || len(grid) == 0 || x >= len(grid[y]) {
		return
	}
	grid[y][x] = r
}

func drawText(grid [][]rune, x, y int, text string) {
	for i, r := range []rune(text) {
		drawRune(grid, x+i, y, r)
	}
}

func defaultControls(kind string) []nodeControl {
	switch kind {
	case "Evaluate Code":
		return []nodeControl{
			{name: "label", label: "Label", value: "Evaluate Code"},
			{name: "language", label: "Language", value: "python"},
			{name: "code", label: "Code", value: `set_result(variable("x"))`},
		}
	case "Variable":
		return []nodeControl{
			{name: "label", label: "Label", value: "Variable"},
			{name: "name", label: "Name", value: "x"},
			{name: "type", label: "Type", value: "Integer"},
			{name: "value", label: "Value", value: "1"},
		}
	case "Output":
		return []nodeControl{
			{name: "label", label: "Label", value: "Output"},
			{name: "status", label: "Status", value: "success"},
		}
	default:
		return []nodeControl{
			{name: "label", label: "Label", value: kind},
		}
	}
}
