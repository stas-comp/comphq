//go:build integration

package questions

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// Editing, removing, restoring and searching through the real pages (gates 8.75,
// 8.76, 8.78): every one is an ordinary form or link, so it works with no script.
func TestEditRemoveRestoreAndSearchPages(t *testing.T) {
	ts, c := newServer(t)
	post(t, c, ts, "/questions", url.Values{"text": {"Where is the stamp book?"}, "details": {"Ask the front desk"}})
	post(t, c, ts, "/questions/1/answers", url.Values{"text": {"In the top drawer"}})

	// The edit forms open from plain links.
	if _, body := get(t, c, ts, "/questions/1?edit=question"); !strings.Contains(body, `name="text"`) || !strings.Contains(body, "Save") || !strings.Contains(body, "Cancel") {
		t.Fatalf("edit form missing: %.400s", body)
	}
	if _, body := get(t, c, ts, "/questions/1?edit_answer=1"); !strings.Contains(body, "Edit this answer") {
		t.Fatalf("answer edit form missing: %.400s", body)
	}

	// A refused edit keeps what was typed.
	_, body := post(t, c, ts, "/questions/1", url.Values{"text": {"   "}, "details": {"keep me"}})
	if !strings.Contains(body, "keep me") || !strings.Contains(body, `role="alert"`) {
		t.Fatalf("refused edit should say why and keep the text: %.500s", body)
	}

	// A good edit shows "edited by".
	_, body = post(t, c, ts, "/questions/1", url.Values{"text": {"Where is the new stamp book?"}, "details": {"Ask the front desk"}})
	if !strings.Contains(body, "Where is the new stamp book?") || !strings.Contains(body, "edited by Sam") {
		t.Fatalf("edit not shown: %.500s", body)
	}
	_, body = post(t, c, ts, "/questions/1/answers/1", url.Values{"text": {"Second drawer"}})
	if !strings.Contains(body, "Second drawer") || strings.Contains(body, "In the top drawer") {
		t.Fatalf("answer edit not shown: %.500s", body)
	}

	// Removing an answer comes back with the Undo message, and Undo brings it back.
	_, body = post(t, c, ts, "/questions/1/answers/1/remove", nil)
	if strings.Contains(body, "Second drawer") || !strings.Contains(body, "Undo") || !strings.Contains(body, "Answer removed") {
		t.Fatalf("remove answer should show Undo: %.600s", body)
	}
	_, body = post(t, c, ts, "/questions/1/answers/1/restore", nil)
	if !strings.Contains(body, "Second drawer") || strings.Contains(body, "Answer removed") {
		t.Fatalf("Undo should bring the answer back: %.600s", body)
	}
	// The Undo message is not offered for an answer that is not removed.
	if _, body := get(t, c, ts, "/questions/1?removed_answer=1"); strings.Contains(body, "Answer removed") {
		t.Fatalf("no Undo for a live answer")
	}

	// Search: the page and the JSON find the question by its words, marked.
	_, body = get(t, c, ts, "/questions/search?q=stamp")
	if !strings.Contains(body, "<mark>stamp</mark>") || !strings.Contains(body, "Search articles instead") || !strings.Contains(body, "Search jobs instead") {
		t.Fatalf("search page: %.800s", body)
	}
	_, raw := get(t, c, ts, "/questions/search.json?q=drawer")
	var out struct {
		Results []struct {
			URL     string `json:"url"`
			Passage string `json:"passage"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || len(out.Results) != 1 || out.Results[0].URL != "/questions/1" || !strings.Contains(out.Results[0].Passage, "<mark>drawer</mark>") {
		t.Fatalf("search json = %s (%v)", raw, err)
	}

	// Removing the question takes it off the Questions page and out of search, and the
	// Removed questions page restores it.
	code, _ := post(t, c, ts, "/questions/1/remove", nil)
	if code != 200 {
		t.Fatalf("remove question = %d", code)
	}
	if _, body := get(t, c, ts, "/questions"); strings.Contains(body, "new stamp book") {
		t.Fatalf("removed question still listed")
	}
	if _, raw := get(t, c, ts, "/questions/search.json?q=stamp"); strings.Contains(raw, `"/questions/1"`) {
		t.Fatalf("removed question still found: %s", raw)
	}
	if _, body := get(t, c, ts, "/questions/removed"); !strings.Contains(body, "new stamp book") || !strings.Contains(body, "Restore question") {
		t.Fatalf("Removed questions: %.600s", body)
	}
	post(t, c, ts, "/questions/1/restore", nil)
	if _, body := get(t, c, ts, "/questions"); !strings.Contains(body, "new stamp book") {
		t.Fatalf("restored question not listed")
	}
}
