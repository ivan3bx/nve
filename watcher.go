package nve

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
)

// StartWatching begins monitoring the notes directory for filesystem changes.
// drawFunc is used to marshal UI updates onto the tview event loop.
// If watching fails to start, a warning is logged but the error is returned
// so the caller can decide how to handle it.
func (n *Notes) StartWatching(drawFunc func(func())) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	n.watcher = watcher
	n.drawFunc = drawFunc

	// Add root directory
	if err := watcher.Add(n.config.Filepath); err != nil {
		watcher.Close()
		n.watcher = nil
		return err
	}

	// Add subdirectories
	filepath.Walk(n.config.Filepath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if watchErr := watcher.Add(path); watchErr != nil {
				log.Printf("[WARN] watcher: could not watch %s: %v", path, watchErr)
			}
		}
		return nil
	})

	go n.watchLoop(watcher)

	log.Printf("[INFO] watcher: started monitoring %s", n.config.Filepath)
	return nil
}

// StopWatching stops the filesystem watcher.
func (n *Notes) StopWatching() {
	if n.watcher != nil {
		n.watcher.Close()
		n.watcher = nil
		log.Printf("[INFO] watcher: stopped")
	}
}

func (n *Notes) watchLoop(watcher *fsnotify.Watcher) {
	debounced := debounce.New(500 * time.Millisecond)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Watch newly created subdirectories
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := watcher.Add(event.Name); err != nil {
						log.Printf("[WARN] watcher: could not watch new dir %s: %v", event.Name, err)
					}
				}
			}

			// Filter to supported file types
			ext := filepath.Ext(event.Name)
			if !SUPPORTED_FILETYPES[ext] {
				continue
			}

			log.Printf("[DEBUG] watcher: event %s on %s", event.Op, event.Name)
			debounced(n.handleWatcherRefresh)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[ERROR] watcher: %v", err)
		}
	}
}

func (n *Notes) handleWatcherRefresh() {
	changed, err := n.Refresh()
	if err != nil {
		log.Printf("[ERROR] watcher: refresh failed: %v", err)
		return
	}

	if changed && n.drawFunc != nil {
		n.drawFunc(func() {
			n.Search(n.LastQuery)
		})
	}
}
