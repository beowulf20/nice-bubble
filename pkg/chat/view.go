package chat

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	return tea.NewView(m.ViewContent())
}

func (m Model) ViewContent() string {
	width := max(1, m.w)
	chatHeight := m.chatHeight()

	lines := m.chatLines()
	lines = m.visibleChatLines(lines, chatHeight)

	chat := lipgloss.NewStyle().
		Width(width).
		Height(chatHeight).
		Render(strings.Join(lines, "\n"))
	status := m.styles.StatusBar.Width(width).Render(m.statusBar.Render(m.status, m.spinner, width))
	input := m.styles.Input.Width(width).Render(m.input.View())
	slashPreview := m.renderSlashPreview(width)

	parts := []string{chat, input}
	if slashPreview != "" {
		parts = append(parts, slashPreview)
	}
	parts = append(parts, status)

	return m.styles.Base.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m Model) visibleChatLines(lines []string, height int) []string {
	if len(lines) <= height {
		return lines
	}

	offset := min(max(0, m.scrollOffset), max(0, len(lines)-height))
	end := len(lines) - offset
	start := max(0, end-height)
	return lines[start:end]
}

func (m Model) chatHeight() int {
	return max(1, max(4, m.h)-2-m.slashPreviewHeight())
}

func (m Model) maxScroll() int {
	return max(0, m.chatLineCount()-m.chatHeight())
}

func (m *Model) clampScroll() {
	m.scrollOffset = min(max(0, m.scrollOffset), m.maxScroll())
}

func (m Model) ScrollOffset() int {
	return m.scrollOffset
}

func (m Model) chatLineCount() int {
	return len(m.chatLines())
}
