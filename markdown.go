package nve

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// MarkdownStyle defines the color palette for markdown syntax highlighting.
type MarkdownStyle struct {
	Heading1   tcell.Color
	Heading2   tcell.Color
	Heading3   tcell.Color
	Bold       tcell.Color
	Italic     tcell.Color
	InlineCode tcell.Color
	LinkText   tcell.Color
	LinkURL    tcell.Color
	ListMarker tcell.Color
	Blockquote tcell.Color
}

// Zenburn is a low-contrast color scheme using muted, earthy tones (256-color palette).
var Zenburn = MarkdownStyle{
	Heading1:   tcell.PaletteColor(187), // pale yellow
	Heading2:   tcell.PaletteColor(186), // olive-yellow
	Heading3:   tcell.PaletteColor(150), // muted green
	Bold:       tcell.PaletteColor(253), // light gray
	Italic:     tcell.PaletteColor(253), // light gray
	InlineCode: tcell.PaletteColor(223), // peachy/tan
	LinkText:   tcell.PaletteColor(174), // muted rose
	LinkURL:    tcell.PaletteColor(67),  // muted blue
	ListMarker: tcell.PaletteColor(187), // pale yellow
	Blockquote: tcell.PaletteColor(108), // sage green
}

// applyMarkdownHighlighting processes visible screen rows and applies
// style-based coloring to markdown syntax elements.
func applyMarkdownHighlighting(screen tcell.Screen, x, y, width, height int, inCodeBlock bool, style MarkdownStyle) {
	for row := 0; row < height; row++ {
		line := extractLine(screen, x, y+row, width)
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		// Code fence toggle
		if strings.HasPrefix(trimmed, "```") {
			colorRange(screen, x, y+row, 0, width, style.InlineCode, false, false)
			inCodeBlock = !inCodeBlock
			continue
		}

		// Inside code block
		if inCodeBlock {
			colorRange(screen, x, y+row, 0, width, style.InlineCode, false, false)
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "# ") {
			colorRange(screen, x, y+row, 0, width, style.Heading1, true, false)
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			colorRange(screen, x, y+row, 0, width, style.Heading2, true, false)
			continue
		}
		if strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "#### ") ||
			strings.HasPrefix(trimmed, "##### ") || strings.HasPrefix(trimmed, "###### ") {
			colorRange(screen, x, y+row, 0, width, style.Heading3, true, false)
			continue
		}

		// Blockquote
		if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
			colorRange(screen, x, y+row, 0, width, style.Blockquote, false, false)
			continue
		}

		// List markers: "- " or "* " at line start (with optional indent)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			colorRange(screen, x, y+row, indent, indent+2, style.ListMarker, true, false)
		}

		// Inline patterns
		applyInlineHighlighting(screen, x, y+row, line, style)
	}
}

// applyInlineHighlighting applies bold, italic, inline code, and link styles within a line.
// Inline code is applied last so it takes precedence (e.g., `**not bold**` stays code-styled).
// The line is converted to []rune once so all index operations use cell coordinates.
func applyInlineHighlighting(screen tcell.Screen, x, row int, line string, style MarkdownStyle) {
	runes := []rune(line)

	// Bold: **text**
	highlightDelimited(screen, x, row, runes, "**", "**", style.Bold, true, false)

	// Italic: *text* (skip if preceded by *)
	applyItalicHighlighting(screen, x, row, runes, style)

	// Links: [text](url)
	applyLinkHighlighting(screen, x, row, runes, style)

	// Inline code: `text` — applied last so code spans win over other styles
	highlightDelimited(screen, x, row, runes, "`", "`", style.InlineCode, false, false)
}

// highlightDelimited finds pairs of start/end delimiters and colors the content between them.
// All positions are in rune (screen cell) coordinates.
func highlightDelimited(screen tcell.Screen, x, row int, runes []rune, startDelim, endDelim string, fg tcell.Color, bold, italic bool) {
	startRunes := []rune(startDelim)
	endRunes := []rune(endDelim)
	offset := 0
	for {
		start := indexRunes(runes[offset:], startRunes)
		if start < 0 {
			break
		}
		start += offset
		searchFrom := start + len(startRunes)
		if searchFrom >= len(runes) {
			break
		}
		end := indexRunes(runes[searchFrom:], endRunes)
		if end < 0 {
			break
		}
		end += searchFrom

		// Color the delimiters and content
		colorRange(screen, x, row, start, end+len(endRunes), fg, bold, italic)
		offset = end + len(endRunes)
	}
}

