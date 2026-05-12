package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	baseStyle      lipgloss.Style
	messageStyle   lipgloss.Style
	inputStyle     lipgloss.Style
	statusBarStyle lipgloss.Style
	statusUpdates  = make(chan StatusUpdate, 8)
	chatUpdates    = make(chan ChatUpdate, 32)
)

type chatModel struct {
	input         textinput.Model
	spinner       spinner.Model
	statusUpdates <-chan StatusUpdate
	chatUpdates   <-chan ChatUpdate
	statusBar     statusBar
	status        StatusState
	spinnerActive bool
	messages      []chatItem
	toolIndex     map[string]int
	toolFormat    toolFormat
	scrollOffset  int
	w             int
	h             int
}

type statusUpdateMsg StatusUpdate
type chatUpdateMsg ChatUpdate

// Init implements tea.Model.
func (c chatModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		c.input.Focus(),
		waitForStatusUpdate(c.statusUpdates),
		waitForChatUpdate(c.chatUpdates),
	}
	if c.spinnerActive {
		cmds = append(cmds, c.spinner.Tick)
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (c chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.h, c.w = msg.Height, msg.Width
		c.input.SetWidth(max(1, msg.Width-2))

	case statusUpdateMsg:
		wasActive := c.spinnerActive
		c.status.Apply(StatusUpdate(msg))
		c.spinnerActive = c.shouldSpin()

		cmds = append(cmds, waitForStatusUpdate(c.statusUpdates))
		if c.spinnerActive && !wasActive {
			cmds = append(cmds, c.spinner.Tick)
		}
		return c, tea.Batch(cmds...)

	case chatUpdateMsg:
		wasActive := c.spinnerActive
		beforeLines := c.chatLineCount()
		c.applyChatUpdate(ChatUpdate(msg))
		if c.scrollOffset > 0 {
			c.scrollOffset += c.chatLineCount() - beforeLines
		}
		c.clampScroll()
		c.spinnerActive = c.shouldSpin()
		cmds = append(cmds, waitForChatUpdate(c.chatUpdates))
		if c.spinnerActive && !wasActive {
			cmds = append(cmds, c.spinner.Tick)
		}
		return c, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		if c.spinnerActive {
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return c, tea.Quit
		}

		switch msg.String() {
		case "pgup", "pageup":
			c.scrollOffset += max(1, c.chatHeight()/2)
			c.clampScroll()
			return c, nil
		case "pgdown", "pagedown":
			c.scrollOffset -= max(1, c.chatHeight()/2)
			c.clampScroll()
			return c, nil
		case "ctrl+home":
			c.scrollOffset = c.maxScroll()
			return c, nil
		case "ctrl+end":
			c.scrollOffset = 0
			return c, nil
		}

		if msg.String() == "enter" {
			text := strings.TrimSpace(c.input.Value())
			if text != "" {
				statusUpdates <- SetStatus("status", "mock tool call")
				statusUpdates <- SetStatus("thinking", true)
				c.addMessage(ChatMessage{Role: "You", Content: text})
				go mockToolCall(text)
				c.input.SetValue("")
			}
			return c, nil
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	cmds = append(cmds, cmd)
	return c, tea.Batch(cmds...)
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

func (c *chatModel) applyChatUpdate(update ChatUpdate) {
	if update.Message != nil {
		c.addMessage(*update.Message)
	}
	if update.Tool != nil {
		c.setToolMessage(*update.Tool)
	}
}

func (c *chatModel) addMessage(message ChatMessage) {
	c.messages = append(c.messages, chatItem{message: &message})
}

func (c *chatModel) setToolMessage(tool ToolMessage) {
	if c.toolIndex == nil {
		c.toolIndex = make(map[string]int)
	}

	if tool.ID == "" {
		c.messages = append(c.messages, chatItem{tool: &tool})
		return
	}

	if index, ok := c.toolIndex[tool.ID]; ok {
		c.messages[index] = chatItem{tool: &tool}
		return
	}

	c.toolIndex[tool.ID] = len(c.messages)
	c.messages = append(c.messages, chatItem{tool: &tool})
}

func (c chatModel) shouldSpin() bool {
	return c.statusBar.SpinnerActive(c.status) || toolSpinnerActive(c.messages, c.toolFormat)
}

func (c chatModel) chatHeight() int {
	return max(1, max(4, c.h)-2)
}

func (c chatModel) maxScroll() int {
	return max(0, c.chatLineCount()-c.chatHeight())
}

func (c *chatModel) clampScroll() {
	c.scrollOffset = min(max(0, c.scrollOffset), c.maxScroll())
}

// View implements tea.Model.
func (c chatModel) View() tea.View {
	width := max(1, c.w)
	chatHeight := c.chatHeight()

	lines := c.chatLines()
	lines = c.visibleChatLines(lines, chatHeight)

	chat := lipgloss.NewStyle().
		Width(width).
		Height(chatHeight).
		Render(strings.Join(lines, "\n"))
	status := statusBarStyle.Width(width).Render(c.statusBar.Render(c.status, c.spinner, width))
	input := inputStyle.Width(width).Render(c.input.View())

	return tea.NewView(baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, chat, input, status)))
}

func (c chatModel) visibleChatLines(lines []string, height int) []string {
	if len(lines) <= height {
		return lines
	}

	offset := min(max(0, c.scrollOffset), max(0, len(lines)-height))
	end := len(lines) - offset
	start := max(0, end-height)
	return lines[start:end]
}

func (c chatModel) chatLines() []string {
	if len(c.messages) == 0 {
		return []string{"assistant: ready"}
	}

	var lines []string
	for _, item := range c.messages {
		lines = append(lines, strings.Split(item.Render(c.toolFormat, c.spinner), "\n")...)
	}
	return lines
}

func (c chatModel) chatLineCount() int {
	return len(c.chatLines())
}

type StatusUpdate struct {
	Key   string
	Value any
}

func SetStatus(key string, value any) StatusUpdate {
	return StatusUpdate{Key: key, Value: value}
}

type ChatUpdate struct {
	Message *ChatMessage
	Tool    *ToolMessage
}

type ChatMessage struct {
	Role    string
	Content string
}

type ToolMessage struct {
	ID      string
	Name    string
	Content string
	Running bool
	Status  ToolStatus
	Format  *toolFormat
}

type ToolStatus string

const (
	ToolStatusRunning ToolStatus = "running"
	ToolStatusError   ToolStatus = "error"
	ToolStatusSuccess ToolStatus = "success"
	ToolStatusSucess  ToolStatus = ToolStatusSuccess
)

func AddMessage(role string, content string) ChatUpdate {
	return ChatUpdate{
		Message: &ChatMessage{
			Role:    role,
			Content: content,
		},
	}
}

func SetToolMessage(id string, name string, content string, running bool) ChatUpdate {
	status := ToolStatusSuccess
	if running {
		status = ToolStatusRunning
	}

	return SetToolStatusMessage(id, name, content, status)
}

func SetToolStatusMessage(id string, name string, content string, status ToolStatus) ChatUpdate {
	return SetToolStatus(id, name, status, ToolMessageContent(content))
}

type ToolMessageOption func(*ToolMessage)

func SetToolStatus(id string, name string, status ToolStatus, opts ...ToolMessageOption) ChatUpdate {
	tool := ToolMessage{
		ID:      id,
		Name:    name,
		Running: status == ToolStatusRunning,
		Status:  status,
	}
	for _, opt := range opts {
		opt(&tool)
	}

	return ChatUpdate{
		Tool: &tool,
	}
}

func ToolMessageContent(content string) ToolMessageOption {
	return func(tool *ToolMessage) {
		tool.Content = content
	}
}

func ToolMessageFormat(format toolFormat) ToolMessageOption {
	return func(tool *ToolMessage) {
		tool.Format = &format
	}
}

func ToolMessageFormatOptions(opts ...ToolFormatOption) ToolMessageOption {
	return ToolMessageFormat(ToolFormat(opts...))
}

func ToolMessageFields(fields ...toolField) ToolMessageOption {
	return ToolMessageFormat(ToolFormat(
		ToolPrefix(""),
		ToolFields(fields...),
	))
}

func mockToolCall(prompt string) {
	tools := []string{"search", "read_file", "run_shell", "fetch_docs"}
	name := tools[rand.Intn(len(tools))]
	id := fmt.Sprintf("mock-%d", time.Now().UnixNano())

	chatUpdates <- SetToolStatusMessage(
		id,
		name,
		fmt.Sprintf("input: %q", prompt),
		ToolStatusRunning,
	)

	time.Sleep(time.Duration(400+rand.Intn(1200)) * time.Millisecond)

	if rand.Intn(4) == 0 {
		chatUpdates <- SetToolStatusMessage(
			id,
			name,
			fmt.Sprintf("mock error from %s for %q", name, prompt),
			ToolStatusError,
		)
		chatUpdates <- AddMessage("assistant", fmt.Sprintf("%s returned an error", name))
	} else {
		chatUpdates <- SetToolStatusMessage(
			id,
			name,
			"", ToolStatusSuccess,
		)
		chatUpdates <- AddMessage("assistant", fmt.Sprintf("handled with %s", name))
	}
	statusUpdates <- SetStatus("thinking", false)
	statusUpdates <- SetStatus("status", "ready")
}

func startFastMockTraffic() {
	for {
		switch rand.Intn(3) {
		case 0:
			addRandomUserMessage()
		case 1:
			addRandomAIMessage()
		default:
			go addRandomToolCall()
		}

		time.Sleep(time.Duration(80+rand.Intn(160)) * time.Millisecond)
	}
}

func addRandomUserMessage() {
	messages := []string{
		"summarize this",
		"check the logs",
		"run the next step",
		"what changed?",
		"try again with tools",
	}
	chatUpdates <- AddMessage("user", messages[rand.Intn(len(messages))])
}

func addRandomAIMessage() {
	messages := []string{
		"I can do that.",
		"Looking at the current state.",
		"Here is a short answer.",
		"I found one likely path.",
		"Continuing with the tool result.",
	}
	chatUpdates <- AddMessage("assistant", messages[rand.Intn(len(messages))])
}

func addRandomToolCall() {
	tools := []string{"search", "read_file", "run_shell", "fetch_docs", "list_files"}
	name := tools[rand.Intn(len(tools))]
	id := fmt.Sprintf("auto-%d", time.Now().UnixNano())

	statusUpdates <- SetStatus("thinking", true)
	statusUpdates <- SetStatus("status", "running "+name)
	chatUpdates <- SetToolStatus(
		id,
		name,
		ToolStatusRunning,
		ToolMessageContent("mock call in progress"),
	)

	time.Sleep(time.Duration(120+rand.Intn(500)) * time.Millisecond)

	if rand.Intn(5) == 0 {
		chatUpdates <- SetToolStatus(
			id,
			name,
			ToolStatusError,
			ToolMessageContent("mock failure"),
		)
		statusUpdates <- SetStatus("status", name+" failed")
		return
	}

	chatUpdates <- SetToolStatus(
		id,
		name,
		ToolStatusSuccess,
		ToolMessageContent("mock result ready"),
	)
	statusUpdates <- SetStatus("status", name+" done")
	statusUpdates <- SetStatus("thinking", false)
}

type chatItem struct {
	message *ChatMessage
	tool    *ToolMessage
}

func (i chatItem) Render(format toolFormat, spin spinner.Model) string {
	switch {
	case i.message != nil:
		return messageStyle.Render(i.message.Role + ": " + i.message.Content)
	case i.tool != nil:
		if i.tool.Format != nil {
			format = *i.tool.Format
		}
		return format.Render(*i.tool, spin)
	default:
		return ""
	}
}

type toolField string

const (
	ToolFieldSpinner toolField = "spinner"
	ToolFieldState   toolField = "state"
	ToolFieldName    toolField = "name"
	ToolFieldID      toolField = "id"
	ToolFieldContent toolField = "content"
)

type toolFormat struct {
	prefix      string
	style       lipgloss.Style
	stateStyles map[ToolStatus]lipgloss.Style
	fields      map[toolField]bool
}

type ToolFormatOption func(*toolFormat)

func ToolFormat(opts ...ToolFormatOption) toolFormat {
	format := toolFormat{
		prefix:      "•",
		style:       lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		stateStyles: make(map[ToolStatus]lipgloss.Style),
		fields: map[toolField]bool{
			ToolFieldSpinner: true,
			ToolFieldState:   true,
			ToolFieldName:    true,
			ToolFieldID:      true,
			ToolFieldContent: true,
		},
	}

	for _, opt := range opts {
		opt(&format)
	}
	return format
}

func ToolPrefix(prefix string) ToolFormatOption {
	return func(format *toolFormat) {
		format.prefix = prefix
	}
}

func ToolColor(color string) ToolFormatOption {
	return func(format *toolFormat) {
		format.style = format.style.Foreground(lipgloss.Color(color))
	}
}

func ToolStyle(style lipgloss.Style) ToolFormatOption {
	return func(format *toolFormat) {
		format.style = style
	}
}

func ToolStateColor(status ToolStatus, color string) ToolFormatOption {
	return func(format *toolFormat) {
		format.stateStyles[status] = format.style.Foreground(lipgloss.Color(color))
	}
}

func ToolStateStyle(status ToolStatus, style lipgloss.Style) ToolFormatOption {
	return func(format *toolFormat) {
		format.stateStyles[status] = style
	}
}

func ToolFields(fields ...toolField) ToolFormatOption {
	return func(format *toolFormat) {
		format.fields = make(map[toolField]bool)
		for _, field := range fields {
			format.fields[field] = true
		}
	}
}

func toolSpinnerActive(messages []chatItem, defaultFormat toolFormat) bool {
	for _, message := range messages {
		if message.tool == nil || !message.tool.IsRunning() {
			continue
		}

		format := defaultFormat
		if message.tool.Format != nil {
			format = *message.tool.Format
		}
		if format.fields[ToolFieldSpinner] {
			return true
		}
	}
	return false
}

func (f toolFormat) Render(tool ToolMessage, spin spinner.Model) string {
	style := f.StyleFor(tool.State())
	var parts []string
	if f.fields[ToolFieldSpinner] && tool.IsRunning() {
		parts = append(parts, spin.View())
	}
	if f.prefix != "" {
		parts = append(parts, f.prefix)
	}
	if f.fields[ToolFieldState] {
		parts = append(parts, string(tool.State()))
	}
	if f.fields[ToolFieldName] && tool.Name != "" {
		parts = append(parts, tool.Name)
	}
	if f.fields[ToolFieldID] && tool.ID != "" {
		parts = append(parts, "#"+tool.ID)
	}

	lines := []string{strings.Join(parts, " ")}
	if f.fields[ToolFieldContent] && tool.Content != "" {
		contentLines := strings.Split(tool.Content, "\n")
		if lines[0] == "" {
			lines = contentLines
		} else {
			lines = append(lines, contentLines...)
		}
	}

	for i, line := range lines {
		lines[i] = style.Render(line)
	}
	return strings.Join(lines, "\n")
}

func (f toolFormat) StyleFor(status ToolStatus) lipgloss.Style {
	if style, ok := f.stateStyles[status]; ok {
		return style
	}
	return f.style
}

func (t ToolMessage) State() ToolStatus {
	if t.Status != "" {
		return t.Status
	}
	if t.Running {
		return ToolStatusRunning
	}
	return ToolStatusSuccess
}

func (t ToolMessage) IsRunning() bool {
	return t.State() == ToolStatusRunning
}

type StatusState map[string]any

func (s StatusState) Apply(update StatusUpdate) {
	s[update.Key] = update.Value
}

func (s StatusState) String(key string) string {
	value, ok := s[key]
	if !ok || value == nil {
		return ""
	}

	switch value := value.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return fmt.Sprint(value)
	}
}

