package nve

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type DB struct {
	*sqlx.DB
}

type SearchResult struct {
	*FileRef
	Snippet string `db:"snippet"`
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
			document_id, filename, text, tokenize = 'porter unicode61'
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
			logger.Printf("DB.Upsert: %v\n", err)
			return errors.WithStack(err)
		}
	} else {
		return db.Update(oldRef, fileRef, data)
	}
}

func (db *DB) Recent(limit int) ([]*SearchResult, error) {
	var (
		res []*SearchResult
		err error
	)

	err = db.Select(&res, `
		SELECT
			docs.id, docs.filename, docs.md5, docs.modified_at,
			substr(cti.text, 0, 120) as snippet
		FROM
			documents docs
		INNER JOIN
			content_index cti
		ON
			cti.document_id = docs.id
		ORDER BY
			docs.modified_at desc
		LIMIT ?
	`, limit)

	if err != nil {
		logger.Printf("DB.Recent: %v\n", err)
		return nil, errors.WithStack(err)
	}

	return res, nil
}

// Search performs FTS on filename and text using default NEAR semantics
// and includes snippet text up to 10 'word' tokens in length.
func (db *DB) Search(text string) ([]*SearchResult, error) {
	var (
		res []*SearchResult
		err error
	)

	term := ftsMatchString(text)
	err = db.Select(&res, `
		SELECT
			docs.id, docs.filename, docs.md5, docs.modified_at,
			snippet(content_index, 2, "**", "**", '...', 10) as snippet
		FROM
			documents docs
		INNER JOIN
			content_index cti
		ON
			cti.document_id = docs.id
		WHERE
			content_index match (?)
	`, fmt.Sprintf("filename:NEAR(%s) OR text:NEAR(%s)", term, term))

	if err != nil {
		logger.Printf("DB.Search: %v\n", err)
		return nil, errors.WithStack(err)
	}

	return res, nil
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
		logger.Printf("DB.Insert: %v\n", err)
		return err
	}

	docId, err = res.LastInsertId()

	if err != nil {
		logger.Printf("DB.Recent: %v\n", err)
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
		logger.Printf("DB.Update: %v\n", err)
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

// GetAllFileRefs returns all files currently in the database
func (db *DB) GetAllFileRefs() ([]*FileRef, error) {
	var files []*FileRef

	err := db.Select(&files, `
		SELECT
			id,
			filename,
			md5,
			modified_at
		FROM
			documents
		ORDER BY filename
	`)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return files, nil
}

// PruneFileRefs removes a file from both documents and content_index tables
func (db *DB) PruneFileRefs(refs []*FileRef) error {

	var (
		query string
		args  []interface{}
		err   error

		docIDs []int64
	)

	for _, ref := range refs {
		docIDs = append(docIDs, ref.DocumentID)
	}

	//
	// Delete from content_index table
	//
	if query, args, err = sqlx.In(`DELETE FROM content_index WHERE document_id IN (?)`, docIDs); err != nil {
		return errors.WithStack(err)
	}

	query = db.Rebind(query)

	if _, err := db.Exec(query, args...); err != nil {
		return errors.WithStack(err)
	}

	//
	// Delete from documents table
	//
	if query, args, err = sqlx.In(`DELETE FROM documents WHERE id IN (?)`, docIDs); err != nil {
		return errors.WithStack(err)
	}

	query = db.Rebind(query)

	if _, err := db.Exec(query, args...); err != nil {
		return errors.WithStack(err)
	}

	log.Printf("[DEBUG] Pruned %d files from database", len(refs))

	return nil
}

// ftsMatchString converts an expression into a wildcard match.
// Each term is quoted, so as to accept non-word characters
// without blowing up SQLite's query parser.
//
// Examples:
//
//	"foo"     => '"foo"*'
//	"foo bar" => '"foo"* "bar"*'
//	"foo-bar" => '"foo-bar"*'
func ftsMatchString(text string) string {
	sb := []string{}

	for _, part := range strings.Split(text, " ") {
		sb = append(sb, fmt.Sprintf(`"%s"*`, part))
	}

	return strings.Join(sb, " ")
}
