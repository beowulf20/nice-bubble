# chat

`pkg/chat` is the first `nice-bubble` component: a Bubble Tea chat surface for
assistant, agent, and tool-heavy terminal apps.

It gives you the UI shell. Your app still owns model calls, tool execution,
persistence, prompts, and business logic.

## What It Handles

- Pinned text input above the status bar.
- No alt-screen requirement, so terminal scrollback stays available.
- User, assistant, system, and thinking messages.
- Streaming assistant and thinking messages.
- Tool call messages with running, success, and error states.
- Optional tool spinner.
- Status bar DSL with left, center, and right sections.
- Slash command registration, preview, arrow selection, and handlers.
- Channel-driven updates for chat messages, status, and slash commands.
- Scrollback controls for long histories.
- Configurable Lip Gloss styles, tool formatting, status bar, and empty state.

## Quick Start

```go
package main

import (
    tea "charm.land/bubbletea/v2"

    "github.com/beowulf20/nice-bubble/pkg/chat"
)

func main() {
    chatUpdates := make(chan chat.ChatUpdate, 16)
    statusUpdates := make(chan chat.StatusUpdate, 8)
    slashUpdates := make(chan chat.SlashCommand, 8)

    m := chat.New(
        chat.WithChatUpdates(chatUpdates),
        chat.WithStatusUpdates(statusUpdates),
        chat.WithSlashUpdates(slashUpdates),
        chat.WithEmptyMessage(""),
    )

    go func() {
        chatUpdates <- chat.AddAIMessage("ready")
    }()

    if _, err := tea.NewProgram(m).Run(); err != nil {
        panic(err)
    }
}
```

## Model Options

Create a model with `chat.New(opts...)`.

```go
m := chat.New(
    chat.WithChatUpdates(chatUpdates),
    chat.WithStatusUpdates(statusUpdates),
    chat.WithSlashUpdates(slashUpdates),
    chat.WithStatus(status),
    chat.WithStatusBar(bar),
    chat.WithToolFormat(format),
    chat.WithStyles(styles),
    chat.WithSpinner(spinnerModel),
    chat.WithEmptyMessage(""),
)
```

Options:

- `WithChatUpdates(<-chan ChatUpdate)`: receive messages, streams, and tools.
- `WithStatusUpdates(<-chan StatusUpdate)`: receive status changes.
- `WithSlashUpdates(<-chan SlashCommand)`: receive dynamic slash commands.
- `WithStatus(StatusState)`: set initial status values.
- `WithStatusBar(StatusBarModel)`: replace the default status bar.
- `WithToolFormat(ToolFormatModel)`: replace the default tool renderer.
- `WithMessagePrefix(role, prefix)`: replace one rendered role prefix.
- `WithMessagePrefixes(MessagePrefixes)`: replace several rendered role
  prefixes.
- `WithRoleStyle(role, lipgloss.Style)`: replace one rendered role style.
- `WithRoleStyles(RoleStyles)`: replace several rendered role styles.
- `WithStyles(Styles)`: replace the default Lip Gloss styles.
- `WithSpinner(spinner.Model)`: replace the spinner model.
- `WithEmptyMessage(string)`: set text shown before any messages. Empty string
  means no placeholder.

## Embedding In Another Model

You can run `chat.Model` as the whole app, or embed it inside a larger Bubble
Tea model.

When embedding, forward `Init`, `Update`, and render `ViewContent()` wherever
you want the chat block to appear.

```go
type Model struct {
    chat chat.Model
}

func (m Model) Init() tea.Cmd {
    return m.chat.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    child, cmd := m.chat.Update(msg)
    if chatModel, ok := child.(chat.Model); ok {
        m.chat = chatModel
    }
    return m, cmd
}

func (m Model) View() tea.View {
    return tea.NewView(m.chat.ViewContent())
}
```

If your parent reserves space for headers, side panels, or footers, call
`SetSize(width, height)` on the chat model with the space assigned to chat.

```go
m.chat = m.chat.SetSize(width, chatHeight)
```

## Channels

The component waits on one update from each configured channel, applies it, then
waits for the next one. Channels may be buffered or unbuffered.

```go
statusUpdates <- chat.SetStatus("status", "thinking")
chatUpdates <- chat.AddUserMessage("summarize this")
slashUpdates <- command
```

You can also mutate the model directly from slash handlers:

```go
m.SetStatus("model", "gpt")
m.AddMessage(chat.ChatMessage{Role: "system", Content: "model changed"})
m.Clear()
```

## Chat Messages

Use the helper constructors for common roles:

```go
chatUpdates <- chat.AddUserMessage("hello")
chatUpdates <- chat.AddAIMessage("hi")
chatUpdates <- chat.AddThinkingMessage("checking context")
chatUpdates <- chat.AddMessage("system", "connected")
```

Or build a `ChatMessage` directly:

