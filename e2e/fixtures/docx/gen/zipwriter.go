package main

import (
	"archive/zip"
	"bytes"
	"os"
	"time"
)

// A fixed timestamp keeps every generated .docx byte-identical across
// runs (PLAN.md P1-11: "npm run fixtures:docx reproduces them byte-
// identically (fixed zip timestamps)").
var fixedZipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// part is one file inside the zip package. Order is significant for
// reproducibility, so parts are built as a slice, never a map.
type part struct {
	name string
	data []byte
}

// writeDocxFile builds a zip from parts, in order, and writes it to path.
func writeDocxFile(path string, parts []part) error {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		fh := &zip.FileHeader{
			Name:     p.name,
			Method:   zip.Deflate,
			Modified: fixedZipTime,
		}
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		if _, err := w.Write(p.data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
