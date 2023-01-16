package nve

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type DB struct {
	*sqlx.DB
}

func MustOpen(file string) *DB {
	db := sqlx.MustOpen("sqlite3", fmt.Sprintf("file:%s?_fk=true&loc=auto", file))

	_, err := db.Exec(`
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
	return &DB{db}
}

func (db *DB) IsIndexed(file, md5 string, modifiedAt time.Time) bool {
	var count int

	if err := db.QueryRow("SELECT count(*) FROM documents WHERE filename = ? AND md5 = ? AND modified_at = ?", file, md5, modifiedAt).Scan(&count); err != nil {
		return false
	}

	return count > 0
}

func (db *DB) Insert(filepath string, md5 string, modifiedAt time.Time, data []byte) error {
	if filepath == "" {
		return errors.New("filename is blank")
	}
	if md5 == "" {
		return errors.New("md5 is blank")
	}
	if modifiedAt.IsZero() {
		return errors.New("modifiedAt is not defined")
	}

	err := db.QueryRow(`
		SELECT 1 FROM documents
		WHERE
			filename = ?
		AND
			md5 = ?
		AND
			modified_at = ?
	`, filepath, md5, modifiedAt).Scan()

	if err == sql.ErrNoRows {
		// Insert
		var (
			res   sql.Result
			docId int64
		)

		res, err = db.Exec(`
			INSERT INTO documents
				(filename, md5, modified_at)
			VALUES
				(?, ?, ?)
			ON CONFLICT(filename) DO NOTHING;
		`, filepath, md5, modifiedAt)

		if err != nil {
			return err
		}

		docId, err = res.LastInsertId()

		if err != nil {
			return err
		}

		_, err = db.Exec(`
			INSERT INTO content_index
				(document_id, filename, text)
			VALUES
				(?, ?, ?);
		`, docId, filepath, string(data))
	}

	return err
}