```go
m.AddMessage(chat.ChatMessage{
    ID:      "msg-1",
    Role:    "assistant",
    Content: "done",
})
```

Roles are rendered with prefixes. Defaults are `user: `, `assistant: `,
`system: `, and `thinking: `. The `thinking` role uses the `Styles.Thinking`
style.

Customize prefixes:

```go
m := chat.New(
    chat.WithMessagePrefix("user", "you: "),
    chat.WithMessagePrefix("assistant", "ai: "),
)
```

Or configure several at once:

```go
m := chat.New(
    chat.WithMessagePrefixes(chat.MessagePrefixes{
        "user":      "> ",
        "assistant": "< ",
    }),
)
```

Use an empty prefix to render content without a role label.

## Streaming

Assistant streams:

```go
id := "answer-1"
chatUpdates <- chat.StartAIStream(id)
chatUpdates <- chat.StreamAIMessage(id, "hello ")
chatUpdates <- chat.StreamAIMessage(id, "world")
chatUpdates <- chat.FinishAIStream(id)
```

Thinking streams:

```go
id := "thinking-1"
chatUpdates <- chat.StartThinkingStream(id)
chatUpdates <- chat.StreamThinkingMessage(id, "checking ")
chatUpdates <- chat.StreamThinkingMessage(id, "tools")
chatUpdates <- chat.FinishThinkingStream(id)
```

Replace the full stream content instead of appending deltas:

```go
chatUpdates <- chat.SetAIStream("answer-1", "replacement text")
chatUpdates <- chat.SetThinkingStream("thinking-1", "replacement thinking")
```

Streaming messages show a cursor marker until `FinishAIStream` or
`FinishThinkingStream` is applied.

## Tool Calls

Tool calls are upserted by `ID`. Sending another tool message with the same ID
updates the existing line instead of appending a new one.

```go
id := "tool-1"

chatUpdates <- chat.SetToolStatus(
    id,
    "search",
    chat.ToolStatusRunning,
    chat.ToolMessageContent("query: nice bubble"),
)

chatUpdates <- chat.SetToolStatus(
    id,
    "search",
    chat.ToolStatusSuccess,
    chat.ToolMessageContent("found 3 results"),
)
```

Simpler helper:

```go
chatUpdates <- chat.SetToolStatusMessage(
    "tool-1",
    "search",
    "found 3 results",
    chat.ToolStatusSuccess,
)
```

Legacy boolean helper:

```go
chatUpdates <- chat.SetToolMessage("tool-1", "search", "running", true)
```

Tool statuses:

- `ToolStatusRunning`
- `ToolStatusSuccess`
- `ToolStatusError`

`ToolStatusSucess` exists as a typo-compatible alias for `ToolStatusSuccess`.

## Tool Formatting DSL

Default tool format:

- color `220`
- prefix `•`
- running color `220`
- success color `82`
- error color `196`
- fields: spinner, state, name, id, content

Customize globally:

```go
format := chat.ToolFormat(
    chat.ToolColor("220"),
    chat.ToolStateColor(chat.ToolStatusRunning, "220"),
    chat.ToolStateColor(chat.ToolStatusSuccess, "82"),
    chat.ToolStateColor(chat.ToolStatusError, "196"),
    chat.ToolPrefix("•"),
    chat.ToolFields(
        chat.ToolFieldSpinner,
        chat.ToolFieldState,
        chat.ToolFieldName,
        chat.ToolFieldID,
        chat.ToolFieldContent,
    ),
)

m := chat.New(chat.WithToolFormat(format))
```

Customize one tool message:

```go
chatUpdates <- chat.SetToolStatus(
    "tool-1",
    "search",
    chat.ToolStatusRunning,
    chat.ToolMessageFormatOptions(
        chat.ToolPrefix(""),
        chat.ToolFields(chat.ToolFieldSpinner, chat.ToolFieldName),
    ),
)
```

Available fields:

- `ToolFieldSpinner`
- `ToolFieldState`
- `ToolFieldName`
- `ToolFieldID`
- `ToolFieldContent`

Formatting options:

- `ToolPrefix(string)`
- `ToolColor(string)`
- `ToolStyle(lipgloss.Style)`
- `ToolStateColor(ToolStatus, string)`
- `ToolStateStyle(ToolStatus, lipgloss.Style)`
- `ToolFields(...ToolField)`

## Status Updates

`StatusState` is a `map[string]any`. Values are converted to strings with
`fmt.Sprint`, except strings and `fmt.Stringer` values use their direct string
form. Booleans can be real `bool` values or string values `"true"`, `"on"`, or
`"yes"`.

```go
status := chat.StatusState{
    "thinking": false,
    "status":   "ready",
    "model":    "llm",
    "tokens":   "0",
}

statusUpdates <- chat.SetStatus("thinking", true)
statusUpdates <- chat.SetStatus("status", "running search")
statusUpdates <- chat.SetStatus("tokens", "128")
```

