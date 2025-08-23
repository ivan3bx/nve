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
	searchCtx        *SearchContext
	updatingFromList bool
}

// SetTextFromList updates the search box text from list selection without triggering search
func (sb *SearchBox) SetTextFromList(text string) {
	log.Printf("[DEBUG] SearchBox: SetTextFromList called with text='%s'", text)
	sb.updatingFromList = true
	sb.SetText(text)
	sb.updatingFromList = false
}

func NewSearchBox(searchCtx *SearchContext, listView *ListBox, contentView *ContentBox, notes *Notes) *SearchBox {
	res := SearchBox{
		InputField:  tview.NewInputField(),
		listView:    listView,
		contentView: contentView,
		notes:       notes,
		searchCtx:   searchCtx,
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
			if len(searchCtx.LastSearchResults) == 0 {
				newNote, err := notes.CreateNote(res.GetText())

				if err != nil {
					log.Println("Error creating new note")
					break
				}

				notes.Search(searchCtx, newNote.DisplayName())
			}
		}
	})

	return &res
}

// syncWithListSelection updates SearchBox text and ContentView with current selection
func (sb *SearchBox) syncWithListSelection(keyAction string) {
	currentItem := sb.listView.GetCurrentItem()
	if currentItem < len(sb.searchCtx.LastSearchResults) {
		filename := sb.searchCtx.LastSearchResults[currentItem].DisplayName()
		log.Printf("[DEBUG] SearchBox: %s, updating text to '%s'", keyAction, filename)
		sb.SetTextFromList(filename)
		result := sb.searchCtx.LastSearchResults[currentItem]
		sb.contentView.SetFile(result.FileRef)
	}
}

// delegateToListView forwards key event to ListBox handler
func (sb *SearchBox) delegateToListView(key tcell.Key, rune rune, modifiers tcell.ModMask, setFocus func(p tview.Primitive)) {
	if handler := sb.listView.InputHandler(); handler != nil {
		handler(tcell.NewEventKey(key, rune, modifiers), setFocus)
	}
}

// handleArrowKey processes up/down arrow keys with proper synchronization
func (sb *SearchBox) handleArrowKey(event *tcell.EventKey, setFocus func(p tview.Primitive), isDown bool) {
	// Early return if no results - just delegate
	if len(sb.searchCtx.LastSearchResults) == 0 {
		sb.delegateToListView(event.Key(), event.Rune(), event.Modifiers(), setFocus)
		return
	}

	sb.listView.SetSelectedFocusOnly(false)

	// Special case: down arrow from empty search should select first item
	if isDown && sb.GetText() == "" && sb.listView.GetCurrentItem() == 0 {
		sb.listView.SetCurrentItem(0)
		log.Printf("[DEBUG] SearchBox: Empty search, selecting first item (index 0)")
	} else {
		// Let ListBox handle the navigation
		sb.delegateToListView(event.Key(), event.Rune(), event.Modifiers(), setFocus)
	}

	// Sync SearchBox and ContentView with the selected item
	keyAction := "Down arrow pressed"
	if !isDown {
		keyAction = "Up arrow pressed"
	}
	sb.syncWithListSelection(keyAction)

	// Transfer focus to ListBox when down arrow is pressed
	if isDown {
		log.Printf("[DEBUG] SearchBox: Down arrow pressed, transferring focus to ListBox")
		setFocus(sb.listView)
	}
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (sb *SearchBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return sb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Handle special keys first
		switch event.Key() {
		case tcell.KeyEnter:
			if sb.GetText() != "" {
				setFocus(sb.contentView)
			}
			return
		case tcell.KeyDown, tcell.KeyCtrlN:
			sb.handleArrowKey(event, setFocus, true)
			return
		case tcell.KeyUp, tcell.KeyCtrlP:
			sb.handleArrowKey(event, setFocus, false)
			return
		}

		// Handle text input and search triggering
		before := sb.GetText()

		if handler := sb.InputField.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		after := sb.GetText()
		if before != after && !sb.updatingFromList {
			log.Printf("[DEBUG] SearchBox: Text changed from '%s' to '%s', triggering search", before, after)
			sb.notes.Search(sb.searchCtx, after)
		} else if before != after && sb.updatingFromList {
			log.Printf("[DEBUG] SearchBox: Text changed from '%s' to '%s' (from list update, skipping search)", before, after)
		}
	})
}
