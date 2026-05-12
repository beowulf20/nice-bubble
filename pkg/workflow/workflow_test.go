package workflow

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderBranchPathsAlignsStepsAtSameDepth(t *testing.T) {
	m := New(
		Paths(
			Route("A", "B", "C"),
			Route("A", "D"),
		),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(60)
	bLine := lineContaining(view, "B")
	dLine := lineContaining(view, "D")

	bColumn := displayColumn(bLine, "B")
	dColumn := displayColumn(dLine, "D")
	if bColumn == -1 || dColumn == -1 {
		t.Fatalf("expected rendered branch labels B and D:\n%s", view)
	}
	if bColumn != dColumn {
		t.Fatalf("expected B and D to align, got B column %d and D column %d:\n%s", bColumn, dColumn, view)
	}
}

func TestWithFitStepsKeepsLongLabelsOnOneLine(t *testing.T) {
	label := "A much longer step label"
	m := New(
		Steps("Plan", label),
		WithFitSteps(),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(20)
	if !strings.Contains(lineContaining(view, label), label) {
		t.Fatalf("expected long label to fit on one line:\n%s", view)
	}
}

func TestWithFitStepsAlignsBranchColumnsWithDifferentLabelLengths(t *testing.T) {
	m := New(
		Paths(
			Route("Start", "B", "Finish"),
			Route("Start", "Longer D"),
		),
		WithFitSteps(),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(20)
	bColumn := labelBoxLeft(view, "B")
	dColumn := labelBoxLeft(view, "Longer D")
	if bColumn == -1 || dColumn == -1 {
		t.Fatalf("expected rendered branch labels B and Longer D:\n%s", view)
	}
	if bColumn != dColumn {
		t.Fatalf("expected B and Longer D boxes to align, got B column %d and Longer D column %d:\n%s", bColumn, dColumn, view)
	}
}

func labelBoxLeft(view string, label string) int {
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		labelIndex := strings.Index(line, label)
		if labelIndex == -1 || i == 0 {
			continue
		}

		top := lines[i-1]
		labelColumn := displayColumn(line, label)
		boxColumn := -1
		for index := strings.Index(top, "╭"); index != -1; {
			column := lipgloss.Width(top[:index])
			if column > labelColumn {
				break
			}
			boxColumn = column
			next := index + len("╭")
			if next >= len(top) {
				break
			}
			offset := strings.Index(top[next:], "╭")
			if offset == -1 {
				break
			}
			index = next + offset
		}
		if boxColumn == -1 {
			return -1
		}
		return boxColumn
	}
	return -1
}

func displayColumn(line string, substr string) int {
	index := strings.Index(line, substr)
	if index == -1 {
		return -1
	}
	return lipgloss.Width(line[:index])
}

func lineContaining(s string, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

func testStyles() Styles {
	step := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Align(lipgloss.Center).
		Height(3)

	return Styles{
		Base:       lipgloss.NewStyle().Align(lipgloss.Center),
		Step:       step,
		ActiveStep: step,
		Arrow:      lipgloss.NewStyle().Height(3).AlignVertical(lipgloss.Center),
		Title:      lipgloss.NewStyle(),
		Help:       lipgloss.NewStyle(),
	}
}
