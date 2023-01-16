package nve

import (
	"database/sql"
	"errors"
	"fmt"

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

func (db *DB) IsIndexed(fileRef *FileRef) bool {
	var count int

	if err := db.QueryRow(`
		SELECT count(*)
		FROM
			documents
		WHERE
			filename = ?
		AND
			md5 = ?
		AND
			modified_at = ?
	`, fileRef.Filename, fileRef.MD5, fileRef.ModifiedAt).Scan(&count); err != nil {
		return false
	}

	return count > 0
}

func (db *DB) Insert(fileRef *FileRef, data []byte) error {
	if fileRef.Filename == "" {
		return errors.New("filename is blank")
	}
	if fileRef.MD5 == "" {
		return errors.New("md5 is blank")
	}
	if fileRef.ModifiedAt.IsZero() {
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
	`, fileRef.Filename, fileRef.MD5, fileRef.ModifiedAt).Scan()

	if err == sql.ErrNoRows {
		// Insert
		var (
			res   sql.Result
			docId int64
		)

		res, err = db.NamedExec(`
			INSERT INTO documents
				(filename, md5, modified_at)
			VALUES
				(:filename, :md5, :modified_at)
			ON CONFLICT(filename) DO NOTHING;
		`, fileRef)

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
				(?, , ?);
		`, docId, fileRef.Filename, string(data))
	}

	return err
}
