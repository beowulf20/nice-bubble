package chatdemo

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/beowulf20/nice-bubble/pkg/chat"
)

var (
	statusUpdates = make(chan chat.StatusUpdate, 8)
	chatUpdates   = make(chan chat.ChatUpdate, 32)
	slashUpdates  = make(chan chat.SlashCommand, 32)
	mockEnabled   atomic.Bool
)

func NewChat() chat.Model {
	status := chat.StatusState{
		"thinking": true,
		"status":   "ready",
		"model":    "llm",
		"tokens":   "0",
	}
	status.Apply(chat.SetStatus("help", "/: commands | arrows: select | pgup/pgdn: scroll | ctrl+c: quit"))

	chatModel := chat.New(
		chat.WithStatusUpdates(statusUpdates),
		chat.WithChatUpdates(chatUpdates),
		chat.WithSlashUpdates(slashUpdates),
		chat.WithStatus(status),
	)

	registerSlashCommands()
	mockEnabled.Store(false)
	go startFastMockTraffic()

	return chatModel
}

func registerSlashCommands() {
	chat.RegisterSlashCommand(slashUpdates, "model", "switch active model", func(m *chat.Model, args string) {
		if args == "" {
			m.AddMessage(chat.ChatMessage{Role: "system", Content: "usage: /model <name>"})
			return
		}
		m.SetStatus("model", args)
		m.AddMessage(chat.ChatMessage{Role: "system", Content: "model set to " + args})
	})
	chat.RegisterSlashCommand(slashUpdates, "clear", "clear chat history", func(m *chat.Model, args string) {
		m.Clear()
		m.SetStatus("status", "cleared")
	})
	chat.RegisterSlashCommand(slashUpdates, "tools", "show available tools", func(m *chat.Model, args string) {
		m.AddMessage(chat.ChatMessage{Role: "system", Content: "tools: search, read_file, run_shell, fetch_docs, list_files"})
	})
	chat.RegisterSlashCommand(slashUpdates, "context", "inspect current context", func(m *chat.Model, args string) {
		m.AddMessage(chat.ChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("messages:%d tools:%d scroll:%d", m.MessageCount(), m.ToolCount(), m.ScrollOffset()),
		})
	})
	chat.RegisterSlashCommand(slashUpdates, "mock", "toggle random mock traffic", func(m *chat.Model, args string) {
		enable := !mockEnabled.Load()
		switch strings.ToLower(strings.TrimSpace(args)) {
		case "on", "true", "1", "yes":
			enable = true
		case "off", "false", "0", "no":
			enable = false
		}

		mockEnabled.Store(enable)
		state := "off"
		if enable {
			state = "on"
		}
		m.SetStatus("status", "mock "+state)
		m.AddMessage(chat.ChatMessage{Role: "system", Content: "mock traffic " + state})
	})
	chat.RegisterSlashCommand(slashUpdates, "help", "show help", func(m *chat.Model, args string) {
		var lines []string
		for _, command := range m.SlashCommands() {
			lines = append(lines, "/"+command.Name+"  "+command.Description)
		}
		m.AddMessage(chat.ChatMessage{Role: "system", Content: strings.Join(lines, "\n")})
	})
}

func startFastMockTraffic() {
	for {
		if !mockEnabled.Load() {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		switch rand.Intn(3) {
		case 0:
			addRandomUserMessage()
		case 1:
			go addRandomAIMessage()
		default:
			if rand.Intn(2) == 0 {
				go addRandomThinkingMessage()
			} else {
				go addRandomToolCall()
			}
		}

		time.Sleep(time.Duration(80+rand.Intn(160)) * time.Millisecond)
	}
}

func addRandomUserMessage() {
	messages := []string{
		"summarize this",
		"check the logs",
		"run the next step",
		"what changed?",
		"try again with tools",
	}
	chatUpdates <- chat.AddUserMessage(messages[rand.Intn(len(messages))])
}

func addRandomAIMessage() {
	messages := []string{
		"I can do that.",
		"Looking at the current state.",
		"Here is a short answer.",
		"I found one likely path.",
		"Continuing with the tool result.",
	}
	streamAIText(messages[rand.Intn(len(messages))])
}

func streamAIText(text string) {
	id := fmt.Sprintf("stream-%d", time.Now().UnixNano())
	chatUpdates <- chat.StartAIStream(id)
	for _, word := range strings.Fields(text) {
		chatUpdates <- chat.StreamAIMessage(id, word+" ")
		time.Sleep(time.Duration(25+rand.Intn(55)) * time.Millisecond)
	}
	chatUpdates <- chat.FinishAIStream(id)
}

func addRandomThinkingMessage() {
	messages := []string{
		"checking available context before answering",
		"deciding which tool should run next",
		"comparing recent tool output with the user request",
		"preparing a concise response",
	}
	streamThinkingText(messages[rand.Intn(len(messages))])
}

func streamThinkingText(text string) {
	id := fmt.Sprintf("thinking-%d", time.Now().UnixNano())
	chatUpdates <- chat.StartThinkingStream(id)
	for _, word := range strings.Fields(text) {
		chatUpdates <- chat.StreamThinkingMessage(id, word+" ")
		time.Sleep(time.Duration(20+rand.Intn(45)) * time.Millisecond)
	}
	chatUpdates <- chat.FinishThinkingStream(id)
}

func addRandomToolCall() {
	tools := []string{"search", "read_file", "run_shell", "fetch_docs", "list_files"}
	name := tools[rand.Intn(len(tools))]
	id := fmt.Sprintf("auto-%d", time.Now().UnixNano())

	statusUpdates <- chat.SetStatus("thinking", true)
	statusUpdates <- chat.SetStatus("status", "running "+name)
	chatUpdates <- chat.SetToolStatus(
		id,
		name,
		chat.ToolStatusRunning,
		chat.ToolMessageContent("mock call in progress"),
	)

	time.Sleep(time.Duration(120+rand.Intn(500)) * time.Millisecond)

	if rand.Intn(5) == 0 {
		chatUpdates <- chat.SetToolStatus(
			id,
			name,
			chat.ToolStatusError,
			chat.ToolMessageContent("mock failure"),
		)
		statusUpdates <- chat.SetStatus("status", name+" failed")
		return
	}

	chatUpdates <- chat.SetToolStatus(
		id,
		name,
		chat.ToolStatusSuccess,
		chat.ToolMessageContent("mock result ready"),
	)
	statusUpdates <- chat.SetStatus("status", name+" done")
	statusUpdates <- chat.SetStatus("thinking", false)
}
