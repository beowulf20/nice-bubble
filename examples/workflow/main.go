package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/beowulf20/nice-bubble/pkg/workflow"
)

func main() {
	opts := []workflow.Option{
		workflow.Paths(
			workflow.Route(
				"Intake",
				"Normalize",
				"Detect intent",
				"Load profile",
				"Check cache",
				"Search docs",
				"Rank context",
				"Draft plan",
				"Pick tool",
				"Call API",
				"Parse result",
				"Validate data",
				"Score confidence",
				"Compose answer",
				"Add citations",
				"Safety review",
				"Tone pass",
				"Send response",
				"Watch reply",
				"Clarify ask",
				"Update memory",
				"Queue follow-up",
				"Log metrics",
				"Summarize trace",
				"Close loop",
			),
			workflow.Route(
				"Intake",
				"Normalize",
				"Detect intent",
				"Load profile",
				"Ask clarifier",
				"Wait reply",
				"Merge answer",
				"Draft plan",
				"Pick tool",
				"Call API",
				"Parse result",
				"Validate data",
				"Score confidence",
				"Compose answer",
				"Add citations",
				"Safety review",
				"Tone pass",
				"Send response",
			),
			workflow.Route(
				"Intake",
				"Normalize",
				"Detect intent",
				"Load profile",
				"Check cache",
				"Search docs",
				"Rank context",
				"Draft plan",
				"Pick tool",
				"Run script",
				"Inspect logs",
				"Patch code",
				"Run tests",
				"Compose answer",
				"Add citations",
				"Safety review",
				"Tone pass",
				"Send response",
			),
			workflow.Route(
				"Intake",
				"Normalize",
				"Detect intent",
				"Load profile",
				"Check cache",
				"Search docs",
				"Rank context",
				"Draft plan",
				"Pick tool",
				"Call API",
				"Parse result",
				"Validate data",
				"Score confidence",
				"Compose answer",
				"Request approval",
				"Apply change",
				"Verify change",
				"Send response",
			),
		),
		workflow.WithTitle("workflow"),
		workflow.WithHelp("25-step workflow with branches along the way"),
		workflow.WithHeight(44),
		workflow.WithFitSteps(),
	}

	m := workflow.New(opts...)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		panic(err)
	}
}
