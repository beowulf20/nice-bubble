// Package chat provides a Bubble Tea chat component for assistant, agent, and
// tool-heavy terminal interfaces.
//
// It owns the terminal interaction surface: transcript rendering, pinned input,
// slash command preview and execution, status bar rendering, tool call display,
// thinking messages, streaming assistant messages, and scrollback. The host
// application owns the actual model calls, tool execution, persistence, and
// domain behavior.
package chat
