package nve

import "log"

type SearchContext struct {
	LastQuery         string
	LastSearchResults []*SearchResult

	observers []Observer
}

func NewSearchContext() *SearchContext {
	return &SearchContext{
		LastQuery:         "",
		LastSearchResults: make([]*SearchResult, 0),
	}
}

func (sc *SearchContext) RegisterObservers(obs ...Observer) {
	if sc.observers != nil {
		sc.observers = obs
	} else {
		sc.observers = append(sc.observers, obs...)
	}
}

func (sc *SearchContext) Notify() {
	log.Printf("[DEBUG] Notes: Notifying %d observers of search results", len(sc.observers))

	for _, obj := range sc.observers {
		obj.SearchResultsUpdate(sc)
	}
}
