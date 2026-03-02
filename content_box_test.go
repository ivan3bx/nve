package nve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
			name:        "enabled snapshots outgoing file when dirty",
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
			cb.dirty = true // simulate user edit
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

func TestSetFile_NotDirty_DoesNotSnapshot(t *testing.T) {
	first := tempFileRef(t, "one.md", "hello")
	second := tempFileRef(t, "two.md", "world")

	cb := NewContentBox()
	cb.versioning = true

	var captured []string
	cb.saveVersion = func(path string) { captured = append(captured, path) }

	cb.SetFile(first)
	// dirty is false — no user edits
	cb.SetFile(second)

	assert.Empty(t, captured, "should not snapshot when file was not edited")
}

func TestSetFile_SameFileDoesNotSnapshot(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = true

	var captured []string
	cb.saveVersion = func(path string) { captured = append(captured, path) }

	cb.SetFile(f)
	cb.dirty = true
	cb.SetFile(f)
	cb.SetFile(f)

	assert.Empty(t, captured, "should not snapshot when re-selecting the same file")
}

func TestClear_DirtyFile_SnapshotsOutgoingFile(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = true

	var captured []string
	cb.saveVersion = func(path string) { captured = append(captured, path) }

	cb.SetFile(f)
	cb.dirty = true // simulate user edit
	captured = nil

	cb.Clear()
	assert.Equal(t, []string{f.Filename}, captured, "should snapshot outgoing file on Clear")
	assert.Nil(t, cb.currentFile, "currentFile should be nil after Clear")
}

func TestClear_NotDirty_DoesNotSnapshot(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = true

	called := false
	cb.saveVersion = func(path string) { called = true }

	cb.SetFile(f)
	// dirty is false — no user edits
	cb.Clear()
	assert.False(t, called, "should not snapshot on Clear when file was not edited")
}

func TestClear_VersioningDisabled_DoesNotSnapshot(t *testing.T) {
	f := tempFileRef(t, "note.md", "hello")

	cb := NewContentBox()
	cb.versioning = false

	called := false
	cb.saveVersion = func(path string) { called = true }

	cb.SetFile(f)
	cb.dirty = true
	cb.Clear()
	assert.False(t, called, "should not snapshot on Clear when versioning is disabled")
}

func TestHandleListContinuation(t *testing.T) {
	testcases := []struct {
		name       string
		filename   string
		text       string
		cursorPos  int
		expectText string
		expectUsed bool
	}{
		{
			name:       "dash bullet continues on next line",
			filename:   "note.md",
			text:       "- hello",
			cursorPos:  7,
			expectText: "- hello\n- ",
			expectUsed: true,
		},
		{
			name:       "star bullet continues on next line",
			filename:   "note.md",
			text:       "* world",
			cursorPos:  7,
			expectText: "* world\n* ",
			expectUsed: true,
		},
		{
			name:       "indented dash preserves indent",
			filename:   "note.md",
			text:       "  - nested",
			cursorPos:  10,
			expectText: "  - nested\n  - ",
			expectUsed: true,
		},
		{
			name:       "numeric ordered increments",
			filename:   "note.md",
			text:       "1. first",
			cursorPos:  8,
			expectText: "1. first\n2. ",
			expectUsed: true,
		},
		{
			name:       "alpha ordered increments",
			filename:   "note.md",
			text:       "a. alpha",
			cursorPos:  8,
			expectText: "a. alpha\nb. ",
			expectUsed: true,
		},
		{
			name:       "empty dash bullet clears to newline",
			filename:   "note.md",
			text:       "- ",
			cursorPos:  2,
			expectText: "\n",
			expectUsed: true,
		},
		{
			name:       "empty star bullet clears to newline",
			filename:   "note.md",
			text:       "* ",
			cursorPos:  2,
			expectText: "\n",
			expectUsed: true,
		},
		{
			name:       "empty numeric bullet clears to newline",
			filename:   "note.md",
			text:       "1. ",
			cursorPos:  3,
			expectText: "\n",
			expectUsed: true,
		},
		{
			name:       "plain text not consumed",
			filename:   "note.md",
			text:       "just text",
			cursorPos:  9,
			expectText: "just text",
			expectUsed: false,
		},
		{
			name:       "cursor right after marker does not clear bullet",
			filename:   "note.md",
			text:       "- item",
			cursorPos:  2,
			expectText: "- \n- item",
			expectUsed: true,
		},
		{
			name:       "cursor mid-line after bullet with trailing text",
			filename:   "note.md",
			text:       "- hello world",
			cursorPos:  7,
			expectText: "- hello\n-  world",
			expectUsed: true,
		},
		{
			name:       "multiline continues second line bullet",
			filename:   "note.md",
			text:       "- first\n- second",
			cursorPos:  16,
			expectText: "- first\n- second\n- ",
			expectUsed: true,
		},
		{
			name:       "multiline empty second bullet clears it",
			filename:   "note.md",
			text:       "- first\n- ",
			cursorPos:  10,
			expectText: "- first\n\n",
			expectUsed: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cb := NewContentBox()
			cb.saveVersion = func(string) {}
			cb.currentFile = &FileRef{Filename: tc.filename}
			cb.SetText(tc.text, false)

			// Position cursor using Replace with empty text.
			cb.Replace(tc.cursorPos, tc.cursorPos, "")

			used := cb.handleListContinuation()
			assert.Equal(t, tc.expectUsed, used, "consumed mismatch")
			assert.Equal(t, tc.expectText, cb.GetText(), "text mismatch")
		})
	}
}

