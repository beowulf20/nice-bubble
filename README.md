# Bubblechat

Bubblechat is a Bubble Tea chat component for building AI and LLM-assisted terminal
interfaces faster.

The goal is to make the hard parts of chat UX feel boring: message rendering,
streaming output, slash commands, command previews, status bars, tool calls,
thinking states, and model activity should be easy to wire into an existing
Bubble Tea program without rebuilding the same chat shell every time.

> Status: early project scaffold. The public API is still being designed.

## Goals

- Easy to embed in existing Bubble Tea applications.
- Configurable chat layout, message styles, key bindings, and status text.
- First-class slash command support, including previews and command metadata.
- Built-in UI patterns for LLM workflows: streaming replies, thinking states,
  tool calls, tool results, errors, and user/system/assistant messages.
- Small surface area for app code: bring your own LLM client, agent loop, and
  tool execution logic.
- Sensible defaults with override points for teams that want a custom terminal
  chat experience.

## Planned Features

- Chat transcript model with typed message roles.
- Composer input for multiline prompts.
- Slash command registry.
- Inline slash command preview panel.
- Status bar for model state, token usage, tool state, or app-specific context.
- Tool call display with pending, running, success, and error states.
- Thinking blocks for reasoning or intermediate assistant progress.
- Streaming assistant responses.
- Configurable rendering through Lip Gloss styles.
- Bubble Tea commands and messages for integration with external agent loops.

## Intended Shape

Bubblechat is meant to be used as a component inside a larger Bubble Tea model:

```go
type Model struct {
    chat bubblechat.Model
}
```

Applications should be able to configure the component with options similar to:

```go
chat := bubblechat.New(
    bubblechat.WithSlashCommands(commands),
    bubblechat.WithStatusBar(status),
    bubblechat.WithStyles(styles),
)
```

The app owns the AI behavior. Bubblechat owns the terminal chat interaction.

## Slash Commands

Slash commands are a core part of the design, not an afterthought. The component
should support command discovery and previews before execution:

```text
/model      Switch active model
/clear      Clear chat history
/tools      Show available tools
/context    Inspect current context
```

Expected command metadata:

- Name and aliases.
- Description.
- Arguments.
- Preview renderer.
- Execute callback or message emission.
- Enabled/disabled state.

## LLM UI States

Bubblechat aims to cover common AI chat states directly:

- `thinking`: assistant is planning or reasoning.
- `streaming`: assistant text is arriving incrementally.
- `tool_call`: assistant requested an external tool.
- `tool_result`: tool execution finished.
- `error`: model, network, or tool failure.
- `idle`: chat is ready for user input.

## Installation

Once the package has an initial public API:

```sh
go get github.com/beowulf20/bubblechat
```

## Development

```sh
go test ./...
```

## License

License TBD.
