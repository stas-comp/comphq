package tasks

import (
	"context"
	"net/http"
	"strings"

	"github.com/stas-comp/comphq/internal/app"
)

// What the Articles and places section needs to be drawn (SPEC B14.2, gates
// 8.10-8.14, 8.19). One partial, "tasks-references", is drawn by the job's own
// page and by the task window alike, the way Linked jobs is.

// refArticleLine is one linked article: it carries the job it is on and whether
// it is drawn in the window, which the line's own form needs.
type refArticleLine struct {
	ArticleLink
	TaskID   int64
	InWindow bool
}

// Remove is the line's icon-only button: gate 4.11's remove icon, so the count
// of icon-only buttons stays at four.
func (l refArticleLine) Remove() app.IconButton {
	return app.NewIconButton(app.IconRemove, "article "+l.Title, false)
}

// refPlaceLine is one COSMOS place.
type refPlaceLine struct {
	Place
	TaskID   int64
	InWindow bool
}

func (l refPlaceLine) Remove() app.IconButton {
	return app.NewIconButton(app.IconRemove, "place "+l.Label, false)
}

// referencesData is the section.
type referencesData struct {
	TaskID   int64
	InWindow bool
	Articles []refArticleLine
	Places   []refPlaceLine
	Notice   string
	// ArticleQuery and ArticleMatches are the no-script picker (?article_q=): the
	// articles that match what was typed, each with a Link button.
	ArticleQuery   string
	ArticleMatches []ArticleRef
	// TypedName and TypedPath are what a refused place had entered.
	TypedName, TypedPath string
}

// buildReferencesData reads a job's articles and places and, for the no-script
// picker, the articles matching query.
func (h *Handlers) buildReferencesData(ctx context.Context, taskID int64, query string, inWindow bool, notice, typedName, typedPath string) (referencesData, error) {
	refs, err := h.tasks.ListReferences(ctx, taskID)
	if err != nil {
		return referencesData{}, err
	}
	data := referencesData{TaskID: taskID, InWindow: inWindow, Notice: notice, ArticleQuery: query, TypedName: typedName, TypedPath: typedPath}
	for _, a := range refs.Articles {
		data.Articles = append(data.Articles, refArticleLine{ArticleLink: a, TaskID: taskID, InWindow: inWindow})
	}
	for _, p := range refs.Places {
		data.Places = append(data.Places, refPlaceLine{Place: p, TaskID: taskID, InWindow: inWindow})
	}
	if strings.TrimSpace(query) != "" {
		data.ArticleMatches, err = h.tasks.SearchArticles(ctx, query, taskID)
		if err != nil {
			return referencesData{}, err
		}
	}
	return data, nil
}

// typedPlace is what a refused "Add a place" form had entered, read back from
// the request so the form comes back with it (gate 8.19).
func typedPlace(r *http.Request) (name, path string) {
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/places") {
		return r.PostFormValue("name"), r.PostFormValue("path")
	}
	return "", ""
}
