package chat

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/spinner"
)

type ReasoningMode string

const (
	ReasoningHidden    ReasoningMode = "hidden"
	ReasoningCollapsed ReasoningMode = "collapsed"
	ReasoningVisible   ReasoningMode = "visible"
)

type ThinkingStatus string

const (
	ThinkingRunning  ThinkingStatus = "running"
	ThinkingDone     ThinkingStatus = "done"
	ThinkingHidden   ThinkingStatus = "hidden"
	ThinkingRedacted ThinkingStatus = "redacted"
	ThinkingError    ThinkingStatus = "error"
)

type ThinkingFilter func(text string) string

type ChatUpdate struct {
	Message *ChatMessage
	Stream  *StreamMessage
	Tool    *ToolMessage
}

type ChatMessage struct {
	ID             string
	Role           string
	Content        string
	Streaming      bool
	Visibility     ReasoningMode
	ThinkingStatus ThinkingStatus
	StartedAt      time.Time
	FinishedAt     time.Time
}

type StreamMessage struct {
	ID             string
	Role           string
	Content        string
	Delta          string
	Replace        bool
	Done           bool
	Visibility     ReasoningMode
	ThinkingStatus ThinkingStatus
	StartedAt      time.Time
	FinishedAt     time.Time
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
	if message.Role == "thinking" {
		if message.Visibility == "" {
			message.Visibility = m.effectiveReasoningMode(message.Visibility)
		}
		if message.ThinkingStatus == "" {
			message.ThinkingStatus = ThinkingDone
		}
	}
	if message.ID != "" {
		if m.messageIndex == nil {
			m.messageIndex = make(map[string]int)
		}
		m.messageIndex[messageIndexKey(message.Role, message.ID)] = len(m.messages)
		if message.Role != "thinking" {
			m.messageIndex[message.ID] = len(m.messages)
		}
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

	index, ok := m.messageIndex[messageIndexKey(role, stream.ID)]
	if !ok && role != "thinking" {
		index, ok = m.messageIndex[stream.ID]
	}
	if ok {
		item := m.messages[index]
		if item.message == nil {
			return
		}
		if stream.Replace {
			item.message.Content = stream.Content
		} else {
			item.message.Content += stream.Delta
		}
		if item.message.Role == "thinking" {
			m.applyThinkingStreamMetadata(item.message, stream)
			if stream.ThinkingStatus != "" && stream.ThinkingStatus != ThinkingRunning {
				item.message.Streaming = false
			}
		}
		if stream.Done {
			item.message.Streaming = false
		}
		m.messages[index] = item
		return
	}

	status := stream.ThinkingStatus
	startedAt := stream.StartedAt
	finishedAt := stream.FinishedAt
	visibility := stream.Visibility
	if role == "thinking" {
		if visibility == "" {
			visibility = m.effectiveReasoningMode(visibility)
		}
		if status == "" {
			if stream.Done {
				status = ThinkingDone
			} else if stream.Visibility != "" && stream.Content == "" && stream.Delta == "" && !stream.Replace {
				status = ThinkingDone
			} else {
				status = ThinkingRunning
			}
		}
		if startedAt.IsZero() && status == ThinkingRunning {
			startedAt = time.Now()
		}
		if stream.Done && finishedAt.IsZero() {
			finishedAt = time.Now()
		}
	}

	message := ChatMessage{
		ID:             stream.ID,
		Role:           role,
		Content:        stream.Content + stream.Delta,
		Streaming:      streamCreatesStreamingMessage(stream, status),
		Visibility:     visibility,
		ThinkingStatus: status,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
	}
	m.messageIndex[messageIndexKey(role, stream.ID)] = len(m.messages)
	if role != "thinking" {
		m.messageIndex[stream.ID] = len(m.messages)
	}
	m.messages = append(m.messages, chatItem{message: &message})
}

func messageIndexKey(role string, id string) string {
	return role + "\x00" + id
}

func streamCreatesStreamingMessage(stream StreamMessage, status ThinkingStatus) bool {
	if stream.Done {
		return false
	}
	if stream.Role != "thinking" {
		return true
	}
	if stream.ThinkingStatus != "" {
		return stream.ThinkingStatus == ThinkingRunning
	}
	if stream.Visibility != "" && stream.Content == "" && stream.Delta == "" && !stream.Replace {
		return false
	}
	return status == ThinkingRunning
}

func (m *Model) applyThinkingStreamMetadata(message *ChatMessage, stream StreamMessage) {
	if stream.Visibility != "" {
		message.Visibility = stream.Visibility
	}
	if message.Visibility == "" {
		message.Visibility = m.effectiveReasoningMode(message.Visibility)
	}
	if stream.StartedAt.IsZero() && message.StartedAt.IsZero() && message.ThinkingStatus == "" && !stream.Done {
		message.StartedAt = time.Now()
	}
	if !stream.StartedAt.IsZero() {
		message.StartedAt = stream.StartedAt
	}
	if !stream.FinishedAt.IsZero() {
		message.FinishedAt = stream.FinishedAt
	}
	if stream.ThinkingStatus != "" {
		message.ThinkingStatus = stream.ThinkingStatus
	} else if stream.Done {
		message.ThinkingStatus = ThinkingDone
	} else if message.ThinkingStatus == "" {
		message.ThinkingStatus = ThinkingRunning
	}
	if stream.Done && message.FinishedAt.IsZero() {
		message.FinishedAt = time.Now()
	}
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
	return ChatUpdate{
		Message: &ChatMessage{
			Role:           "thinking",
			Content:        content,
			ThinkingStatus: ThinkingDone,
		},
	}
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
			ID:             id,
			Role:           "thinking",
			Done:           true,
			ThinkingStatus: ThinkingDone,
		},
	}
}

