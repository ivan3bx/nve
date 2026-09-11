package nve

import (
	"log"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bep/debounce"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	HighlightBackground = tcell.ColorYellow
	HighlightForeground = tcell.ColorBlack

	// moreMatchesLabel is drawn on the bottom border when search terms match
	// text below the visible rows.
	moreMatchesLabel = " >> more "
)

type ContentBox struct {
	*tview.TextArea
	debounce       func(func())
	currentFile    *FileRef
	pendingRefresh bool
	dirty          bool
	highlightTerms []string
	versioning     bool
	saveVersion    func(string)
}

func NewContentBox(config ...*Config) *ContentBox {
	textArea := ContentBox{
		TextArea:    tview.NewTextArea(),
		debounce:    debounce.New(300 * time.Millisecond),
		saveVersion: SaveFileVersion,
	}

	if len(config) > 0 && config[0] != nil {
		textArea.versioning = config[0].Versioning
	}

	textArea.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	textArea.SetFocusFunc(func() {
		// ignore edits if there is no current file
		if textArea.currentFile == nil {
			textArea.Blur()
		}
	})

	textArea.SetBlurFunc(func() {
		textArea.flushRefresh()
	})
	return &textArea
}

func (b *ContentBox) Clear() {
	if b.versioning && b.currentFile != nil {
		b.flushAndSnapshot()
	}
	b.currentFile = nil
	b.dirty = false
	b.SetText("", true)
}

func (b *ContentBox) SetFile(f *FileRef) {
	// Snapshot outgoing file before switching, but only if changing files.
	if b.versioning && b.currentFile != nil {
		if f == nil || f.Filename != b.currentFile.Filename {
			b.flushAndSnapshot()
		}
	}

	b.currentFile = f
	b.dirty = false
	b.SetText(GetContent(f.Filename), false)
}

// Shutdown snapshots the current file. Called on app exit.
func (b *ContentBox) Shutdown() {
	if b.versioning && b.currentFile != nil {
		b.flushAndSnapshot()
	}
}

// flushAndSnapshot persists the current editor buffer to disk and then
// registers a version snapshot. Only acts when the user has made edits
// (dirty flag is set), to avoid overwriting external changes.
func (b *ContentBox) flushAndSnapshot() {
	if !b.dirty {
		return
	}
	if err := SaveContent(b.currentFile.Filename, b.GetText()); err != nil {
		log.Printf("[WARN] flushAndSnapshot: failed to save %s: %v; skipping snapshot", b.currentFile.Filename, err)
		return
	}
	b.dirty = false
	b.saveVersion(b.currentFile.Filename)
}

// RefreshFile marks that the file may have changed on disk. The actual
// reload is deferred until the user leaves the editor (via flushRefresh)
// because calling SetText on a focused TextArea corrupts tview's
// internal cursor state and causes panics.
func (b *ContentBox) RefreshFile() {
	b.pendingRefresh = true
}

// flushRefresh reloads the current file from disk if a refresh is pending
// and the content actually changed. Called when ContentBox loses focus.
func (b *ContentBox) flushRefresh() {
	defer func() { b.pendingRefresh = false }()

	if !b.pendingRefresh || b.currentFile == nil {
		return
	}
	diskContent := GetContent(b.currentFile.Filename)
	if diskContent != b.GetText() {
		b.SetText(diskContent, false)
	}
}

// SetSearchQuery updates the current search query used for highlighting.
// The query is split into terms, each of which is highlighted independently.
func (b *ContentBox) SetSearchQuery(query string) {
	b.highlightTerms = strings.Fields(strings.ToLower(query))
}

// Draw renders the text area, applies markdown syntax highlighting for .md files,
// and then highlights any occurrences of the search terms.
func (b *ContentBox) Draw(screen tcell.Screen) {
	b.TextArea.Draw(screen)

	x, y, width, height := b.GetInnerRect()

	// Apply markdown highlighting before search highlighting
	if b.isMarkdown() {
		inCodeBlock := b.codeBlockStateAtVisibleTop()
		applyMarkdownHighlighting(screen, x, y, width, height, inCodeBlock, Zenburn)
	}

	highlightSearchTerms(screen, x, y, width, height, b.highlightTerms)

	if b.hasMatchesBelow(screen, x, y, width, height) {
		b.drawMoreIndicator(screen)
	}
}

