package nve

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

var SUPPORTED_FILETYPES = map[string]bool{
	".txt":   true,
	".md":    true,
	".mdown": true,
	".go":    true,
	".rb":    true,
}

type FileRef struct {
	Filename   string
	ModifiedAt time.Time
}

func (f *FileRef) DisplayName() string {
	return strings.TrimSuffix(filepath.Base(f.Filename), filepath.Ext(f.Filename))
}

func GetContent(filename string) string {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		logger.Printf("GetContent: %v", err)
	}

	return string(bytes)
}

func SaveContent(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
