# nice-bubble

`nice-bubble` is a small collection of Bubble Tea components I like to reuse in
my projects: polished little terminal UI pieces that are useful, configurable,
and a bit uncommon.

> Status: early project. APIs are still free to move.

## Components

### `pkg/chat`

A chat surface for assistant, agent, and tool-heavy terminal apps.

Includes pinned input, scrollback without alt-screen, streaming messages,
thinking messages, slash command previews, status bar composition, and tool call
rendering.

Docs: [pkg/chat/README.md](pkg/chat/README.md)

Example:

```sh
go run ./examples/chat
```

### `pkg/workflow`

A horizontal workflow/progress component.

Includes one or many paths, active path/step state, step styling, optional
auto-ticking, and simple composition above another component.

Docs: [pkg/workflow/README.md](pkg/workflow/README.md)

Example:

```sh
go run ./examples/workflow
```

### Composition Example

```sh
go run ./examples/chat-workflow
```

## Installation

```sh
go get github.com/beowulf20/nice-bubble
```

## Development

```sh
go test ./...
```

## License

License TBD.
