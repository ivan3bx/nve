package nve

import (
	"log"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bep/debounce"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContentBox struct {
	EditArea *tview.TextArea
	TextView *tview.TextView

	debounce    func(func())
	currentFile *FileRef
	isEditMode  bool

	searchCtx *SearchContext
}

func NewContentBox(sctx *SearchContext) *ContentBox {
	content := ContentBox{
		EditArea:   tview.NewTextArea(),
		TextView:   tview.NewTextView(),
		debounce:   debounce.New(300 * time.Millisecond),
		isEditMode: false,
		searchCtx:  sctx,
	}

	// Configure TextArea
	content.EditArea.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	// Configure TextView for highlighting
	content.TextView.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	// Enable color markup processing
	content.TextView.SetDynamicColors(true)

	content.EditArea.SetFocusFunc(func() {
		// ignore edits if there is no current file
		if content.currentFile == nil {
			content.EditArea.Blur()
		}
	})
	return &content
}

// highlightText highlights all instances of a search term in the given text
func highlightText(content, searchTerm string) string {
	if searchTerm == "" {
		return content
	}

	// Create case-insensitive regex pattern
	pattern := regexp.QuoteMeta(searchTerm)
	re := regexp.MustCompile("(?i)" + pattern)

	// Replace all matches with highlighted version
	highlighted := re.ReplaceAllStringFunc(content, func(match string) string {
		return "[orange::b]" + match + "[white::-]"
	})

	return highlighted
}

// GetCurrentView returns the currently active view component
func (b *ContentBox) GetCurrentView() tview.Primitive {
	if b.isEditMode {
		return b.EditArea
	}
	return b.TextView
}

// Implement tview.Primitive interface by delegating to the current view
func (b *ContentBox) Draw(screen tcell.Screen) {
	b.GetCurrentView().Draw(screen)
}

func (b *ContentBox) GetRect() (int, int, int, int) {
	return b.GetCurrentView().GetRect()
}

func (b *ContentBox) SetRect(x, y, width, height int) {
	b.EditArea.SetRect(x, y, width, height)
	b.TextView.SetRect(x, y, width, height)
}

func (b *ContentBox) Focus(delegate func(p tview.Primitive)) {
	// Switch to edit mode when content box gains focus
	if !b.isEditMode {
		b.switchToEditMode()
	}
	b.GetCurrentView().Focus(delegate)
}

func (b *ContentBox) switchToEditMode() {
	if b.currentFile == nil {
		return
	}

	b.isEditMode = true

	// Copy current content to TextArea for editing
	content := GetContent(b.currentFile.Filename)
	b.EditArea.SetText(content, false)
}

func (b *ContentBox) switchToViewMode() {
	if b.currentFile == nil {
		return
	}

	b.isEditMode = false

	// Copy current content from TextArea and apply highlighting
	content := b.EditArea.GetText()
	highlighted := highlightText(content, b.searchCtx.LastQuery)
	b.TextView.SetText(highlighted)
}

func (b *ContentBox) Blur() {
	// Switch back to view mode when losing focus
	if b.isEditMode {
		b.switchToViewMode()
		b.EditArea.Blur()
	} else {
		b.TextView.Blur()
	}
}

func (b *ContentBox) HasFocus() bool {
	return b.GetCurrentView().HasFocus()
}

func (b *ContentBox) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return b.GetCurrentView().MouseHandler()
}

func (b *ContentBox) Clear() {
	b.currentFile = nil
	if b.isEditMode {
		b.EditArea.SetText("", true)
	} else {
		b.TextView.SetText("")
	}
}

func (b *ContentBox) SetFile(f *FileRef) {
	b.currentFile = f
	content := GetContent(f.Filename)

	if b.isEditMode {
		b.EditArea.SetText(content, false)
	} else {
		// Apply highlighting for current search query
		highlighted := highlightText(content, b.searchCtx.LastQuery)
		b.TextView.SetText(highlighted)
	}
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (b *ContentBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if b.isEditMode {
		return b.EditArea.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
			event = b.mapSpecialKeys(event)

			before := b.EditArea.GetText()

			if handler := b.EditArea.InputHandler(); handler != nil {
				handler(event, setFocus)
			}

			if after := b.EditArea.GetText(); before != after {
				b.queueSave(after)
			}
		})
	} else {
		// In view mode, just return TextView's input handler
		return b.TextView.InputHandler()
	}
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
		if b.isEditMode {
			fromRow, fromCol, toRow, toCol := b.EditArea.GetCursor()

			if fromRow == toRow && fromCol == toCol && fromCol == 0 {
				if _, start, end := b.EditArea.GetSelection(); start == end {
					r, _ := utf8.DecodeRuneInString(b.EditArea.GetText()[start:])
					if !unicode.IsLetter(r) {
						event = tcell.NewEventKey(tcell.KeyDelete, event.Rune(), event.Modifiers())
					}
				}
			}
		}

	}

	return event
}

func (b *ContentBox) queueSave(content string) {
	b.debounce(func() {
		err := SaveContent(b.currentFile.Filename, content)

		if err != nil {
			log.Println("Error saving content:", err)
		}
	})
}
