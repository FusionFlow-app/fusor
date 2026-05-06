package tui

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	defaultServer       = "localhost:4000"
	connectTimeout      = 15 * time.Second
	minWidth            = 72
	minHeight           = 16
	minSidebarWidth     = 28
	maxSidebarWidth     = 36
	composerHeight      = 5
	footerHeight        = 1
	sectionGapSize      = 1
	panelPaddingX       = 2
	panelPaddingTop     = 1
	panelPaddingBottom  = 1
	connectCardWidth    = 56
	connectCardHeight   = 11
	connectInputWidth   = 34
	connectButtonWidth  = 14
	defaultWorkflowName = "Untitled workflow"
)

type screen string

const (
	screenConnect        screen = "connect"
	screenApp            screen = "app"
	screenWorkflowEditor screen = "workflow_editor"
)

type workflow struct {
	id         int
	name       string
	status     string
	lastRun    string
	nodes      int
	recentRuns []string
}

type editorMode string

const (
	editorModeNormal  editorMode = "normal"
	editorModeAdd     editorMode = "add"
	editorModeConnect editorMode = "connect"
	editorModeModal   editorMode = "modal"
)

type editorOverlay string

const (
	editorOverlayNone    editorOverlay = ""
	editorOverlayContext editorOverlay = "context"
	editorOverlayCommand editorOverlay = "command"
	editorOverlayCreate  editorOverlay = "create"
)

type editorAction string

const (
	editorActionOpenCreate editorAction = "open_create"
	editorActionCreate     editorAction = "create"
	editorActionDelete     editorAction = "delete"
	editorActionEdit       editorAction = "edit"
	editorActionConnect    editorAction = "connect"
	editorActionBack       editorAction = "back"
)

type editorMenuItem struct {
	label  string
	action editorAction
	kind   string
}

type editorNode struct {
	id       string
	kind     string
	label    string
	x        int
	y        int
	w        int
	h        int
	controls []nodeControl
}

type nodeControl struct {
	name  string
	label string
	value string
}

type editorConnection struct {
	source string
	target string
}

type connectionResultMsg struct {
	host string
	err  error
}

type workflowsLoadedMsg struct {
	workflows []workflow
	err       error
}

type model struct {
	screen screen

	hostInput     textinput.Model
	composerInput textinput.Model
	spinner       spinner.Model

	width  int
	height int

	connecting       bool
	connected        bool
	loadingWorkflows bool
	terminalFocused  bool

	activeHost     string
	connectStatus  string
	connectError   string
	workflowsError string

	workflows        []workflow
	selectedWorkflow int
	workflowScroll   int
	activities       []string
	commandOutput    []string

	editorWorkflowName  string
	editorNodes         []editorNode
	editorConnections   []editorConnection
	selectedNode        int
	draggingNode        int
	dragOffsetX         int
	dragOffsetY         int
	connectSource       int
	paletteIndex        int
	editorMode          editorMode
	editorOverlay       editorOverlay
	editorMenuX         int
	editorMenuY         int
	editorMenuSelected  int
	editorMenuTarget    int
	editorCreateAtMenu  bool
	modalFocusedControl int
	modalDraft          []nodeControl
	nodeRects           map[string]rect

	connectCardRect   rect
	connectInputRect  rect
	connectButtonRect rect
	sidebarRect       rect
	navRect           rect
	workflowListRect  rect
	mainRect          rect
	commandRect       rect
	commandInputRect  rect
	canvasRect        rect
	paletteRect       rect
	inspectorRect     rect
	modalRect         rect
	footerRect        rect
}

func initialModel() model {
	hostInput := textinput.New()
	hostInput.SetValue(defaultServer)
	hostInput.SetWidth(0)
	hostInput.CharLimit = len(defaultServer) + 64
	hostInput.Placeholder = defaultServer
	hostInput.Prompt = ""
	hostInput.SetVirtualCursor(true)
	hostInput.SetStyles(textInputStyles())
	hostInput.Focus()

	composerInput := textinput.New()
	composerInput.SetWidth(0)
	composerInput.CharLimit = 256
	composerInput.Placeholder = "Type a command, search workflow, or ask the agent..."
	composerInput.Prompt = ""
	composerInput.SetVirtualCursor(true)
	composerInput.SetStyles(textInputStyles())

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accentColor)

	return model{
		screen:           screenConnect,
		hostInput:        hostInput,
		composerInput:    composerInput,
		spinner:          sp,
		connectStatus:    "Enter server host to continue.",
		terminalFocused:  true,
		selectedNode:     -1,
		draggingNode:     -1,
		connectSource:    -1,
		editorMenuTarget: -1,
		editorMode:       editorModeNormal,
		nodeRects:        map[string]rect{},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}
