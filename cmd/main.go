package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	var (
		topBox     = tview.NewBox().SetBorder(true).SetTitle("Search Box").SetTitleColor(tcell.ColorYellow).SetTitleAlign(tview.AlignLeft)
		listBox    = tview.NewBox().SetBorder(true).SetTitle("List Box").SetTitleColor(tcell.ColorOrange).SetTitleAlign(tview.AlignLeft)
		contentBox = tview.NewBox().SetBorder(true).SetTitle("Content").SetTitleColor(tcell.ColorDarkOrange).SetTitleAlign(tview.AlignLeft)
	)

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(topBox, 3, 0, true).
				AddItem(listBox, 0, 1, false).
				AddItem(contentBox, 0, 3, false), 0, 2, false,
		)
	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}
