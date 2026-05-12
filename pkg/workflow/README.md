# workflow

`pkg/workflow` is a small Bubble Tea component for rendering a horizontal
workflow/progress header.

It is intentionally simple: provide steps, set the active step directly or let
it tick, and render `ViewContent()` above another component.

## Quick Start

```go
wf := workflow.New(
    workflow.Steps("Plan", "Retrieve", "Tool", "Answer"),
    workflow.WithTitle("workflow"),
    workflow.WithHelp("composable header view above chat"),
)
```

Multiple paths are supported too:

```go
wf := workflow.New(
    workflow.Paths(
        workflow.Route("A", "B", "C"),
        workflow.Route("A", "D"),
    ),
)
```

Common prefixes render once. For example, `A -> B -> C` and `A -> D` share the
`A` box, then branch with separate arrows into `B` and `D`.

In a parent Bubble Tea model:

```go
func (m Model) Init() tea.Cmd {
    return m.workflow.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    child, cmd := m.workflow.Update(msg)
    if workflowModel, ok := child.(workflow.Model); ok {
        m.workflow = workflowModel
    }
    return m, cmd
}

func (m Model) View() tea.View {
    return tea.NewView(m.workflow.ViewContent())
}
```

## Options

- `Steps(...string)`: set workflow step labels.
- `Paths(...Path)`: set multiple possible workflow routes.
- `Route(...string)`: build one route for `Paths`.
- `WithActive(int)`: set the active step index.
- `WithActivePath(int)`: set the active route index.
- `WithHeight(int)`: set rendered height.
- `WithTitle(string)`: set title text.
- `WithHelp(string)`: set help text.
- `WithTickInterval(time.Duration)`: set auto-advance interval. Use `0` to
  disable ticking.
- `WithStyles(Styles)`: replace default Lip Gloss styles.

## Helpers

- `SetWidth(int) Model`
- `Height() int`
- `Active() int`
- `ActivePath() int`
- `Steps() []string`
- `Paths() []Path`
- `Next()`
- `NextPath()`
- `SetActive(int)`
- `SetActivePath(int)`
- `ViewContent() string`

## Running Examples

From the repository root:

```sh
go run ./examples/workflow
go run ./examples/chat-workflow
```