Direct model mutation is also available:

```go
m.SetStatus("status", "ready")
```

## Status Bar DSL

The default status bar is:

```go
chat.StatusBar(
    chat.Left(chat.Spinner("thinking"), chat.Text("status")),
    chat.Right(chat.Badge("model"), chat.Badge("tokens"), chat.Text("help")),
)
```

Build your own:

```go
bar := chat.StatusBar(
    chat.Left(
        chat.Spinner("thinking"),
        chat.Text("status"),
    ),
    chat.Center(
        chat.Badge("session"),
    ),
    chat.Right(
        chat.Badge("model"),
        chat.Badge("tokens"),
    ),
)

m := chat.New(chat.WithStatusBar(bar))
```

Status parts:

- `Text(key)`: renders only the value.
- `Badge(key)`: renders `key:value`.
- `Spinner(key)`: renders the spinner while the status key is truthy.

The status bar spinner uses the same status bar style as the rest of the bar.

## Slash Commands

Register commands through the slash channel:

```go
chat.RegisterSlashCommand(
    slashUpdates,
    "model",
    "switch active model",
    func(m *chat.Model, args string) {
        if args == "" {
            m.AddMessage(chat.ChatMessage{Role: "system", Content: "usage: /model <name>"})
            return
        }
        m.SetStatus("model", args)
    },
)
```

Or create a command value:

```go
command, ok := chat.NewSlashCommand("clear", "clear chat history", func(m *chat.Model, args string) {
    m.Clear()
})
if ok {
    slashUpdates <- command
}
```

Command names are sanitized:

- leading `/` is ignored
- spaces are removed
- only letters, digits, `-`, and `_` remain
- names are lowercased

Typing `/` opens a command preview below the input and above the status bar. Use
up/down arrows to select a command and enter to execute it. Typing `/name args`
and pressing enter passes `args` to the handler.

Unknown commands set the `status` key to `unknown command: /name`.

## Styles

Default styles are returned by `DefaultStyles()`.

```go
styles := chat.DefaultStyles()
styles.Message = styles.Message.Foreground(lipgloss.Color("252"))
styles.Thinking = styles.Thinking.Foreground(lipgloss.Color("245")).Italic(true)
styles.Input = styles.Input.Foreground(lipgloss.Color("87"))
styles.SlashPick = styles.SlashPick.Foreground(lipgloss.Color("220"))
styles.StatusBar = styles.StatusBar.
    Foreground(lipgloss.Color("231")).
    Background(lipgloss.Color("236"))

m := chat.New(chat.WithStyles(styles))
```

Messages also have per-role styles. Defaults give `user`, `assistant`, and
`system` different colors.

```go
m := chat.New(
    chat.WithRoleStyle("user", lipgloss.NewStyle().Foreground(lipgloss.Color("87"))),
    chat.WithRoleStyle("assistant", lipgloss.NewStyle().Foreground(lipgloss.Color("252"))),
    chat.WithRoleStyle("system", lipgloss.NewStyle().Foreground(lipgloss.Color("220"))),
)
```

Or configure several at once:

```go
m := chat.New(
    chat.WithRoleStyles(chat.RoleStyles{
        "user":      lipgloss.NewStyle().Foreground(lipgloss.Color("87")),
        "assistant": lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
        "system":    lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
    }),
)
```

Style fields:

- `Base`
- `Message`
- `Thinking`
- `Input`
- `Slash`
- `SlashPick`
- `StatusBar`

Role style helpers:

- `DefaultRoleStyles()`
- `WithRoleStyle(role, style)`
- `WithRoleStyles(styles)`

## Input And Key Bindings

The input is focused by default and stays directly above the status bar.

Built-in keys:

- `ctrl+c`: quit.
- `enter`: submit input or execute slash command.
- `up` / `down`: select slash command when preview is visible.
- `pgup` / `pageup`: scroll up half the chat viewport.
- `pgdown` / `pagedown`: scroll down half the chat viewport.
- `ctrl+home`: jump to oldest visible history.
- `ctrl+end`: jump back to newest messages.

Normal terminal scrollback also works because the component does not require
alt-screen.

## Public Model Helpers

```go
m.AddMessage(chat.ChatMessage{Role: "system", Content: "hello"})
m.ApplyChatUpdate(chat.AddAIMessage("done"))
m.Clear()
m.SetStatus("status", "ready")

messages := m.MessageCount()
tools := m.ToolCount()
offset := m.ScrollOffset()
commands := m.SlashCommands()

m = m.SetSize(width, height)
content := m.ViewContent()
```

## Running The Example

From the repository root:

```sh
go run ./examples/chat
go run ./examples/chat-workflow
go run ./examples/workflow
```

Useful commands in the example:

- `/help`
- `/mock`
- `/mock on`
- `/mock off`
- `/tool`
- `/model <name>`
- `/context`
- `/tools`
- `/clear`
