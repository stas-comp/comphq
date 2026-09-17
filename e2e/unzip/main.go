// Command unzip extracts a zip file to a directory, for e2e tests that
// need to inspect an exported archive as real files on disk — SPEC gate
// 1.34's check opens the export with a file:// URL in an offline browser
// context, which needs actual extracted files, not zip entries. Built
// once in global setup and reused by every worker, the same as
// e2e/seed.
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: unzip <zip-path> <dest-dir>")
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

func run(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		path := filepath.Join(destDir, f.Name)
		if path != destDir && !strings.HasPrefix(path, destDir+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := extractFile(f, path); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}
	return nil
}

func extractFile(f *zip.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
