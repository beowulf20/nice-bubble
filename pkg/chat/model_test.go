package chat

import (
	"strings"
	"testing"
	"time"

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

func TestThinkingRenderModes(t *testing.T) {
	t.Run("visible", func(t *testing.T) {
		m := New()
		m.ApplyChatUpdate(AddThinkingMessage("checking context"))

		lines := strings.Join(m.chatLines(), "\n")
		if !strings.Contains(lines, "thinking: checking context") {
			t.Fatalf("expected visible thinking content, got %q", lines)
		}
	})

	t.Run("collapsed", func(t *testing.T) {
		started := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
		m := New(WithReasoningMode(ReasoningCollapsed))
		m.AddMessage(ChatMessage{
			Role:           "thinking",
			Content:        "reasoning",
			ThinkingStatus: ThinkingDone,
			StartedAt:      started,
			FinishedAt:     started.Add(1800 * time.Millisecond),
		})

		lines := strings.Join(m.chatLines(), "\n")
		if !strings.Contains(lines, "thinking 1.8s, 9 chars") {
			t.Fatalf("expected collapsed thinking summary, got %q", lines)
		}
		if strings.Contains(lines, "reasoning") {
			t.Fatalf("expected collapsed mode to hide raw content, got %q", lines)
		}
	})

	t.Run("hidden", func(t *testing.T) {
		m := New(WithReasoningMode(ReasoningHidden))
		m.ApplyChatUpdate(AddUserMessage("hello"))
		m.ApplyChatUpdate(AddThinkingMessage("private reasoning"))
		m.ApplyChatUpdate(AddAIMessage("done"))

		lines := strings.Join(m.chatLines(), "\n")
		if strings.Contains(lines, "private reasoning") || strings.Contains(lines, "thinking:") {
			t.Fatalf("expected hidden thinking to be omitted, got %q", lines)
		}
		if !strings.Contains(lines, "user: hello") || !strings.Contains(lines, "assistant: done") {
			t.Fatalf("expected non-thinking messages to remain, got %q", lines)
		}
	})
}

func TestThinkingStreamLifecycleAndStatusOverrides(t *testing.T) {
	m := New()

	m.ApplyChatUpdate(StartThinkingStream("thinking:1"))
	if got := m.messages[0].message.ThinkingStatus; got != ThinkingRunning {
		t.Fatalf("expected running status on start, got %q", got)
	}
	if m.messages[0].message.StartedAt.IsZero() {
		t.Fatal("expected started timestamp on thinking stream start")
	}
	if !m.messages[0].message.Streaming {
		t.Fatal("expected thinking stream to be marked streaming")
	}

	m.ApplyChatUpdate(StreamThinkingMessage("thinking:1", "checking"))
	m.ApplyChatUpdate(FinishThinkingStream("thinking:1"))
	message := m.messages[0].message
	if got := message.Content; got != "checking" {
		t.Fatalf("expected streamed content, got %q", got)
	}
	if got := message.ThinkingStatus; got != ThinkingDone {
		t.Fatalf("expected done status on finish, got %q", got)
	}
	if message.FinishedAt.IsZero() {
		t.Fatal("expected finished timestamp on thinking stream finish")
	}
	if message.Streaming {
		t.Fatal("expected finished thinking stream not to be streaming")
	}

	m.ApplyChatUpdate(SetThinkingStatus("thinking:1", ThinkingError))
	if got := m.messages[0].message.ThinkingStatus; got != ThinkingError {
		t.Fatalf("expected explicit status override, got %q", got)
	}
}

func TestCtrlTTogglesLatestThinkingVisibility(t *testing.T) {
	m := New(WithReasoningMode(ReasoningCollapsed))
	m.ApplyChatUpdate(AddThinkingMessage("older"))
	m.ApplyChatUpdate(AddAIMessage("answer"))
	m.ApplyChatUpdate(AddThinkingMessage("latest"))

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 't', Mod: tea.ModCtrl}))
	m = updated.(Model)

	if got := m.messages[0].message.Visibility; got != ReasoningCollapsed {
		t.Fatalf("expected older thinking to stay collapsed, got %q", got)
	}
	if got := m.messages[2].message.Visibility; got != ReasoningVisible {
		t.Fatalf("expected latest thinking to become visible, got %q", got)
	}
	lines := strings.Join(m.chatLines(), "\n")
	if !strings.Contains(lines, "thinking: latest") {
		t.Fatalf("expected latest thinking content after toggle, got %q", lines)
	}

	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 't', Mod: tea.ModCtrl}))
	m = updated.(Model)
	if got := m.messages[2].message.Visibility; got != ReasoningCollapsed {
		t.Fatalf("expected latest thinking to collapse again, got %q", got)
	}
}

