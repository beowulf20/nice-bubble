package chat

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Model struct {
	input         textinput.Model
	spinner       spinner.Model
	statusUpdates <-chan StatusUpdate
	chatUpdates   <-chan ChatUpdate
	slashUpdates  <-chan SlashCommand
	statusBar     StatusBarModel
	status        StatusState
	spinnerActive bool
	slashCommands []SlashCommand
	slashSelected int
	messages      []chatItem
	messageIndex  map[string]int
	toolIndex     map[string]int
	toolFormat    ToolFormatModel
	messagePrefix MessagePrefixes
	roleStyles    RoleStyles
	styles        Styles
	emptyMessage  string
	scrollOffset  int
	w             int
	h             int
}

type Option func(*Model)

func New(opts ...Option) Model {
	input := textinput.New()
	input.Focus()

	status := StatusState{
		"thinking": false,
		"status":   "ready",
		"model":    "llm",
		"tokens":   "0",
	}

	m := Model{
		input:         input,
		spinner:       spinner.New(spinner.WithSpinner(spinner.Dot)),
		statusBar:     DefaultStatusBar(),
		status:        status,
		messageIndex:  make(map[string]int),
		toolIndex:     make(map[string]int),
		toolFormat:    DefaultToolFormat(),
		messagePrefix: DefaultMessagePrefixes(),
		roleStyles:    DefaultRoleStyles(),
		styles:        DefaultStyles(),
	}

	for _, opt := range opts {
		opt(&m)
	}
	m.spinnerActive = m.shouldSpin()
	return m
}

func WithStatusUpdates(updates <-chan StatusUpdate) Option {
	return func(m *Model) {
		m.statusUpdates = updates
	}
}

func WithChatUpdates(updates <-chan ChatUpdate) Option {
	return func(m *Model) {
		m.chatUpdates = updates
	}
}

func WithSlashUpdates(updates <-chan SlashCommand) Option {
	return func(m *Model) {
		m.slashUpdates = updates
	}
}

func WithStatusBar(bar StatusBarModel) Option {
	return func(m *Model) {
		m.statusBar = bar
	}
}

func WithStatus(status StatusState) Option {
	return func(m *Model) {
		m.status = status
	}
}

func WithToolFormat(format ToolFormatModel) Option {
	return func(m *Model) {
		m.toolFormat = format
	}
}

func WithMessagePrefix(role string, prefix string) Option {
	return func(m *Model) {
		if m.messagePrefix == nil {
			m.messagePrefix = DefaultMessagePrefixes()
		}
		m.messagePrefix[role] = prefix
	}
}

func WithMessagePrefixes(prefixes MessagePrefixes) Option {
	return func(m *Model) {
		if m.messagePrefix == nil {
			m.messagePrefix = DefaultMessagePrefixes()
		}
		for role, prefix := range prefixes {
			m.messagePrefix[role] = prefix
		}
	}
}

func WithRoleStyle(role string, style lipgloss.Style) Option {
	return func(m *Model) {
		if m.roleStyles == nil {
			m.roleStyles = DefaultRoleStyles()
		}
		m.roleStyles[role] = style
	}
}

func WithRoleStyles(styles RoleStyles) Option {
	return func(m *Model) {
		if m.roleStyles == nil {
			m.roleStyles = DefaultRoleStyles()
		}
		for role, style := range styles {
			m.roleStyles[role] = style
		}
	}
}

func WithStyles(styles Styles) Option {
	return func(m *Model) {
		m.styles = styles
	}
}

func WithSpinner(spin spinner.Model) Option {
	return func(m *Model) {
		m.spinner = spin
	}
}

func WithEmptyMessage(message string) Option {
	return func(m *Model) {
		m.emptyMessage = message
	}
}

func (m Model) SetSize(width int, height int) Model {
	m.w, m.h = width, height
	m.input.SetWidth(max(1, width-2))
	return m
}

type statusUpdateMsg StatusUpdate
type chatUpdateMsg ChatUpdate
type slashCommandMsg SlashCommand

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.input.Focus(),
		waitForStatusUpdate(m.statusUpdates),
		waitForChatUpdate(m.chatUpdates),
		waitForSlashCommand(m.slashUpdates),
	}
	if m.spinnerActive {
		cmds = append(cmds, m.spinner.Tick)
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	wasSpinnerActive := m.spinnerActive

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.h, m.w = msg.Height, msg.Width
		m.input.SetWidth(max(1, msg.Width-2))

	case statusUpdateMsg:
		m.status.Apply(StatusUpdate(msg))
		cmds = append(cmds, waitForStatusUpdate(m.statusUpdates))
		return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)

	case chatUpdateMsg:
		beforeLines := m.chatLineCount()
		m.ApplyChatUpdate(ChatUpdate(msg))
		if m.scrollOffset > 0 {
			m.scrollOffset += m.chatLineCount() - beforeLines
		}
		m.clampScroll()
		cmds = append(cmds, waitForChatUpdate(m.chatUpdates))
		return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)

	case slashCommandMsg:
		m.RegisterSlashCommand(SlashCommand(msg))
		m.clampSlashSelection()
		cmds = append(cmds, waitForSlashCommand(m.slashUpdates))
		return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.spinnerActive {
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch msg.String() {
		case "pgup", "pageup":
			m.scrollOffset += max(1, m.chatHeight()/2)
			m.clampScroll()
			return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
		case "pgdown", "pagedown":
			m.scrollOffset -= max(1, m.chatHeight()/2)
			m.clampScroll()
			return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
		case "ctrl+home":
			m.scrollOffset = m.maxScroll()
			return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
		case "ctrl+end":
			m.scrollOffset = 0
			return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
		}

		if m.slashPreviewVisible() {
			switch msg.String() {
			case "up":
				m.slashSelected--
				m.wrapSlashSelection()
				return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
			case "down":
				m.slashSelected++
				m.wrapSlashSelection()
				return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
			}
		}

		if msg.String() == "enter" {
			if m.slashPreviewVisible() && m.executeSelectedSlashCommand() {
				return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
			}
			if m.executeSlashInput() {
				return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
			}

			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				m.AddMessage(ChatMessage{Role: "user", Content: text})
				m.input.SetValue("")
			}
			return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.clampSlashSelection()
	cmds = append(cmds, cmd)
	return m, tea.Batch(m.syncSpinner(wasSpinnerActive, cmds)...)
}

func waitForStatusUpdate(updates <-chan StatusUpdate) tea.Cmd {
	if updates == nil {
		return nil
	}
	return func() tea.Msg {
		update, ok := <-updates
		if !ok {
			return nil
		}
		return statusUpdateMsg(update)
	}
}

func waitForChatUpdate(updates <-chan ChatUpdate) tea.Cmd {
	if updates == nil {
		return nil
	}
	return func() tea.Msg {
		update, ok := <-updates
		if !ok {
			return nil
		}
		return chatUpdateMsg(update)
	}
}

func waitForSlashCommand(updates <-chan SlashCommand) tea.Cmd {
	if updates == nil {
		return nil
	}
	return func() tea.Msg {
		command, ok := <-updates
		if !ok {
			return nil
		}
		return slashCommandMsg(command)
	}
}

func (m Model) shouldSpin() bool {
	return m.statusBar.SpinnerActive(m.status) || toolSpinnerActive(m.messages, m.toolFormat)
}

func (m *Model) syncSpinner(wasActive bool, cmds []tea.Cmd) []tea.Cmd {
	m.spinnerActive = m.shouldSpin()
	if m.spinnerActive && !wasActive {
		cmds = append(cmds, m.spinner.Tick)
	}
	return cmds
}
