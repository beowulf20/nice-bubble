package chat

import "charm.land/lipgloss/v2"

type Styles struct {
	Base      lipgloss.Style
	Message   lipgloss.Style
	Thinking  lipgloss.Style
	Input     lipgloss.Style
	Slash     lipgloss.Style
	SlashPick lipgloss.Style
	StatusBar lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Base:      lipgloss.NewStyle(),
		Message:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Thinking:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),
		Input:     lipgloss.NewStyle().Foreground(lipgloss.Color("87")),
		Slash:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		SlashPick: lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		StatusBar: lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("236")),
	}
}
