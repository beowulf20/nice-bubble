package chat

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
)

type StatusUpdate struct {
	Key   string
	Value any
}

func SetStatus(key string, value any) StatusUpdate {
	return StatusUpdate{Key: key, Value: value}
}

func (m *Model) SetStatus(key string, value any) {
	m.status.Apply(SetStatus(key, value))
	m.spinnerActive = m.shouldSpin()
}

type StatusState map[string]any

func (s StatusState) Apply(update StatusUpdate) {
	s[update.Key] = update.Value
}

func (s StatusState) String(key string) string {
	value, ok := s[key]
	if !ok || value == nil {
		return ""
	}

	switch value := value.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return fmt.Sprint(value)
	}
}

func (s StatusState) Bool(key string) bool {
	value, ok := s[key]
	if !ok || value == nil {
		return false
	}

	switch value := value.(type) {
	case bool:
		return value
	case string:
		return value == "true" || value == "on" || value == "yes"
	default:
		return false
	}
}

type StatusBarModel struct {
	left   []StatusPart
	center []StatusPart
	right  []StatusPart
}

type StatusSection struct {
	name  string
	parts []StatusPart
}

type StatusContext struct {
	state   StatusState
	spinner spinner.Model
}

func (ctx StatusContext) State() StatusState {
	return ctx.state
}

func (ctx StatusContext) String(key string) string {
	return ctx.state.String(key)
}

func (ctx StatusContext) Bool(key string) bool {
	return ctx.state.Bool(key)
}

func (ctx StatusContext) SpinnerView() string {
	return ctx.spinner.View()
}

type StatusPart interface {
	Render(StatusContext) string
	SpinnerActive(StatusState) bool
}

func DefaultStatusBar() StatusBarModel {
	return StatusBar(
		Left(Spinner("thinking"), Text("status")),
		Right(Badge("model"), Badge("tokens"), Text("help")),
	)
}

func StatusBar(sections ...StatusSection) StatusBarModel {
	var bar StatusBarModel
	for _, section := range sections {
		switch section.name {
		case "left":
			bar.left = section.parts
		case "center":
			bar.center = section.parts
		case "right":
			bar.right = section.parts
		}
	}
	return bar
}

func Left(parts ...StatusPart) StatusSection {
	return StatusSection{name: "left", parts: parts}
}

func Center(parts ...StatusPart) StatusSection {
	return StatusSection{name: "center", parts: parts}
}

func Right(parts ...StatusPart) StatusSection {
	return StatusSection{name: "right", parts: parts}
}

func (b StatusBarModel) Render(state StatusState, spin spinner.Model, width int) string {
	ctx := StatusContext{state: state, spinner: spin}
	left := renderStatusParts(ctx, b.left)
	center := renderStatusParts(ctx, b.center)
	right := renderStatusParts(ctx, b.right)

	if center == "" {
		return joinStatusEdges(width, left, right)
	}

	leftWidth := max(1, width/3)
	centerWidth := max(1, width-leftWidth*2)
	rightWidth := max(1, width-leftWidth-centerWidth)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.PlaceHorizontal(leftWidth, lipgloss.Left, left),
		lipgloss.PlaceHorizontal(centerWidth, lipgloss.Center, center),
		lipgloss.PlaceHorizontal(rightWidth, lipgloss.Right, right),
	)
}

func (b StatusBarModel) SpinnerActive(state StatusState) bool {
	for _, part := range append(append(b.left, b.center...), b.right...) {
		if part.SpinnerActive(state) {
			return true
		}
	}
	return false
}

func renderStatusParts(ctx StatusContext, parts []StatusPart) string {
	var rendered []string
	for _, part := range parts {
		text := part.Render(ctx)
		if text != "" {
			rendered = append(rendered, text)
		}
	}
	return strings.Join(rendered, " ")
}

func joinStatusEdges(width int, left string, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

type textPart struct {
	key string
}

func Text(key string) StatusPart {
	return textPart{key: key}
}

func (p textPart) Render(ctx StatusContext) string {
	return ctx.String(p.key)
}

func (p textPart) SpinnerActive(StatusState) bool {
	return false
}

type badgePart struct {
	key string
}

func Badge(key string) StatusPart {
	return badgePart{key: key}
}

func (p badgePart) Render(ctx StatusContext) string {
	value := ctx.String(p.key)
	if value == "" {
		return ""
	}
	return p.key + ":" + value
}

func (p badgePart) SpinnerActive(StatusState) bool {
	return false
}

type spinnerPart struct {
	key string
}

func Spinner(key string) StatusPart {
	return spinnerPart{key: key}
}

func (p spinnerPart) Render(ctx StatusContext) string {
	if !ctx.Bool(p.key) {
		return ""
	}
	return ctx.SpinnerView()
}

func (p spinnerPart) SpinnerActive(state StatusState) bool {
	return state.Bool(p.key)
}
