package nve

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Disable logging during tests
	log.SetOutput(io.Discard)
}

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		snippet    string
		maxWidth   int
		modifiedAt string
		expected   string
	}{
		{
			name:       "simple test",
			filename:   "test.txt",
			snippet:    "This is a test snippet",
			maxWidth:   -1,
			modifiedAt: "2023-10-01T15:04:05Z",
			expected:   "test                     This is a test snippet",
		},
		{
			name:       "width includes date",
			filename:   "test_file.txt",
			snippet:    "This is a test snippet",
			maxWidth:   70,
			modifiedAt: "2006-01-02T15:04:05Z",
			expected:   "test_file                This is a test snippet           Jan 02, 2006",
		},
		{
			name:     "no width specified",
			filename: "test_file.txt",
			snippet:  "package nve\n\nimport ( \"fmt\"\n\"log\"\n\"math\"\n\"strings\"\n\"time\"\n\"github.com/gdamore/tcell/v2\"",
			expected: `test_file                package nve import ( "fmt" "log" "math" "strings" "time" "github.com/gdamore/tcell/v2"         Jan 01, 2001`,
		},
		{
			name:     "width truncates",
			filename: "test_file.txt",
			snippet:  "package nve\n\nimport ( \"fmt\"\n\"log\"\n\"math\"\n\"strings\"\n\"time\"\n\"github.com/gdamore/tcell/v2\"",
			expected: `test_file                package nve import ( "fmt" "log" "math" "strings" "time" "github.com/gda..         Jan 01, 2001`,
			maxWidth: 120,
		},
		{
			name:     "width is narrow",
			filename: "test_file.txt",
			snippet:  "package nve\n\nimport ( \"fmt\"\n\"log\"\n\"math\"\n\"strings\"\n\"time\"\n\"github.com/gdamore/tcell/v2\"",
			expected: `test_file                package nve import ( "fmt" "log" "math" "strings" "time" "github.com/gda..         Jan 01, 2001`,
			maxWidth: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.modifiedAt == "" {
				tt.modifiedAt = "2001-01-01T15:04:05Z"
			}

			modifiedAt, _ := time.Parse(time.RFC3339, tt.modifiedAt)

			fileRef := &FileRef{
				DocumentID: 1,
				Filename:   tt.filename,
				MD5:        "abc123",
				ModifiedAt: modifiedAt,
			}

			result := &SearchResult{
				FileRef: fileRef,
				Snippet: tt.snippet,
			}

			actual := formatResult(result, tt.maxWidth)

			if len(tt.snippet) > tt.maxWidth && tt.maxWidth > 0 {
				// Ensure that the snippet was truncated appropriately
				assert.Equal(t, len(actual), tt.maxWidth)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}