func TestCtrlTDoesNotRevealHiddenThinking(t *testing.T) {
	m := New(WithReasoningMode(ReasoningHidden))
	m.ApplyChatUpdate(AddThinkingMessage("private"))

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 't', Mod: tea.ModCtrl}))
	m = updated.(Model)

	if got := m.messages[0].message.Visibility; got != ReasoningHidden {
		t.Fatalf("expected hidden thinking to remain hidden, got %q", got)
	}
	if lines := strings.Join(m.chatLines(), "\n"); lines != "" {
		t.Fatalf("expected no rendered thinking lines, got %q", lines)
	}
}

func TestThinkingFilterRedactsRenderAndTranscript(t *testing.T) {
	m := New(WithThinkingFilter(func(text string) string {
		return ""
	}))
	m.ApplyChatUpdate(AddThinkingMessage("secret reasoning"))

	lines := strings.Join(m.chatLines(), "\n")
	if strings.Contains(lines, "secret reasoning") {
		t.Fatalf("expected filtered thinking not to render raw content, got %q", lines)
	}
	if !strings.Contains(lines, "[redacted]") {
		t.Fatalf("expected redacted render marker, got %q", lines)
	}

	transcript := m.Transcript()
	if len(transcript) != 1 {
		t.Fatalf("expected one transcript item, got %d", len(transcript))
	}
	if transcript[0].Content != "" {
		t.Fatalf("expected filtered transcript content, got %q", transcript[0].Content)
	}
	if transcript[0].ThinkingStatus != ThinkingRedacted {
		t.Fatalf("expected redacted transcript status, got %q", transcript[0].ThinkingStatus)
	}
	if transcript[0].Visibility != ReasoningVisible {
		t.Fatalf("expected visible transcript visibility, got %q", transcript[0].Visibility)
	}
}

func TestTokenUsageString(t *testing.T) {
	usage := TokenUsage{
		Input:     12000,
		Output:    1200,
		Cached:    9000,
		Reasoning: 600,
	}

	if got := usage.String(); got != "in 12k | out 1.2k | cached 9k | reasoning 600" {
		t.Fatalf("unexpected token usage string: %q", got)
	}
}

func TestThinkingHelperCompatibility(t *testing.T) {
	m := New()
	m.ApplyChatUpdate(AddThinkingMessage("static"))
	m.ApplyChatUpdate(StartThinkingStream("thinking:stream"))
	m.ApplyChatUpdate(SetThinkingStream("thinking:stream", "replacement"))
	m.ApplyChatUpdate(SetThinkingStreamStatus("thinking:stream", "replacement hidden", ThinkingHidden))
	m.ApplyChatUpdate(SetThinkingVisibility("thinking:stream", ReasoningCollapsed))
	m.ApplyChatUpdate(FinishThinkingStream("thinking:stream"))

	if got := m.MessageCount(); got != 2 {
		t.Fatalf("expected two thinking messages, got %d", got)
	}
	streamed := m.messages[1].message
	if streamed.Role != "thinking" {
		t.Fatalf("expected thinking stream role, got %q", streamed.Role)
	}
	if streamed.Content != "replacement hidden" {
		t.Fatalf("expected replace helper content, got %q", streamed.Content)
	}
	if streamed.Visibility != ReasoningCollapsed {
		t.Fatalf("expected explicit collapsed visibility, got %q", streamed.Visibility)
	}
	if streamed.ThinkingStatus != ThinkingDone {
		t.Fatalf("expected finish helper to preserve done compatibility, got %q", streamed.ThinkingStatus)
	}
}

func TestThinkingAndAssistantStreamsWithSameIDStaySeparate(t *testing.T) {
	m := New()

	m.ApplyChatUpdate(StartThinkingStream("turn-1"))
	m.ApplyChatUpdate(StreamThinkingMessage("turn-1", "checking"))
	m.ApplyChatUpdate(StartAIStream("turn-1"))
	m.ApplyChatUpdate(StreamAIMessage("turn-1", "answer"))

	if got := m.MessageCount(); got != 2 {
		t.Fatalf("expected separate thinking and assistant messages, got %d", got)
	}
	if got := m.messages[0].message.Role; got != "thinking" {
		t.Fatalf("expected first message to be thinking, got %q", got)
	}
	if got := m.messages[0].message.Content; got != "checking" {
		t.Fatalf("expected thinking content to stay separate, got %q", got)
	}
	if got := m.messages[1].message.Role; got != "assistant" {
		t.Fatalf("expected second message to be assistant, got %q", got)
	}
	if got := m.messages[1].message.Content; got != "answer" {
		t.Fatalf("expected assistant content to stay separate, got %q", got)
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
