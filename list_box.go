package nve

import (
	"math"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ListBox struct {
	*tview.List
	contentView *ContentBox
}

func NewListBox(contentView *ContentBox, notes *Notes) *ListBox {
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

	box.SetSelectedFocusOnly(true)

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

	b.Clear()

	b.SetSelectedFocusOnly(emptyQuery)

	if len(lastResult) == 0 {
		b.contentView.Clear()
	}

	selectedIndex := -1

	for index, result := range lastResult {
		displayName := strings.TrimSuffix(filepath.Base(result.Filename), filepath.Ext(result.Filename))
		b.AddItem(displayName, "", 0, nil)

		if strings.HasPrefix(displayName, notes.LastQuery) {
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
