package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/beowulf20/nice-bubble/pkg/workflow"
)

func main() {
	m := workflow.New(
		workflow.Paths(
			workflow.Route("A", "B", "C"),
			workflow.Route("A", "D"),
		),
		workflow.WithTitle("workflow"),
		workflow.WithHelp("multiple paths: A -> B -> C or A -> D"),
	)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		panic(err)
	}
}
