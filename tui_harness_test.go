//go:build integration

package nve

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// binaryPath is set once by TestMain and reused by all TUI tests.
var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all integration tests.
	tmp, err := os.MkdirTemp("", "nve-tui-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "nve")
	cmd := exec.Command("go", "build", "--tags=fts5", "-o", binaryPath, "./cmd/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TUIHarness drives the real nve binary inside a tmux session for functional testing.
type TUIHarness struct {
	t       *testing.T
	dir     string // temp dir where the app runs
	session string // tmux session name
	mu      sync.Mutex
}

// NewTUIHarness builds and launches nve in a tmux session.
// seedFiles is a map of filename -> content to pre-populate the test directory.
func NewTUIHarness(t *testing.T, seedFiles map[string]string) *TUIHarness {
	t.Helper()

	dir := t.TempDir()

	// Write seed files
	for name, content := range seedFiles {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write seed file %s: %v", name, err)
		}
	}

	// Use test name as tmux session name (sanitized)
	session := strings.ReplaceAll(t.Name(), "/", "-")

	h := &TUIHarness{
		t:       t,
		dir:     dir,
		session: session,
	}

	// Kill any stale session with the same name
	exec.Command("tmux", "kill-session", "-t", session).Run()

	// Launch tmux session running nve
	launchCmd := fmt.Sprintf("cd %s && %s", h.dir, binaryPath)
	cmd := exec.Command("tmux", "new-session", "-d", "-s", session, "-x", "120", "-y", "30", launchCmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to start tmux session: %v\n%s", err, out)
	}

	// Wait for app to be ready (Search Box title visible)
	h.WaitFor(func(screen string) bool {
		return strings.Contains(screen, "Search Box")
	}, 10*time.Second)

	t.Cleanup(h.Cleanup)

	return h
}

// SendKeys sends each key string to the tmux session with a pause between them.
func (h *TUIHarness) SendKeys(keys ...string) {
	h.t.Helper()
	for _, key := range keys {
		cmd := exec.Command("tmux", "send-keys", "-t", h.session, key)
		if out, err := cmd.CombinedOutput(); err != nil {
			h.t.Fatalf("SendKeys(%q) failed: %v\n%s", key, err, out)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// Capture returns the current tmux pane content.
func (h *TUIHarness) Capture() string {
	h.t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()

	cmd := exec.Command("tmux", "capture-pane", "-t", h.session, "-p")
	out, err := cmd.Output()
	if err != nil {
		h.t.Fatalf("Capture failed: %v", err)
	}
	return string(out)
}

// WaitFor polls the screen every 200ms until predicate returns true or timeout is reached.
// Returns the final captured screen. Fails the test on timeout.
func (h *TUIHarness) WaitFor(predicate func(screen string) bool, timeout time.Duration) string {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	var screen string
	for time.Now().Before(deadline) {
		screen = h.Capture()
		if predicate(screen) {
			return screen
		}
		time.Sleep(200 * time.Millisecond)
	}
	h.t.Logf("WaitFor timeout — final screen:\n%s", screen)
	h.t.Fatalf("WaitFor timed out after %v", timeout)
	return screen
}

// WriteFile writes a file into the test directory (simulates external edit).
func (h *TUIHarness) WriteFile(name, content string) {
	h.t.Helper()
	path := filepath.Join(h.dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		h.t.Fatalf("WriteFile mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		h.t.Fatalf("WriteFile failed: %v", err)
	}
}

// ReadFile reads a file from the test directory.
func (h *TUIHarness) ReadFile(name string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.dir, name))
	if err != nil {
		h.t.Fatalf("ReadFile(%s) failed: %v", name, err)
	}
	return string(data)
}

// RemoveFile deletes a file from the test directory.
func (h *TUIHarness) RemoveFile(name string) {
	h.t.Helper()
	if err := os.Remove(filepath.Join(h.dir, name)); err != nil {
		h.t.Fatalf("RemoveFile(%s) failed: %v", name, err)
	}
}

// Snapshot captures and logs the screen with a label (useful for debugging mid-test).
func (h *TUIHarness) Snapshot(label string) {
	h.t.Helper()
	screen := h.Capture()
	h.t.Logf("=== Snapshot [%s] ===\n%s", label, screen)
}

// Cleanup kills the tmux session. On test failure, logs the final screen and debug log.
func (h *TUIHarness) Cleanup() {
	if h.t.Failed() {
		// Log final screen
		if screen, err := exec.Command("tmux", "capture-pane", "-t", h.session, "-p").Output(); err == nil {
			h.t.Logf("=== Final screen on failure ===\n%s", screen)
		}
		// Log debug log contents
		logPath := filepath.Join(h.dir, "nve-debug.log")
		if data, err := os.ReadFile(logPath); err == nil {
			h.t.Logf("=== nve-debug.log ===\n%s", string(data))
		}
	}
	exec.Command("tmux", "kill-session", "-t", h.session).Run()
}