// indexRunes finds the first occurrence of needle in haystack, returning the rune index or -1.
func indexRunes(haystack, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// applyItalicHighlighting handles *text* while avoiding **bold** markers.
// All positions are in rune (screen cell) coordinates.
func applyItalicHighlighting(screen tcell.Screen, x, row int, runes []rune, style MarkdownStyle) {
	offset := 0
	for {
		start := indexRunes(runes[offset:], []rune{'*'})
		if start < 0 {
			break
		}
		start += offset

		// Skip bold markers (**)
		if start+1 < len(runes) && runes[start+1] == '*' {
			offset = start + 2
			continue
		}
		// Skip if preceded by * (end of bold)
		if start > 0 && runes[start-1] == '*' {
			offset = start + 1
			continue
		}

		searchFrom := start + 1
		if searchFrom >= len(runes) {
			break
		}
		end := indexRunes(runes[searchFrom:], []rune{'*'})
		if end < 0 {
			break
		}
		end += searchFrom

		// Skip if followed by * (start of bold)
		if end+1 < len(runes) && runes[end+1] == '*' {
			offset = end + 2
			continue
		}

		colorRange(screen, x, row, start, end+1, style.Italic, false, true)
		offset = end + 1
	}
}

// applyLinkHighlighting handles [text](url) patterns.
// All positions are in rune (screen cell) coordinates.
func applyLinkHighlighting(screen tcell.Screen, x, row int, runes []rune, style MarkdownStyle) {
	offset := 0
	for {
		bracketOpen := indexRunes(runes[offset:], []rune{'['})
		if bracketOpen < 0 {
			break
		}
		bracketOpen += offset

		bracketClose := indexRunes(runes[bracketOpen+1:], []rune{']', '('})
		if bracketClose < 0 {
			break
		}
		bracketClose += bracketOpen + 1

		parenClose := indexRunes(runes[bracketClose+2:], []rune{')'})
		if parenClose < 0 {
			break
		}
		parenClose += bracketClose + 2

		// [text] in rose
		colorRange(screen, x, row, bracketOpen, bracketClose+1, style.LinkText, false, false)
		// ](url) in blue underline
		modifyStyleRange(screen, x, row, bracketClose+1, parenClose+1, func(s tcell.Style) tcell.Style {
			return s.Foreground(style.LinkURL).Underline(true)
		})

		offset = parenClose + 1
	}
}

// extractLine reads a row of screen cells and returns the text as a string.
func extractLine(screen tcell.Screen, x, row, width int) string {
	runes := make([]rune, width)
	for col := 0; col < width; col++ {
		mainc, _, _, _ := screen.GetContent(x+col, row)
		runes[col] = mainc
	}
	return string(runes)
}

// modifyStyleRange applies a style transformation to a range of screen cells.
func modifyStyleRange(screen tcell.Screen, x, row, from, to int, fn func(tcell.Style) tcell.Style) {
	for col := from; col < to; col++ {
		mainc, combc, style, _ := screen.GetContent(x+col, row)
		screen.SetContent(x+col, row, mainc, combc, fn(style))
	}
}

// colorRange applies foreground color and explicit bold/italic to a range of screen cells.
func colorRange(screen tcell.Screen, x, row, from, to int, fg tcell.Color, bold, italic bool) {
	modifyStyleRange(screen, x, row, from, to, func(s tcell.Style) tcell.Style {
		return s.Foreground(fg).Bold(bold).Italic(italic)
	})
}

// countCodeFences counts the number of code fence (```) lines in text
// to determine if we start inside a code block.
func countCodeFences(text string, maxLine int) int {
	count := 0
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i >= maxLine {
			break
		}
		if strings.HasPrefix(strings.TrimLeft(line, " "), "```") {
			count++
		}
	}
	return count
}
