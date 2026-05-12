package main

import (
	"flag"

	tea "charm.land/bubbletea/v2"

	"github.com/beowulf20/nice-bubble/pkg/workflow"
)

func main() {
	fitSteps := flag.Bool("fit-steps", false, "size workflow steps to fit their labels")
	flag.Parse()

	opts := []workflow.Option{
		workflow.Paths(
			workflow.Route("A", "B", "C"),
			workflow.Route("A", "D"),
		),
		workflow.WithTitle("workflow"),
		workflow.WithHelp("multiple paths: A -> B -> C or A -> D"),
	}
	if *fitSteps {
		opts = append(opts, workflow.WithFitSteps())
	}

	m := workflow.New(opts...)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		panic(err)
	}
}
