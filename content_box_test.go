package nve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func tempFileRef(t *testing.T, name, content string) *FileRef {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return &FileRef{Filename: path}
}

func TestSetFile_Versioning(t *testing.T) {
	first := tempFileRef(t, "one.md", "hello")
	second := tempFileRef(t, "two.md", "world")

	testcases := []struct {
		name        string
		versioning  bool
		expectEmpty bool
	}{
		{
			name:        "enabled snapshots outgoing file",
			versioning:  true,
			expectEmpty: false,
		},
		{
			name:        "disabled does not snapshot",
			versioning:  false,
			expectEmpty: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cb := NewContentBox()
			cb.versioning = tc.versioning

			var captured []string
			cb.saveVersion = func(path string) { captured = append(captured, path) }

			cb.SetFile(first)
			assert.Empty(t, captured, "no snapshot on first file open")

			captured = nil
			cb.SetFile(second)

			if tc.expectEmpty {
				assert.Empty(t, captured)
			} else {
				assert.Equal(t, []string{first.Filename}, captured)
			}
		})
	}
}

func TestSetFile_SameFileDoesNotSnapshot(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = true

	var captured []string
	cb.saveVersion = func(path string) { captured = append(captured, path) }

	cb.SetFile(f)
	cb.SetFile(f)
	cb.SetFile(f)

	assert.Empty(t, captured, "should not snapshot when re-selecting the same file")
}

func TestClear_VersioningEnabled_SnapshotsOutgoingFile(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = true

	var captured []string
	cb.saveVersion = func(path string) { captured = append(captured, path) }

	cb.SetFile(f)
	captured = nil

	cb.Clear()
	assert.Equal(t, []string{f.Filename}, captured, "should snapshot outgoing file on Clear")
	assert.Nil(t, cb.currentFile, "currentFile should be nil after Clear")
}

func TestClear_VersioningDisabled_DoesNotSnapshot(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = false

	called := false
	cb.saveVersion = func(path string) { called = true }

	cb.SetFile(f)
	cb.Clear()
	assert.False(t, called, "should not snapshot on Clear when versioning is disabled")
}

func TestShutdown_Versioning(t *testing.T) {
	testcases := []struct {
		name        string
		versioning  bool
		hasFile     bool
		expectEmpty bool
	}{
		{
			name:        "enabled with file snapshots on shutdown",
			versioning:  true,
			hasFile:     true,
			expectEmpty: false,
		},
		{
			name:        "disabled does not snapshot",
			versioning:  false,
			hasFile:     true,
			expectEmpty: true,
		},
		{
			name:        "enabled without file does not snapshot",
			versioning:  true,
			hasFile:     false,
			expectEmpty: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cb := NewContentBox()
			cb.versioning = tc.versioning

			var captured []string
			cb.saveVersion = func(path string) { captured = append(captured, path) }

			if tc.hasFile {
				cb.SetFile(tempFileRef(t, "note.md", "content"))
			}

			captured = nil
			cb.Shutdown()

			if tc.expectEmpty {
				assert.Empty(t, captured)
			} else {
				assert.NotEmpty(t, captured)
			}
		})
	}
}
