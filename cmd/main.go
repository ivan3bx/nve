package main

import (
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/ivan3bx/nve"
	"github.com/rivo/tview"
)

func main() {
	// Setup debug logging to file
	logFile, err := os.OpenFile("nve-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	var (
		app   = tview.NewApplication()
		notes = nve.NewNotes(nve.NotesConfig{
			Filepath: "./",
		})

		// View hierarchy
		contentBox = nve.NewContentBox()
		listBox    = nve.NewListBox(contentBox, notes)
		searchBox  = nve.NewSearchBox(listBox, contentBox, notes)
	)

	notes.RegisterObservers(listBox)
	notes.Notify()

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
			app.SetFocus(searchBox)
			searchBox.SetText("")
			notes.Search("")
			return &tcell.EventKey{}
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
