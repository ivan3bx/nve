package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/ivan3bx/nve"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "nve",
	Short: "A terminal-based note-taking app inspired by Notational Velocity",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringP("directory", "d", ".", "Directory containing notes")
	viper.BindPFlag("directory", rootCmd.Flags().Lookup("directory"))
}

func run(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("directory")

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("cannot access directory %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absDir)
	}

	// Setup debug logging to file
	logFile, err := os.OpenFile(filepath.Join(absDir, "nve-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("cannot create log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	config := nve.LoadConfig()
	log.Printf("[INFO] config: versioning=%v", config.Versioning)
	log.Printf("[INFO] notes directory: %s", absDir)

	var (
		app   = tview.NewApplication()
		notes = nve.NewNotes(nve.NotesConfig{
			Filepath: absDir,
			DBPath:   filepath.Join(absDir, "nve.db"),
		})

		// View hierarchy
		contentBox = nve.NewContentBox(config)
		listBox    = nve.NewListBox(contentBox, notes)
		searchBox  = nve.NewSearchBox(listBox, contentBox, notes)
	)

	defer contentBox.Shutdown()

	notes.RegisterObservers(listBox)
	notes.Notify()

	if err := notes.StartWatching(func(f func()) { app.QueueUpdateDraw(f) }); err != nil {
		log.Printf("[WARN] filesystem watcher not available: %v", err)
	}
	defer notes.StopWatching()

	// global input events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			if searchBox.HasFocus() {
				app.SetFocus(listBox)
			} else if listBox.HasFocus() {
				app.SetFocus(contentBox)
			} else {
				break
			}
			return &tcell.EventKey{}
		case tcell.KeyEscape:
			app.SetFocus(searchBox)
			searchBox.SetText("")
			notes.Search("")
			return &tcell.EventKey{}
		}

		return event
	})

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(searchBox, 3, 0, true).
				AddItem(listBox, 0, 1, false).
				AddItem(contentBox, 0, 3, false), 0, 2, true,
		)

	if err := app.SetRoot(flex, true).SetFocus(flex).EnableMouse(true).Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
