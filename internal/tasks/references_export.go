package tasks

import (
	"context"
	"database/sql"
)

// ArticlesCSV is the export's job-articles.csv (gate 8.20): the job, the
// article, who linked it and when, and whether it was removed. Removed links
// are included, and so are archived articles, whose titles come through the
// finder like any other.
type ArticlesCSV struct {
	DB       *sql.DB
	Articles ArticleFinder
}

// ExportRows returns a header row in plain English, then one row per link in
// the order they were made.
func (c ArticlesCSV) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT t.title, a.article_id, p.name, a.created_at, a.removed_at
		FROM task_articles a
		JOIN tasks t ON t.id = a.task_id
		JOIN people p ON p.id = a.created_by
		ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	type row struct {
		job, who, created string
		article           int64
		removed           *string
	}
	var raw []row
	ids := map[int64]bool{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.job, &r.article, &r.who, &r.created, &r.removed); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, r)
		ids[r.article] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	idList := make([]int64, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	known := map[int64]ArticleRef{}
	if c.Articles != nil && len(idList) > 0 {
		if m := c.Articles.Articles(ctx, idList); m != nil {
			known = m
		}
	}
	out := [][]string{{"Job", "Article", "Linked by", "Linked", "Removed"}}
	for _, r := range raw {
		title := known[r.article].Title
		if title == "" {
			title = missingArticle
		}
		out = append(out, []string{r.job, title, r.who, exportDateTime(r.created), exportDateTimePtr(r.removed)})
	}
	return out, nil
}

// PlacesCSV is the export's job-places.csv (gate 8.20).
type PlacesCSV struct{ DB *sql.DB }

// ExportRows returns a header row in plain English, then one row per place in
// the order they were added, removed ones included.
func (c PlacesCSV) ExportRows(ctx context.Context) ([][]string, error) {
	rows, err := c.DB.QueryContext(ctx, `
		SELECT t.title, l.name, l.path, p.name, l.created_at, l.removed_at
		FROM task_places l
		JOIN tasks t ON t.id = l.task_id
		JOIN people p ON p.id = l.created_by
		ORDER BY l.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{{"Job", "Name", "Folder path", "Added by", "Added", "Removed"}}
	for rows.Next() {
		var job, name, folder, who, created string
		var removed *string
		if err := rows.Scan(&job, &name, &folder, &who, &created, &removed); err != nil {
			return nil, err
		}
		out = append(out, []string{job, name, folder, who, exportDateTime(created), exportDateTimePtr(removed)})
	}
	return out, rows.Err()
}