func (s StatusState) Bool(key string) bool {
	value, ok := s[key]
	if !ok || value == nil {
		return false
	}

	switch value := value.(type) {
	case bool:
		return value
	case string:
		return value == "true" || value == "on" || value == "yes"
	default:
		return false
	}
}

type statusBar struct {
	left   []statusPart
	center []statusPart
	right  []statusPart
}

type statusSection struct {
	name  string
	parts []statusPart
}

type statusContext struct {
	state   StatusState
	spinner spinner.Model
}

type statusPart interface {
	Render(statusContext) string
	SpinnerActive(StatusState) bool
}

func StatusBar(sections ...statusSection) statusBar {
	var bar statusBar
	for _, section := range sections {
		switch section.name {
		case "left":
			bar.left = section.parts
		case "center":
			bar.center = section.parts
		case "right":
			bar.right = section.parts
		}
	}
	return bar
}

func Left(parts ...statusPart) statusSection {
	return statusSection{name: "left", parts: parts}
}

func Center(parts ...statusPart) statusSection {
	return statusSection{name: "center", parts: parts}
}

func Right(parts ...statusPart) statusSection {
	return statusSection{name: "right", parts: parts}
}

func (b statusBar) Render(state StatusState, spin spinner.Model, width int) string {
	ctx := statusContext{state: state, spinner: spin}
	left := renderStatusParts(ctx, b.left)
	center := renderStatusParts(ctx, b.center)
	right := renderStatusParts(ctx, b.right)

	if center == "" {
		return joinStatusEdges(width, left, right)
	}

	leftWidth := max(1, width/3)
	centerWidth := max(1, width-leftWidth*2)
	rightWidth := max(1, width-leftWidth-centerWidth)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.PlaceHorizontal(leftWidth, lipgloss.Left, left),
		lipgloss.PlaceHorizontal(centerWidth, lipgloss.Center, center),
		lipgloss.PlaceHorizontal(rightWidth, lipgloss.Right, right),
	)
}

