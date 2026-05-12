package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/beowulf20/nice-bubble/examples/internal/chatdemo"
	"github.com/beowulf20/nice-bubble/pkg/chat"
	"github.com/beowulf20/nice-bubble/pkg/workflow"
)

type appModel struct {
	chat     chat.Model
	workflow workflow.Model
}

func main() {
	m := appModel{
		chat: chatdemo.NewChat(),
		workflow: workflow.New(
			workflow.Paths(
				workflow.Route("Plan", "Retrieve", "Tool", "Answer"),
				workflow.Route("Plan", "Answer"),
			),
			workflow.WithTitle("workflow"),
			workflow.WithHelp("chat can compose with one of many workflow paths"),
		),
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		panic(err)
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(m.chat.Init(), m.workflow.Init())
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.workflow = m.workflow.SetWidth(msg.Width)
		workflowModel, workflowCmd := m.workflow.Update(msg)
		if model, ok := workflowModel.(workflow.Model); ok {
			m.workflow = model
		}

		chatHeight := max(4, msg.Height-m.workflow.Height())
		m.chat = m.chat.SetSize(msg.Width, chatHeight)
		child, cmd := m.chat.Update(tea.WindowSizeMsg{Width: msg.Width, Height: chatHeight})
		if chatModel, ok := child.(chat.Model); ok {
			m.chat = chatModel
		}
		cmds = append(cmds, workflowCmd, cmd)
		return m, tea.Batch(cmds...)
	}

	workflowModel, workflowCmd := m.workflow.Update(msg)
	if model, ok := workflowModel.(workflow.Model); ok {
		m.workflow = model
	}
	cmds = append(cmds, workflowCmd)

	child, cmd := m.chat.Update(msg)
	if chatModel, ok := child.(chat.Model); ok {
		m.chat = chatModel
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m appModel) View() tea.View {
	return tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		m.workflow.ViewContent(),
		m.chat.ViewContent(),
	))
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
