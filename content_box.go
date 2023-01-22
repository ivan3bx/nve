package nve

import (
	"log"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bep/debounce"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContentBox struct {
	*tview.TextArea
	debounce    func(func())
	currentFile *FileRef
}

func NewContentBox() *ContentBox {
	textArea := ContentBox{
		TextArea: tview.NewTextArea(),
		debounce: debounce.New(300 * time.Millisecond),
	}

	textArea.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	return &textArea
}

func (b *ContentBox) Clear() {
	b.currentFile = nil
	b.SetText("", true)
}

func (b *ContentBox) SetFile(f *FileRef) {
	b.currentFile = f
	b.SetText(GetContent(f.Filename), false)
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (b *ContentBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		event = b.mapSpecialKeys(event)

		before := b.GetText()

		if handler := b.TextArea.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		if after := b.GetText(); before != after {
			b.queueSave(after)
		}
	})
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
	b.debounce(func() {
		err := SaveContent(b.currentFile.Filename, content)

		if err != nil {
			log.Println("Error saving content:", err)
		}
	})
}
