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
)

type ContentBox struct {
	*tview.TextArea
	debounce       func(func())
	currentFile    *FileRef
	pendingRefresh bool
	dirty          bool
	searchQuery    string
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
func (b *ContentBox) SetSearchQuery(query string) {
	b.searchQuery = query
}

// Draw renders the text area, applies markdown syntax highlighting for .md files,
// and then highlights any occurrences of the search query.
func (b *ContentBox) Draw(screen tcell.Screen) {
	b.TextArea.Draw(screen)

	x, y, width, height := b.GetInnerRect()

	// Apply markdown highlighting before search highlighting
	if b.isMarkdown() {
		inCodeBlock := b.codeBlockStateAtVisibleTop()
		applyMarkdownHighlighting(screen, x, y, width, height, inCodeBlock, Zenburn)
	}

	if b.searchQuery == "" {
		return
	}

	query := strings.ToLower(b.searchQuery)

	for row := y; row < y+height; row++ {
		line := strings.ToLower(extractLine(screen, x, row, width))

		// Find all occurrences of the query in this line
		offset := 0
		for {
			idx := strings.Index(line[offset:], query)
			if idx < 0 {
				break
			}
			matchStart := offset + idx
			for i := 0; i < len(query); i++ {
				cx := x + matchStart + i
				mainc, combc, style, _ := screen.GetContent(cx, row)
				screen.SetContent(cx, row, mainc, combc, style.Background(HighlightBackground).Foreground(HighlightForeground).Bold(true))
			}
			offset = matchStart + len(query)
		}
	}
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
