package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/beowulf20/nice-bubble/examples/internal/chatdemo"
)

func main() {
	if _, err := tea.NewProgram(chatdemo.NewChat()).Run(); err != nil {
		panic(err)
	}
}
