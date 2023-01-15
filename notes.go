package nve

type Notes struct {
	Query     string
	observers []Observer
}

func NewNotes() *Notes {
	return &Notes{}
}

func (n *Notes) Search(text string) {
	// 1. perform the search on local FS
	n.Query = text

	// 2. update results (save in field)
	n.Notify()
}

func (n *Notes) RegisterObservers(obs ...Observer) {
	if n.observers != nil {
		n.observers = obs
	} else {
		n.observers = append(n.observers, obs...)
	}
}

func (n *Notes) Notify() {
	for _, obj := range n.observers {
		obj.SearchResultsUpdate(n)
	}
}
