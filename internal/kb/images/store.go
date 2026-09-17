// Package images stores and serves the pictures articles embed (SPEC B3,
// B4). Files live under DataDir/images/<first 2 hex>/<sha256>.<ext> and are
// named by content hash, so re-uploading the same bytes collapses to the
// same file; nothing is ever deleted.
package images

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MaxBytes is the upload size limit (SPEC B4).
const MaxBytes = 20 * 1024 * 1024

// ErrTooLarge is returned by Save when the input exceeds MaxBytes.
var ErrTooLarge = errors.New("file is larger than 20 MB")

// ErrUnsupportedType is returned by Save for anything that doesn't sniff as
// JPEG, PNG, GIF or WebP (SPEC B4: "no SVG").
var ErrUnsupportedType = errors.New("file isn't a JPEG, PNG, GIF or WebP image")

// Image is one row of the images table.
type Image struct {
	SHA256 string
	Ext    string
	MIME   string
	Bytes  int64
}

// Store saves and looks up images. DataDir is the app's data directory
// (SPEC B3); files live under DataDir/images/.
type Store struct {
	DB      *sql.DB
	DataDir string
}

// Save reads r (capped at MaxBytes+1 so oversize input is rejected instead
// of exhausting memory), sniffs its type, and stores it under its SHA-256.
// A duplicate upload (same bytes already on disk) collapses to the
// existing file; the DB insert is idempotent for the same reason.
func (s *Store) Save(ctx context.Context, r io.Reader, personID int64) (Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Image{}, err
	}
	if len(data) > MaxBytes {
		return Image{}, ErrTooLarge
	}

	mime, ext, ok := sniff(data)
	if !ok {
		return Image{}, ErrUnsupportedType
	}

	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])

	if err := s.writeFile(sha, ext, data); err != nil {
		return Image{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO images (sha256, ext, mime, bytes, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(sha256) DO NOTHING`,
		sha, ext, mime, len(data), personID, now,
	); err != nil {
		return Image{}, err
	}

	return Image{SHA256: sha, Ext: ext, MIME: mime, Bytes: int64(len(data))}, nil
}

// Path returns the on-disk path for an image (SPEC B3 layout).
func (s *Store) Path(sha, ext string) string {
	return filepath.Join(s.DataDir, "images", sha[:2], sha+"."+ext)
}

// writeFile writes data to Path(sha, ext), skipping the write entirely if
// that exact file already exists (the duplicate-collapse case), and
// otherwise via a temp file + rename so a failed write never leaves a
// half-written file at the final path.
func (s *Store) writeFile(sha, ext string, data []byte) error {
	finalPath := s.Path(sha, ext)
	if _, err := os.Stat(finalPath); err == nil {
		return nil
	}

	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "upload-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// Get returns the stored mime type for an image, so the handler can serve
// it with the right Content-Type without trusting the request's extension.
func (s *Store) Get(sha, ext string) (Image, error) {
	var img Image
	img.SHA256, img.Ext = sha, ext
	err := s.DB.QueryRow(`SELECT mime, bytes FROM images WHERE sha256 = ? AND ext = ?`, sha, ext).
		Scan(&img.MIME, &img.Bytes)
	return img, err
}

// sniff identifies the image type from content, not from any client-
// supplied name or header. http.DetectContentType doesn't recognise WebP,
// so its signature ("RIFF"...."WEBP") is checked first.
func sniff(data []byte) (mime, ext string, ok bool) {
	if isWebP(data) {
		return "image/webp", "webp", true
	}
	switch detected := http.DetectContentType(data); {
	case strings.HasPrefix(detected, "image/jpeg"):
		return "image/jpeg", "jpg", true
	case strings.HasPrefix(detected, "image/png"):
		return "image/png", "png", true
	case strings.HasPrefix(detected, "image/gif"):
		return "image/gif", "gif", true
	default:
		return "", "", false
	}
}

func isWebP(data []byte) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}