// hasMatchesBelow reports whether any search term occurs in the text that
// follows the visible rows.
func (b *ContentBox) hasMatchesBelow(screen tcell.Screen, x, y, width, height int) bool {
	if len(b.highlightTerms) == 0 {
		return false
	}

	rows := make([]string, 0, height)
	for row := y; row < y+height; row++ {
		rows = append(rows, extractLine(screen, x, row, width))
	}

	text := b.GetText()
	end := textAfterVisible(text, rows)
	if end < 0 {
		return false
	}

	rest := strings.ToLower(text[end:])
	for _, term := range b.highlightTerms {
		if strings.Contains(rest, term) {
			return true
		}
	}

	return false
}

// drawMoreIndicator draws moreMatchesLabel, right-aligned on the bottom border,
// in the same style used for highlighted matches.
func (b *ContentBox) drawMoreIndicator(screen tcell.Screen) {
	x, y, width, height := b.GetRect()
	if height < 2 || width < len(moreMatchesLabel)+2 {
		return
	}

	row := y + height - 1
	col := x + width - 1 - len(moreMatchesLabel)
	style := highlightStyle(tcell.StyleDefault)

	for i, r := range moreMatchesLabel {
		screen.SetContent(col+i, row, r, nil, style)
	}
}

// textAfterVisible returns the byte offset in text immediately following the
// content rendered on rows, or -1 if that content cannot be located in text.
//
// The TextArea wraps lines and pads rows, so whitespace is ignored on both
// sides when locating the visible content. The wrapped-row offset exposed by
// TextArea cannot be mapped back to a text position directly.
func textAfterVisible(text string, rows []string) int {
	visible := stripSpace(strings.Join(rows, ""))
	if visible == "" {
		return -1
	}

	// Build a whitespace-free copy of text, recording the original offset of
	// every byte so a match position can be mapped back.
	var stripped strings.Builder
	offsets := make([]int, 0, len(text))

	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsSpace(r) {
			stripped.WriteString(text[i : i+n])
			for k := 0; k < n; k++ {
				offsets = append(offsets, i+k)
			}
		}
		i += n
	}

	idx := strings.Index(stripped.String(), visible)
	if idx < 0 {
		return -1
	}

	return offsets[idx+len(visible)-1] + 1
}

// stripSpace removes whitespace and the zero runes tcell reports for the
// trailing cell of wide characters.
func stripSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// highlightSearchTerms applies the highlight style to every occurrence of each
// term on the visible rows. Matching is case-insensitive and terms are lower-case.
func highlightSearchTerms(screen tcell.Screen, x, y, width, height int, terms []string) {
	if len(terms) == 0 {
		return
	}

	for row := y; row < y+height; row++ {
		line := []rune(extractLine(screen, x, row, width))
		for i, r := range line {
			line[i] = unicode.ToLower(r)
		}

		for _, term := range terms {
			needle := []rune(term)
			for col := 0; col < len(line); {
				i := indexRunes(line[col:], needle)
				if i < 0 {
					break
				}
				col += i
				modifyStyleRange(screen, x, row, col, col+len(needle), highlightStyle)
				col += len(needle)
			}
		}
	}
}

func highlightStyle(s tcell.Style) tcell.Style {
	return s.Background(HighlightBackground).Foreground(HighlightForeground).Bold(true)
}

// isMarkdown returns true if the current file is a markdown file.
func (b *ContentBox) isMarkdown() bool {
	if b.currentFile == nil {
		return false
	}
	ext := filepath.Ext(b.currentFile.Filename)
	return ext == ".md" || ext == ".mdown"
}

// codeBlockStateAtVisibleTop determines whether the first visible row
// is inside a code block by using the TextArea's scroll offset to count
// code fences above the visible area.
func (b *ContentBox) codeBlockStateAtVisibleTop() bool {
	topRow, _ := b.GetOffset()
	if topRow <= 0 {
		return false
	}
	return countCodeFences(b.GetText(), topRow)%2 == 1
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (b *ContentBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		event = b.mapSpecialKeys(event)

		// Intercept Enter in markdown files for list continuation.
		if event.Key() == tcell.KeyEnter && b.isMarkdown() {
			if b.handleListContinuation() {
				return
			}
		}

		// Intercept Tab/Shift-Tab on bullet lines for indentation.
		if b.isMarkdown() {
			if event.Key() == tcell.KeyTab {
				if b.handleListIndent() {
					return
				}
			}
			if event.Key() == tcell.KeyBacktab {
				if b.handleListDedent() {
					return
				}
			}
		}

		before := b.GetText()

		if handler := b.TextArea.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		if after := b.GetText(); before != after {
			b.dirty = true
			b.queueSave(after)
		}
	})
}

