package kb

import (
	"context"
	"database/sql"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

// ArticleVersion is one row of kb_article_versions, with the editor's name
// joined in live (SPEC gate 1.05: renaming a person changes their name in
// History too; removing one still shows it, since deactivation never
// deletes the row). BodyHTML is only populated by Version, not History —
// the list doesn't need each version's full content.
type ArticleVersion struct {
	ID           int64
	VersionNo    int
	Title        string
	CategoryID   int64
	BodyHTML     string
	Action       string
	RestoredFrom sql.NullInt64
	EditedByName string
	EditedAt     string // pre-formatted, SPEC A3: "Sat 19 Sep 2026, 14:30"
}

// History returns every version of an article, newest first (SPEC gate
// 1.22).
func (s *ArticleStore) History(articleID int64) ([]ArticleVersion, error) {
	rows, err := s.DB.Query(`
		SELECT v.id, v.version_no, v.title, v.category_id, v.action, v.restored_from_version, COALESCE(p.name, ''), v.edited_at
		FROM kb_article_versions v
		LEFT JOIN people p ON p.id = v.edited_by
		WHERE v.article_id = ?
		ORDER BY v.id DESC
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ArticleVersion
	for rows.Next() {
		v, editedAt, err := scanVersion(rows.Scan)
		if err != nil {
			return nil, err
		}
		v.EditedAt = formatEditedAt(editedAt)
		out = append(out, v)
	}
	return out, rows.Err()
}

// Version returns one specific version's full content, as it was
// published at the time (SPEC gate 1.23: "opening an old version shows it
// as it was").
func (s *ArticleStore) Version(articleID int64, versionNo int) (ArticleVersion, error) {
	var v ArticleVersion
	var editedAt string
	err := s.DB.QueryRow(`
		SELECT v.id, v.version_no, v.title, v.category_id, v.body_html, v.action, v.restored_from_version, COALESCE(p.name, ''), v.edited_at
		FROM kb_article_versions v
		LEFT JOIN people p ON p.id = v.edited_by
		WHERE v.article_id = ? AND v.version_no = ?
		ORDER BY v.id DESC
		LIMIT 1
	`, articleID, versionNo).Scan(&v.ID, &v.VersionNo, &v.Title, &v.CategoryID, &v.BodyHTML, &v.Action, &v.RestoredFrom, &v.EditedByName, &editedAt)
	if err != nil {
		return ArticleVersion{}, err
	}
	v.EditedAt = formatEditedAt(editedAt)
	return v, nil
}

func scanVersion(scan func(dest ...any) error) (v ArticleVersion, editedAt string, err error) {
	err = scan(&v.ID, &v.VersionNo, &v.Title, &v.CategoryID, &v.Action, &v.RestoredFrom, &v.EditedByName, &editedAt)
	return v, editedAt, err
}

func formatEditedAt(raw string) string {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return format.DateTime(t)
	}
	return raw
}

// Restore makes an old version current again: a new version_no, the
// restored title/category/body, and a history entry recording who
// restored it and from which version (SPEC gate 1.23). Search rows are
// rewritten only if the article is currently published — an archived
// article stays out of search either way (SPEC gate 1.24).
func (s *ArticleStore) Restore(ctx context.Context, articleID int64, versionNo int, personID int64) (Article, error) {
	version, err := s.Version(articleID, versionNo)
	if err != nil {
		return Article{}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Article{}, err
	}
	defer tx.Rollback()

	var currentVersion int
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT version_no, status FROM kb_articles WHERE id = ?`, articleID).Scan(&currentVersion, &status); err != nil {
		return Article{}, err
	}
	newVersion := currentVersion + 1
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := tx.ExecContext(ctx,
		`UPDATE kb_articles SET category_id = ?, title = ?, body_html = ?, version_no = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		version.CategoryID, version.Title, version.BodyHTML, newVersion, personID, now, articleID,
	); err != nil {
		return Article{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kb_article_versions (article_id, version_no, title, category_id, body_html, action, restored_from_version, edited_by, edited_at)
		 VALUES (?, ?, ?, ?, ?, 'restored', ?, ?, ?)`,
		articleID, newVersion, version.Title, version.CategoryID, version.BodyHTML, versionNo, personID, now,
	); err != nil {
		return Article{}, err
	}

	if status == "published" {
		if err := rewriteSearchRows(ctx, tx, articleID, version.Title, version.BodyHTML); err != nil {
			return Article{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Article{}, err
	}
	return Article{
		ID: articleID, CategoryID: version.CategoryID, Title: version.Title,
		BodyHTML: version.BodyHTML, Status: status, VersionNo: newVersion,
	}, nil
}

// Archive removes an article from its category and from search, and
// lists it in Archived (SPEC gate 1.24). The content and version_no are
// unchanged — this is a status change, not an edit — but it still gets
// its own History entry.
func (s *ArticleStore) Archive(ctx context.Context, articleID, personID int64) error {
	return s.setStatus(ctx, articleID, personID, "archived", "archived")
}

// Unarchive puts an article back in its category and back in search
// (SPEC gate 1.24).
func (s *ArticleStore) Unarchive(ctx context.Context, articleID, personID int64) error {
	return s.setStatus(ctx, articleID, personID, "published", "unarchived")
}

func (s *ArticleStore) setStatus(ctx context.Context, articleID, personID int64, newStatus, action string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var title string
	var categoryID, versionNo int64
	var bodyHTML string
	if err := tx.QueryRowContext(ctx,
		`SELECT title, category_id, body_html, version_no FROM kb_articles WHERE id = ?`, articleID,
	).Scan(&title, &categoryID, &bodyHTML, &versionNo); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE kb_articles SET status = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
		newStatus, personID, now, articleID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kb_article_versions (article_id, version_no, title, category_id, body_html, action, edited_by, edited_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		articleID, versionNo, title, categoryID, bodyHTML, action, personID, now,
	); err != nil {
		return err
	}

	if newStatus == "archived" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM kb_search WHERE article_id = ?`, articleID); err != nil {
			return err
		}
	} else if err := rewriteSearchRows(ctx, tx, articleID, title, bodyHTML); err != nil {
		return err
	}

	return tx.Commit()
}

// ArchivedArticle is one row of the Archived articles listing.
type ArchivedArticle struct {
	ID    int64
	Title string
}

// ListArchived returns every archived article (SPEC gate 1.24: "lists it
// in Archived articles").
func (s *ArticleStore) ListArchived() ([]ArchivedArticle, error) {
	rows, err := s.DB.Query(`SELECT id, title FROM kb_articles WHERE status = 'archived' ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ArchivedArticle
	for rows.Next() {
		var a ArchivedArticle
		if err := rows.Scan(&a.ID, &a.Title); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
