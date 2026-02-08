//go:build integration

package nve

import (
	"strings"
	"testing"
	"time"
)

func TestTUI_AppStartsWithFiles(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha.md": "Alpha content here",
		"beta.md":  "Beta content here",
	})

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha") && strings.Contains(s, "beta")
	}, 5*time.Second)
}

func TestTUI_EditAndSave(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"notes.md": "original content",
	})

	// Wait for file to appear in list
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "notes")
	}, 5*time.Second)

	// Navigate: Down arrow moves to ListBox with selection, Enter opens in ContentBox
	h.SendKeys("Down", "Enter")

	// Wait for ContentBox to show the file content
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "original content")
	}, 3*time.Second)

	// Type some text — cursor starts at beginning, so just type directly
	h.SendKeys("h", "i")

	// Wait for debounced save (300ms save + buffer)
	time.Sleep(1 * time.Second)

	// Verify text persisted to disk
	content := h.ReadFile("notes.md")
	if !strings.Contains(content, "hi") {
		t.Errorf("expected 'hi' in saved file, got: %s", content)
	}

	// Escape back to SearchBox, then re-open the file
	h.SendKeys("Escape")
	time.Sleep(500 * time.Millisecond)
	h.SendKeys("Down", "Enter")

	// Verify content still has our edit
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "hi")
	}, 3*time.Second)
}

func TestTUI_ExternalFileCreate(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"existing.md": "already here",
	})

	// Wait for initial file
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "existing")
	}, 5*time.Second)

	// Create a new file externally
	h.WriteFile("newfile.md", "externally created content")

	// Wait for it to appear in the list (watcher debounce is 500ms)
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "newfile")
	}, 5*time.Second)
}

func TestTUI_ExternalFileDelete(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"keeper.md":  "I stay",
		"goner.md":   "I go away",
	})

	// Wait for both files
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "keeper") && strings.Contains(s, "goner")
	}, 5*time.Second)

	// Delete the file externally
	h.RemoveFile("goner.md")

	// Wait for it to disappear from the list
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "keeper") && !strings.Contains(s, "goner")
	}, 5*time.Second)
}

func TestTUI_ExternalEditWhileViewing(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"watched.md": "version one",
	})

	// Wait for file in list
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "watched")
	}, 5*time.Second)

	// Open the file in ContentBox
	h.SendKeys("Down", "Enter")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "version one")
	}, 3*time.Second)

	// Externally overwrite the file
	h.WriteFile("watched.md", "version two")

	// Escape to SearchBox (triggers flushRefresh on blur), then re-open
	h.SendKeys("Escape")
	time.Sleep(1 * time.Second) // wait for watcher debounce + refresh
	h.SendKeys("Down", "Enter")

	// Verify new content is shown
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "version two")
	}, 5*time.Second)
}

func TestTUI_SelfEditNoContentClearing(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"stable.md": "initial text",
	})

	// Wait for file in list
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "stable")
	}, 5*time.Second)

	// Open the file: Down arrow selects in ListBox, Enter opens ContentBox
	h.SendKeys("Down", "Enter")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "initial text")
	}, 3*time.Second)

	// Type additional text (cursor starts at beginning)
	h.SendKeys("x", "y", "z")

	// Wait for save to complete (debounce 300ms + watcher 500ms + buffer)
	time.Sleep(2 * time.Second)

	// Verify ContentBox still shows both the original and new text
	screen := h.Capture()
	if !strings.Contains(screen, "initial text") {
		t.Errorf("expected 'initial text' still visible, got:\n%s", screen)
	}
	if !strings.Contains(screen, "xyz") {
		t.Errorf("expected 'xyz' still visible, got:\n%s", screen)
	}
}
