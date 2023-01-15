package nve

type Observer interface {
	SearchResultsUpdate(*Notes)
}
