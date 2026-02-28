package nve

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// setupScreen writes text into a simulated screen so we can test highlighting.
func setupScreen(t *testing.T, width, height int, lines []string) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	screen.Init()
	screen.SetSize(width, height)

	for row, line := range lines {
		for col, r := range line {
			if col >= width {
				break
			}
			screen.SetContent(col, row, r, nil, tcell.StyleDefault)
		}
		// Fill remaining cols with spaces
		for col := len(line); col < width; col++ {
			screen.SetContent(col, row, ' ', nil, tcell.StyleDefault)
		}
	}
	return screen
}

func getFg(screen tcell.Screen, col, row int) tcell.Color {
	_, _, style, _ := screen.GetContent(col, row)
	fg, _, _ := style.Decompose()
	return fg
}

func isBold(screen tcell.Screen, col, row int) bool {
	_, _, style, _ := screen.GetContent(col, row)
	_, _, attrs := style.Decompose()
	return attrs&tcell.AttrBold != 0
}

func isItalic(screen tcell.Screen, col, row int) bool {
	_, _, style, _ := screen.GetContent(col, row)
	_, _, attrs := style.Decompose()
	return attrs&tcell.AttrItalic != 0
}

func TestMarkdown_Headings(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantColor tcell.Color
	}{
		{"heading1", "# Hello World", Zenburn.Heading1},
		{"heading2", "## Subheading", Zenburn.Heading2},
		{"heading3", "### Third Level", Zenburn.Heading3},
		{"heading4", "#### Fourth Level", Zenburn.Heading3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := setupScreen(t, 40, 1, []string{tt.line})
			applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)
			if fg := getFg(screen, 0, 0); fg != tt.wantColor {
				t.Errorf("color: got %v, want %v", fg, tt.wantColor)
			}
			if !isBold(screen, 0, 0) {
				t.Error("heading should be bold")
			}
		})
	}
}

func TestMarkdown_CodeFence(t *testing.T) {
	lines := []string{
		"```go",
		"func main() {}",
		"```",
	}
	screen := setupScreen(t, 40, 3, lines)
	applyMarkdownHighlighting(screen, 0, 0, 40, 3, false, Zenburn)

	// Fence line should be code color
	if fg := getFg(screen, 0, 0); fg != Zenburn.InlineCode {
		t.Errorf("code fence color: got %v, want %v", fg, Zenburn.InlineCode)
	}
	// Content inside fence should be code color
	if fg := getFg(screen, 0, 1); fg != Zenburn.InlineCode {
		t.Errorf("code block content color: got %v, want %v", fg, Zenburn.InlineCode)
	}
	// Closing fence should be code color
	if fg := getFg(screen, 0, 2); fg != Zenburn.InlineCode {
		t.Errorf("closing fence color: got %v, want %v", fg, Zenburn.InlineCode)
	}
}

func TestMarkdown_CodeFenceStartInside(t *testing.T) {
	lines := []string{
		"some code here",
		"```",
	}
	screen := setupScreen(t, 40, 2, lines)
	// Start inside a code block (fence was above visible area)
	applyMarkdownHighlighting(screen, 0, 0, 40, 2, true, Zenburn)

	// First line should be code color (we're inside a code block)
	if fg := getFg(screen, 0, 0); fg != Zenburn.InlineCode {
		t.Errorf("in-code-block content: got %v, want %v", fg, Zenburn.InlineCode)
	}
}

func TestMarkdown_Bold(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"some **bold** text"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	// "b" in "bold" is at index 7
	if !isBold(screen, 7, 0) {
		t.Error("bold text should have bold attribute")
	}
	if fg := getFg(screen, 7, 0); fg != Zenburn.Bold {
		t.Errorf("bold color: got %v, want %v", fg, Zenburn.Bold)
	}
}

func TestMarkdown_Italic(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"some *italic* text"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	// "i" in "italic" is at index 6
	if !isItalic(screen, 6, 0) {
		t.Error("italic text should have italic attribute")
	}
}

func TestMarkdown_InlineCode(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"use `code` here"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	// "c" in "code" is at index 5
	if fg := getFg(screen, 5, 0); fg != Zenburn.InlineCode {
		t.Errorf("inline code color: got %v, want %v", fg, Zenburn.InlineCode)
	}
}

func TestMarkdown_Link(t *testing.T) {
	screen := setupScreen(t, 50, 1, []string{"click [here](http://example.com) now"})
	applyMarkdownHighlighting(screen, 0, 0, 50, 1, false, Zenburn)

	// "h" in "here" at index 7
	if fg := getFg(screen, 7, 0); fg != Zenburn.LinkText {
		t.Errorf("link text color: got %v, want %v", fg, Zenburn.LinkText)
	}
	// "h" in "http" at index 14
	if fg := getFg(screen, 14, 0); fg != Zenburn.LinkURL {
		t.Errorf("link URL color: got %v, want %v", fg, Zenburn.LinkURL)
	}
}

func TestMarkdown_ListMarker(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"- list item"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	if fg := getFg(screen, 0, 0); fg != Zenburn.ListMarker {
		t.Errorf("list marker color: got %v, want %v", fg, Zenburn.ListMarker)
	}
	if !isBold(screen, 0, 0) {
		t.Error("list marker should be bold")
	}
}

func TestMarkdown_Blockquote(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"> quoted text"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	if fg := getFg(screen, 0, 0); fg != Zenburn.Blockquote {
		t.Errorf("blockquote color: got %v, want %v", fg, Zenburn.Blockquote)
	}
}

func TestMarkdown_PlainTextUnchanged(t *testing.T) {
	screen := setupScreen(t, 40, 1, []string{"just plain text"})
	applyMarkdownHighlighting(screen, 0, 0, 40, 1, false, Zenburn)

	// Plain text should retain default style
	_, _, style, _ := screen.GetContent(0, 0)
	if style != tcell.StyleDefault {
		t.Errorf("plain text style should be default, got %v", style)
	}
}

func TestCountCodeFences(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		maxLine int
		want    int
	}{
		{"no fences", "hello\nworld", 10, 0},
		{"one fence", "hello\n```\ncode", 10, 1},
		{"two fences", "```\ncode\n```\n", 10, 2},
		{"maxLine limits", "```\ncode\n```\nmore", 2, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countCodeFences(tt.text, tt.maxLine)
			if got != tt.want {
				t.Errorf("countCodeFences() = %d, want %d", got, tt.want)
			}
		})
	}
}
