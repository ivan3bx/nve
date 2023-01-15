package nve

import (
	"database/sql"
	"fmt"
	"time"
)

func initializeDB() *sql.DB {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_fk=true&loc=auto", DBNAME))

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
	return db
}

func insertDocument(db *sql.DB, filepath string, md5 string, modififiedAt time.Time, data []byte) error {
	res, err := db.Exec(`
		INSERT INTO documents
			(filename, md5, modified_at)
		VALUES
			(?, ?, ?)
		ON CONFLICT(filename) DO NOTHING;
	`, filepath, md5, modififiedAt)

	if err != nil {
		return err
	}

	docId, err := res.LastInsertId()

	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO content_index
			(document_id, filename, text)
		VALUES
			(?, ?, ?);
	`, docId, filepath, string(data))

	return err
}
