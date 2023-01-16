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
	Filename   string    `db:"filename"`
	MD5        string    `db:"md5"`
	ModifiedAt time.Time `db:"modified_at"`
}

func scanCurrentDirectory() []string {
	var files []string

	cwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(cwd, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		if !info.IsDir() && SUPPORTED_FILETYPES[filepath.Ext(path)] {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return files
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
