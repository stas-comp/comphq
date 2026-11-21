package kb

import (
	"context"
	"sort"

	"github.com/stas-comp/comphq/internal/app/wordmatch"
)

// What the Knowledge Base shares with Tasks (SPEC B14.2, D-105), without
// either importing the other (SPEC B2): cmd/comphq wires one section's
// implementation into the other's interface at start-up.

// JobRef is a job that links an article, as the article's page lists it.
type JobRef struct {
	ID         int64
	Title      string
	StageLabel string
	Finished   bool
}

// JobsUsing is what the article page needs from Tasks: the jobs linking an
// article, leaving out removed jobs and removed links. A nil implementation
// means no jobs.
type JobsUsing interface {
	JobsForArticle(ctx context.Context, articleID int64) []JobRef
}

// SectionOption changes how Section is wired.
type SectionOption func(*Handlers)

// WithJobsUsing gives the article page its "Jobs using this article" panel.
func WithJobsUsing(j JobsUsing) SectionOption {
	return func(h *Handlers) { h.jobs = j }
}

// ArticleRef is what Tasks needs to know about an article.
type ArticleRef struct {
	ID       int64
	Title    string
	Category string
	Archived bool
}

// FindByTitle returns published articles whose title matches q, best match
// first (gate 8.11): every word of q must appear in the title, the last one
// possibly half-typed, with the same Unicode-folding match as job search
// (D-94). Titles that start with the first word come before the others, then
// alphabetical. At most limit are returned. A query with fewer than two
// letters finds nothing.
func (s *ArticleStore) FindByTitle(ctx context.Context, q string, limit int) []ArticleRef {
	query := wordmatch.Parse(q)
	if query.Empty() || limit <= 0 {
		return nil
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT a.id, a.title, c.name
		FROM kb_articles a JOIN kb_categories c ON c.id = a.category_id
		WHERE a.status = 'published'`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	type hit struct {
		ref    ArticleRef
		prefix bool
		lower  string
	}
	var hits []hit
	for rows.Next() {
		var r ArticleRef
		if err := rows.Scan(&r.ID, &r.Title, &r.Category); err != nil {
			return nil
		}
		if !query.MatchesAll(r.Title) {
			continue
		}
		hits = append(hits, hit{ref: r, prefix: query.StartsWithFirst(r.Title), lower: string(wordmatch.Lower(r.Title))})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].prefix != hits[j].prefix {
			return hits[i].prefix
		}
		if hits[i].lower != hits[j].lower {
			return hits[i].lower < hits[j].lower
		}
		return hits[i].ref.ID < hits[j].ref.ID
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]ArticleRef, len(hits))
	for i, h := range hits {
		out[i] = h.ref
	}
	return out
}

// Articles returns the articles with these ids, archived ones included, for
// the titles, categories and the Archived mark on a job.
func (s *ArticleStore) Articles(ctx context.Context, ids []int64) map[int64]ArticleRef {
	out := make(map[int64]ArticleRef, len(ids))
	for _, id := range ids {
		var r ArticleRef
		var status string
		err := s.DB.QueryRowContext(ctx, `
			SELECT a.id, a.title, c.name, a.status
			FROM kb_articles a JOIN kb_categories c ON c.id = a.category_id
			WHERE a.id = ?`, id).Scan(&r.ID, &r.Title, &r.Category, &status)
		if err != nil {
			continue
		}
		r.Archived = status == "archived"
		out[id] = r
	}
	return out
}
