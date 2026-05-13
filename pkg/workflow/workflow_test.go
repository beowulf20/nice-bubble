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

func TestRenderLongPathWrapsToNextRowWithDownArrow(t *testing.T) {
	m := New(
		Steps("A", "B", "C", "D", "E", "F"),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(48)
	if !strings.Contains(view, "↓") {
		t.Fatalf("expected wrapped workflow to include a down arrow:\n%s", view)
	}
	if strings.Contains(lineContaining(view, "A"), "D") {
		t.Fatalf("expected D to wrap to the next row:\n%s", view)
	}
	if lineContaining(view, "D") == "" {
		t.Fatalf("expected wrapped row to include D:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > 48 {
			t.Fatalf("expected wrapped line width <= 48, got %d for %q:\n%s", width, line, view)
		}
	}
}

func TestRenderLongPathWrapsSerpentineFromRowEnd(t *testing.T) {
	m := New(
		Steps(
			"Plan",
			"Retrieve context",
			"Pick tool",
			"Run tool",
			"Review result",
			"Answer",
			"Follow up",
		),
		WithFitSteps(),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(80)
	reviewCenter := labelCenter(view, "Review result")
	answerCenter := labelCenter(view, "Answer")
	arrowCenter := labelCenter(view, "↓")
	if reviewCenter == -1 || answerCenter == -1 || arrowCenter == -1 {
		t.Fatalf("expected Review result, down arrow, and Answer:\n%s", view)
	}
	if intAbs(reviewCenter-arrowCenter) > 2 {
		t.Fatalf("expected down arrow under Review result, got review center %d and arrow %d:\n%s", reviewCenter, arrowCenter, view)
	}
	if intAbs(answerCenter-arrowCenter) > 4 {
		t.Fatalf("expected down arrow to point to Answer, got answer center %d and arrow %d:\n%s", answerCenter, arrowCenter, view)
	}

	answerLine := lineContaining(view, "Answer")
	if !strings.Contains(answerLine, "Follow up") || !strings.Contains(answerLine, "<-") {
		t.Fatalf("expected wrapped row to continue right-to-left:\n%s", view)
	}
	if displayColumn(answerLine, "Follow up") > displayColumn(answerLine, "Answer") {
		t.Fatalf("expected Follow up left of Answer in reverse row:\n%s", view)
	}
}

func TestRenderLongBranchPathsWrapWithinWidth(t *testing.T) {
	m := New(
		Paths(
			Route("A", "B", "C", "D", "E", "F", "G", "H"),
			Route("A", "B", "C", "Alt D", "E", "F", "G", "H"),
			Route("A", "B", "C", "D", "E", "Alt F", "G", "H"),
		),
		WithFitSteps(),
		WithStyles(testStyles()),
	)

	view := m.renderPaths(52)
	if !strings.Contains(view, "↓") {
		t.Fatalf("expected long branch paths to wrap:\n%s", view)
	}
	if !strings.Contains(view, "Alt D") || !strings.Contains(view, "Alt F") {
		t.Fatalf("expected branch labels to render:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > 52 {
			t.Fatalf("expected wrapped branch line width <= 52, got %d for %q:\n%s", width, line, view)
		}
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

func labelCenter(view string, label string) int {
	line := lineContaining(view, label)
	if line == "" {
		return -1
	}
	left := displayColumn(line, label)
	if left == -1 {
		return -1
	}
	return left + lipgloss.Width(label)/2
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

func intAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
