package nve

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ListBox struct {
	*tview.List
	contentView *ContentBox
}

func NewListBox(contentView *ContentBox) *ListBox {
	box := ListBox{
		List:        tview.NewList(),
		contentView: contentView,
	}

	box.ShowSecondaryText(false).
		SetWrapAround(false).
		SetHighlightFullLine(true).
		SetSelectedStyle(
			tcell.StyleDefault.
				Background(tcell.ColorDarkBlue).
				Foreground(tcell.ColorLightSkyBlue),
		)

	box.SetBorder(true).
		SetTitle("List Box").
		SetTitleColor(tcell.ColorOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(0, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	// sample data
	box.AddItem("Main text goes here", "", 0, nil)
	box.AddItem("Second item here", "", 0, nil)

	return &box
}

func (b *ListBox) SearchResultsUpdate(notes *Notes) {
	// TODO: remove items not present in Notes search results
	// fmt.Printf("Searched for '%s'\n", notes.Query)
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (lb *ListBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return lb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyEnter {
			setFocus(lb.contentView)
		} else {
			if handler := lb.List.InputHandler(); handler != nil {
				handler(event, setFocus)
			}
		}
	})
}