// handleListContinuation checks the current line for a bullet prefix and
// either continues the list or clears an empty bullet. Returns true if the
// Enter key was consumed and should not be passed to TextArea.
func (b *ContentBox) handleListContinuation() bool {
	text := b.GetText()
	_, start, _ := b.GetSelection()

	// Find the start of the current line.
	lineStart := strings.LastIndex(text[:start], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++ // skip past the newline
	}

	// Use the full line (not just up to the cursor) so that bullet detection
	// works correctly when the cursor is positioned within the marker.
	lineEnd := strings.Index(text[start:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += start
	}

	fullLine := text[lineStart:lineEnd]
	indent, marker, rest, ok := parseBulletPrefix(fullLine)
	if !ok {
		return false
	}

	if strings.TrimSpace(rest) == "" {
		// Empty bullet line: remove the bullet and replace with a plain newline.
		b.Replace(lineStart, lineEnd, "\n")
	} else {
		// Continue the list on the next line.
		insertion := "\n" + nextBulletPrefix(indent, marker)
		b.Replace(start, start, insertion)
	}

	b.dirty = true
	b.queueSave(b.GetText())
	return true
}

// handleListIndent inserts a tab at the beginning of the current line when
// the cursor is on a bullet line. Returns true if the key was consumed.
func (b *ContentBox) handleListIndent() bool {
	text := b.GetText()
	_, cursorPos, _ := b.GetSelection()

	lineStart := strings.LastIndex(text[:cursorPos], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++
	}

	currentLine := text[lineStart:cursorPos]
	// Also consider text after the cursor on the same line.
	lineEnd := strings.Index(text[cursorPos:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += cursorPos
	}
	fullLine := text[lineStart:lineEnd]

	if _, _, _, ok := parseBulletPrefix(currentLine); !ok {
		if _, _, _, ok := parseBulletPrefix(fullLine); !ok {
			return false
		}
	}

	b.Replace(lineStart, lineStart, "\t")
	// Restore cursor to its original relative position (shifted by the inserted tab).
	b.Replace(cursorPos+1, cursorPos+1, "")
	b.dirty = true
	b.queueSave(b.GetText())
	return true
}

// handleListDedent removes one leading tab from the current line when the
// cursor is on a bullet line. Returns true if the key was consumed.
func (b *ContentBox) handleListDedent() bool {
	text := b.GetText()
	_, cursorPos, _ := b.GetSelection()

	lineStart := strings.LastIndex(text[:cursorPos], "\n")
	if lineStart < 0 {
		lineStart = 0
	} else {
		lineStart++
	}

	currentLine := text[lineStart:cursorPos]
	lineEnd := strings.Index(text[cursorPos:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += cursorPos
	}
	fullLine := text[lineStart:lineEnd]

	if _, _, _, ok := parseBulletPrefix(currentLine); !ok {
		if _, _, _, ok := parseBulletPrefix(fullLine); !ok {
			return false
		}
	}

	// Only dedent if the line starts with a tab.
	if lineStart < len(text) && text[lineStart] == '\t' {
		b.Replace(lineStart, lineStart+1, "")
		// Restore cursor to its original relative position (shifted back by the removed tab).
		// Clamp to lineStart to avoid underflow when cursor is at the start of the line.
		newCursorPos := cursorPos - 1
		if newCursorPos < lineStart {
			newCursorPos = lineStart
		}
		b.Replace(newCursorPos, newCursorPos, "")
		b.dirty = true
		b.queueSave(b.GetText())
	}
	return true
}

func (b *ContentBox) mapSpecialKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	// navigate up
	case tcell.KeyCtrlP:
		event = tcell.NewEventKey(tcell.KeyUp, event.Rune(), event.Modifiers())

	// navigate down
	case tcell.KeyCtrlN:
		event = tcell.NewEventKey(tcell.KeyDown, event.Rune(), event.Modifiers())

	// navigate forward
	case tcell.KeyCtrlF:
		event = tcell.NewEventKey(tcell.KeyRight, event.Rune(), event.Modifiers())

	// delete empty line
	case tcell.KeyCtrlK:
		fromRow, fromCol, toRow, toCol := b.GetCursor()

		if fromRow == toRow && fromCol == toCol && fromCol == 0 {
			if _, start, end := b.GetSelection(); start == end {
				r, _ := utf8.DecodeRuneInString(b.GetText()[start:])
				if !unicode.IsLetter(r) {
					event = tcell.NewEventKey(tcell.KeyDelete, event.Rune(), event.Modifiers())
				}
			}
		}

	}

	return event
}

func (b *ContentBox) queueSave(content string) {
	if b.currentFile == nil {
		return
	}
	filename := b.currentFile.Filename
	b.debounce(func() {
		err := SaveContent(filename, content)

		if err != nil {
			log.Println("Error saving content:", err)
		}
	})
}
