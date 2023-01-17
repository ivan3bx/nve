package nve

import (
	"os"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type NotesConfig struct {
	Filepath string
	DBPath   string
}

type Notes struct {
	LastSearchResults []*SearchResult

	config    NotesConfig
	db        *DB
	observers []Observer
}

var DefaultDBPath = "./nve.db"

func NewNotes(config NotesConfig) *Notes {
	if config.Filepath == "" {
		config.Filepath, _ = os.Getwd()
	}

	if config.DBPath == "" {
		config.DBPath = DefaultDBPath
	}

	notes := &Notes{config: config}
	notes.db = MustOpen(config.DBPath)

	if err := notes.refresh(); err != nil {
		panic(err)
	}

	return notes
}

// Search returns a set of filepaths matching the given search string.
func (n *Notes) Search(text string) ([]string, error) {
	var (
		res []string
		err error
	)

	// 1. perform the search
	if n.LastSearchResults, err = n.db.Search(text); err != nil {
		return nil, err
	}

	// 2. update results (save in field)
	n.Notify()

	// 3. return results
	res = make([]string, 0)

	for _, file := range n.LastSearchResults {
		res = append(res, file.Filename)
	}

	return res, nil
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

func (n *Notes) refresh() error {
	var db = n.db

	files, err := scanDirectory(n.config.Filepath)

	if err != nil {
		return err
	}

	for _, file := range files {
		md5, _ := calculateMD5(file)
		stats, _ := os.Stat(file)

		ref := FileRef{
			Filename:   file,
			MD5:        md5,
			ModifiedAt: stats.ModTime(),
		}

		// Skip unmodified documents
		if db.IsUnmodified(&ref) {
			continue
		}

		bytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		if err := db.Upsert(&ref, bytes); err != nil {
			return err
		}
	}

	return nil
}
