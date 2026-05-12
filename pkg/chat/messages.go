package chat

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
)

type ChatUpdate struct {
	Message *ChatMessage
	Stream  *StreamMessage
	Tool    *ToolMessage
}

type ChatMessage struct {
	ID        string
	Role      string
	Content   string
	Streaming bool
}

type StreamMessage struct {
	ID      string
	Role    string
	Content string
	Delta   string
	Replace bool
	Done    bool
}

type MessagePrefixes map[string]string

func DefaultMessagePrefixes() MessagePrefixes {
	return MessagePrefixes{
		"user":      "user: ",
		"assistant": "assistant: ",
		"system":    "system: ",
		"thinking":  "thinking: ",
	}
}

func (p MessagePrefixes) Prefix(role string) string {
	if prefix, ok := p[role]; ok {
		return prefix
	}
	if role == "" {
		return ""
	}
	return role + ": "
}

func (m *Model) ApplyChatUpdate(update ChatUpdate) {
	if update.Message != nil {
		m.AddMessage(*update.Message)
	}
	if update.Stream != nil {
		m.setStreamMessage(*update.Stream)
	}
	if update.Tool != nil {
		m.setToolMessage(*update.Tool)
	}
}

func (m *Model) AddMessage(message ChatMessage) {
	if message.ID != "" {
		if m.messageIndex == nil {
			m.messageIndex = make(map[string]int)
		}
		m.messageIndex[message.ID] = len(m.messages)
	}
	m.messages = append(m.messages, chatItem{message: &message})
}

func (m *Model) Clear() {
	m.messages = nil
	m.messageIndex = make(map[string]int)
	m.toolIndex = make(map[string]int)
	m.scrollOffset = 0
}

func (m Model) MessageCount() int {
	return len(m.messages)
}

func (m *Model) setStreamMessage(stream StreamMessage) {
	if m.messageIndex == nil {
		m.messageIndex = make(map[string]int)
	}

	role := stream.Role
	if role == "" {
		role = "assistant"
	}

	if index, ok := m.messageIndex[stream.ID]; ok {
		item := m.messages[index]
		if item.message == nil {
			return
		}
		if stream.Replace {
			item.message.Content = stream.Content
		} else {
			item.message.Content += stream.Delta
		}
		if stream.Done {
			item.message.Streaming = false
		}
		m.messages[index] = item
		return
	}

	message := ChatMessage{
		ID:        stream.ID,
		Role:      role,
		Content:   stream.Content + stream.Delta,
		Streaming: !stream.Done,
	}
	m.messageIndex[stream.ID] = len(m.messages)
	m.messages = append(m.messages, chatItem{message: &message})
}

func AddMessage(role string, content string) ChatUpdate {
	return ChatUpdate{
		Message: &ChatMessage{
			Role:    role,
			Content: content,
		},
	}
}

func AddUserMessage(content string) ChatUpdate {
	return AddMessage("user", content)
}

func AddAIMessage(content string) ChatUpdate {
	return AddMessage("assistant", content)
}

func AddThinkingMessage(content string) ChatUpdate {
	return AddMessage("thinking", content)
}

func StartAIStream(id string) ChatUpdate {
	return StreamAIMessage(id, "")
}

func StreamAIMessage(id string, delta string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:    id,
			Role:  "assistant",
			Delta: delta,
		},
	}
}

func SetAIStream(id string, content string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:      id,
			Role:    "assistant",
			Content: content,
			Replace: true,
		},
	}
}

func FinishAIStream(id string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:   id,
			Role: "assistant",
			Done: true,
		},
	}
}

func StartThinkingStream(id string) ChatUpdate {
	return StreamThinkingMessage(id, "")
}

func StreamThinkingMessage(id string, delta string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:    id,
			Role:  "thinking",
			Delta: delta,
		},
	}
}

func SetThinkingStream(id string, content string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:      id,
			Role:    "thinking",
			Content: content,
			Replace: true,
		},
	}
}

func FinishThinkingStream(id string) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:   id,
			Role: "thinking",
			Done: true,
		},
	}
}

type chatItem struct {
	message *ChatMessage
	tool    *ToolMessage
}

func (i chatItem) Render(format ToolFormatModel, spin spinner.Model, styles Styles, prefixes MessagePrefixes, roleStyles RoleStyles) string {
	switch {
	case i.message != nil:
		content := i.message.Content
		if i.message.Streaming {
			content += "▌"
		}
		style := roleStyles.StyleFor(i.message.Role, styles)
		return style.Render(prefixes.Prefix(i.message.Role) + content)
	case i.tool != nil:
		if i.tool.Format != nil {
			format = *i.tool.Format
		}
		return format.Render(*i.tool, spin)
	default:
		return ""
	}
}

func (m Model) chatLines() []string {
	if len(m.messages) == 0 {
		if m.emptyMessage == "" {
			return nil
		}
		return []string{m.emptyMessage}
	}

	var lines []string
	for _, item := range m.messages {
		lines = append(lines, strings.Split(item.Render(m.toolFormat, m.spinner, m.styles, m.messagePrefix, m.roleStyles), "\n")...)
	}
	return lines
}