func SetThinkingStatus(id string, status ThinkingStatus) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:             id,
			Role:           "thinking",
			ThinkingStatus: status,
		},
	}
}

func SetThinkingVisibility(id string, visibility ReasoningMode) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:         id,
			Role:       "thinking",
			Visibility: visibility,
		},
	}
}

func SetThinkingStreamStatus(id string, content string, status ThinkingStatus) ChatUpdate {
	return ChatUpdate{
		Stream: &StreamMessage{
			ID:             id,
			Role:           "thinking",
			Content:        content,
			Replace:        true,
			ThinkingStatus: status,
		},
	}
}

type chatItem struct {
	message *ChatMessage
	tool    *ToolMessage
}

func (i chatItem) Render(format ToolFormatModel, spin spinner.Model, styles Styles, prefixes MessagePrefixes, roleStyles RoleStyles, reasoningMode ReasoningMode, filter ThinkingFilter) string {
	switch {
	case i.message != nil:
		if i.message.Role == "thinking" {
			return renderThinkingMessage(*i.message, styles, prefixes, roleStyles, reasoningMode, filter)
		}
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
		rendered := item.Render(m.toolFormat, m.spinner, m.styles, m.messagePrefix, m.roleStyles, m.reasoningMode, m.thinkingFilter)
		if rendered == "" {
			continue
		}
		lines = append(lines, strings.Split(rendered, "\n")...)
	}
	return lines
}

func renderThinkingMessage(message ChatMessage, styles Styles, prefixes MessagePrefixes, roleStyles RoleStyles, reasoningMode ReasoningMode, filter ThinkingFilter) string {
	visibility := effectiveReasoningMode(reasoningMode, message.Visibility)
	if visibility == ReasoningHidden {
		return ""
	}

	content, status := filteredThinkingContent(message, filter)
	style := roleStyles.StyleFor(message.Role, styles)
	if visibility == ReasoningCollapsed {
		return style.Render(collapsedThinkingLine(message, content, status))
	}

	if status == ThinkingRedacted && content == "" {
		content = "[redacted]"
	}
	if message.Streaming {
		content += "▌"
	}
	return style.Render(prefixes.Prefix(message.Role) + content)
}

func collapsedThinkingLine(message ChatMessage, content string, status ThinkingStatus) string {
	if status == ThinkingRedacted && content == "" {
		return "thinking redacted"
	}
	if status == ThinkingHidden {
		return "thinking hidden"
	}
	if status == ThinkingError {
		return "thinking error"
	}

	line := fmt.Sprintf("thinking %s, %d chars", formatThinkingDuration(message), utf8.RuneCountInString(content))
	if message.Streaming || status == ThinkingRunning {
		line += " ▌"
	}
	return line
}

func filteredThinkingContent(message ChatMessage, filter ThinkingFilter) (string, ThinkingStatus) {
	content := message.Content
	if filter != nil {
		content = filter(content)
	}
	status := message.ThinkingStatus
	if status == "" {
		if message.Streaming {
			status = ThinkingRunning
		} else {
			status = ThinkingDone
		}
	}
	if message.Content != "" && content == "" {
		status = ThinkingRedacted
	}
	return content, status
}

func formatThinkingDuration(message ChatMessage) string {
	var duration time.Duration
	switch {
	case !message.StartedAt.IsZero() && !message.FinishedAt.IsZero():
		duration = message.FinishedAt.Sub(message.StartedAt)
	case !message.StartedAt.IsZero() && message.Streaming:
		duration = time.Since(message.StartedAt)
	}
	if duration < 0 {
		duration = 0
	}
	if duration < time.Second {
		return "0s"
	}
	tenths := int((duration + 50*time.Millisecond) / (100 * time.Millisecond))
	if tenths%10 == 0 {
		return fmt.Sprintf("%ds", tenths/10)
	}
	return fmt.Sprintf("%d.%ds", tenths/10, tenths%10)
}

func (m Model) effectiveReasoningMode(visibility ReasoningMode) ReasoningMode {
	return effectiveReasoningMode(m.reasoningMode, visibility)
}

func effectiveReasoningMode(defaultMode ReasoningMode, visibility ReasoningMode) ReasoningMode {
	if visibility != "" {
		return visibility
	}
	if defaultMode != "" {
		return defaultMode
	}
	return ReasoningVisible
}

func (m *Model) toggleLatestThinking() {
	for i := len(m.messages) - 1; i >= 0; i-- {
		message := m.messages[i].message
		if message == nil || message.Role != "thinking" {
			continue
		}
		switch m.effectiveReasoningMode(message.Visibility) {
		case ReasoningHidden:
			return
		case ReasoningCollapsed:
			message.Visibility = ReasoningVisible
		default:
			message.Visibility = ReasoningCollapsed
		}
		return
	}
}
