# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`nve` (Note, View, Edit) is a terminal-based note-taking application inspired by Notational Velocity. It provides a fast, keyboard-driven interface for searching, viewing, creating, and editing plain-text files using the `tview` TUI framework.

## Build and Development Commands

### Building
```bash
# Build for current platform
make build-local

# Build for all platforms (requires goreleaser)
make build

# Create local release archives
make release-local
```

### Testing
```bash
# Run all tests
make test

# Run tests directly with Go
go test ./... --count=1

# Run a specific test
go test -run TestName

# Run TUI integration tests (requires tmux)
make test-tui
```

### Running the Application
```bash
# Run from source
go run cmd/main.go

# Or run the built binary
./dist/nve_linux_amd64_v1/nve
```

### Committing code

Prefer descriptive commits listing the important changes.

* The title describes the intent of the change. It will be retained when squashing commits. (example: "Performance improvements on initial startup")
* The description is a bulleted list of changes, prefixed with a dash. (example: "- Refactors notes.go to lazy-load file contents.")
* If changes to go.mod, list the package being updated or introduced with no further comment.

### Addressing PR feedback

When addressing PR review feedback, critically evaluate each comment before applying changes. If you disagree with the feedback or believe it's incorrect, explain your reasoning and ask before making the change.

### Writing Go Tests

- When a test has multiple variations with the same basic setup, use a table-driven test
- Use `testcases` as the variable name for the slice of test case structs
- Use `tc` as the loop variable: `for _, tc := range testcases`
- Use multi-line struct literals for each test case, with every field on its own line:
  ```go
  {
      name:       "descriptive name",
      input:      "value",
      expectFoo:  true,
  },
  ```

## Architecture

### Three-Pane UI Structure

The application uses a single-threaded TUI with three main components arranged vertically:

1. **SearchBox** (top) - Input field for searching and filtering notes
2. **ListBox** (middle) - Displays search results with filenames, snippets, and timestamps
3. **ContentBox** (bottom) - Shows/edits the selected note's full content

Navigation flows: SearchBox → ListBox → ContentBox (using Tab), with Escape returning focus to SearchBox.

### Core Components

- **Notes** (notes.go): Central coordinator that manages the note collection, search operations, and notifies observers of changes. Uses the Observer pattern to update UI components.

- **Search** (search.go): File-based search with no index. Each query walks the notes directory, reads every supported file, and matches whitespace-separated terms as case-insensitive substrings of the content or display name. Terms are unordered and independent; all must match somewhere in the file. Results are ordered by modification time, newest first.

- **UI Boxes**: Each inherits from a `tview` primitive and implements custom input handlers:
  - `SearchBox`: Debounced search triggering, note creation on Enter when no results
  - `ListBox`: Displays search results with custom formatting and navigation
  - `ContentBox`: Editable text area with debounced auto-save (300ms)

### Key Interaction Patterns

1. **Observer Pattern**: Notes notifies ListBox when search results change via `SearchResultsUpdate()`

2. **Focus Coordination**: Components use `setFocus` callbacks to transfer focus between panes. Non-navigational keypresses in ListBox forward to SearchBox for seamless typing.

3. **Debounced Operations**:
   - Search queries are triggered immediately on text change
   - File saves are debounced (300ms) to avoid excessive disk writes

4. **Filesystem Refresh**: A watcher (watcher.go) re-runs the last query when files under the notes directory change, so results always reflect what is on disk.

### File Support

Supported file types: `.txt`, `.md`, `.mdown`, `.go`, `.rb` (see `SUPPORTED_FILETYPES` in files.go)

All files are expected to be plain text.

## Development Notes

- **Logging**: Debug logs are written to `nve-debug.log` in the working directory
- **Test Data**: The `test_data/` directory contains sample markdown files for testing
- **CGO**: Only needed on macOS, for native file versioning (versions_darwin.m). Linux builds are pure Go.

## Build Configuration

The project uses goreleaser for multi-platform builds:
- Targets: Linux and macOS (amd64 + arm64)
- Binary name: `nve`
- Releases are drafted but not auto-published
