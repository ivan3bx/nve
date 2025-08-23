package nve

type Observer interface {
	SearchResultsUpdate(*SearchContext)
}
