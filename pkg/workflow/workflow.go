package workflow

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	paths      []Path
	activePath int
	active     int
	width      int
	height     int
	title      string
	help       string
	interval   time.Duration
	fitSteps   bool
	styles     Styles
}

type Option func(*Model)

type Path []string

type Styles struct {
	Base       lipgloss.Style
	Step       lipgloss.Style
	ActiveStep lipgloss.Style
	Arrow      lipgloss.Style
	Title      lipgloss.Style
	Help       lipgloss.Style
}

type tickMsg struct{}

func New(opts ...Option) Model {
	m := Model{
		paths:    []Path{Route("Plan", "Retrieve", "Tool", "Answer")},
		height:   20,
		title:    "workflow",
		help:     "composable header view above chat",
		interval: time.Second,
		styles:   DefaultStyles(),
	}

	for _, opt := range opts {
		opt(&m)
	}
	m.clampActive()
	return m
}

func Route(steps ...string) Path {
	return append(Path(nil), steps...)
}

func Paths(paths ...Path) Option {
	return func(m *Model) {
		m.paths = cleanPaths(paths)
		m.clampActive()
	}
}

func Steps(steps ...string) Option {
	return func(m *Model) {
		if len(steps) == 0 {
			return
		}
		m.paths = []Path{Route(steps...)}
		m.clampActive()
	}
}

func WithActivePath(index int) Option {
	return func(m *Model) {
		m.activePath = index
	}
}

func WithActive(index int) Option {
	return func(m *Model) {
		m.active = index
	}
}

func WithHeight(height int) Option {
	return func(m *Model) {
		m.height = max(1, height)
	}
}

func WithTitle(title string) Option {
	return func(m *Model) {
		m.title = title
	}
}

func WithHelp(help string) Option {
	return func(m *Model) {
		m.help = help
	}
}

func WithTickInterval(interval time.Duration) Option {
	return func(m *Model) {
		m.interval = interval
	}
}

func WithFitSteps() Option {
	return func(m *Model) {
		m.fitSteps = true
	}
}

func WithStyles(styles Styles) Option {
	return func(m *Model) {
		m.styles = styles
	}
}

func DefaultStyles() Styles {
	step := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Foreground(lipgloss.Color("245")).
		Align(lipgloss.Center).
		Height(3)

	return Styles{
		Base: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("238")).
			Align(lipgloss.Center),
		Step: step,
		ActiveStep: step.
			BorderForeground(lipgloss.Color("82")).
			Foreground(lipgloss.Color("82")).
			Bold(true),
		Arrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Height(3).
			AlignVertical(lipgloss.Center),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
	}
}

func (m Model) Init() tea.Cmd {
	if m.interval <= 0 {
		return nil
	}
	return tick(m.interval)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tickMsg:
		m.Next()
		if m.interval > 0 {
			return m, tick(m.interval)
		}
	}
	return m, nil
}

func (m Model) SetWidth(width int) Model {
	m.width = width
	return m
}

func (m Model) Height() int {
	return max(1, m.height)
}

func (m Model) Active() int {
	return m.active
}

func (m Model) ActivePath() int {
	return m.activePath
}

func (m Model) Steps() []string {
	return append([]string(nil), m.activeSteps()...)
}

func (m Model) Paths() []Path {
	paths := make([]Path, len(m.paths))
	for i, path := range m.paths {
		paths[i] = Route(path...)
	}
	return paths
}

func (m *Model) Next() {
	if len(m.paths) == 0 {
		m.active = 0
		m.activePath = 0
		return
	}

	path := m.activeSteps()
	if len(path) == 0 {
		m.active = 0
		m.activePath = (m.activePath + 1) % len(m.paths)
		return
	}

	if m.active < len(path)-1 {
		m.active++
		return
	}

	m.active = 0
	m.activePath = (m.activePath + 1) % len(m.paths)
}

func (m *Model) NextPath() {
	if len(m.paths) == 0 {
		m.activePath = 0
		m.active = 0
		return
	}
	m.activePath = (m.activePath + 1) % len(m.paths)
	m.clampActive()
}

func (m *Model) SetActivePath(index int) {
	m.activePath = index
	m.clampActive()
}

func (m *Model) SetActive(index int) {
	m.active = index
	m.clampActive()
}

func (m Model) View() tea.View {
	return tea.NewView(m.ViewContent())
}

func (m Model) ViewContent() string {
	width := max(1, m.width)
	contentParts := []string{""}
	if m.title != "" {
		contentParts = append(contentParts, m.styles.Title.Render(m.title), "")
	}
	contentParts = append(contentParts, m.renderPaths(width))
	if m.help != "" {
		contentParts = append(contentParts, "", m.styles.Help.Render(m.help))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, contentParts...)
	return m.styles.Base.
		Width(width).
		Height(m.Height()).
		Render(content)
}

