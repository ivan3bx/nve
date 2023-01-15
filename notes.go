package nve

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

type Notes struct {
	Query     string
	db        *sql.DB
	observers []Observer
}

func NewNotes() *Notes {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:nve.db?_fk=true&loc=auto"))

	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id 					INTEGER PRIMARY KEY AUTOINCREMENT,
			filename 			varchar(255) NOT NULL UNIQUE,
			md5 				TEXT,
			modified_at			DATETIME,
			last_indexed_at 	DATETIME
		);
	`,
	)

	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS content_index USING FTS5 (
			document_id, filename, text
		);

	`)

	if err != nil {
		panic(err)
	}

	// populate database
	files := scanCurrentDirectory()

	for _, file := range files {
		md5, _ := calculateMD5(file)
		stats, _ := os.Stat(file)

		var count int
		db.QueryRow("SELECT count(*) FROM documents WHERE filename = ? AND md5 = ? AND modified_at = ?", file, md5, stats.ModTime()).Scan(&count)

		// Skip unmodified documents
		if count > 0 {
			continue
		}

		if bytes, err := os.ReadFile(file); err == nil {
			res, err := db.Exec(`
				INSERT INTO documents
					(filename, md5, modified_at)
				VALUES
					(?, ?, ?)
				ON CONFLICT(filename) DO NOTHING;
			`, file, md5, stats.ModTime())

			if err != nil {
				panic(err)
			}

			docId, err := res.LastInsertId()

			if err != nil {
				panic(err)
			}

			_, err = db.Exec(`
				INSERT INTO content_index
					(document_id, filename, text)
				VALUES
					(?, ?, ?);
			`, docId, file, string(bytes))

			if err != nil {
				panic(err)
			}
		}
	}

	notes := &Notes{db: db}

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

func scanCurrentDirectory() []string {
	var files []string

	supported_types := map[string]bool{
		".txt":   true,
		".md":    true,
		".mdown": true,
		".go":    true,
	}

	cwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(cwd, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if !info.IsDir() && supported_types[filepath.Ext(path)] {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return files
}

func calculateMD5(path string) (string, error) {
	file, err := os.Open(path)

	if err != nil {
		return "", err
	}

	defer file.Close()

	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
