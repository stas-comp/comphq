package kb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/stas-comp/comphq/internal/kb/search"
)

// searchResultView is one row of a search results list, ready to render
// or encode as JSON (SPEC gate 1.27: title, category, and the matching
// passage highlighted).
type searchResultView struct {
	ArticleID int64         `json:"articleId"`
	Title     string        `json:"title"`
	Category  string        `json:"category"`
	Snippet   template.HTML `json:"snippet"`
	URL       string        `json:"url"`
}

// searchResults runs the query and looks up each result's live title and
// category (SPEC gate 1.05: a category rename shows up immediately, the
// same as everywhere else that joins to it rather than storing a copy).
func (h *Handlers) searchResults(query string) ([]searchResultView, error) {
	raw, err := search.Search(h.srv.DB, query, 20)
	if err != nil {
		return nil, err
	}
	out := make([]searchResultView, 0, len(raw))
	for _, r := range raw {
		var title, category string
		err := h.srv.DB.QueryRow(`
			SELECT a.title, c.name FROM kb_articles a JOIN kb_categories c ON c.id = a.category_id WHERE a.id = ?
		`, r.ArticleID).Scan(&title, &category)
		if err != nil {
			continue
		}
		out = append(out, searchResultView{
			ArticleID: r.ArticleID,
			Title:     title,
			Category:  category,
			Snippet:   template.HTML(r.Snippet),
			// SPEC B4: "/kb/articles/<id>?q=<query>#b-<block>".
			URL: fmt.Sprintf("/kb/articles/%d?q=%s#b-%d", r.ArticleID, url.QueryEscape(query), r.BlockID),
		})
	}
	return out, nil
}

// handleSearchJSON serves the live panel (SPEC gate 1.26): up to 20
// results, most relevant first, for the 250ms-debounced fetch search.js
// issues after the box stops changing.
func (h *Handlers) handleSearchJSON(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := h.searchResults(query)
	if err != nil {
		log.Printf("search.json %q: %v", query, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"query": query, "results": results})
}

type searchPageData struct {
	Query   string
	Results []searchResultView
}

// handleSearchResultsPage serves the full results page Enter goes to
// (SPEC B4: "the same content" as the live panel).
func (h *Handlers) handleSearchResultsPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := h.searchResults(query)
	if err != nil {
		log.Printf("search results page %q: %v", query, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "kb-search-results.html", "Search", searchPageData{
		Query: query, Results: results,
	})
}