func TestListContinuation_NonMarkdownSkipped(t *testing.T) {
	cb := NewContentBox()
	cb.saveVersion = func(string) {}
	cb.currentFile = &FileRef{Filename: "note.txt"}
	cb.SetText("- hello", false)
	cb.Replace(7, 7, "")

	handler := cb.InputHandler()
	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	handler(enter, func(tview.Primitive) {})

	// TextArea's default Enter inserts a newline, but no bullet prefix.
	assert.Equal(t, "- hello\n", cb.GetText(), "non-markdown should get plain newline")
}

func TestListIndent(t *testing.T) {
	testcases := []struct {
		name            string
		text            string
		cursorPos       int
		expectText      string
		expectUsed      bool
		expectCursorPos int
	}{
		{
			name:            "tab indents dash bullet",
			text:            "- item",
			cursorPos:       6,
			expectText:      "\t- item",
			expectUsed:      true,
			expectCursorPos: 7,
		},
		{
			name:            "tab indents star bullet",
			text:            "* item",
			cursorPos:       6,
			expectText:      "\t* item",
			expectUsed:      true,
			expectCursorPos: 7,
		},
		{
			name:            "tab indents numeric bullet",
			text:            "1. item",
			cursorPos:       7,
			expectText:      "\t1. item",
			expectUsed:      true,
			expectCursorPos: 8,
		},
		{
			name:            "tab stacks on existing tab",
			text:            "\t- item",
			cursorPos:       7,
			expectText:      "\t\t- item",
			expectUsed:      true,
			expectCursorPos: 8,
		},
		{
			name:       "tab on non-bullet line is not consumed",
			text:       "plain text",
			cursorPos:  10,
			expectText: "plain text",
			expectUsed: false,
		},
		{
			name:            "tab indents second line only",
			text:            "- first\n- second",
			cursorPos:       16,
			expectText:      "- first\n\t- second",
			expectUsed:      true,
			expectCursorPos: 17,
		},
		{
			name:            "tab on empty bullet line",
			text:            "- ",
			cursorPos:       2,
			expectText:      "\t- ",
			expectUsed:      true,
			expectCursorPos: 3,
		},
		{
			name:            "cursor mid-line preserves relative position",
			text:            "- hello world",
			cursorPos:       7,
			expectText:      "\t- hello world",
			expectUsed:      true,
			expectCursorPos: 8,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cb := NewContentBox()
			cb.saveVersion = func(string) {}
			cb.currentFile = &FileRef{Filename: "note.md"}
			cb.SetText(tc.text, false)
			cb.Replace(tc.cursorPos, tc.cursorPos, "")

			used := cb.handleListIndent()
			assert.Equal(t, tc.expectUsed, used, "consumed mismatch")
			assert.Equal(t, tc.expectText, cb.GetText(), "text mismatch")
			if tc.expectUsed {
				_, pos, _ := cb.GetSelection()
				assert.Equal(t, tc.expectCursorPos, pos, "cursor position mismatch")
			}
		})
	}
}

