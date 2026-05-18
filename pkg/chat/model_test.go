package chat

import (
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type submitObservedMsg struct{}

func TestEnterEchoesSubmittedTextByDefault(t *testing.T) {
	m := New()
	m.input.SetValue("  hello chat  ")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(Model)

	if m.input.Value() != "" {
		t.Fatalf("expected input to clear, got %q", m.input.Value())
	}
	if got := m.MessageCount(); got != 1 {
		t.Fatalf("expected one echoed message, got %d", got)
	}

	message := m.messages[0].message
	if message == nil {
		t.Fatal("expected chat message, got nil")
	}
	if message.Role != "user" || message.Content != "hello chat" {
		t.Fatalf("expected echoed user message, got role=%q content=%q", message.Role, message.Content)
	}
}

func TestEnterCanSubmitWithoutEcho(t *testing.T) {
	m := New(WithEchoSubmit(false))
	m.input.SetValue("hello parent")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(Model)

	if m.input.Value() != "" {
		t.Fatalf("expected input to clear, got %q", m.input.Value())
	}
	if got := m.MessageCount(); got != 0 {
		t.Fatalf("expected no echoed messages, got %d", got)
	}
}

func TestEnterSchedulesSubmitHandlerCommand(t *testing.T) {
	var event SubmitEvent
	m := New(WithSubmitHandler(func(ev SubmitEvent) tea.Cmd {
		event = ev
		return func() tea.Msg {
			return submitObservedMsg{}
		}
	}))
	m.input.SetValue("  run this  ")

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if event.Text != "run this" {
		t.Fatalf("expected trimmed submit event text, got %q", event.Text)
	}
	if !containsSubmitObserved(cmd) {
		t.Fatalf("expected submit handler command, got %T", cmd())
	}
}

func TestEnterWithEmptyInputDoesNotSubmitOrClear(t *testing.T) {
	called := false
	m := New(WithSubmitHandler(func(SubmitEvent) tea.Cmd {
		called = true
		return nil
	}))
	m.input.SetValue("   ")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(Model)

	if called {
		t.Fatal("expected empty input not to call submit handler")
	}
	if m.input.Value() != "   " {
		t.Fatalf("expected empty input to stay unchanged, got %q", m.input.Value())
	}
	if got := m.MessageCount(); got != 0 {
		t.Fatalf("expected no echoed messages, got %d", got)
	}
}

func TestSlashCommandDoesNotSubmit(t *testing.T) {
	called := false
	m := New(WithSubmitHandler(func(SubmitEvent) tea.Cmd {
		called = true
		return nil
	}))
	m.RegisterSlashCommand(SlashCommand{Name: "clear"})
	m.input.SetValue("/clear")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(Model)

	if called {
		t.Fatal("expected slash command not to call submit handler")
	}
	if m.input.Value() != "" {
		t.Fatalf("expected known slash command to clear input, got %q", m.input.Value())
	}
}

func TestUnknownSlashCommandDoesNotSubmit(t *testing.T) {
	called := false
	m := New(WithSubmitHandler(func(SubmitEvent) tea.Cmd {
		called = true
		return nil
	}))
	m.input.SetValue("/missing")

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = updated.(Model)

	if called {
		t.Fatal("expected unknown slash command not to call submit handler")
	}
	if m.input.Value() != "/missing" {
		t.Fatalf("expected unknown slash command to keep input, got %q", m.input.Value())
	}
}

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

func containsSubmitObserved(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	switch msg := msg.(type) {
	case submitObservedMsg:
		return true
	case tea.BatchMsg:
		for _, cmd := range msg {
			if containsSubmitObserved(cmd) {
				return true
			}
		}
	}
	return false
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
