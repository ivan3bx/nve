package nve

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type Notes struct {
	Query     string
	db        *sql.DB
	observers []Observer
}

var DBNAME = "nve.db"

func NewNotes() *Notes {
	notes := &Notes{db: initializeDB()}

	if err := notes.refresh(); err != nil {
		panic(err)
	}

	return notes
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

func (n *Notes) refresh() error {
	var db = n.db

	for _, file := range scanCurrentDirectory() {
		md5, _ := calculateMD5(file)
		stats, _ := os.Stat(file)

		// Skip unmodified documents
		var count int

		if err := db.QueryRow("SELECT count(*) FROM documents WHERE filename = ? AND md5 = ? AND modified_at = ?", file, md5, stats.ModTime()).Scan(&count); err != nil {
			return err
		}

		if count > 0 {
			// skip unmodified documents
			continue
		}

		bytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		if err := insertDocument(db, file, md5, stats.ModTime(), bytes); err != nil {
			return err
		}
	}

	return nil
}
