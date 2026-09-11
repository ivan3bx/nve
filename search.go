package nve

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type SearchResult struct {
	*FileRef
	Snippet string
}

const (
	// snippetLength is the maximum number of bytes shown for a result snippet.
	snippetLength = 180

	// snippetLead is how many bytes of context precede the first match in a snippet.
	snippetLead = 40

	snippetEllipsis = "..."
)

// scanFiles walks root and returns a FileRef for every supported file found.
func scanFiles(root string) ([]*FileRef, error) {
	var refs []*FileRef

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !SUPPORTED_FILETYPES[filepath.Ext(path)] {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		refs = append(refs, &FileRef{Filename: path, ModifiedAt: info.ModTime()})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return refs, nil
}

// recentFiles returns the most recently modified files under root, up to limit,
// each with a snippet taken from the start of the file.
func recentFiles(root string, limit int) ([]*SearchResult, error) {
	refs, err := scanFiles(root)
	if err != nil {
		return nil, err
	}

	sortByModified(refs)

	if len(refs) > limit {
		refs = refs[:limit]
	}

	results := make([]*SearchResult, 0, len(refs))

	for _, ref := range refs {
		content, err := os.ReadFile(ref.Filename)
		if err != nil {
			logger.Printf("recentFiles: %v", err)
			continue
		}

		results = append(results, &SearchResult{FileRef: ref, Snippet: leadSnippet(content)})
	}

	return results, nil
}

// searchFiles returns every file under root that matches query, ordered by
// modification time (newest first).
//
// The query is split into whitespace-separated terms. A file matches when every
// term appears, case-insensitively, somewhere in its content or display name.
// Terms are unordered and independent: "foo bar" and "bar foo" both match a
// file containing "foobar", or one containing "foo" and "bar" on separate lines.
func searchFiles(root, query string) ([]*SearchResult, error) {
	terms := searchTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	refs, err := scanFiles(root)
	if err != nil {
		return nil, err
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []*SearchResult
		work    = make(chan *FileRef)
	)

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ref := range work {
				if res := matchFile(ref, terms); res != nil {
					mu.Lock()
					results = append(results, res)
					mu.Unlock()
				}
			}
		}()
	}

	for _, ref := range refs {
		work <- ref
	}
	close(work)
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return lessByModified(results[i].FileRef, results[j].FileRef)
	})

	return results, nil
}

// searchTerms splits a query into lower-cased search terms.
func searchTerms(query string) [][]byte {
	fields := strings.Fields(strings.ToLower(query))
	terms := make([][]byte, len(fields))

	for i, f := range fields {
		terms[i] = []byte(f)
	}

	return terms
}

// matchFile reads ref and returns a SearchResult if every term is present in
// either the file's display name or its content. Returns nil otherwise.
func matchFile(ref *FileRef, terms [][]byte) *SearchResult {
	content, err := os.ReadFile(ref.Filename)
	if err != nil {
		logger.Printf("matchFile: %v", err)
		return nil
	}

	lower := bytes.ToLower(content)
	name := []byte(strings.ToLower(ref.DisplayName()))

	for _, term := range terms {
		if !bytes.Contains(lower, term) && !bytes.Contains(name, term) {
			return nil
		}
	}

	return &SearchResult{FileRef: ref, Snippet: matchSnippet(content, lower, terms)}
}

// matchSnippet builds a snippet around the earliest term match in content.
// Every term occurrence within the snippet window is wrapped in "**".
// If no term matches the content (i.e. the file matched by name only), the
// snippet is taken from the start of the file instead.
func matchSnippet(content, lower []byte, terms [][]byte) string {
	first := -1

	for _, term := range terms {
		if i := bytes.Index(lower, term); i >= 0 && (first < 0 || i < first) {
			first = i
		}
	}

	if first < 0 {
		return leadSnippet(content)
	}

	// Offsets are computed against the lower-cased copy. Unicode case folding
	// can change byte lengths, in which case those offsets are only valid
	// against the lower-cased copy itself.
	src := content
	if len(lower) != len(content) {
		src = lower
	}

	start := first - snippetLead
	if start < 0 {
		start = 0
	}

	// Snap the window start to the next word boundary so it doesn't open mid-word.
	if start > 0 {
		if i := bytes.IndexAny(src[start:first], " \t\n"); i >= 0 {
			start += i + 1
		}
		for start < first && !utf8.RuneStart(src[start]) {
			start++
		}
	}

	end := start + snippetLength
	if end > len(src) {
		end = len(src)
	}
	for end < len(src) && !utf8.RuneStart(src[end]) {
		end++
	}

	var sb strings.Builder

	if start > 0 {
		sb.WriteString(snippetEllipsis)
	}

	sb.Write(boldTerms(src[start:end], lower[start:end], terms))

	if end < len(src) {
		sb.WriteString(snippetEllipsis)
	}

	return strings.ReplaceAll(sb.String(), "\n", " ")
}

// boldTerms wraps every occurrence of terms within window in "**". Overlapping
// or adjacent occurrences are merged into a single bold span.
func boldTerms(window, lowerWindow []byte, terms [][]byte) []byte {
	type span struct{ start, end int }
	var spans []span

	for _, term := range terms {
		offset := 0
		for {
			i := bytes.Index(lowerWindow[offset:], term)
			if i < 0 {
				break
			}
			spans = append(spans, span{offset + i, offset + i + len(term)})
			offset += i + len(term)
		}
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	var out bytes.Buffer
	pos := 0

	for i := 0; i < len(spans); i++ {
		s := spans[i]
		for i+1 < len(spans) && spans[i+1].start <= s.end {
			i++
			if spans[i].end > s.end {
				s.end = spans[i].end
			}
		}
		out.Write(window[pos:s.start])
		out.WriteString("**")
		out.Write(window[s.start:s.end])
		out.WriteString("**")
		pos = s.end
	}

	out.Write(window[pos:])
	return out.Bytes()
}

// leadSnippet returns the first snippetLength bytes of content on a single line.
func leadSnippet(content []byte) string {
	end := len(content)
	if end > snippetLength {
		end = snippetLength
		for end < len(content) && !utf8.RuneStart(content[end]) {
			end++
		}
	}

	return strings.ReplaceAll(string(content[:end]), "\n", " ")
}

// sortByModified orders refs newest first, breaking ties by filename.
func sortByModified(refs []*FileRef) {
	sort.Slice(refs, func(i, j int) bool { return lessByModified(refs[i], refs[j]) })
}

func lessByModified(a, b *FileRef) bool {
	if !a.ModifiedAt.Equal(b.ModifiedAt) {
		return a.ModifiedAt.After(b.ModifiedAt)
	}
	return a.Filename < b.Filename
}
