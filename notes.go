package nve

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

var logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)

type NotesConfig struct {
	Filepath string
	DBPath   string
}

type Notes struct {
	LastQuery         string
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

	notes := &Notes{
		config: config,
		db:     MustOpen(config.DBPath),
	}

	if err := notes.Refresh(); err != nil {
		panic(err)
	}

	return notes
}

// Search returns a set of filepaths matching the given search string.
func (n *Notes) Search(text string) ([]string, error) {
	var (
		searchResults []*SearchResult
		err           error
	)

	log.Printf("[DEBUG] Notes: Search called with text='%s'", text)
	n.LastQuery = text

	if text == "" {
		searchResults, err = n.db.Recent(20)
	} else {
		searchResults, err = n.db.Search(text)
	}

	if err != nil {
		return nil, err
	}

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

func (n *Notes) CreateNote(name string) (*FileRef, error) {
	path := filepath.Join(n.config.Filepath, fmt.Sprintf("%s.%s", name, "md"))
	newFile, err := os.OpenFile(path, os.O_CREATE, 0644)

	if err != nil {
		return nil, err
	}

	md5, err := calculateMD5(path)

	if err != nil {
		return nil, err
	}

	stat, err := newFile.Stat()

	if err != nil {
		return nil, err
	}

	fileRef := FileRef{
		Filename:   newFile.Name(),
		MD5:        md5,
		ModifiedAt: stat.ModTime(),
	}

	if err := n.db.Insert(&fileRef, []byte{}); err != nil {
		return nil, err
	}

	return &fileRef, nil
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

func (n *Notes) Refresh() error {
	var db = n.db

	// Get all files currently on disk
	files, err := scanDirectory(n.config.Filepath)
	if err != nil {
		return err
	}

	// Create a map of existing files for quick lookup
	existingFiles := make(map[string]bool)
	for _, file := range files {
		existingFiles[file] = true
	}

	// Get all files currently in the database
	dbFiles, err := db.GetAllFileRefs()
	if err != nil {
		return err
	}

	// Prune files from database that no longer exist on disk
	refsToPrune := []*FileRef{}

	for _, dbFile := range dbFiles {
		if !existingFiles[dbFile.Filename] {
			refsToPrune = append(refsToPrune, dbFile)
		}
	}

	if len(refsToPrune) > 0 {
		if err := db.PruneFileRefs(refsToPrune); err != nil {
			logger.Printf("Error pruning files from database: %v", err)
			return err
		}
	}

	// Process files that exist on disk (existing logic)
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

	n.Search("")
	return nil
}
