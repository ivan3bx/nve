package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func searchBox() *tview.TextArea {
	textArea := tview.NewTextArea().SetPlaceholder(">")

	textArea.SetBorder(true).
		SetTitle("Search Box").
		SetBackgroundColor(tcell.ColorBlack).
		SetTitleColor(tcell.ColorYellow).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetTitleAlign(tview.AlignLeft)

	return textArea
}

func listBox() *tview.List {
	listView := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedStyle(
			tcell.StyleDefault.
				Background(tcell.ColorDarkBlue).
				Foreground(tcell.ColorLightSkyBlue),
		)

	listView.SetBorder(true).
		SetTitle("List Box").
		SetTitleColor(tcell.ColorOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetTitleAlign(tview.AlignLeft)

	// sample data
	listView.AddItem("Main text goes here", "", 0, nil)
	listView.AddItem("Second item here", "", 0, nil)

	return listView
}

func contentBox() *tview.TextArea {
	textArea := tview.NewTextArea()

	textArea.SetBorder(true).
		SetTitle("Content").
		SetTitleColor(tcell.ColorDarkOrange).
		SetBorderStyle(tcell.StyleDefault.Dim(true)).
		SetTitleAlign(tview.AlignLeft)

	textArea.SetText("this could be lots of content\n\n# Separated by other stuff.\n\n* one item\n* two item\n", true)
	return textArea
}

func main() {
	app := tview.NewApplication()

	var (
		searchBox  = searchBox()
		listBox    = listBox()
		contentBox = contentBox()
	)

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(searchBox, 3, 0, false).
				AddItem(listBox, 0, 1, true).
				AddItem(contentBox, 0, 3, false), 0, 2, false,
		)
	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}
