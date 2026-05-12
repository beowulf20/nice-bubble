package chat

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
		input:        input,
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot)),
		statusBar:    DefaultStatusBar(),
		status:       status,
		messageIndex: make(map[string]int),
		toolIndex:    make(map[string]int),
		toolFormat:   DefaultToolFormat(),
		styles:       DefaultStyles(),
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.h, m.w = msg.Height, msg.Width
		m.input.SetWidth(max(1, msg.Width-2))

	case statusUpdateMsg:
		wasActive := m.spinnerActive
		m.status.Apply(StatusUpdate(msg))
		m.spinnerActive = m.shouldSpin()

		cmds = append(cmds, waitForStatusUpdate(m.statusUpdates))
		if m.spinnerActive && !wasActive {
			cmds = append(cmds, m.spinner.Tick)
		}
		return m, tea.Batch(cmds...)

	case chatUpdateMsg:
		wasActive := m.spinnerActive
		beforeLines := m.chatLineCount()
		m.ApplyChatUpdate(ChatUpdate(msg))
		if m.scrollOffset > 0 {
			m.scrollOffset += m.chatLineCount() - beforeLines
		}
		m.clampScroll()
		m.spinnerActive = m.shouldSpin()
		cmds = append(cmds, waitForChatUpdate(m.chatUpdates))
		if m.spinnerActive && !wasActive {
			cmds = append(cmds, m.spinner.Tick)
		}
		return m, tea.Batch(cmds...)

	case slashCommandMsg:
		m.RegisterSlashCommand(SlashCommand(msg))
		m.clampSlashSelection()
		return m, waitForSlashCommand(m.slashUpdates)

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
			return m, nil
		case "pgdown", "pagedown":
			m.scrollOffset -= max(1, m.chatHeight()/2)
			m.clampScroll()
			return m, nil
		case "ctrl+home":
			m.scrollOffset = m.maxScroll()
			return m, nil
		case "ctrl+end":
			m.scrollOffset = 0
			return m, nil
		}

		if m.slashPreviewVisible() {
			switch msg.String() {
			case "up":
				m.slashSelected--
				m.wrapSlashSelection()
				return m, nil
			case "down":
				m.slashSelected++
				m.wrapSlashSelection()
				return m, nil
			}
		}

		if msg.String() == "enter" {
			if m.slashPreviewVisible() && m.executeSelectedSlashCommand() {
				return m, nil
			}
			if m.executeSlashInput() {
				return m, nil
			}

			text := strings.TrimSpace(m.input.Value())
			if text != "" {
				m.AddMessage(ChatMessage{Role: "user", Content: text})
				m.input.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.clampSlashSelection()
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
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
