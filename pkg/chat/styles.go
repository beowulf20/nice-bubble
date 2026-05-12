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

type RoleStyles map[string]lipgloss.Style

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

func DefaultRoleStyles() RoleStyles {
	styles := DefaultStyles()
	return RoleStyles{
		"user":      lipgloss.NewStyle().Foreground(lipgloss.Color("87")),
		"assistant": styles.Message.Foreground(lipgloss.Color("252")),
		"system":    lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		"thinking":  styles.Thinking,
	}
}

func (s RoleStyles) StyleFor(role string, fallback Styles) lipgloss.Style {
	if style, ok := s[role]; ok {
		return style
	}
	if role == "thinking" {
		return fallback.Thinking
	}
	return fallback.Message
}
