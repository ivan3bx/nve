package nve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var notes *Notes

func init() {
	notes = NewNotes(NotesConfig{
		Filepath: "./test_data",
		DBPath:   "./nve_test.db", // generateTempDBPath(),
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
			name:     "locates files by partial name match",
			input:    "apple",
			expected: []string{"test_data/apples in zoo.md"},
		},
		{
			name:     "locates files by fragment match",
			input:    "app zoo",
			expected: []string{"test_data/apples in zoo.md"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := notes.Search(tc.input)
			assert.Equal(t, tc.expected, results)
		})
	}
}
