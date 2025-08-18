package nve

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ListBox struct {
	*tview.List
	contentView *ContentBox
	searchView  *SearchBox
	notes       *Notes
}

func NewListBox(contentView *ContentBox, notes *Notes) *ListBox {
	box := ListBox{
		List:        tview.NewList(),
		contentView: contentView,
		notes:       notes,
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

	box.SetSelectedFocusOnly(false)

	box.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if (!box.HasFocus() && notes.LastQuery == "") || len(notes.LastSearchResults) == 0 {
			box.contentView.Clear()
		} else {
			result := notes.LastSearchResults[index]
			box.contentView.SetFile(result.FileRef)
		}
	})

	box.SetFocusFunc(func() {
		if notes.LastQuery == "" {
			result := notes.LastSearchResults[box.GetCurrentItem()]
			box.contentView.SetFile(result.FileRef)
		}
	})

	return &box
}

func (b *ListBox) SearchResultsUpdate(notes *Notes) {
	emptyQuery := notes.LastQuery == ""
	lastResult := notes.LastSearchResults

	log.Printf("[DEBUG] ListBox: SearchResultsUpdate called - query='%s', emptyQuery=%t, results=%d", notes.LastQuery, emptyQuery, len(lastResult))

	b.Clear()

	b.SetSelectedFocusOnly(emptyQuery)

	if len(lastResult) == 0 {
		b.contentView.Clear()
	}

	selectedIndex := -1

	for index, result := range lastResult {
		displayName := result.DisplayName()
		var formattedName string
		if len(displayName) > 14 {
			formattedName = fmt.Sprintf("%-20.20s..", displayName)
		} else {
			formattedName = fmt.Sprintf("%-22.22s", displayName)
		}

		b.AddItem(strings.Join([]string{formattedName, result.Snippet}, " : "), "", 0, nil)

		if selectedIndex == -1 && strings.HasPrefix(displayName, notes.LastQuery) {
			selectedIndex = index
		}
	}

	_, _, _, height := b.GetInnerRect()

	if selectedIndex >= 0 {
		// highlights row with exact prefix match to search query.
		b.SetCurrentItem(selectedIndex)

		// scroll to view; use height of list box
		b.SetOffset(int(math.Max(float64(selectedIndex-height+1), 0)), 0)
	} else {
		// highlight any selected row if not in visible rect
		if !b.InRect(b.GetCurrentItem(), 0) {
			b.SetOffset(b.GetCurrentItem(), 0)
		}
	}
}

// isNavigationalKey returns true if the key is for navigation purposes
func (lb *ListBox) isNavigationalKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyCtrlP, tcell.KeyCtrlN,
		tcell.KeyHome, tcell.KeyEnd, tcell.KeyPgUp, tcell.KeyPgDn,
		tcell.KeyEnter, tcell.KeyEscape, tcell.KeyTab:
		return true
	default:
		return false
	}
}

// InputHandler overrides default handling to switch focus away from search box when necessary.
func (lb *ListBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return lb.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {

		// Handle Enter key press
		if event.Key() == tcell.KeyEnter {
			setFocus(lb.contentView)
			log.Printf("[DEBUG] ListBox: Enter pressed, setting focus to content view")
			return
		}

		// Forward non-navigational characters to SearchBox
		if !lb.isNavigationalKey(event) {
			log.Printf("[DEBUG] ListBox: Non-navigational key pressed, forwarding to SearchBox")
			setFocus(lb.searchView)
			// Replace SearchBox text with the new character
			if event.Rune() != 0 {
				lb.searchView.SetText(string(event.Rune()))
				log.Printf("[DEBUG] ListBox: Set SearchBox text to '%s'", string(event.Rune()))
			}
			return
		}

		// Store the current item before handling the event
		before := lb.GetCurrentItem()

		// Allow the underlying List to handle input events
		if handler := lb.List.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		// For arrow keys, always sync SearchBox and ContentView regardless of selection change
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown || event.Key() == tcell.KeyCtrlP || event.Key() == tcell.KeyCtrlN {
			lb.SetSelectedFocusOnly(false)
			currentItem := lb.GetCurrentItem()
			if currentItem < len(lb.notes.LastSearchResults) {
				filename := lb.notes.LastSearchResults[currentItem].DisplayName()
				log.Printf("[DEBUG] ListBox: Arrow key pressed, updating search box to '%s'", filename)
				lb.searchView.SetTextFromList(filename)
				result := lb.notes.LastSearchResults[currentItem]
				lb.contentView.SetFile(result.FileRef)
			}
			return
		}

		// Check if selection has changed for other events
		if before != lb.GetCurrentItem() {
			log.Printf("[DEBUG] ListBox: Selection changed from %d to %d", before, lb.GetCurrentItem())
			lb.SetSelectedFocusOnly(false)
			if lb.GetCurrentItem() < len(lb.notes.LastSearchResults) {
				filename := lb.notes.LastSearchResults[lb.GetCurrentItem()].DisplayName()
				log.Printf("[DEBUG] ListBox: Updating search box to '%s'", filename)
				lb.searchView.SetTextFromList(filename)
			}
			return
		}

		// no change in selection (example: entering arrow key when already at top/bottom of list)
		log.Printf("[DEBUG] ListBox: No change in selection, current item remains %d", before)
	})
}
