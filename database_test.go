package nve

import (
	"fmt"
	"os"
	"testing"
	"time"
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
				if err == os.ErrNotExist {
					t.Error("DB does not exist")
				}
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
					t.Error("DB did not open")
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
			assert: func(t *testing.T, d *DB) { checkIsIndexed(t, db, fileRef) },
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

			db.Insert(fileRef, data)
			tc.assert(t, db)
		})
	}
}

// func TestDocumentUpdate(t *testing.T) {
// 	var (
// 		filename   = "/tmp/some_file.txt"
// 		md5        = "b9fe6c5ee4966accc23e32adea6f537d"
// 		modifiedAt = time.Now()
// 		data       = []byte("some data")
// 	)

// 	db = MustOpen(dbPath)
// 	db.Insert(filename, md5, modifiedAt, data)

// 	db.Insert(filename, "NEW_MD5", modifiedAt, []byte("fresher data"))

// 	var currentData []string
// 	if err := db.Select(&currentData, "SELECT text from content_index"); err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	if len(currentData) > 1 {
// 		t.Errorf("invalid number of rows. expected '1', was '%v'", len(currentData))
// 	}

// 	if currentData[0] != "fresher data" {
// 		t.Error("document data was not updated")
// 	}

// }

func checkCount(t *testing.T, db *DB, expected int) {
	var count int
	db.QueryRow("SELECT count(*) from documents").Scan(&count)

	if count != expected {
		t.Errorf("document count '%v' does not match expected '%v'\n", count, expected)
	}
}

func checkIsIndexed(t *testing.T, db *DB, fileRef *FileRef) {
	if !db.IsIndexed(fileRef) {
		t.Errorf("expected file '%s' to appear in index", fileRef.Filename)
	}
}
