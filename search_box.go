package nve

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBox struct {
	*tview.InputField
	listView    *ListBox
	contentView *ContentBox
}

func NewSearchBox(listView *ListBox, contentView *ContentBox, notes *Notes) *SearchBox {
	res := SearchBox{
		InputField:  tview.NewInputField(),
		listView:    listView,
		contentView: contentView,
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
			if len(notes.LastSearchResults) == 0 {
				newNote, err := notes.CreateNote(res.GetText())

				if err != nil {
					log.Println("Error creating new note")
					break
				}

				log.Println("Searching for", newNote.DisplayName())
				notes.Search(newNote.DisplayName())
			}
		}
	})

	return &res
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (sb *SearchBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return sb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyEnter {
			setFocus(sb.contentView)
		} else if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyCtrlN {
			if handler := sb.listView.InputHandler(); handler != nil {
				handler(tcell.NewEventKey(tcell.KeyDown, event.Rune(), event.Modifiers()), setFocus)
			}
		} else if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyCtrlP {
			if handler := sb.listView.InputHandler(); handler != nil {
				handler(tcell.NewEventKey(tcell.KeyUp, event.Rune(), event.Modifiers()), setFocus)
			}
		}

		if handler := sb.InputField.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}
