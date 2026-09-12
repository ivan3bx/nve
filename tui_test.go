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
		"keeper.md": "I stay",
		"goner.md":  "I go away",
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

func TestTUI_RenameNote(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha.md": "alpha content",
		"beta.md":  "beta content",
	})

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha")
	}, 5*time.Second)

	// Filter to the target note, then start a rename with Ctrl-R
	h.SendKeys("a", "l", "p", "h", "a")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha") && !strings.Contains(s, "beta")
	}, 3*time.Second)

	h.SendKeys("C-r")

	// Rename mode: the title changes and the input is pre-filled
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Rename")
	}, 3*time.Second)

	// Typing previews the new name in the list row. The list row is
	// distinguished from the search box by its single-line border (│ vs ║).
	h.SendKeys("s")
	h.WaitFor(func(s string) bool {
		for _, line := range strings.Split(s, "\n") {
			if strings.Contains(line, "│ alphas") {
				return true
			}
		}
		return false
	}, 3*time.Second)

	// Enter commits the rename
	h.SendKeys("Enter")

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alphas") && strings.Contains(s, "Search Box")
	}, 3*time.Second)

	if !h.FileExists("alphas.md") {
		t.Errorf("expected alphas.md on disk after rename")
	}
	if h.FileExists("alpha.md") {
		t.Errorf("expected alpha.md to be gone after rename")
	}
	if content := h.ReadFile("alphas.md"); content != "alpha content" {
		t.Errorf("expected renamed file to keep its content, got: %s", content)
	}
}

// seedRenameFixtures creates three notes with known modtime ordering
// (newest first: three, two, one).
func seedRenameFixtures(h *TUIHarness) {
	now := time.Now()
	h.SetModTime("alpha one.md", now.Add(-3*time.Hour))
	h.SetModTime("alpha two.md", now.Add(-2*time.Hour))
	h.SetModTime("alpha three.md", now.Add(-1*time.Hour))
}

// searchBoxLineHas reports whether the SearchBox line (double-line border ║)
// contains want and excludes unwanted.
func searchBoxLineHas(screen, want, unwanted string) bool {
	for _, line := range strings.Split(screen, "\n") {
		if strings.Contains(line, "║") && strings.Contains(line, want) {
			return unwanted == "" || !strings.Contains(line, unwanted)
		}
	}
	return false
}

func TestTUI_RenameFromListBox(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha one.md":   "first note",
		"alpha two.md":   "second note",
		"alpha three.md": "third note",
	})
	seedRenameFixtures(h)

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha")
	}, 5*time.Second)

	// Filter, then move into the list and select the second match
	h.SendKeys("a", "l", "p", "h", "a")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha two")
	}, 3*time.Second)
	h.SendKeys("Down")

	// Ctrl-R from the list: focus moves to the search box in rename mode,
	// pre-filled with the selected (second) note
	h.SendKeys("C-r")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Rename") && searchBoxLineHas(s, "alpha two", "")
	}, 3*time.Second)

	// Commit a rename of the second note
	h.SendKeys("x", "Enter")

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Search Box") && strings.Contains(s, "alpha twox")
	}, 3*time.Second)

	if !h.FileExists("alpha twox.md") {
		t.Errorf("expected alpha twox.md on disk after rename")
	}
	if h.FileExists("alpha two.md") {
		t.Errorf("expected alpha two.md to be gone after rename")
	}
	if !h.FileExists("alpha one.md") || !h.FileExists("alpha three.md") {
		t.Errorf("expected other notes to be untouched")
	}

	// Focus returned to the list: typing a character forwards to the search
	// box, replacing its text rather than appending
	h.SendKeys("q")
	h.WaitFor(func(s string) bool {
		return searchBoxLineHas(s, " q ", "alpha")
	}, 3*time.Second)
}

func TestTUI_RenameFromListBoxCancel(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha one.md":   "first note",
		"alpha two.md":   "second note",
		"alpha three.md": "third note",
	})
	seedRenameFixtures(h)

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha")
	}, 5*time.Second)

	// Filter, move into the list, start a rename of the second match
	h.SendKeys("a", "l", "p", "h", "a")
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha two")
	}, 3*time.Second)
	h.SendKeys("Down", "C-r")

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Rename")
	}, 3*time.Second)

	// Type a change, then cancel with Escape
	h.SendKeys("x", "Escape")

	// Rename mode exits with the original name restored
	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Search Box") && searchBoxLineHas(s, "alpha two", "")
	}, 3*time.Second)

	if !h.FileExists("alpha two.md") {
		t.Errorf("expected alpha two.md to be untouched after cancel")
	}
	if h.FileExists("alpha twox.md") {
		t.Errorf("expected no alpha twox.md after cancel")
	}

	// Focus returned to the list: typing a character forwards to the search
	// box, replacing its text rather than appending
	h.SendKeys("q")
	h.WaitFor(func(s string) bool {
		return searchBoxLineHas(s, " q ", "alpha")
	}, 3*time.Second)
}

func TestTUI_RenameCancel(t *testing.T) {
	h := NewTUIHarness(t, map[string]string{
		"alpha.md": "alpha content",
	})

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "alpha")
	}, 5*time.Second)

	h.SendKeys("a", "l", "p", "h", "a", "C-r")

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Rename")
	}, 3*time.Second)

	// Type a change, then cancel with Escape
	h.SendKeys("x", "Escape")

	h.WaitFor(func(s string) bool {
		return strings.Contains(s, "Search Box") && !strings.Contains(s, "alphax")
	}, 3*time.Second)

	if !h.FileExists("alpha.md") {
		t.Errorf("expected alpha.md to be untouched after cancel")
	}
	if h.FileExists("alphax.md") {
		t.Errorf("expected no alphax.md after cancel")
	}
}
