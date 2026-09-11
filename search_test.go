package nve

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFixtures creates files (name -> content) under a temp dir and returns the dir.
func writeFixtures(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	return dir
}

// relativeNames returns result filenames relative to dir, sorted.
func relativeNames(t *testing.T, dir string, results []*SearchResult) []string {
	t.Helper()
	names := make([]string, 0, len(results))

	for _, r := range results {
		rel, err := filepath.Rel(dir, r.Filename)
		require.NoError(t, err)
		names = append(names, rel)
	}

	sort.Strings(names)
	return names
}

func TestSearchFiles(t *testing.T) {
	dir := writeFixtures(t, map[string]string{
		"foobar.md":      "nothing relevant here",
		"notes.md":       "foo on line one\nsomething else\nbar on line three",
		"mixed.md":       "FOOD and BARN",
		"plain.md":       "no terms at all",
		"nested/deep.md": "foo bar together",
		"image.png":      "foo bar",
	})

	testcases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "matches every term in any order",
			query:    "bar foo",
			expected: []string{"foobar.md", "mixed.md", "nested/deep.md", "notes.md"},
		},
		{
			name:     "matches terms on different lines",
			query:    "line three",
			expected: []string{"notes.md"},
		},
		{
			name:     "is case-insensitive",
			query:    "food barn",
			expected: []string{"mixed.md"},
		},
		{
			name:     "matches inside words",
			query:    "ooba",
			expected: []string{"foobar.md"},
		},
		{
			name:     "matches display name",
			query:    "foobar",
			expected: []string{"foobar.md"},
		},
		{
			name:     "requires every term to match",
			query:    "foo missing",
			expected: []string{},
		},
		{
			name:     "ignores unsupported file types",
			query:    "png",
			expected: []string{},
		},
		{
			name:     "empty query matches nothing",
			query:    "",
			expected: []string{},
		},
		{
			name:     "whitespace-only query matches nothing",
			query:    "   ",
			expected: []string{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := searchFiles(dir, tc.query)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, relativeNames(t, dir, results))
		})
	}
}

func TestSearchFilesOrder(t *testing.T) {
	dir := writeFixtures(t, map[string]string{
		"old.md":    "needle",
		"newest.md": "needle",
		"middle.md": "needle",
	})

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, age := range map[string]time.Duration{"old.md": 2 * time.Hour, "middle.md": time.Hour, "newest.md": 0} {
		mtime := base.Add(-age)
		require.NoError(t, os.Chtimes(filepath.Join(dir, name), mtime, mtime))
	}

	results, err := searchFiles(dir, "needle")
	require.NoError(t, err)

	names := []string{}
	for _, r := range results {
		names = append(names, filepath.Base(r.Filename))
	}

	assert.Equal(t, []string{"newest.md", "middle.md", "old.md"}, names)
}

func TestRecentFiles(t *testing.T) {
	dir := writeFixtures(t, map[string]string{
		"a.md": "first line\nsecond line",
		"b.md": "b content",
		"c.md": "c content",
	})

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"c.md", "b.md", "a.md"} {
		mtime := base.Add(time.Duration(i) * time.Hour)
		require.NoError(t, os.Chtimes(filepath.Join(dir, name), mtime, mtime))
	}

	results, err := recentFiles(dir, 2)
	require.NoError(t, err)

	require.Len(t, results, 2)
	assert.Equal(t, "a.md", filepath.Base(results[0].Filename))
	assert.Equal(t, "first line second line", results[0].Snippet)
	assert.Equal(t, "b.md", filepath.Base(results[1].Filename))
}

func TestMatchSnippet(t *testing.T) {
	testcases := []struct {
		name     string
		content  string
		query    string
		expected string
	}{
		{
			name:     "bolds the match",
			content:  "alpha beta gamma",
			query:    "gamma",
			expected: "alpha beta **gamma**",
		},
		{
			name:     "bolds every occurrence of every term",
			content:  "foo and bar and foo",
			query:    "foo bar",
			expected: "**foo** and **bar** and **foo**",
		},
		{
			name:     "merges overlapping matches",
			content:  "foobar",
			query:    "foo oba",
			expected: "**fooba**r",
		},
		{
			name:     "preserves original case",
			content:  "Hello WORLD",
			query:    "world",
			expected: "Hello **WORLD**",
		},
		{
			name:     "collapses newlines",
			content:  "one\nneedle\ntwo",
			query:    "needle",
			expected: "one **needle** two",
		},
		{
			name:     "leads with context and ellipsis when match is deep in the file",
			content:  strings.Repeat("filler ", 20) + "needle",
			query:    "needle",
			expected: "...filler filler filler filler filler **needle**",
		},
		{
			name:     "trails with ellipsis when content exceeds the window",
			content:  "needle " + strings.Repeat("x", 200),
			query:    "needle",
			expected: "**needle** " + strings.Repeat("x", snippetLength-len("needle ")) + "...",
		},
		{
			name:     "falls back to leading content when only the name matched",
			content:  "just some text",
			query:    "zzz",
			expected: "just some text",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.content)
			lower := []byte(strings.ToLower(tc.content))
			assert.Equal(t, tc.expected, matchSnippet(content, lower, searchTerms(tc.query)))
		})
	}
}

func TestLeadSnippet(t *testing.T) {
	testcases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "returns short content as-is",
			content:  "short",
			expected: "short",
		},
		{
			name:     "collapses newlines",
			content:  "a\nb\nc",
			expected: "a b c",
		},
		{
			name:     "truncates long content",
			content:  strings.Repeat("x", 300),
			expected: strings.Repeat("x", snippetLength),
		},
		{
			name:     "does not split a multi-byte rune",
			content:  strings.Repeat("x", snippetLength-1) + "éé",
			expected: strings.Repeat("x", snippetLength-1) + "é",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, leadSnippet([]byte(tc.content)))
		})
	}
}
