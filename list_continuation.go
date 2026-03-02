package nve

import (
	"fmt"
	"strings"
	"unicode"
)

// parseBulletPrefix parses a line into its indentation, bullet marker, and
// remaining text. It recognises unordered markers (- and *) and ordered
// markers (numeric like "1." or alphabetic like "a."). Returns ok=false if
// the line does not start with a recognised bullet pattern.
func parseBulletPrefix(line string) (indent, marker, rest string, ok bool) {
	// Extract leading whitespace.
	trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
	indent = line[:len(line)-len(trimmed)]

	// Unordered: "- " or "* "
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		marker = trimmed[:1]
		rest = trimmed[2:]
		return indent, marker, rest, true
	}

	// Ordered: digits followed by ". " (e.g. "1. ") or one or more lowercase letters followed by ". " (e.g. "a. ", "aa. ")
	dotIdx := strings.Index(trimmed, ". ")
	if dotIdx < 1 {
		return "", "", "", false
	}

	prefix := trimmed[:dotIdx]

	// Numeric ordered list: all digits.
	allDigits := true
	for _, r := range prefix {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		marker = prefix + "."
		rest = trimmed[dotIdx+2:]
		return indent, marker, rest, true
	}

	// Alpha ordered list: all lowercase letters.
	allAlpha := true
	for _, r := range prefix {
		if r < 'a' || r > 'z' {
			allAlpha = false
			break
		}
	}
	if allAlpha {
		marker = prefix + "."
		rest = trimmed[dotIdx+2:]
		return indent, marker, rest, true
	}

	return "", "", "", false
}

// nextBulletPrefix returns the bullet prefix for the next list line,
// including the trailing space. For unordered markers the same marker is
// returned. For ordered markers the value is incremented.
func nextBulletPrefix(indent, marker string) string {
	// Unordered markers.
	if marker == "-" || marker == "*" {
		return indent + marker + " "
	}

	// Must end with "." for ordered markers.
	if !strings.HasSuffix(marker, ".") {
		return indent + marker + " "
	}

	val := marker[:len(marker)-1]

	// Numeric: "1." → "2.", "99." → "100."
	allDigits := true
	for _, r := range val {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		n := 0
		for _, r := range val {
			n = n*10 + int(r-'0')
		}
		return indent + fmt.Sprintf("%d. ", n+1)
	}

	// Alpha: "a." → "b.", "z." → "aa."
	return indent + incrementAlpha(val) + ". "
}

// incrementAlpha increments an alphabetic string: "a" → "b", "z" → "aa", "az" → "ba".
func incrementAlpha(s string) string {
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'z' {
			runes[i]++
			return string(runes)
		}
		runes[i] = 'a'
	}
	return "a" + string(runes)
}
