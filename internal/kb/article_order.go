package kb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/stas-comp/comphq/internal/app"
)

// Articles in their own order within a category (SPEC B13.8, D-98, gates
// 7.55-7.59). The order is the position column, 1..n among a category's
// published articles; categories are ordered the same way (sort_order).
// Reordering is not an edit: it records no History, bumps no version and
// leaves updated_at alone, so Recently updated is about content only.

// ErrArticleNotInOrder is returned when an article or its neighbour is not a
// published article in the category (archived, gone, or somewhere else now).
var ErrArticleNotInOrder = errors.New("that article is not in this list any more")

// nextPosition is the number a new arrival in the category gets: the bottom.
func nextPosition(ctx context.Context, tx *sql.Tx, categoryID int64) (int, error) {
	var max sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM kb_articles WHERE category_id = ? AND status = 'published'`, categoryID,
	).Scan(&max); err != nil {
		return 0, err
	}
	return int(max.Int64) + 1, nil
}

// publishedOrder is a category's published articles, in order.
func publishedOrder(ctx context.Context, tx *sql.Tx, categoryID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM kb_articles WHERE category_id = ? AND status = 'published'
		 ORDER BY position IS NULL, position, created_at, id`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func writeOrder(ctx context.Context, tx *sql.Tx, ids []int64) error {
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE kb_articles SET position = ? WHERE id = ? AND (position IS NULL OR position <> ?)`, i+1, id, i+1,
		); err != nil {
			return err
		}
	}
	return nil
}

// renumberCategory tidies one category to 1..n.
func renumberCategory(ctx context.Context, tx *sql.Tx, categoryID int64) error {
	ids, err := publishedOrder(ctx, tx, categoryID)
	if err != nil {
		return err
	}
	return writeOrder(ctx, tx, ids)
}

// moveToBottomOfCategory is for an article that has just arrived in toCategory
// (a category change on save or on a version restore, or an unarchive): it goes
// to the bottom, and the category it left is closed up.
func moveToBottomOfCategory(ctx context.Context, tx *sql.Tx, articleID, fromCategory, toCategory int64) error {
	// With no number it sorts after everything numbered; then tidy both.
	if _, err := tx.ExecContext(ctx, `UPDATE kb_articles SET position = NULL WHERE id = ?`, articleID); err != nil {
		return err
	}
	if err := renumberCategory(ctx, tx, toCategory); err != nil {
		return err
	}
	if fromCategory != 0 && fromCategory != toCategory {
		return renumberCategory(ctx, tx, fromCategory)
	}
	return nil
}

// RenumberAll tidies every category's published articles to 1..n by
// (position IS NULL, position, created_at, id), and clears the number of
// anything not published. main.go calls it at every start-up, after
// migrations, as D-85 does for the half-typed-word list: it is what makes a
// rollback to 1.3.0 and an upgrade again safe. An article created while
// rolled back has no number and lands at the bottom (gate 7.59); one that was
// moved to another category keeps a stale number and lands somewhere in its
// new category, never twice and never missing.
func RenumberAll(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT category_id FROM kb_articles WHERE status = 'published'`)
	if err != nil {
		return err
	}
	var categories []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		categories = append(categories, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range categories {
		if err := renumberCategory(ctx, tx, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE kb_articles SET position = NULL WHERE status <> 'published' AND position IS NOT NULL`); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveInput says where a moved article goes: before or after another
// article of the same category, or a step up or down.
type MoveInput struct {
	BeforeID, AfterID int64
	Direction         string // "up" or "down"
}

// Move puts an article in a new place in its category (gate 7.55) and
// closes the gap. It changes nothing but positions (gate 7.56).
func (s *ArticleStore) Move(ctx context.Context, articleID int64, in MoveInput) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var categoryID int64
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT category_id, status FROM kb_articles WHERE id = ?`, articleID).Scan(&categoryID, &status); err != nil {
		if err == sql.ErrNoRows {
			return ErrArticleNotInOrder
		}
		return err
	}
	if status != "published" {
		return ErrArticleNotInOrder
	}
	order, err := publishedOrder(ctx, tx, categoryID)
	if err != nil {
		return err
	}
	from := -1
	for i, id := range order {
		if id == articleID {
			from = i
		}
	}
	if from < 0 {
		return ErrArticleNotInOrder
	}
	rest := append(append([]int64(nil), order[:from]...), order[from+1:]...)

	insertAt := -1
	switch {
	case in.BeforeID != 0:
		for i, id := range rest {
			if id == in.BeforeID {
				insertAt = i
			}
		}
	case in.AfterID != 0:
		for i, id := range rest {
			if id == in.AfterID {
				insertAt = i + 1
			}
		}
	case in.Direction == "up":
		insertAt = max(from-1, 0)
	case in.Direction == "down":
		insertAt = min(from+1, len(rest))
	default:
		return ErrArticleNotInOrder
	}
	if insertAt < 0 {
		return ErrArticleNotInOrder
	}
	newOrder := append(append(append([]int64(nil), rest[:insertAt]...), articleID), rest[insertAt:]...)
	if err := writeOrder(ctx, tx, newOrder); err != nil {
		return err
	}
	return tx.Commit()
}

// OrderedArticle is one row of a category's list, with the neighbours the
// move icons need.
type OrderedArticle struct {
	CategoryArticle
	PrevID int64
	NextID int64
}

// MoveUp and MoveDown are the two icon-only buttons of a row (gate 4.11),
// disabled at the top and bottom of the category.
func (a OrderedArticle) MoveUp() app.IconButton {
	return app.NewIconButton(app.IconMoveUp, a.Title, a.PrevID == 0)
}

func (a OrderedArticle) MoveDown() app.IconButton {
	return app.NewIconButton(app.IconMoveDown, a.Title, a.NextID == 0)
}

// WithNeighbours adds each article's neighbours in the list.
func WithNeighbours(list []CategoryArticle) []OrderedArticle {
	out := make([]OrderedArticle, len(list))
	for i, a := range list {
		out[i].CategoryArticle = a
		if i > 0 {
			out[i].PrevID = list[i-1].ID
		}
		if i < len(list)-1 {
			out[i].NextID = list[i+1].ID
		}
	}
	return out
}
