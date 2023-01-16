package nve

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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

func (db *DB) IsUnmodified(fileRef *FileRef) bool {
	var count int

	err := db.QueryRow(`
		SELECT count(*)
		FROM
			documents
		WHERE
			filename = ?
		AND
			md5 = ?
		AND
			modified_at = ?
	`, fileRef.Filename, fileRef.MD5, fileRef.ModifiedAt).Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

func (db *DB) GetFileRef(filename string) (*FileRef, error) {
	var ref FileRef

	err := db.QueryRowx(`
		SELECT
			id,
			filename,
			md5,
			modified_at
		FROM
			documents
		WHERE
			filename = ?
	`, filename).StructScan(&ref)

	if err != nil {
		return nil, err
	}

	return &ref, nil
}

func (db *DB) Upsert(fileRef *FileRef, data []byte) error {
	if fileRef.Filename == "" {
		return errors.New("filename is blank")
	}
	if fileRef.MD5 == "" {
		return errors.New("md5 is blank")
	}
	if fileRef.ModifiedAt.IsZero() {
		return errors.New("modifiedAt is not defined")
	}

	if db.IsUnmodified(fileRef) {
		return nil
	}

	oldRef, err := db.GetFileRef(fileRef.Filename)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return db.Insert(fileRef, data)
		default:
			panic(err)
		}
	} else {
		return db.Update(oldRef, fileRef, data)
	}
}

func (db *DB) Insert(fileRef *FileRef, data []byte) error {
	// Insert
	var (
		res   sql.Result
		docId int64
	)

	res, err := db.NamedExec(`
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

	fileRef.DocumentID = docId

	_, err = db.Exec(`
		INSERT INTO content_index
			(document_id, filename, text)
		VALUES
			(?, ?, ?);
	`, fileRef.DocumentID, fileRef.Filename, string(data))

	return errors.WithStack(err)
}

func (db *DB) Update(oldRef, newRef *FileRef, data []byte) error {
	newRef.Filename = oldRef.Filename

	res, err := db.NamedExec(`
		UPDATE documents
		SET
			md5         = :md5,
			modified_at = :modified_at
		WHERE
			filename = :filename
	`, newRef)

	if err != nil {
		return errors.WithStack(err)
	}

	if count, _ := res.RowsAffected(); count != 1 {
		return errors.New("update did not change any rows")
	}

	_, err = db.Exec(`
		UPDATE content_index
		SET
			text        = ?
		WHERE
			document_id = ?
	`, string(data), oldRef.DocumentID)

	return errors.WithStack(err)
}
