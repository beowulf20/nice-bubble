package chat

import (
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func TestSlashHandlerStartingThinkingSchedulesSpinnerTick(t *testing.T) {
	m := New()
	m.RegisterSlashCommand(SlashCommand{
		Name: "think",
		Handler: func(m *Model, args string) {
			m.SetStatus("thinking", true)
		},
	})
	m.input.SetValue("/think")

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("expected spinner tick command, got nil")
	}

	if !containsSpinnerTick(cmd) {
		t.Fatalf("expected spinner.TickMsg, got %T", cmd())
	}
}

func containsSpinnerTick(cmd tea.Cmd) bool {
	msg := cmd()
	switch msg := msg.(type) {
	case spinner.TickMsg:
		return true
	case tea.BatchMsg:
		for _, cmd := range msg {
			if containsSpinnerTick(cmd) {
				return true
			}
		}
	}
	return false
}
