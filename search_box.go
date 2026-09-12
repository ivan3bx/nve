package nve

import (
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBox struct {
	*tview.InputField
	listView         *ListBox
	contentView      *ContentBox
	notes            *Notes
	updatingFromList bool

	// renaming tracks rename mode: the input edits the selected note's name
	// instead of searching, and the list previews the new name as it is typed.
	renaming     bool
	renameIndex  int
	renameTarget *FileRef
	renameOrigin tview.Primitive // focus returns here when the rename ends
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
		if key != tcell.KeyEnter || len(notes.LastSearchResults) > 0 {
			return
		}

		newNote, err := notes.CreateNote(res.GetText())
		if err != nil {
			log.Println("Error creating new note:", err)
			return
		}

		notes.Search(newNote.DisplayName())
	})

	return &res
}

// syncWithListSelection updates SearchBox text and ContentView with current selection
func (sb *SearchBox) syncWithListSelection(keyAction string) {
	currentItem := sb.listView.GetCurrentItem()
	if currentItem < len(sb.notes.LastSearchResults) {
		filename := sb.notes.LastSearchResults[currentItem].DisplayName()
		log.Printf("[DEBUG] SearchBox: %s, updating text to '%s'", keyAction, filename)
		sb.SetTextFromList(filename)
		result := sb.notes.LastSearchResults[currentItem]
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
	if len(sb.notes.LastSearchResults) == 0 {
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

// IsRenaming reports whether rename mode is active, so the global input
// handler can leave keys (like Escape) for the rename to handle.
func (sb *SearchBox) IsRenaming() bool {
	return sb.renaming
}

// startRename enters rename mode for the currently selected note, pre-filling
// the input with its display name. origin receives focus when the rename ends.
func (sb *SearchBox) startRename(origin tview.Primitive) {
	if sb.renaming {
		return
	}

	index := sb.listView.GetCurrentItem()
	if index < 0 || index >= len(sb.notes.LastSearchResults) {
		return
	}

	sb.renaming = true
	sb.renameIndex = index
	sb.renameTarget = sb.notes.LastSearchResults[index].FileRef
	sb.renameOrigin = origin
	sb.SetTitle("Rename")
	sb.SetTextFromList(sb.renameTarget.DisplayName())
	sb.listView.SetRenamePreview(index, sb.renameTarget.DisplayName())
}

// commitRename applies the typed name, then searches for it so the renamed
// note stays selected.
func (sb *SearchBox) commitRename(setFocus func(p tview.Primitive)) {
	name := strings.TrimSpace(sb.GetText())

	if name == "" || name == sb.renameTarget.DisplayName() {
		sb.cancelRename(setFocus)
		return
	}

	if _, err := sb.notes.RenameNote(sb.renameTarget, name); err != nil {
		log.Printf("[WARN] SearchBox: rename failed: %v", err)
		sb.SetTitle("Rename (name unavailable)")
		return
	}

	origin := sb.renameOrigin
	sb.endRename()
	sb.notes.Search(name)

	if origin != nil {
		setFocus(origin)
	}
}

// cancelRename exits rename mode, restoring the input to the note's name.
func (sb *SearchBox) cancelRename(setFocus func(p tview.Primitive)) {
	target := sb.renameTarget
	origin := sb.renameOrigin
	sb.endRename()
	if target != nil {
		sb.SetTextFromList(target.DisplayName())
	}
	if origin != nil {
		setFocus(origin)
	}
}

func (sb *SearchBox) endRename() {
	sb.renaming = false
	sb.renameIndex = -1
	sb.renameTarget = nil
	sb.renameOrigin = nil
	sb.SetTitle("Search Box")
	sb.listView.ClearRenamePreview()
}

// handleRenameInput processes keys while renaming: typing updates the list
// preview instead of searching, Enter commits, Escape cancels.
func (sb *SearchBox) handleRenameInput(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	switch event.Key() {
	case tcell.KeyEnter:
		sb.commitRename(setFocus)
		return
	case tcell.KeyEscape:
		sb.cancelRename(setFocus)
		return
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyCtrlN, tcell.KeyCtrlP, tcell.KeyCtrlR:
		// Keep the list selection pinned to the rename target.
		return
	}

	before := sb.GetText()

	if handler := sb.InputField.InputHandler(); handler != nil {
		handler(event, setFocus)
	}

	if after := sb.GetText(); after != before {
		sb.SetTitle("Rename") // clear any previous error
		sb.listView.SetRenamePreview(sb.renameIndex, after)
	}
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (sb *SearchBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return sb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if sb.renaming {
			sb.handleRenameInput(event, setFocus)
			return
		}

		// Handle special keys first
		switch event.Key() {
		case tcell.KeyCtrlR:
			sb.startRename(sb)
			return
		case tcell.KeyEnter:
			if sb.GetText() == "" {
				return
			}
			if len(sb.notes.LastSearchResults) == 0 {
				// No matches — create a new note, then focus ContentBox
				if handler := sb.InputField.InputHandler(); handler != nil {
					handler(event, setFocus)
				}
			}
			setFocus(sb.contentView)
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
			sb.notes.Search(after)
		} else if before != after && sb.updatingFromList {
			log.Printf("[DEBUG] SearchBox: Text changed from '%s' to '%s' (from list update, skipping search)", before, after)
		}
	})
}
