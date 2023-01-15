package nve

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ContentBox struct {
	*tview.TextArea
}

func NewContentBox() *ContentBox {
	textArea := ContentBox{tview.NewTextArea()}

	textArea.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(1, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	textArea.SetText("this could be lots of content\n\n# Separated by other stuff.\n\n* one item\n* two item\n", true)

	return &textArea
}

func (b *ContentBox) SearchResultsUpdate(_ *Notes) {
	// TODO: if selected note changes, update content.
}
