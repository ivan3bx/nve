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
# Run all tests (IMPORTANT: must include --tags=fts5 for SQLite FTS support)
make test

# Run tests directly with Go
go test ./... --tags=fts5 --count=1

# Run a specific test
go test -run TestName --tags=fts5
```

### Running the Application
```bash
# Run from source
go run --tags=fts5 cmd/main.go

# Or run the built binary
./dist/nve_linux_amd64_v1/nve
```

**Critical**: Always include `--tags=fts5` when building or testing. This enables SQLite's Full-Text Search (FTS5) extension, which is essential for the application's search functionality.

### Committing code

Prefer desscriptive commits listing the important changes.

* The title describes the intent of the change. It will be retained when squashing commits. (example: "Performance improvements on initial startup")
* The description is a bulleted list of changes, prefixed with a dash. (example: "- Refactors notes.go to lazy-load file contents.")
* If changes to go.mod, list the package being updated or introduced with no further comment.

## Architecture

### Three-Pane UI Structure

The application uses a single-threaded TUI with three main components arranged vertically:

1. **SearchBox** (top) - Input field for searching and filtering notes
2. **ListBox** (middle) - Displays search results with filenames, snippets, and timestamps
3. **ContentBox** (bottom) - Shows/edits the selected note's full content

Navigation flows: SearchBox → ListBox → ContentBox (using Tab), with Escape returning focus to SearchBox.

### Core Components

- **Notes** (notes.go): Central coordinator that manages the note collection, search operations, and notifies observers of changes. Uses the Observer pattern to update UI components.

- **Database** (database.go): SQLite wrapper with FTS5 for full-text search. Maintains two tables:
  - `documents`: Stores file metadata (filename, MD5 hash, modification time)
  - `content_index`: FTS5 virtual table for searching filename and text content

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

4. **File Indexing**: On startup and refresh, the app scans the directory, calculates MD5 hashes, and updates the database only for modified files. Deleted files are pruned from the database.

### File Support

Supported file types: `.txt`, `.md`, `.mdown`, `.go`, `.rb` (see `SUPPORTED_FILETYPES` in files.go)

All files are expected to be plain text and searchable via FTS5.

## Development Notes

- **Logging**: Debug logs are written to `nve-debug.log` in the working directory
- **Database**: `nve.db` is created in the working directory and persists the search index
- **Test Data**: The `test_data/` directory contains sample markdown files for testing
- **CGO**: SQLite driver requires CGO, which is enabled by default but may need special handling for cross-compilation

## Build Configuration

The project uses goreleaser for multi-platform builds:
- Targets: Linux and macOS (amd64 + arm64)
- Binary name: `nve`
- Build flags: `--tags=fts5` (critical for SQLite FTS support)
- Releases are drafted but not auto-published
