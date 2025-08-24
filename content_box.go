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

// ContentMode represents the interface for different content display modes
type ContentMode interface {
	tview.Primitive
	SetContent(content string)
	GetContent() string
	Clear()
}

type ContentBox struct {
	editMode    *EditMode
	viewMode    *ViewMode
	currentMode ContentMode

	debounce    func(func())
	currentFile *FileRef
	isEditMode  bool

	searchCtx *SearchContext
}

// Ensure ContentBox implements tview.Primitive
var _ tview.Primitive = (*ContentBox)(nil)

func NewContentBox(sctx *SearchContext) *ContentBox {
	// Create the underlying components
	content := ContentBox{
		debounce:   debounce.New(300 * time.Millisecond),
		isEditMode: false,
		searchCtx:  sctx,
	}

	// Content modes share bAttrs attributes
	bAttrs := tview.NewBox().SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	// Create mode strategies
	content.editMode = NewEditMode(&content, bAttrs)
	content.viewMode = NewViewMode(sctx, bAttrs)

	content.currentMode = content.viewMode // Start in view mode

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
	return b.currentMode
}

// switchToEditMode switches to edit mode
func (b *ContentBox) switchToEditMode() {
	if b.currentFile == nil {
		return
	}

	b.isEditMode = true
	b.currentMode = b.editMode

	// Copy current content to edit mode
	content := GetContent(b.currentFile.Filename)
	b.currentMode.SetContent(content)
}

// switchToViewMode switches to view mode
func (b *ContentBox) switchToViewMode() {
	if b.currentFile == nil {
		return
	}

	b.isEditMode = false

	// Get content from current mode before switching
	content := b.currentMode.GetContent()
	b.currentMode = b.viewMode

	// Apply content to view mode (which handles highlighting)
	b.currentMode.SetContent(content)
}

// Implement tview.Primitive interface by delegating to the current mode
func (b *ContentBox) Draw(screen tcell.Screen)      { b.currentMode.Draw(screen) }
func (b *ContentBox) GetRect() (int, int, int, int) { return b.currentMode.GetRect() }
func (b *ContentBox) HasFocus() bool                { return b.currentMode.HasFocus() }
func (b *ContentBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.currentMode.InputHandler()
}
func (b *ContentBox) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return b.currentMode.MouseHandler()
}

func (b *ContentBox) SetRect(x, y, width, height int) {
	// Set rect on both components so they're ready when switched
	b.editMode.SetRect(x, y, width, height)
	b.viewMode.SetRect(x, y, width, height)
}

func (b *ContentBox) Focus(delegate func(p tview.Primitive)) {
	// Switch to edit mode when content box gains focus
	if !b.isEditMode {
		b.switchToEditMode()
	}
	b.currentMode.Focus(delegate)
}

func (b *ContentBox) Blur() {
	// Switch back to view mode when losing focus
	if b.isEditMode {
		b.switchToViewMode()
	}
	b.currentMode.Blur()
}

func (b *ContentBox) Clear() {
	b.currentFile = nil
	b.currentMode.Clear()
}

func (b *ContentBox) SetFile(f *FileRef) {
	b.currentFile = f
	content := GetContent(f.Filename)
	b.currentMode.SetContent(content)
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
			editMode := b.currentMode.(*EditMode)
			fromRow, fromCol, toRow, toCol := editMode.GetCursor()

			if fromRow == toRow && fromCol == toCol && fromCol == 0 {
				if _, start, end := editMode.GetSelection(); start == end {
					r, _ := utf8.DecodeRuneInString(editMode.GetText()[start:])
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
