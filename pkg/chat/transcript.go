package chat

type TranscriptItem struct {
	Role           string         `json:"role"`
	ID             string         `json:"id,omitempty"`
	Visibility     ReasoningMode  `json:"visibility,omitempty"`
	ThinkingStatus ThinkingStatus `json:"thinking_status,omitempty"`
	Content        string         `json:"content,omitempty"`
}

func (m Model) Transcript() []TranscriptItem {
	items := make([]TranscriptItem, 0, len(m.messages))
	for _, item := range m.messages {
		if item.message == nil {
			continue
		}
		message := *item.message
		transcript := TranscriptItem{
			Role:    message.Role,
			ID:      message.ID,
			Content: message.Content,
		}
		if message.Role == "thinking" {
			content, status := filteredThinkingContent(message, m.thinkingFilter)
			transcript.Content = content
			transcript.Visibility = m.effectiveReasoningMode(message.Visibility)
			transcript.ThinkingStatus = status
		}
		items = append(items, transcript)
	}
	return items
}
