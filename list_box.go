package nve

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

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

	// Custom Draw function will append timestamp to each line
	box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		innerX, innerY, innerWidth, innerHeight := box.GetInnerRect()
		offsetX, _ := box.GetOffset()

		for i := offsetX; i < offsetX+innerHeight && i < box.GetItemCount(); i++ {
			result := notes.LastSearchResults[i]
			log.Printf("[DEBUG] ListBox: DrawFunc called - Item %d: %d", i, len(result.Snippet))
			box.SetItemText(i, formatResult(result, innerWidth), "")

		}

		return innerX, innerY, innerWidth, innerHeight
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
		mainText := formatResult(result, -1)
		b.AddItem(mainText, "", 0, nil)

		if selectedIndex == -1 && strings.HasPrefix(result.DisplayName(), notes.LastQuery) {
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

func formatResult(result *SearchResult, maxWidth int) string {
	// Format of a single line in the list box:
	// <filename> : <snippet> <timestamp>
	//
	//   Filename is left-aligned, max 22 characters (20 + ".." if truncated)
	//   Snippet is left-aligned, max width depends on overall maxWidth
	//   Timestamp is right-aligned, fixed width of 20 characters (e.g., "Aug 16, 2025 12:15PM", or "5 min ago", or "now")
	//
	// If maxWidth is -1, no truncation or padding is applied to snippet or timestamp.
	// If maxWidth < 60, filename is omitted.

	filename := result.DisplayName()
	snippet := result.Snippet
	timestamp := formatModifiedTime(result.ModifiedAt)

	if len(filename) > 20 {
		filename = strings.TrimSpace(filename[:20])
		log.Printf("[DEBUG] ListBox: truncated filename to '%s'", filename)
		filename = fmt.Sprintf("%s...", filename)
	} else {
		filename = fmt.Sprintf("%-22s", filename)
	}

	timestamp = fmt.Sprintf("%20s", timestamp)

	// Replace newlines, tabs with spaces and collapse multiple spaces
	snippet = strings.Join(strings.Fields(snippet), " ")

	maxWidthOfSnippet := maxWidth

	if maxWidth < 0 {
		maxWidthOfSnippet = maxWidth - len(filename) - len(" | ")
	} else {
		maxWidthOfSnippet = maxWidth - len(filename) - len(timestamp) - len(" | ") - len(" ")
	}

	if maxWidthOfSnippet > 0 && len(snippet) > maxWidthOfSnippet {
		snippet = snippet[:maxWidthOfSnippet-2] + ".."
	} else {
		// pad snippet to maxWidthOfSnippet
		snippet = fmt.Sprintf("%-*s", maxWidthOfSnippet, snippet)
	}

	mainText := ""
	if maxWidth < 0 {
		mainText = strings.TrimSpace(fmt.Sprintf("%s   %s", filename, snippet))
	} else {
		mainText = fmt.Sprintf("%s   %s %s", filename, snippet, timestamp)
	}

	return tview.Escape(mainText)
}

// isNavigationalKey returns true if the key is for navigation purposes
func (lb *ListBox) isNavigationalKey(event *tcell.EventKey) bool {
	switch event.Key() {
	case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyCtrlP, tcell.KeyCtrlN,
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

		// Handle left arrow to move focus to SearchBox with cursor at start
		if event.Key() == tcell.KeyLeft {
			log.Printf("[DEBUG] ListBox: Left arrow pressed, moving focus to SearchBox")
			setFocus(lb.searchView)
			// Move cursor to start by simulating Home key press
			if handler := lb.searchView.InputHandler(); handler != nil {
				handler(tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone), setFocus)
			}
			return
		}

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

func formatModifiedTime(modTime time.Time) string {
	now := time.Now()
	diff := now.Sub(modTime)

	// Less than 1 day: show relative time
	if diff < 24*time.Hour {
		return modTime.Format("3:04PM")
	}

	// 1 week or older: show "Aug 16, 2025 12:15PM" format
	return modTime.Format("Jan 02, 2006")
}
