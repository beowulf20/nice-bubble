package chat

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

type ToolMessage struct {
	ID      string
	Name    string
	Content string
	Running bool
	Status  ToolStatus
	Format  *ToolFormatModel
}

type ToolStatus string

const (
	ToolStatusRunning ToolStatus = "running"
	ToolStatusError   ToolStatus = "error"
	ToolStatusSuccess ToolStatus = "success"
	ToolStatusSucess  ToolStatus = ToolStatusSuccess
)

type ToolMessageOption func(*ToolMessage)

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
	return ChatUpdate{Tool: &tool}
}

func ToolMessageContent(content string) ToolMessageOption {
	return func(tool *ToolMessage) {
		tool.Content = content
	}
}

func ToolMessageFormat(format ToolFormatModel) ToolMessageOption {
	return func(tool *ToolMessage) {
		tool.Format = &format
	}
}

func ToolMessageFormatOptions(opts ...ToolFormatOption) ToolMessageOption {
	return ToolMessageFormat(ToolFormat(opts...))
}

func ToolMessageFields(fields ...ToolField) ToolMessageOption {
	return ToolMessageFormat(ToolFormat(
		ToolPrefix(""),
		ToolFields(fields...),
	))
}

func (m *Model) setToolMessage(tool ToolMessage) {
	if m.toolIndex == nil {
		m.toolIndex = make(map[string]int)
	}

	if tool.ID == "" {
		m.messages = append(m.messages, chatItem{tool: &tool})
		return
	}

	if index, ok := m.toolIndex[tool.ID]; ok {
		m.messages[index] = chatItem{tool: &tool}
		return
	}

	m.toolIndex[tool.ID] = len(m.messages)
	m.messages = append(m.messages, chatItem{tool: &tool})
}

func (m Model) ToolCount() int {
	return len(m.toolIndex)
}

type ToolField string

const (
	ToolFieldSpinner ToolField = "spinner"
	ToolFieldState   ToolField = "state"
	ToolFieldName    ToolField = "name"
	ToolFieldID      ToolField = "id"
	ToolFieldContent ToolField = "content"
)

type ToolFormatModel struct {
	prefix      string
	style       lipgloss.Style
	stateStyles map[ToolStatus]lipgloss.Style
	fields      map[ToolField]bool
}

type ToolFormatOption func(*ToolFormatModel)

func DefaultToolFormat() ToolFormatModel {
	return ToolFormat(
		ToolColor("220"),
		ToolStateColor(ToolStatusRunning, "220"),
		ToolStateColor(ToolStatusSuccess, "82"),
		ToolStateColor(ToolStatusError, "196"),
		ToolPrefix("•"),
		ToolFields(ToolFieldSpinner, ToolFieldState, ToolFieldName, ToolFieldID, ToolFieldContent),
	)
}

func ToolFormat(opts ...ToolFormatOption) ToolFormatModel {
	format := ToolFormatModel{
		prefix:      "•",
		style:       lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		stateStyles: make(map[ToolStatus]lipgloss.Style),
		fields: map[ToolField]bool{
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
	return func(format *ToolFormatModel) {
		format.prefix = prefix
	}
}

func ToolColor(color string) ToolFormatOption {
	return func(format *ToolFormatModel) {
		format.style = format.style.Foreground(lipgloss.Color(color))
	}
}

func ToolStyle(style lipgloss.Style) ToolFormatOption {
	return func(format *ToolFormatModel) {
		format.style = style
	}
}

func ToolStateColor(status ToolStatus, color string) ToolFormatOption {
	return func(format *ToolFormatModel) {
		format.stateStyles[status] = format.style.Foreground(lipgloss.Color(color))
	}
}

func ToolStateStyle(status ToolStatus, style lipgloss.Style) ToolFormatOption {
	return func(format *ToolFormatModel) {
		format.stateStyles[status] = style
	}
}

func ToolFields(fields ...ToolField) ToolFormatOption {
	return func(format *ToolFormatModel) {
		format.fields = make(map[ToolField]bool)
		for _, field := range fields {
			format.fields[field] = true
		}
	}
}

func toolSpinnerActive(messages []chatItem, defaultFormat ToolFormatModel) bool {
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

func (f ToolFormatModel) Render(tool ToolMessage, spin spinner.Model) string {
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

func (f ToolFormatModel) StyleFor(status ToolStatus) lipgloss.Style {
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
