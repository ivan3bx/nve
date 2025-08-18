package nve

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBox struct {
	*tview.InputField
	listView         *ListBox
	contentView      *ContentBox
	notes            *Notes
	updatingFromList bool
}

// SetTextFromList updates the search box text from list selection without triggering search
func (sb *SearchBox) SetTextFromList(text string) {
	log.Printf("[DEBUG] SearchBox: SetTextFromList called with text='%s'", text)
	sb.updatingFromList = true
	sb.SetText(text)
	sb.updatingFromList = false
}

func NewSearchBox(listView *ListBox, contentView *ContentBox, notes *Notes) *SearchBox {
	res := SearchBox{
		InputField:  tview.NewInputField(),
		listView:    listView,
		contentView: contentView,
		notes:       notes,
	}

	listView.searchView = &res

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

	res.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if len(notes.LastSearchResults) == 0 {
				newNote, err := notes.CreateNote(res.GetText())

				if err != nil {
					log.Println("Error creating new note")
					break
				}

				notes.Search(newNote.DisplayName())
			}
		}
	})

	return &res
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (sb *SearchBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return sb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyEnter && sb.GetText() != "" {
			setFocus(sb.contentView)
		} else if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyCtrlN {
			if len(sb.notes.LastSearchResults) > 0 {
				sb.listView.SetSelectedFocusOnly(false)

				// Check if we're starting from an unselected state (empty search)
				currentItem := sb.listView.GetCurrentItem()
				if sb.GetText() == "" && currentItem == 0 {
					// When starting from empty search, select the first item directly
					sb.listView.SetCurrentItem(0)
					log.Printf("[DEBUG] SearchBox: Empty search, selecting first item (index 0)")
				} else {
					// Otherwise let the ListBox handle the down arrow to update selection
					if handler := sb.listView.InputHandler(); handler != nil {
						handler(tcell.NewEventKey(tcell.KeyDown, event.Rune(), event.Modifiers()), setFocus)
					}
				}

				// Now get the updated current item and sync everything
				currentItem = sb.listView.GetCurrentItem()
				if currentItem < len(sb.notes.LastSearchResults) {
					filename := sb.notes.LastSearchResults[currentItem].DisplayName()
					log.Printf("[DEBUG] SearchBox: Down arrow pressed, updating text to '%s'", filename)
					sb.SetTextFromList(filename)
					result := sb.notes.LastSearchResults[currentItem]
					sb.contentView.SetFile(result.FileRef)
				}
			} else {
				if handler := sb.listView.InputHandler(); handler != nil {
					handler(tcell.NewEventKey(tcell.KeyDown, event.Rune(), event.Modifiers()), setFocus)
				}
			}
		} else if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyCtrlP {
			if len(sb.notes.LastSearchResults) > 0 {
				sb.listView.SetSelectedFocusOnly(false)

				// Let the ListBox handle the up arrow to update selection
				if handler := sb.listView.InputHandler(); handler != nil {
					handler(tcell.NewEventKey(tcell.KeyUp, event.Rune(), event.Modifiers()), setFocus)
				}

				// Now get the updated current item and sync everything
				currentItem := sb.listView.GetCurrentItem()
				if currentItem < len(sb.notes.LastSearchResults) {
					filename := sb.notes.LastSearchResults[currentItem].DisplayName()
					log.Printf("[DEBUG] SearchBox: Up arrow pressed, updating text to '%s'", filename)
					sb.SetTextFromList(filename)
					result := sb.notes.LastSearchResults[currentItem]
					sb.contentView.SetFile(result.FileRef)
				}
			} else {
				if handler := sb.listView.InputHandler(); handler != nil {
					handler(tcell.NewEventKey(tcell.KeyUp, event.Rune(), event.Modifiers()), setFocus)
				}
			}
		}

		before := sb.GetText()

		if handler := sb.InputField.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		after := sb.GetText()
		if before != after && !sb.updatingFromList {
			log.Printf("[DEBUG] SearchBox: Text changed from '%s' to '%s', triggering search", before, after)
			sb.notes.Search(after)
		} else if before != after && sb.updatingFromList {
			log.Printf("[DEBUG] SearchBox: Text changed from '%s' to '%s' (from list update, skipping search)", before, after)
		}
	})
}
