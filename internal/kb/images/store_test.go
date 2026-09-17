package images

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/people"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"people", "kb"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatalf("LoadMigrations(%s): %v", section, err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatalf("RunMigrations(%s): %v", section, err)
		}
	}

	return &Store{DB: sqlDB, DataDir: t.TempDir()}
}

func testPerson(t *testing.T, store *Store) int64 {
	t.Helper()
	p, err := (&people.Store{DB: store.DB}).Create("Sam")
	if err != nil {
		t.Fatalf("create person: %v", err)
	}
	return p.ID
}

var (
	jpegBytes = append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 20)...)
	pngBytes  = append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, bytes.Repeat([]byte{0}, 20)...)
	gifBytes  = append([]byte("GIF89a"), bytes.Repeat([]byte{0}, 20)...)
	webpBytes = append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBPVP8 ")...)
)

func TestSniffAcceptsEachSupportedType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		mime string
		ext  string
	}{
		{"jpeg", jpegBytes, "image/jpeg", "jpg"},
		{"png", pngBytes, "image/png", "png"},
		{"gif", gifBytes, "image/gif", "gif"},
		{"webp", webpBytes, "image/webp", "webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime, ext, ok := sniff(tt.data)
			if !ok || mime != tt.mime || ext != tt.ext {
				t.Errorf("sniff(%s) = (%q, %q, %v), want (%q, %q, true)", tt.name, mime, ext, ok, tt.mime, tt.ext)
			}
		})
	}
}

func TestSniffRefusesSVG(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if _, _, ok := sniff(svg); ok {
		t.Error("sniff accepted an SVG; SPEC B4 says no SVG")
	}
}

func TestSniffRefusesRenamedTextFile(t *testing.T) {
	// A .txt file's actual bytes, regardless of what name or extension a
	// client claims for it — Save never looks at either.
	plain := []byte("just some plain text, not a picture")
	if _, _, ok := sniff(plain); ok {
		t.Error("sniff accepted plain text")
	}
}

func TestSaveRejectsUnsupportedType(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)
	_, err := store.Save(context.Background(), bytes.NewReader([]byte("not a picture")), personID)
	if err != ErrUnsupportedType {
		t.Errorf("Save(text) error = %v, want ErrUnsupportedType", err)
	}
}

func TestSaveRejectsOversize(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)
	oversize := append(append([]byte{}, pngBytes...), make([]byte, MaxBytes)...)
	_, err := store.Save(context.Background(), bytes.NewReader(oversize), personID)
	if err != ErrTooLarge {
		t.Errorf("Save(oversize) error = %v, want ErrTooLarge", err)
	}
}

func TestSaveCollapsesDuplicateUploads(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	first, err := store.Save(context.Background(), bytes.NewReader(pngBytes), personID)
	if err != nil {
		t.Fatalf("Save (first): %v", err)
	}
	second, err := store.Save(context.Background(), bytes.NewReader(pngBytes), personID)
	if err != nil {
		t.Fatalf("Save (second): %v", err)
	}
	if first.SHA256 != second.SHA256 {
		t.Errorf("two uploads of the same bytes got different hashes: %q vs %q", first.SHA256, second.SHA256)
	}

	var count int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM images WHERE sha256 = ?`, first.SHA256).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("images rows for that hash = %d, want 1", count)
	}

	path := store.Path(first.SHA256, first.Ext)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if !bytes.Equal(data, pngBytes) {
		t.Error("stored file content doesn't match the uploaded bytes")
	}
	if got := filepath.Base(filepath.Dir(path)); got != first.SHA256[:2] {
		t.Errorf("stored file's directory = %q, want the hash's first two hex chars %q", got, first.SHA256[:2])
	}
}
