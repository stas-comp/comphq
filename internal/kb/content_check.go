package kb

import (
	"context"

	kbhtml "github.com/stas-comp/comphq/internal/kb/html"
)

// ContentCheckArticle is one article Settings > Content check should
// list (SPEC gates 1.36, 1.49).
type ContentCheckArticle struct {
	ID    int64
	Title string
}

// ListContentCheck returns every published or archived article whose
// body still has something that couldn't be brought in — an image that
// failed to fetch, or a Word-import placeholder (SPEC B7: "until the
// boxes are removed, Settings > Content check lists the article").
func (s *ArticleStore) ListContentCheck(ctx context.Context) ([]ContentCheckArticle, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, title, body_html FROM kb_articles WHERE status IN ('published', 'archived') ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ContentCheckArticle
	for rows.Next() {
		var id int64
		var title, body string
		if err := rows.Scan(&id, &title, &body); err != nil {
			return nil, err
		}
		if kbhtml.HasUnresolvedContent(body) {
			out = append(out, ContentCheckArticle{ID: id, Title: title})
		}
	}
	return out, rows.Err()
}
