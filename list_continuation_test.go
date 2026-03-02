package nve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBulletPrefix(t *testing.T) {
	testcases := []struct {
		name         string
		line         string
		expectOK     bool
		expectIndent string
		expectMarker string
		expectRest   string
	}{
		{
			name:         "dash bullet",
			line:         "- hello",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "-",
			expectRest:   "hello",
		},
		{
			name:         "star bullet",
			line:         "* world",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "*",
			expectRest:   "world",
		},
		{
			name:         "indented dash",
			line:         "  - indented",
			expectOK:     true,
			expectIndent: "  ",
			expectMarker: "-",
			expectRest:   "indented",
		},
		{
			name:         "deeply indented star",
			line:         "    * deep",
			expectOK:     true,
			expectIndent: "    ",
			expectMarker: "*",
			expectRest:   "deep",
		},
		{
			name:         "numeric ordered",
			line:         "1. first item",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "1.",
			expectRest:   "first item",
		},
		{
			name:         "multi-digit numeric",
			line:         "99. ninety-ninth",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "99.",
			expectRest:   "ninety-ninth",
		},
		{
			name:         "alpha ordered",
			line:         "a. alpha item",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "a.",
			expectRest:   "alpha item",
		},
		{
			name:         "indented numeric",
			line:         "   3. third",
			expectOK:     true,
			expectIndent: "   ",
			expectMarker: "3.",
			expectRest:   "third",
		},
		{
			name:         "empty dash bullet",
			line:         "- ",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "-",
			expectRest:   "",
		},
		{
			name:         "empty star bullet",
			line:         "* ",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "*",
			expectRest:   "",
		},
		{
			name:         "empty numeric bullet",
			line:         "1. ",
			expectOK:     true,
			expectIndent: "",
			expectMarker: "1.",
			expectRest:   "",
		},
		{
			name:     "plain text",
			line:     "just plain text",
			expectOK: false,
		},
		{
			name:     "empty string",
			line:     "",
			expectOK: false,
		},
		{
			name:     "blockquote",
			line:     "> quoted text",
			expectOK: false,
		},
		{
			name:     "heading",
			line:     "# heading",
			expectOK: false,
		},
		{
			name:     "dash without space",
			line:     "-nospace",
			expectOK: false,
		},
		{
			name:     "number without space after dot",
			line:     "1.nospace",
			expectOK: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			indent, marker, rest, ok := parseBulletPrefix(tc.line)
			assert.Equal(t, tc.expectOK, ok, "ok mismatch")
			if ok {
				assert.Equal(t, tc.expectIndent, indent, "indent mismatch")
				assert.Equal(t, tc.expectMarker, marker, "marker mismatch")
				assert.Equal(t, tc.expectRest, rest, "rest mismatch")
			}
		})
	}
}

func TestNextBulletPrefix(t *testing.T) {
	testcases := []struct {
		name   string
		indent string
		marker string
		expect string
	}{
		{
			name:   "dash",
			indent: "",
			marker: "-",
			expect: "- ",
		},
		{
			name:   "star",
			indent: "",
			marker: "*",
			expect: "* ",
		},
		{
			name:   "indented dash",
			indent: "  ",
			marker: "-",
			expect: "  - ",
		},
		{
			name:   "numeric 1 to 2",
			indent: "",
			marker: "1.",
			expect: "2. ",
		},
		{
			name:   "numeric 99 to 100",
			indent: "",
			marker: "99.",
			expect: "100. ",
		},
		{
			name:   "alpha a to b",
			indent: "",
			marker: "a.",
			expect: "b. ",
		},
		{
			name:   "alpha z to aa",
			indent: "",
			marker: "z.",
			expect: "aa. ",
		},
		{
			name:   "indented numeric",
			indent: "   ",
			marker: "3.",
			expect: "   4. ",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := nextBulletPrefix(tc.indent, tc.marker)
			assert.Equal(t, tc.expect, result)
		})
	}
}

func TestIncrementAlpha(t *testing.T) {
	testcases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "a to b",
			input:  "a",
			expect: "b",
		},
		{
			name:   "y to z",
			input:  "y",
			expect: "z",
		},
		{
			name:   "z to aa",
			input:  "z",
			expect: "aa",
		},
		{
			name:   "az to ba",
			input:  "az",
			expect: "ba",
		},
		{
			name:   "zz to aaa",
			input:  "zz",
			expect: "aaa",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := incrementAlpha(tc.input)
			assert.Equal(t, tc.expect, result)
		})
	}
}
