package nve

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type EditMode struct {
	*tview.TextArea
	contentBox *ContentBox
}

func NewEditMode(cbox *ContentBox, box *tview.Box) *EditMode {
	editMode := EditMode{
		TextArea:   tview.NewTextArea(),
		contentBox: cbox,
	}

	boxCopy := *box
	editMode.Box = &boxCopy

	editMode.SetFocusFunc(func() {
		// ignore edits if there is no current file
		if cbox.currentFile == nil {
			editMode.Blur()
		}
	})

	return &editMode
}

func (em *EditMode) GetContent() string {
	return em.GetText()
}

func (em *EditMode) SetContent(content string) {
	em.SetText(content, false)
}

func (em *EditMode) Clear() {
	em.SetText("", true)
}

func (em *EditMode) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	parent := em.TextArea

	return em.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		event = em.contentBox.mapSpecialKeys(event)

		before := parent.GetText()

		if handler := parent.InputHandler(); handler != nil {
			handler(event, setFocus)
		}

		if after := parent.GetText(); before != after {
			em.contentBox.queueSave(after)
		}
	})
}
