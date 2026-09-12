package nve

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var notes *Notes

func init() {
	notes = NewNotes(NotesConfig{
		Filepath: "./test_data",
	})
}

func TestSearch(t *testing.T) {
	/*
		Following tests rely on the fixture files within "./test_data"
	*/
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "handles empty input",
			input: "",
			expected: []string{
				"test_data/apples in zoo.md",
				"test_data/bananas_in_zoo.md",
				"test_data/cats.md",
				"test_data/nested/cucumbers.md",
				"test_data/zebra in zoo.md",
			},
		},
		{
			name:     "handles quote characters",
			input:    "\"",
			expected: []string{},
		},
		{
			name:     "locates no files",
			input:    "nothing-matches-this-string~~",
			expected: []string{},
		},
		{
			name:     "locates files by partial name match",
			input:    "apple",
			expected: []string{"test_data/apples in zoo.md"},
		},
		{
			name:     "locates files by fragment match",
			input:    "app zoo",
			expected: []string{"test_data/apples in zoo.md"},
		},
		{
			name:     "locates files by content match",
			input:    "new york",
			expected: []string{"test_data/apples in zoo.md"},
		},
		{
			name:     "locates files by partial content match",
			input:    "yor",
			expected: []string{"test_data/apples in zoo.md"},
		},
		{
			name:     "locates files by case-insensitive content match",
			input:    "YOR",
			expected: []string{"test_data/apples in zoo.md"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := notes.Search(tc.input)
			assert.NoError(t, err)

			// sort both arrays before assertion
			sort.Strings(tc.expected)
			sort.Strings(results)
			assert.Equal(t, tc.expected, results)
		})
	}
}

type mockObserver struct {
	lastResult []*SearchResult
}

func (m *mockObserver) SearchResultsUpdate(notes *Notes) {
	m.lastResult = notes.LastSearchResults
}

func TestNotifyObservers(t *testing.T) {
	mock := mockObserver{}
	notes.RegisterObservers(&mock)

	notes.Search("seattle")

	if assert.Len(t, mock.lastResult, 1) {
		res := mock.lastResult[0]

		// assert snippet
		assert.Contains(t, res.Snippet, "**seattle**")

		// assert filename
		assert.Equal(t, "test_data/apples in zoo.md", res.Filename)
	}
}

// newTempNotes creates a Notes over a temp directory seeded with files.
func newTempNotes(t *testing.T, files map[string]string) (*Notes, string) {
	t.Helper()
	dir := writeFixtures(t, files)
	return NewNotes(NotesConfig{Filepath: dir}), dir
}

func names(results []string) []string {
	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, filepath.Base(r))
	}
	sort.Strings(out)
	return out
}

func TestSearchNarrowing(t *testing.T) {
	fixtures := map[string]string{
		"apples.md":  "apple pie",
		"apricot.md": "apricot jam",
		"zoo.md":     "apples in the zoo",
	}

	testcases := []struct {
		name     string
		queries  []string                                 // run in order; the last one is asserted
		between  func(t *testing.T, n *Notes, dir string) // runs before the last query
		expected []string
	}{
		{
			name:     "extending the query narrows the previous results",
			queries:  []string{"ap", "app"},
			expected: []string{"apples.md", "zoo.md"},
		},
		{
			name:     "adding a term narrows the previous results",
			queries:  []string{"app", "app zoo"},
			expected: []string{"zoo.md"},
		},
		{
			name:    "an extended query does not rescan the directory",
			queries: []string{"ap", "app"},
			between: func(t *testing.T, n *Notes, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "applause.md"), []byte("applause"), 0644))
			},
			expected: []string{"apples.md", "zoo.md"},
		},
		{
			name:    "a shortened query rescans the directory",
			queries: []string{"app", "ap"},
			between: func(t *testing.T, n *Notes, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "applause.md"), []byte("applause"), 0644))
			},
			expected: []string{"applause.md", "apples.md", "apricot.md", "zoo.md"},
		},
		{
			name:    "narrowed results drop deleted files",
			queries: []string{"ap", "app"},
			between: func(t *testing.T, n *Notes, dir string) {
				require.NoError(t, os.Remove(filepath.Join(dir, "zoo.md")))
			},
			expected: []string{"apples.md"},
		},
		{
			name:    "narrowed results reflect edited content",
			queries: []string{"ap", "app"},
			between: func(t *testing.T, n *Notes, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "apricot.md"), []byte("apricot and apple"), 0644))
			},
			expected: []string{"apples.md", "apricot.md", "zoo.md"},
		},
		{
			name:     "a whitespace-only query is not narrowed from",
			queries:  []string{"   ", "   ap"},
			expected: []string{"apples.md", "apricot.md", "zoo.md"},
		},
		{
			name:    "creating a note invalidates the previous results",
			queries: []string{"banana", "banana"},
			between: func(t *testing.T, n *Notes, dir string) {
				_, err := n.CreateNote("banana")
				require.NoError(t, err)
			},
			expected: []string{"banana.md"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			n, dir := newTempNotes(t, fixtures)

			var results []string
			for i, q := range tc.queries {
				if i == len(tc.queries)-1 && tc.between != nil {
					tc.between(t, n, dir)
				}
				var err error
				results, err = n.Search(q)
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expected, names(results))
		})
	}
}

func TestSearchNarrowingRefreshesModifiedTime(t *testing.T) {
	n, dir := newTempNotes(t, map[string]string{"apples.md": "apple pie"})

	_, err := n.Search("ap")
	require.NoError(t, err)
	before := n.LastSearchResults[0].ModifiedAt

	later := before.Add(time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "apples.md"), later, later))

	_, err = n.Search("app")
	require.NoError(t, err)

	require.Len(t, n.LastSearchResults, 1)
	assert.True(t, n.LastSearchResults[0].ModifiedAt.Equal(later), "expected narrowed result to carry the new modification time")
}
