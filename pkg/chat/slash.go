package chat

import (
	"strings"
	"unicode"
)

type SlashCommand struct {
	Name        string
	Description string
	Handler     SlashCommandHandler
}

type SlashCommandHandler func(*Model, string)

func RegisterSlashCommand(updates chan<- SlashCommand, name string, description string, handlers ...SlashCommandHandler) {
	command, ok := NewSlashCommand(name, description, handlers...)
	if !ok {
		return
	}
	updates <- command
}

func NewSlashCommand(name string, description string, handlers ...SlashCommandHandler) (SlashCommand, bool) {
	name = sanitizeSlashName(strings.TrimPrefix(strings.TrimSpace(name), "/"))
	if name == "" {
		return SlashCommand{}, false
	}

	var handler SlashCommandHandler
	if len(handlers) > 0 {
		handler = handlers[0]
	}
	return SlashCommand{Name: name, Description: strings.TrimSpace(description), Handler: handler}, true
}

func (m *Model) RegisterSlashCommand(command SlashCommand) {
	if command.Name == "" {
		return
	}

	for i := range m.slashCommands {
		if m.slashCommands[i].Name == command.Name {
			m.slashCommands[i] = command
			return
		}
	}
	m.slashCommands = append(m.slashCommands, command)
}

func (m Model) SlashCommands() []SlashCommand {
	commands := make([]SlashCommand, len(m.slashCommands))
	copy(commands, m.slashCommands)
	return commands
}

func (m Model) slashPreviewHeight() int {
	return len(m.filteredSlashCommands())
}

func (m Model) slashPreviewVisible() bool {
	return m.slashPreviewHeight() > 0
}

func (m Model) slashQuery() (string, bool) {
	value := m.input.Value()
	if !strings.HasPrefix(value, "/") {
		return "", false
	}

	query := strings.TrimPrefix(value, "/")
	if strings.ContainsAny(query, " \t\r\n") {
		return "", false
	}
	return sanitizeSlashName(query), true
}

func (m Model) filteredSlashCommands() []SlashCommand {
	query, ok := m.slashQuery()
	if !ok {
		return nil
	}

	var commands []SlashCommand
	for _, command := range m.slashCommands {
		if strings.HasPrefix(command.Name, query) {
			commands = append(commands, command)
		}
	}
	return commands
}

func (m *Model) clampSlashSelection() {
	commands := m.filteredSlashCommands()
	if len(commands) == 0 {
		m.slashSelected = 0
		return
	}
	m.slashSelected = min(max(0, m.slashSelected), len(commands)-1)
}

func (m *Model) wrapSlashSelection() {
	commands := m.filteredSlashCommands()
	if len(commands) == 0 {
		m.slashSelected = 0
		return
	}
	if m.slashSelected < 0 {
		m.slashSelected = len(commands) - 1
	}
	if m.slashSelected >= len(commands) {
		m.slashSelected = 0
	}
}

func (m *Model) executeSelectedSlashCommand() bool {
	commands := m.filteredSlashCommands()
	if len(commands) == 0 {
		return false
	}

	m.clampSlashSelection()
	m.executeSlashCommand(commands[m.slashSelected], "")
	m.input.SetValue("")
	return true
}

func (m *Model) executeSlashInput() bool {
	value := strings.TrimSpace(m.input.Value())
	if !strings.HasPrefix(value, "/") {
		return false
	}

	value = strings.TrimPrefix(value, "/")
	name, args, _ := strings.Cut(value, " ")
	command, ok := m.findSlashCommand(sanitizeSlashName(name))
	if !ok {
		m.SetStatus("status", "unknown command: /"+name)
		return true
	}

	m.executeSlashCommand(command, strings.TrimSpace(args))
	m.input.SetValue("")
	return true
}

func (m Model) findSlashCommand(name string) (SlashCommand, bool) {
	for _, command := range m.slashCommands {
		if command.Name == name {
			return command, true
		}
	}
	return SlashCommand{}, false
}

func (m *Model) executeSlashCommand(command SlashCommand, args string) {
	if command.Handler == nil {
		m.SetStatus("status", "ran /"+command.Name)
		return
	}
	command.Handler(m, args)
}

func (m Model) renderSlashPreview(width int) string {
	commands := m.filteredSlashCommands()
	if len(commands) == 0 {
		return ""
	}

	lines := make([]string, 0, len(commands))
	for i, command := range commands {
		prefix := "  "
		style := m.styles.Slash
		if i == m.slashSelected {
			prefix = "> "
			style = m.styles.SlashPick
		}

		line := prefix + "/" + command.Name
		if command.Description != "" {
			line += "  " + command.Description
		}
		lines = append(lines, style.Width(width).Render(line))
	}
	return strings.Join(lines, "\n")
}

func sanitizeSlashName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}
