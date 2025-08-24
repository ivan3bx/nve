package nve

import "github.com/rivo/tview"

// ViewMode handles read-only content with highlighting using TextView
type ViewMode struct {
	*tview.TextView
	searchCtx *SearchContext
}

func NewViewMode(sctx *SearchContext, box *tview.Box) *ViewMode {
	viewMode := ViewMode{
		TextView:  tview.NewTextView(),
		searchCtx: sctx,
	}

	viewMode.SetDynamicColors(true)

	boxCopy := *box
	viewMode.Box = &boxCopy

	return &viewMode
}

func (vm *ViewMode) GetContent() string {
	return vm.GetText(false)
}

func (vm *ViewMode) SetContent(content string) {
	highlighted := highlightText(content, vm.searchCtx.LastQuery)
	vm.SetText(highlighted)
}

func (vm *ViewMode) Clear() {
	vm.SetText("")
}
