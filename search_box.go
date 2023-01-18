package nve

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBox struct {
	*tview.InputField
	listView *ListBox
}

func NewSearchBox(listView *ListBox, notes *Notes) *SearchBox {
	res := SearchBox{
		InputField: tview.NewInputField(),
		listView:   listView,
	}

	// input field attributes
	res.SetFieldBackgroundColor(tcell.ColorBlack).
		SetPlaceholderStyle(tcell.StyleDefault.Background(tcell.ColorBlack))

	// other attributes
	res.SetBorder(true).
		SetTitle("Search Box").
		SetBackgroundColor(tcell.ColorBlack).
		SetTitleColor(tcell.ColorYellow).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetBorderPadding(0, 0, 1, 1).
		SetTitleAlign(tview.AlignLeft)

	// input handling
	res.SetChangedFunc(func(text string) {
		notes.Search(text)
	})

	res.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			notes.Search(res.GetText())
		case tcell.KeyEsc:
			notes.Search("")
			res.SetText("")
		}
	})

	return &res
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (sb *SearchBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return sb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyEnter {
			setFocus(sb.listView)
		} else {
			if handler := sb.InputField.InputHandler(); handler != nil {
				handler(event, setFocus)
			}
		}
	})
}
