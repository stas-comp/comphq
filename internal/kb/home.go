package kb

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

// RecentArticle is one row of the KB home page's "recently updated" list
// (SPEC gate 1.13). Empty until P1-19 starts writing kb_articles rows.
type RecentArticle struct {
	ID            int64
	Title         string
	UpdatedAt     string // pre-formatted, SPEC A3: "Sat 19 Sep 2026, 14:30"
	UpdatedByName string
	CategoryName  string
}

func recentArticles(sqlDB *sql.DB, limit int) ([]RecentArticle, error) {
	rows, err := sqlDB.Query(`
		SELECT a.id, a.title, a.updated_at, COALESCE(p.name, ''), c.name
		FROM kb_articles a
		JOIN kb_categories c ON c.id = a.category_id
		LEFT JOIN people p ON p.id = a.updated_by
		WHERE a.status = 'published'
		ORDER BY a.updated_at DESC, a.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RecentArticle
	for rows.Next() {
		var r RecentArticle
		var updatedAt string
		if err := rows.Scan(&r.ID, &r.Title, &updatedAt, &r.UpdatedByName, &r.CategoryName); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			r.UpdatedAt = format.DateTime(t)
		} else {
			r.UpdatedAt = updatedAt
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// homeTile is one category's tile: its name and count, and its first three
// articles in the category's own order (gate 7.70).
type homeTile struct {
	Category
	First []CategoryArticle
}

type homePageData struct {
	Tiles  []homeTile
	Recent []RecentArticle
}

func (h *Handlers) handleHome(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categories.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	first, err := h.articles.FirstByCategory(3)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tiles := make([]homeTile, len(categories))
	for i, c := range categories {
		tiles[i] = homeTile{Category: c, First: first[c.ID]}
	}
	recent, err := recentArticles(h.srv.DB, 10)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-home.html", "Knowledge Base", homePageData{
		Tiles:  tiles,
		Recent: recent,
	})
}
