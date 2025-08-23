package nve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockObserver struct {
	lastResults []*SearchResult
}

func (m *mockObserver) SearchResultsUpdate(sc *SearchContext) {
	m.lastResults = sc.LastSearchResults

}

func TestNotifyObservers(t *testing.T) {
	mock := mockObserver{}

	searchCtx := NewSearchContext()
	searchCtx.RegisterObservers(&mock)

	// Notes search will trigger an update to any observers
	notes.Search(searchCtx, "seattle")

	if assert.Len(t, mock.lastResults, 1) {
		res := mock.lastResults[0]

		// assert snippet
		assert.Equal(t, "new york **seattle**", res.Snippet)

		// assert filename
		assert.Equal(t, "test_data/apples in zoo.md", res.Filename)
	}
}
