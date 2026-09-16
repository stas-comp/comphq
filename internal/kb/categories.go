package kb

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Category is one row of kb_categories, with its published article count
// for the home page's tiles (SPEC gate 1.13).
type Category struct {
	ID           int64
	Name         string
	SortOrder    int
	ArticleCount int
}

// ErrCategoryHasArticles is returned by Delete when the category still has
// articles — published or archived, since archived ones can be restored
// (SPEC gate 1.12).
var ErrCategoryHasArticles = errors.New("this category still has articles")

// ErrEmptyName is returned by Create/Rename for blank input.
var ErrEmptyName = errors.New("name can't be empty")

type CategoryStore struct {
	DB *sql.DB
}

// List returns every category in sort_order, with its published article
// count.
func (s *CategoryStore) List() ([]Category, error) {
	rows, err := s.DB.Query(`
		SELECT c.id, c.name, c.sort_order,
			(SELECT COUNT(*) FROM kb_articles a WHERE a.category_id = c.id AND a.status = 'published')
		FROM kb_categories c
		ORDER BY c.sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.ArticleCount); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// Create adds a category at the end of the sort order.
func (s *CategoryStore) Create(name string) (Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Category{}, ErrEmptyName
	}

	var maxOrder sql.NullInt64
	if err := s.DB.QueryRow(`SELECT MAX(sort_order) FROM kb_categories`).Scan(&maxOrder); err != nil {
		return Category{}, err
	}
	nextOrder := int(maxOrder.Int64) + 1

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.DB.Exec(
		`INSERT INTO kb_categories (name, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name, nextOrder, now, now,
	)
	if err != nil {
		return Category{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Category{}, err
	}
	return Category{ID: id, Name: name, SortOrder: nextOrder}, nil
}

// Rename changes a category's name.
func (s *CategoryStore) Rename(id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	_, err := s.DB.Exec(
		`UPDATE kb_categories SET name = ?, updated_at = ? WHERE id = ?`,
		name, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

// MoveUp swaps a category with the one immediately above it in sort_order.
func (s *CategoryStore) MoveUp(id int64) error {
	return s.swapWithNeighbor(id, true)
}

// MoveDown swaps a category with the one immediately below it in
// sort_order.
func (s *CategoryStore) MoveDown(id int64) error {
	return s.swapWithNeighbor(id, false)
}

func (s *CategoryStore) swapWithNeighbor(id int64, up bool) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var order int64
	if err := tx.QueryRow(`SELECT sort_order FROM kb_categories WHERE id = ?`, id).Scan(&order); err != nil {
		return err
	}

	var neighborID, neighborOrder int64
	var query string
	if up {
		query = `SELECT id, sort_order FROM kb_categories WHERE sort_order < ? ORDER BY sort_order DESC LIMIT 1`
	} else {
		query = `SELECT id, sort_order FROM kb_categories WHERE sort_order > ? ORDER BY sort_order ASC LIMIT 1`
	}
	err = tx.QueryRow(query, order).Scan(&neighborID, &neighborOrder)
	if err == sql.ErrNoRows {
		return nil // already at the end; nothing to do
	}
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`UPDATE kb_categories SET sort_order = ?, updated_at = ? WHERE id = ?`, neighborOrder, now, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE kb_categories SET sort_order = ?, updated_at = ? WHERE id = ?`, order, now, neighborID); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete removes an empty category. If it still has any article
// (published or archived), it refuses and returns the count, so the
// handler can show a plain explanation (SPEC gate 1.12).
func (s *CategoryStore) Delete(id int64) (articleCount int, err error) {
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM kb_articles WHERE category_id = ?`, id).Scan(&articleCount); err != nil {
		return 0, err
	}
	if articleCount > 0 {
		return articleCount, ErrCategoryHasArticles
	}
	_, err = s.DB.Exec(`DELETE FROM kb_categories WHERE id = ?`, id)
	return 0, err
}
