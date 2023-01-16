package nve

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var dbPath string
var db *DB

func init() {
	tempFile, err := os.CreateTemp(os.TempDir(), "*.db")
	if err != nil {
		panic(err)
	}
	dbPath = tempFile.Name()
	fmt.Println(dbPath)
	os.Remove(dbPath)
}

func teardown() {
	if db != nil {
		db.Close()
	}
	os.Remove(dbPath)
}

func TestDatabaseInitialization(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(*DB)
		assert func(*testing.T, *DB)
	}{
		{
			name: "Creates a new database",

			assert: func(t *testing.T, d *DB) {
				_, err := os.Stat(dbPath)
				assert.NoError(t, err, "DB does not exist")
			},
		},
		{
			name: "Re-opens xisting database",

			setup: func(d *DB) {
				d.Close()
				db = MustOpen(dbPath)
			},

			assert: func(t *testing.T, d *DB) {
				if err := db.Ping(); err != nil {
					assert.NoError(t, err, "DB did not open")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db = MustOpen(dbPath)
			defer teardown()

			if tc.setup != nil {
				tc.setup(db)
			}

			tc.assert(t, db)
		})
	}
}

func TestDocumentInsertion(t *testing.T) {
	var (
		fileRef *FileRef
		data    []byte
	)

	testCases := []struct {
		name   string
		setup  func()
		assert func(*testing.T, *DB)
	}{
		{
			name:   "is successful",
			assert: func(t *testing.T, db *DB) { checkCount(t, db, 1) },
		},
		{
			name:   "is indexed",
			assert: func(t *testing.T, d *DB) { checkIsUnmodified(t, db, fileRef) },
		},
		{
			name:   "requires filename",
			setup:  func() { fileRef.Filename = "" },
			assert: func(t *testing.T, db *DB) { checkCount(t, db, 0) },
		},
		{
			name:   "requires md5",
			setup:  func() { fileRef.MD5 = "" },
			assert: func(t *testing.T, db *DB) { checkCount(t, db, 0) },
		},
		{
			name:   "requires last modified date",
			setup:  func() { fileRef.ModifiedAt = time.Time{} },
			assert: func(t *testing.T, db *DB) { checkCount(t, db, 0) },
		},
		{
			name:   "allows empty data",
			setup:  func() { data = nil },
			assert: func(t *testing.T, db *DB) { checkCount(t, db, 1) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileRef = &FileRef{
				Filename:   "/tmp/some_file.txt",
				MD5:        "b9fe6c5ee4966accc23e32adea6f537d",
				ModifiedAt: time.Now(),
			}
			data = []byte("some data")

			db = MustOpen(dbPath)

			defer teardown()

			if tc.setup != nil {
				tc.setup()
			}

			db.Upsert(fileRef, data)
			tc.assert(t, db)
		})
	}
}

func TestDocumentUpdate(t *testing.T) {
	var (
		data    = []byte("some data")
		fileRef = FileRef{
			Filename:   "/tmp/some_file.txt",
			MD5:        "b9fe6c5ee4966accc23e32adea6f537d",
			ModifiedAt: time.Now(),
		}
	)

	db = MustOpen(dbPath)

	// Insert initial record
	err := db.Insert(&fileRef, data)

	if err != nil {
		assert.FailNow(t, "Query failed", err)
	}

	newRef := fileRef
	newRef.MD5 = "NEW_MD5"

	if err := db.Upsert(&newRef, []byte("fresher data")); err != nil {
		assert.FailNow(t, "upsert failed", err)
	}

	var currentData []string
	if err := db.Select(&currentData, "SELECT text from content_index"); err != nil {
		assert.FailNow(t, "index did not return results", err)
	}

	if len(currentData) != 1 {
		assert.FailNowf(t, "invalid number of rows", "expected '1', was '%v'", len(currentData))
	}

	assert.Equal(t, "fresher data", currentData[0], "document data was not updated")

}

func checkCount(t *testing.T, db *DB, expected int) {
	var count int
	db.QueryRow("SELECT count(*) from documents").Scan(&count)

	if count != expected {
		t.Errorf("document count '%v' does not match expected '%v'\n", count, expected)
	}
}

func checkIsUnmodified(t *testing.T, db *DB, fileRef *FileRef) {
	if !db.IsUnmodified(fileRef) {
		t.Errorf("expected file '%s' to appear in index", fileRef.Filename)
	}
}
