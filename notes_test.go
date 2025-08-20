package nve

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

var notes *Notes

func init() {
	notes = NewNotes(NotesConfig{
		Filepath: "./test_data",
		DBPath:   generateTempDBPath(),
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
		assert.Equal(t, "new york **seattle**", res.Snippet)

		// assert filename
		assert.Equal(t, "test_data/apples in zoo.md", res.Filename)
	}
}
