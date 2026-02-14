//go:build integration

package nve

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
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

func TestTUI_CreateNewNote(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"existing.md": "some content",
	})

	// Wait for app to be ready with existing file
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "existing")
	}, 5*time.Second)

	// Type a name that doesn't match any file, then hit Enter
	h.SendKeys("m", "y", "n", "e", "w", "n", "o", "t", "e", "Enter")

	// Should land in ContentBox with a new empty file
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "mynewnote")
	}, 5*time.Second)

	// Type some content
	h.SendKeys("h", "e", "l", "l", "o")

	// Wait for save
	time.Sleep(1 * time.Second)

	// Verify the file was created on disk
	content := h.ReadFile("mynewnote.md")
	if !strings.Contains(content, "hello") {
		t.Errorf("expected 'hello' in new file, got: %s", content)
	}
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

func TestTUI_SearchHighlightsContent(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha.md": "The food fight was fantastic",
	})

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha")
	}, 5*time.Second)

	// contentLine returns the line from the Content pane (single-line border │)
	// that contains the file text, skipping the ListBox snippet (contains "alpha")
	// and the SearchBox (double-line border ║).
	contentLine := func(screen string) string {
		for _, line := range strings.Split(screen, "\n") {
			hasContent := strings.Contains(line, "food") || strings.Contains(line, "fight")
			isListRow := strings.Contains(line, "alpha")
			isSearchBox := strings.Contains(line, "║")
			if hasContent && !isListRow && !isSearchBox {
				return line
			}
		}
		return ""
	}

	// Derive the ANSI SGR background escape from the HighlightBackground constant.
	// tcell's first 16 named colors (ColorBlack..ColorWhite) map to SGR codes:
	//   indices 0-7  → bg 40-47
	//   indices 8-15 → bg 100-107
	colorIndex := int(HighlightBackground - tcell.ColorBlack)
	var sgrBg int
	if colorIndex < 8 {
		sgrBg = 40 + colorIndex
	} else {
		sgrBg = 100 + (colorIndex - 8)
	}
	highlightBgEsc := fmt.Sprintf("\x1b[%dm", sgrBg)

	// hasHighlight reports whether word appears with the highlight background SGR code.
	hasHighlight := func(line, word string) bool {
		pat := regexp.QuoteMeta(highlightBgEsc) + `[^\x1b]*` + regexp.QuoteMeta(word)
		return regexp.MustCompile(pat).MatchString(line)
	}

	// Search for "foo" — expect "foo" highlighted yellow in the content pane
	h.SendKeys("f", "o", "o")

	screen := h.WaitForWithColors(func(s string) bool {
		line := contentLine(s)
		return line != "" && hasHighlight(line, "foo")
	}, 5*time.Second)

	// Non-matched words must not be yellow
	line := contentLine(screen)
	if hasHighlight(line, "The") {
		t.Errorf("non-matched word 'The' should not be yellow, got:\n%s", line)
	}

	// Narrow search to "food" — expect "food" highlighted
	h.SendKeys("d")

	h.WaitForWithColors(func(s string) bool {
		line := contentLine(s)
		return line != "" && hasHighlight(line, "food")
	}, 3*time.Second)

	// Clear search — expect no yellow highlighting in the content pane.
	// With an empty query the content pane may be empty, which also satisfies this.
	h.SendKeys("BSpace", "BSpace", "BSpace", "BSpace")

	h.WaitForWithColors(func(s string) bool {
		line := contentLine(s)
		return line == "" || !strings.Contains(line, highlightBgEsc)
	}, 3*time.Second)
}
