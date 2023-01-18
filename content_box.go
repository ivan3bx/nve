package nve

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContentBox struct {
	*tview.TextArea
	currentFile *FileRef
}

func NewContentBox() *ContentBox {
	textArea := ContentBox{TextArea: tview.NewTextArea()}

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

func (b *ContentBox) SearchResultsUpdate(_ *Notes) {
	// TODO: if selected note changes, update content.
}
