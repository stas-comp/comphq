package settings

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/kb"
)

// handleExport streams comphq-export-YYYY-MM-DD.zip (SPEC B5, gate 1.34):
// index.html plus one self-contained page per article (kb.BuildExport),
// every image those pages reference, and a fresh comphq.db copy. It's
// read-only — there's nothing to roll back if it fails partway through
// writing the response, so unlike Publish this needs no transaction.
func (h *Handlers) handleExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, archived, err := h.articles.ListForExport(ctx)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	files, imageSrcs := kb.BuildExport(categories, archived)

	dbBytes, err := freshDatabaseCopy(ctx, h.srv.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("comphq-export-%s.zip", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, f := range files {
		fw, err := zw.Create(f.Path)
		if err != nil {
			return
		}
		if _, err := fw.Write(f.Content); err != nil {
			return
		}
	}

	// A file missing from disk is skipped rather than failing the whole
	// export — Settings > Content check (P1-36) is where that gets
	// surfaced and fixed.
	for _, src := range imageSrcs {
		sha, ext, ok := parseImageSrc(src)
		if !ok {
			continue
		}
		data, err := os.ReadFile(h.images.Path(sha, ext))
		if err != nil {
			continue
		}
		fw, err := zw.Create("images/" + sha + "." + ext)
		if err != nil {
			return
		}
		if _, err := fw.Write(data); err != nil {
			return
		}
	}

	for _, f := range []struct {
		name string
		src  CSVSource
	}{{"tasks.csv", h.tasks}, {"events.csv", h.events}, {"steps.csv", h.steps}} {
		csvRows, err := f.src.ExportRows(ctx)
		if err != nil {
			return
		}
		fw, err := zw.Create(f.name)
		if err != nil {
			return
		}
		if err := writeCSV(fw, csvRows); err != nil {
			return
		}
	}

	if fw, err := zw.Create("comphq.db"); err == nil {
		fw.Write(dbBytes)
	}
}

// freshDatabaseCopy VACUUM INTOs a temp file and returns its bytes, the
// same "clean snapshot" approach the nightly/pre-update backups use
// (internal/db.backupInto), but without writing anywhere under DataDir
// since this copy only exists to go straight into the zip.
func freshDatabaseCopy(ctx context.Context, sqlDB *sql.DB) ([]byte, error) {
	tmp, err := os.CreateTemp("", "comphq-export-*.db")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	os.Remove(tmpPath) // VACUUM INTO refuses to write to a file that already exists
	defer os.Remove(tmpPath)

	if _, err := sqlDB.ExecContext(ctx, `VACUUM INTO ?`, tmpPath); err != nil {
		return nil, err
	}
	return os.ReadFile(tmpPath)
}

func parseImageSrc(src string) (sha, ext string, ok bool) {
	name := strings.TrimPrefix(src, "/images/")
	if name == src {
		return "", "", false
	}
	dot := strings.LastIndex(name, ".")
	if dot < 0 {
		return "", "", false
	}
	return name[:dot], name[dot+1:], true
}

// writeCSV writes rows the way Excel expects to open a spreadsheet by
// double-click (SPEC B5, gate 2.22): a UTF-8 byte-order mark so accented
// names aren't garbled, and CRLF line endings.
func writeCSV(w io.Writer, rows [][]string) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	if err := cw.WriteAll(rows); err != nil {
		return err
	}
	return cw.Error()
}