func TestListDedent(t *testing.T) {
	testcases := []struct {
		name            string
		text            string
		cursorPos       int
		expectText      string
		expectUsed      bool
		expectDirty     bool
		expectCursorPos int
	}{
		{
			name:            "shift-tab removes leading tab",
			text:            "\t- item",
			cursorPos:       7,
			expectText:      "- item",
			expectUsed:      true,
			expectDirty:     true,
			expectCursorPos: 6,
		},
		{
			name:            "shift-tab removes one tab when multiple",
			text:            "\t\t- item",
			cursorPos:       8,
			expectText:      "\t- item",
			expectUsed:      true,
			expectDirty:     true,
			expectCursorPos: 7,
		},
		{
			name:        "shift-tab on unindented bullet is consumed but no change",
			text:        "- item",
			cursorPos:   6,
			expectText:  "- item",
			expectUsed:  true,
			expectDirty: false,
		},
		{
			name:        "shift-tab on non-bullet line not consumed",
			text:        "\tplain text",
			cursorPos:   11,
			expectText:  "\tplain text",
			expectUsed:  false,
			expectDirty: false,
		},
		{
			name:            "shift-tab dedents second line only",
			text:            "- first\n\t- second",
			cursorPos:       17,
			expectText:      "- first\n- second",
			expectUsed:      true,
			expectDirty:     true,
			expectCursorPos: 16,
		},
		{
			name:            "shift-tab with cursor at line start clamps position",
			text:            "\t- item",
			cursorPos:       0,
			expectText:      "- item",
			expectUsed:      true,
			expectDirty:     true,
			expectCursorPos: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cb := NewContentBox()
			cb.saveVersion = func(string) {}
			cb.currentFile = &FileRef{Filename: "note.md"}
			cb.SetText(tc.text, false)
			cb.Replace(tc.cursorPos, tc.cursorPos, "")
			cb.dirty = false

			used := cb.handleListDedent()
			assert.Equal(t, tc.expectUsed, used, "consumed mismatch")
			assert.Equal(t, tc.expectText, cb.GetText(), "text mismatch")
			assert.Equal(t, tc.expectDirty, cb.dirty, "dirty mismatch")
			if tc.expectDirty {
				_, pos, _ := cb.GetSelection()
				assert.Equal(t, tc.expectCursorPos, pos, "cursor position mismatch")
			}
		})
	}
}

func TestListIndent_ViaInputHandler(t *testing.T) {
	cb := NewContentBox()
	cb.saveVersion = func(string) {}
	cb.currentFile = &FileRef{Filename: "note.md"}
	cb.SetText("- item", false)
	cb.Replace(6, 6, "")

	handler := cb.InputHandler()
	tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	handler(tab, func(tview.Primitive) {})

	assert.Equal(t, "\t- item", cb.GetText(), "Tab via InputHandler should indent bullet line")
}

func TestListDedent_ViaInputHandler(t *testing.T) {
	cb := NewContentBox()
	cb.saveVersion = func(string) {}
	cb.currentFile = &FileRef{Filename: "note.md"}
	cb.SetText("\t- item", false)
	cb.Replace(7, 7, "")

	handler := cb.InputHandler()
	backtab := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
	handler(backtab, func(tview.Primitive) {})

	assert.Equal(t, "- item", cb.GetText(), "Shift-Tab via InputHandler should dedent bullet line")
}

func TestListIndent_NonMarkdownSkipped(t *testing.T) {
	cb := NewContentBox()
	cb.saveVersion = func(string) {}
	cb.currentFile = &FileRef{Filename: "note.txt"}
	cb.SetText("- item", false)
	cb.Replace(6, 6, "")

	handler := cb.InputHandler()
	tab := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	handler(tab, func(tview.Primitive) {})

	// TextArea's default Tab inserts a tab at cursor, not at line start.
	assert.Equal(t, "- item\t", cb.GetText(), "non-markdown should get default tab behavior")
}

func TestListContinuation_SetsDirtyFlag(t *testing.T) {
	cb := NewContentBox()
	cb.saveVersion = func(string) {}
	cb.currentFile = &FileRef{Filename: "note.md"}
	cb.SetText("- item", false)
	cb.Replace(6, 6, "")
	cb.dirty = false

	cb.handleListContinuation()

	assert.True(t, cb.dirty, "dirty flag should be set after list continuation")
}

func TestShutdown_Versioning(t *testing.T) {
	testcases := []struct {
		name        string
		versioning  bool
		hasFile     bool
		dirty       bool
		expectEmpty bool
	}{
		{
			name:        "enabled with dirty file snapshots on shutdown",
			versioning:  true,
			hasFile:     true,
			dirty:       true,
			expectEmpty: false,
		},
		{
			name:        "enabled with clean file does not snapshot",
			versioning:  true,
			hasFile:     true,
			dirty:       false,
			expectEmpty: true,
		},
		{
			name:        "disabled does not snapshot",
			versioning:  false,
			hasFile:     true,
			dirty:       true,
			expectEmpty: true,
		},
		{
			name:        "enabled without file does not snapshot",
			versioning:  true,
			hasFile:     false,
			dirty:       false,
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
				cb.dirty = tc.dirty
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
