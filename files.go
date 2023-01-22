package nve

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

var SUPPORTED_FILETYPES = map[string]bool{
	".txt":   true,
	".md":    true,
	".mdown": true,
	".go":    true,
}

type FileRef struct {
	DocumentID int64     `db:"id"`
	Filename   string    `db:"filename"`
	MD5        string    `db:"md5"`
	ModifiedAt time.Time `db:"modified_at"`
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

func scanDirectory(dirname string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirname, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && SUPPORTED_FILETYPES[filepath.Ext(path)] {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func calculateMD5(path string) (string, error) {
	file, err := os.Open(path)

	if err != nil {
		return "", err
	}

	defer file.Close()

	hash := md5.New()

	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
