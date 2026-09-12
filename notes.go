package nve

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)

// recentLimit is the number of results shown for an empty query.
const recentLimit = 20

type NotesConfig struct {
	Filepath string
}

type Notes struct {
	LastQuery         string
	LastSearchResults []*SearchResult

	config    NotesConfig
	observers []Observer
	watcher   io.Closer
	drawFunc  func(func())

	// stale forces the next search to rescan the directory instead of
	// narrowing the previous results.
	stale bool
}

func NewNotes(config NotesConfig) *Notes {
	if config.Filepath == "" {
		config.Filepath, _ = os.Getwd()
	}

	notes := &Notes{
		config: config,
	}

	notes.Search("")
	return notes
}

// Search returns a set of filepaths matching the given search string.
func (n *Notes) Search(text string) ([]string, error) {
	var (
		searchResults []*SearchResult
		err           error
	)

	log.Printf("[DEBUG] Notes: Search called with text='%s'", text)

	switch {
	case text == "":
		searchResults, err = recentFiles(n.config.Filepath, recentLimit)
	case n.canNarrow(text):
		log.Printf("[DEBUG] Notes: narrowing %d previous results", len(n.LastSearchResults))
		searchResults = searchRefs(restatRefs(n.LastSearchResults), searchTerms(text))
	default:
		searchResults, err = searchFiles(n.config.Filepath, text)
	}

	if err != nil {
		return nil, err
	}

	n.LastQuery = text
	n.stale = false

	// 1. perform the search
	n.LastSearchResults = searchResults

	// 2. update results (save in field)
	log.Printf("[DEBUG] Notes: Notifying %d observers of search results", len(n.observers))
	n.Notify()

	// 3. return results
	res := make([]string, 0)

	for _, file := range n.LastSearchResults {
		res = append(res, file.Filename)
	}

	return res, nil
}

// canNarrow reports whether text extends the previous query. Extending a query
// (adding characters or terms) can only shrink the result set, so only the
// previous results need to be searched.
func (n *Notes) canNarrow(text string) bool {
	return !n.stale && strings.TrimSpace(n.LastQuery) != "" && strings.HasPrefix(text, n.LastQuery)
}

// invalidate marks the previous results as unreliable, typically because the
// directory changed, so the next search performs a full rescan.
func (n *Notes) invalidate() {
	n.stale = true
}

func (n *Notes) CreateNote(name string) (*FileRef, error) {
	path := filepath.Join(n.config.Filepath, fmt.Sprintf("%s.%s", name, "md"))
	newFile, err := os.OpenFile(path, os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	defer newFile.Close()

	stat, err := newFile.Stat()

	if err != nil {
		return nil, err
	}

	n.invalidate()

	return &FileRef{
		Filename:   newFile.Name(),
		ModifiedAt: stat.ModTime(),
	}, nil
}

func (n *Notes) RegisterObservers(obs ...Observer) {
	n.observers = obs
}

func (n *Notes) Notify() {
	for _, obj := range n.observers {
		obj.SearchResultsUpdate(n)
	}
}
