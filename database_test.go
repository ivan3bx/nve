package nve

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func withNewDB(runtest func(db *DB)) {
	withNewDBPath(func(db *DB, dbPath string) {
		runtest(db)
	})
}

func withNewDBPath(runtest func(db *DB, dbPath string)) {
	tempFile, err := os.CreateTemp(os.TempDir(), "*.db")
	if err != nil {
		panic(err)
	}
	dbPath := tempFile.Name()
	os.Remove(dbPath)

	db := MustOpen(dbPath)
	defer func() {
		if db != nil {
			db.Close()
		}
		os.Remove(dbPath)
	}()

	runtest(db, dbPath)
}

func TestDatabaseInitialization(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(db *DB, dbPath string) *DB
		assert func(*testing.T, *DB, string)
	}{
		{
			name: "Creates a new database",

			assert: func(t *testing.T, d *DB, dbPath string) {
				_, err := os.Stat(dbPath)
				assert.NoError(t, err, "DB does not exist")
			},
		},
		{
			name: "Re-opens existing database",

			setup: func(db *DB, dbPath string) *DB {
				db.Close()
				return MustOpen(dbPath)
			},

			assert: func(t *testing.T, db *DB, dbPath string) {
				if err := db.Ping(); err != nil {
					assert.NoError(t, err, "DB did not open")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withNewDBPath(func(db *DB, dbPath string) {
				if tc.setup != nil {
					db = tc.setup(db, dbPath)
				}

				tc.assert(t, db, dbPath)
			})
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
			assert: func(t *testing.T, db *DB) { checkIsUnmodified(t, db, fileRef) },
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

			withNewDB(func(db *DB) {
				if tc.setup != nil {
					tc.setup()
				}

				db.Upsert(fileRef, data)
				tc.assert(t, db)
			})
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

	testCases := []struct {
		name          string
		update        *FileRef
		expectChanged bool
	}{
		{
			name: "updates when MD5 changes",
			update: &FileRef{
				Filename:   fileRef.Filename,
				MD5:        "NEW MD5____",
				ModifiedAt: fileRef.ModifiedAt,
			},
			expectChanged: true,
		},
		{
			name: "updates when modified timestamp changes",
			update: &FileRef{
				Filename:   fileRef.Filename,
				MD5:        fileRef.MD5,
				ModifiedAt: time.Now().Add(5 * time.Second),
			},
			expectChanged: true,
		},
		{
			name:          "does not update with zero changes",
			update:        &fileRef,
			expectChanged: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withNewDB(func(db *DB) {
				// Insert initial record
				err := db.Insert(&fileRef, data)

				if err != nil {
					assert.FailNow(t, "Query failed", err)
				}

				newRef := tc.update

				if err := db.Upsert(newRef, []byte("fresher data")); err != nil {
					assert.FailNow(t, "upsert failed", err)
				}

				var currentData []string
				if err := db.Select(&currentData, "SELECT text from content_index"); err != nil {
					assert.FailNow(t, "index did not return results", err)
				}

				if tc.expectChanged {
					assert.Equal(t, "fresher data", currentData[0], "document data was not updated")
				} else {
					assert.Equal(t, "some data", currentData[0], "document data was updated")
				}

			})
		})
	}
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
