package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/ivan3bx/nve"
	"github.com/rivo/tview"
)

func main() {
	var (
		app   = tview.NewApplication()
		notes = nve.NewNotes(nve.NotesConfig{
			Filepath: "./",
		})

		// View hierarchy
		contentBox = nve.NewContentBox()
		listBox    = nve.NewListBox(contentBox)
		searchBox  = nve.NewSearchBox(listBox, notes)
	)

	notes.RegisterObservers(contentBox, listBox)

	// global input events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			if searchBox.HasFocus() {
				app.SetFocus(listBox)
			} else if listBox.HasFocus() {
				app.SetFocus(contentBox)
			} else {
				break
			}
			return &tcell.EventKey{}
		case tcell.KeyEscape:
			if contentBox.HasFocus() {
				app.SetFocus(searchBox)
				return &tcell.EventKey{}
			}
		}

		return event
	})

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(searchBox, 3, 0, true).
				AddItem(listBox, 0, 1, false).
				AddItem(contentBox, 0, 3, false), 0, 2, true,
		)

	if err := app.SetRoot(flex, true).SetFocus(flex).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
