package kb

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	kbhtml "github.com/stas-comp/comphq/internal/kb/html"
	"github.com/stas-comp/comphq/internal/kb/images"
)

// Article is one row of kb_articles. ImageFailures is set only on the
// Article Publish just returned (SPEC gate 1.19: "the editor response
// lists the failures") — it's never stored.
type Article struct {
	ID            int64
	CategoryID    int64
	Title         string
	BodyHTML      string
	Status        string
	VersionNo     int
	ImageFailures []string
}

// ArticleInput is what a publish form submits. ID is 0 for a new article.
// ExpectedVersion is the version_no the editor started from — checked
// against the article's current version_no on an edit (ID != 0) unless
// Override is set (SPEC gate 1.21: "Publish mine anyway").
type ArticleInput struct {
	ID              int64
	CategoryID      int64
	Title           string
	BodyHTML        string
	ExpectedVersion int
	Override        bool
}

// ErrArticleEmptyTitle is returned by Publish for blank input.
var ErrArticleEmptyTitle = errors.New("title can't be empty")

// ErrVersionConflict is returned by Publish when someone else published a
// newer version of the same article first (SPEC gate 1.21).
var ErrVersionConflict = errors.New("someone else changed this article while you were editing")

type ArticleStore struct {
	DB     *sql.DB
	Images *images.Store
}

// Publish copies external images onto the NAS, sanitises the body, assigns
// block ids, saves the article (a new row for a new article, or an
// incremented version for an existing one), records a version, and
// rewrites its search rows — all in one transaction (SPEC B3/B4: "Rows are
// rewritten in the same transaction as the article save").
func (s *ArticleStore) Publish(ctx context.Context, input ArticleInput, personID int64) (Article, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Article{}, ErrArticleEmptyTitle
	}

	// SPEC B4: "for every img whose src isn't /images/...", fetch or
	// decode it before Sanitize, which unconditionally drops anything
	// still pointing elsewhere afterward.
	withLocalImages, imageFailures := kbhtml.RewriteExternalImages(input.BodyHTML, s.fetchImage(ctx, personID))
	withBlocks := kbhtml.AssignBlocks(kbhtml.Sanitize(withLocalImages))
	blockTexts := kbhtml.BlockTexts(withBlocks)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Article{}, err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	action := "edited"
	article := Article{
		ID: input.ID, CategoryID: input.CategoryID, Title: title,
		BodyHTML: withBlocks, Status: "published",
	}

	if input.ID == 0 {
		action = "created"
		article.VersionNo = 1
		res, err := tx.ExecContext(ctx,
			`INSERT INTO kb_articles (category_id, title, body_html, status, version_no, created_by, created_at, updated_by, updated_at)
			 VALUES (?, ?, ?, 'published', 1, ?, ?, ?, ?)`,
			input.CategoryID, title, withBlocks, personID, now, personID, now,
		)
		if err != nil {
			return Article{}, err
		}
		article.ID, err = res.LastInsertId()
		if err != nil {
			return Article{}, err
		}
	} else {
		var currentVersion int
		if err := tx.QueryRowContext(ctx, `SELECT version_no FROM kb_articles WHERE id = ?`, input.ID).Scan(&currentVersion); err != nil {
			return Article{}, err
		}
		if !input.Override && currentVersion != input.ExpectedVersion {
			return Article{}, ErrVersionConflict
		}
		article.VersionNo = currentVersion + 1
		if _, err := tx.ExecContext(ctx,
			`UPDATE kb_articles SET category_id = ?, title = ?, body_html = ?, version_no = ?, updated_by = ?, updated_at = ? WHERE id = ?`,
			input.CategoryID, title, withBlocks, article.VersionNo, personID, now, input.ID,
		); err != nil {
			return Article{}, err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kb_article_versions (article_id, version_no, title, category_id, body_html, action, edited_by, edited_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		article.ID, article.VersionNo, title, input.CategoryID, withBlocks, action, personID, now,
	); err != nil {
		return Article{}, err
	}

	if err := rewriteSearchRowsFromBlocks(ctx, tx, article.ID, title, blockTexts); err != nil {
		return Article{}, err
	}

	if err := tx.Commit(); err != nil {
		return Article{}, err
	}
	article.ImageFailures = imageFailures
	return article, nil
}

// rewriteSearchRows recomputes an article's kb_search rows from its
// already-sanitised, already-block-id'd body (SPEC B4). Used by Restore
// and Unarchive, which don't already have blockTexts computed the way
// Publish does.
func rewriteSearchRows(ctx context.Context, tx *sql.Tx, articleID int64, title, bodyHTML string) error {
	return rewriteSearchRowsFromBlocks(ctx, tx, articleID, title, kbhtml.BlockTexts(bodyHTML))
}

// rewriteSearchRowsFromBlocks deletes and re-inserts an article's
// kb_search rows in the same transaction as the change that caused them
// to need updating (SPEC B4: "the same transaction as the article save").
// Block 0 always holds the title; blocks 1..n hold each body block's text.
func rewriteSearchRowsFromBlocks(ctx context.Context, tx *sql.Tx, articleID int64, title string, blockTexts map[int]string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM kb_search WHERE article_id = ?`, articleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kb_search (title, body, article_id, block_id) VALUES (?, '', ?, 0)`,
		title, articleID,
	); err != nil {
		return err
	}
	for blockID, text := range blockTexts {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO kb_search (title, body, article_id, block_id) VALUES ('', ?, ?, ?)`,
			text, articleID, blockID,
		); err != nil {
			return err
		}
	}
	return nil
}

// fetchImage returns a RewriteExternalImages callback that decodes a
// data: URL or fetches an http(s) one through s.Images, whichever the src
// is; anything else (an unrecognised scheme) is treated as a failure.
func (s *ArticleStore) fetchImage(ctx context.Context, personID int64) func(src string) (string, bool) {
	return func(src string) (string, bool) {
		var img images.Image
		var err error
		switch {
		case strings.HasPrefix(src, "data:"):
			img, err = s.Images.StoreDataURL(ctx, src, personID)
		case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
			img, err = s.Images.FetchAndStore(ctx, src, personID)
		default:
			return "", false
		}
		if err != nil {
			return "", false
		}
		return "/images/" + img.SHA256 + "." + img.Ext, true
	}
}

// Get returns one article by id, regardless of status.
func (s *ArticleStore) Get(id int64) (Article, error) {
	var a Article
	err := s.DB.QueryRow(
		`SELECT id, category_id, title, body_html, status, version_no FROM kb_articles WHERE id = ?`, id,
	).Scan(&a.ID, &a.CategoryID, &a.Title, &a.BodyHTML, &a.Status, &a.VersionNo)
	return a, err
}

// CategoryArticle is one row of a category's published article listing.
type CategoryArticle struct {
	ID    int64
	Title string
}

// ListByCategory returns a category's published articles, most recently
// updated first.
func (s *ArticleStore) ListByCategory(categoryID int64) ([]CategoryArticle, error) {
	rows, err := s.DB.Query(
		`SELECT id, title FROM kb_articles WHERE category_id = ? AND status = 'published' ORDER BY updated_at DESC`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CategoryArticle
	for rows.Next() {
		var a CategoryArticle
		if err := rows.Scan(&a.ID, &a.Title); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
