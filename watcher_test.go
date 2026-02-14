package nve

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWatcherTest(t *testing.T) (*Notes, string) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	n := NewNotes(NotesConfig{
		Filepath: dir,
		DBPath:   dbPath,
	})

	return n, dir
}

// startWatching starts the watcher and returns a channel that receives
// a value each time the watcher triggers a UI refresh.
func startWatching(t *testing.T, n *Notes) chan struct{} {
	t.Helper()
	refreshed := make(chan struct{}, 1)
	err := n.StartWatching(func(f func()) {
		f()
		select {
		case refreshed <- struct{}{}:
		default:
		}
	})
	require.NoError(t, err)
	t.Cleanup(n.StopWatching)
	return refreshed
}

func TestWatcher_CreateFile(t *testing.T) {
	n, dir := setupWatcherTest(t)
	refreshed := startWatching(t, n)

	// Create a new .md file
	testFile := filepath.Join(dir, "new_note.md")
	require.NoError(t, os.WriteFile(testFile, []byte("hello watcher"), 0644))

	// Wait for the debounced refresh
	select {
	case <-refreshed:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watcher refresh")
	}

	// Verify the file is in the DB
	results, err := n.db.Search("hello watcher")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, testFile, results[0].Filename)
}

func TestWatcher_DeleteFile(t *testing.T) {
	n, dir := setupWatcherTest(t)

	// Create a file first and refresh to index it
	testFile := filepath.Join(dir, "to_delete.md")
	require.NoError(t, os.WriteFile(testFile, []byte("delete me"), 0644))
	_, err := n.Refresh()
	require.NoError(t, err)

	// Verify it's indexed
	results, err := n.db.Search("delete me")
	require.NoError(t, err)
	require.Len(t, results, 1)

	refreshed := startWatching(t, n)

	// Delete the file
	require.NoError(t, os.Remove(testFile))

	// Wait for the debounced refresh
	select {
	case <-refreshed:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for watcher refresh")
	}

	// Verify it's been pruned from the DB
	results, err = n.db.Search("delete me")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestWatcher_IgnoresUnsupportedTypes(t *testing.T) {
	n, dir := setupWatcherTest(t)
	refreshed := startWatching(t, n)

	// Create an unsupported file type
	pngFile := filepath.Join(dir, "image.png")
	require.NoError(t, os.WriteFile(pngFile, []byte("not a real png"), 0644))

	// The watcher should filter this out; wait briefly to confirm no refresh
	select {
	case <-refreshed:
		t.Fatal("watcher should not have triggered refresh for .png file")
	case <-time.After(1 * time.Second):
		// Expected: no refresh triggered
	}
}