func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) clampActive() {
	if len(m.paths) == 0 {
		m.active = 0
		m.activePath = 0
		return
	}
	m.activePath = min(max(0, m.activePath), len(m.paths)-1)
	m.active = min(max(0, m.active), max(0, len(m.activeSteps())-1))
}

func (m Model) activeSteps() Path {
	if len(m.paths) == 0 {
		return nil
	}
	index := min(max(0, m.activePath), len(m.paths)-1)
	return m.paths[index]
}

func (m Model) renderPaths(width int) string {
	paths := m.paths
	if len(paths) == 0 {
		paths = []Path{{""}}
	}
	if len(paths) == 1 {
		return m.renderPath(width, paths[0], true)
	}

	return m.renderBranchPaths(width, paths)
}

func (m Model) renderBranchPaths(width int, paths []Path) string {
	boxWidths := m.branchStepWidths(width, paths)
	activeKey := pathKey(m.activeSteps(), m.active)
	rendered := make(map[string]bool)
	rows := make([]string, 0, len(paths))
	for _, path := range paths {
		row, ok := m.renderBranchPath(path, boxWidths, activeKey, rendered)
		if ok {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return m.renderPath(width, Path{""}, true)
	}
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}

func (m Model) renderBranchPath(path Path, boxWidths []int, activeKey string, rendered map[string]bool) (string, bool) {
	if len(path) == 0 {
		return "", false
	}

	arrow := m.styles.Arrow.Render(" -> ")
	arrowBlank := blankLike(arrow)
	parts := make([]string, 0, len(boxWidths)*2-1)
	prefix := make(Path, 0, len(path))
	hasNewStep := false
	for i, boxWidth := range boxWidths {
		stepBlank := blankLike(m.styles.Step.Width(boxWidth).Render(""))
		if i > 0 {
			if i < len(path) {
				parts = append(parts, arrow)
			} else {
				parts = append(parts, arrowBlank)
			}
		}
		if i >= len(path) {
			parts = append(parts, stepBlank)
			continue
		}

		step := path[i]
		prefix = append(prefix, step)
		key := pathKey(prefix, i)
		if rendered[key] {
			parts = append(parts, stepBlank)
			continue
		}

		style := m.styles.Step.Width(boxWidth)
		if key == activeKey {
			style = m.styles.ActiveStep.Width(boxWidth)
		}
		parts = append(parts, style.Render(step))
		rendered[key] = true
		hasNewStep = true
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...), hasNewStep
}

func (m Model) renderPath(width int, path Path, activePath bool) string {
	if len(path) == 0 {
		path = Path{""}
	}

	boxWidths := m.pathStepWidths(width, path)
	parts := make([]string, 0, len(path)*2-1)
	for i, step := range path {
		style := m.styles.Step.Width(boxWidths[i])
		if activePath && i == m.active {
			style = m.styles.ActiveStep.Width(boxWidths[i])
		}
		parts = append(parts, style.Render(step))
		if i < len(path)-1 {
			parts = append(parts, m.styles.Arrow.Render(" -> "))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func (m Model) branchStepWidths(width int, paths []Path) []int {
	depth := maxPathDepth(paths)
	if !m.fitSteps {
		boxWidth := max(10, (width-(depth-1)*4)/max(1, depth))
		widths := make([]int, depth)
		for i := range widths {
			widths[i] = boxWidth
		}
		return widths
	}

	widths := make([]int, depth)
	for i := range widths {
		widths[i] = 10
	}
	for _, path := range paths {
		for i, step := range path {
			widths[i] = max(widths[i], m.fittedStepWidth(step))
		}
	}
	return widths
}

func (m Model) pathStepWidths(width int, path Path) []int {
	if !m.fitSteps {
		boxWidth := max(10, (width-(len(path)-1)*4)/len(path))
		widths := make([]int, len(path))
		for i := range widths {
			widths[i] = boxWidth
		}
		return widths
	}

	widths := make([]int, len(path))
	for i, step := range path {
		widths[i] = m.fittedStepWidth(step)
	}
	return widths
}

func (m Model) fittedStepWidth(step string) int {
	frameWidth := max(m.styles.Step.GetHorizontalFrameSize(), m.styles.ActiveStep.GetHorizontalFrameSize())
	return max(10, lipgloss.Width(step)+frameWidth)
}

func blankLike(block string) string {
	return lipgloss.NewStyle().
		Width(lipgloss.Width(block)).
		Height(lipgloss.Height(block)).
		Render("")
}

func maxPathDepth(paths []Path) int {
	depth := 1
	for _, path := range paths {
		depth = max(depth, len(path))
	}
	return depth
}

func pathKey(path Path, index int) string {
	if len(path) == 0 || index < 0 {
		return ""
	}
	index = min(index, len(path)-1)
	return strings.Join(path[:index+1], "\x00")
}

func cleanPaths(paths []Path) []Path {
	if len(paths) == 0 {
		return nil
	}

	cleaned := make([]Path, 0, len(paths))
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		cleaned = append(cleaned, Route(path...))
	}
	return cleaned
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