func (b statusBar) SpinnerActive(state StatusState) bool {
	for _, part := range append(append(b.left, b.center...), b.right...) {
		if part.SpinnerActive(state) {
			return true
		}
	}
	return false
}

func renderStatusParts(ctx statusContext, parts []statusPart) string {
	var rendered []string
	for _, part := range parts {
		text := part.Render(ctx)
		if text != "" {
			rendered = append(rendered, text)
		}
	}
	return strings.Join(rendered, " ")
}

func joinStatusEdges(width int, left string, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

type textPart struct {
	key string
}

func Text(key string) statusPart {
	return textPart{key: key}
}

func (p textPart) Render(ctx statusContext) string {
	return ctx.state.String(p.key)
}

func (p textPart) SpinnerActive(StatusState) bool {
	return false
}

type badgePart struct {
	key string
}

func Badge(key string) statusPart {
	return badgePart{key: key}
}

func (p badgePart) Render(ctx statusContext) string {
	value := ctx.state.String(p.key)
	if value == "" {
		return ""
	}
	return p.key + ":" + value
}

func (p badgePart) SpinnerActive(StatusState) bool {
	return false
}

type spinnerPart struct {
	key string
}

func Spinner(key string) statusPart {
	return spinnerPart{key: key}
}

func (p spinnerPart) Render(ctx statusContext) string {
	if !ctx.state.Bool(p.key) {
		return ""
	}
	return ctx.spinner.View()
}

func (p spinnerPart) SpinnerActive(state StatusState) bool {
	return state.Bool(p.key)
}

func main() {
	baseStyle = lipgloss.NewStyle()

	messageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("87"))

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("231")).
		Background(lipgloss.Color("236"))

	input := textinput.New()
	input.Focus()
	spin := spinner.New(
		spinner.WithSpinner(spinner.Dot),
	)

	status := StatusState{
		"thinking": true,
		"status":   "ready",
		"model":    "llm",
		"tokens":   "0",
	}
	bar := StatusBar(
		Left(Spinner("thinking"), Text("status")),
		Right(Badge("model"), Badge("tokens"), Text("help")),
	)
	status.Apply(SetStatus("help", "pgup/pgdn: scroll | ctrl+end: bottom | ctrl+c: quit"))

	m := chatModel{
		input:         input,
		spinner:       spin,
		statusUpdates: statusUpdates,
		chatUpdates:   chatUpdates,
		statusBar:     bar,
		status:        status,
		spinnerActive: bar.SpinnerActive(status),
		toolIndex:     make(map[string]int),
		toolFormat: ToolFormat(
			ToolColor("220"),
			ToolStateColor(ToolStatusRunning, "220"),
			ToolStateColor(ToolStatusSuccess, "82"),
			ToolStateColor(ToolStatusError, "196"),
			ToolPrefix("•"),
			ToolFields(ToolFieldSpinner, ToolFieldState, ToolFieldName, ToolFieldID, ToolFieldContent),
		),
	}

	go startFastMockTraffic()

	prog := tea.NewProgram(m)
	_, err := prog.Run()
	if err != nil {
		panic(err)
	}
}
