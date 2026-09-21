//go:build integration

package tasks

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// The steps routes over HTTP (SPEC B10.2, gates 5.08, 5.12, 5.13): every one
// is an ordinary form post answering a redirect back to the task page, so
// none of this needs a script.

type stepsHTTP struct {
	t      *testing.T
	client *http.Client
	sqlDB  *sql.DB
	taskID int64
	base   string
}

// newStepsHTTP starts a server, signs Sam in, and gives Sam a job. The
// client does not follow redirects, so a test can see where one goes.
func newStepsHTTP(t *testing.T) (*stepsHTTP, func(name string) *http.Client) {
	t.Helper()
	ts, sqlDB := newTestServer(t)
	newClient := func(name string) *http.Client {
		c := &http.Client{
			Jar:           mustCookieJar(t),
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
		signIn(t, c, ts, name)
		return c
	}
	sam := newClient("Sam")
	resp := postForm(t, sam, ts, "/tasks", url.Values{"title": {"Concert"}})
	resp.Body.Close()
	s := &stepsHTTP{t: t, client: sam, sqlDB: sqlDB, taskID: taskID(t, sqlDB, "Concert"), base: ts.URL}
	return s, newClient
}

func (s *stepsHTTP) path(suffix string) string {
	return "/tasks/" + strconv.FormatInt(s.taskID, 10) + suffix
}

func (s *stepsHTTP) post(client *http.Client, suffix string, form url.Values) (*http.Response, string) {
	s.t.Helper()
	req, err := http.NewRequest(http.MethodPost, s.base+s.path(suffix), strings.NewReader(form.Encode()))
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", s.base)
	resp, err := client.Do(req)
	if err != nil {
		s.t.Fatalf("POST %s: %v", suffix, err)
	}
	return resp, readBody(s.t, resp)
}

func (s *stepsHTTP) get(client *http.Client, suffix string) string {
	s.t.Helper()
	resp, err := client.Get(s.base + s.path(suffix))
	if err != nil {
		s.t.Fatalf("GET %s: %v", suffix, err)
	}
	if resp.StatusCode != http.StatusOK {
		s.t.Fatalf("GET %s = %d, want 200", suffix, resp.StatusCode)
	}
	return readBody(s.t, resp)
}

// add posts a step as client and returns its id.
func (s *stepsHTTP) add(client *http.Client, text string) int64 {
	s.t.Helper()
	resp, _ := s.post(client, "/steps", url.Values{"text": {text}})
	if resp.StatusCode != http.StatusFound {
		s.t.Fatalf("add %q = %d, want a redirect", text, resp.StatusCode)
	}
	var id int64
	if err := s.sqlDB.QueryRow(`SELECT id FROM task_checklist_items WHERE task_id = ? AND text = ? ORDER BY id DESC LIMIT 1`, s.taskID, text).Scan(&id); err != nil {
		s.t.Fatalf("find step %q: %v", text, err)
	}
	return id
}

func (s *stepsHTTP) stepPath(id int64, suffix string) string {
	return "/steps/" + strconv.FormatInt(id, 10) + suffix
}

func (s *stepsHTTP) count(query string, args ...any) int {
	s.t.Helper()
	var n int
	if err := s.sqlDB.QueryRow(query, args...).Scan(&n); err != nil {
		s.t.Fatal(err)
	}
	return n
}

func wantRedirect(t *testing.T, resp *http.Response, wantLocation string) {
	t.Helper()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != wantLocation {
		t.Errorf("redirected to %q, want %q", got, wantLocation)
	}
}

func wantIn(t *testing.T, body string, snippets ...string) {
	t.Helper()
	for _, sn := range snippets {
		if !strings.Contains(body, sn) {
			t.Errorf("page is missing %q", sn)
		}
	}
}

func wantNotIn(t *testing.T, body string, snippets ...string) {
	t.Helper()
	for _, sn := range snippets {
		if strings.Contains(body, sn) {
			t.Errorf("page should not contain %q", sn)
		}
	}
}

// Gate 5.01: a job with no steps shows the Add a step line and nothing else.
func TestStepsHTTPEmptyJobShowsOnlyTheAddLine(t *testing.T) {
	s, _ := newStepsHTTP(t)
	page := s.get(s.client, "")
	wantIn(t, page, `id="steps"`, `>Steps</h2>`, `placeholder="Add a step"`)
	wantNotIn(t, page, `class="steps-list"`, `steps-count`, `<progress`)
}

func TestStepsHTTPAddRedirectsBackAndTheStepAppears(t *testing.T) {
	s, _ := newStepsHTTP(t)
	resp, _ := s.post(s.client, "/steps", url.Values{"text": {"Print exam papers"}})
	wantRedirect(t, resp, s.path("#steps"))

	page := s.get(s.client, "")
	wantIn(t, page, `Print exam papers`, `0 of 1 done`, `<progress`)
}

// Gate 5.13: a refusal keeps what was typed, says why, and changes nothing.
func TestStepsHTTPRefusedAddKeepsTheTypedWordsAndSaysWhy(t *testing.T) {
	s, _ := newStepsHTTP(t)

	resp, body := s.post(s.client, "/steps", url.Values{"text": {"   "}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("blank add = %d, want the page again (200)", resp.StatusCode)
	}
	wantIn(t, body, `role="alert"`, `Type what the step is first.`)

	long := strings.Repeat("a", MaxStepChars+1)
	_, body = s.post(s.client, "/steps", url.Values{"text": {long}})
	wantIn(t, body, `A step can be up to 200 characters.`, `value="`+long+`"`)

	for i := 0; i < MaxSteps; i++ {
		s.add(s.client, "Step "+strconv.Itoa(i))
	}
	_, body = s.post(s.client, "/steps", url.Values{"text": {"One more"}})
	wantIn(t, body, `A job can have up to 50 steps.`, `value="One more"`)

	if n := s.count(`SELECT COUNT(*) FROM task_checklist_items WHERE task_id = ?`, s.taskID); n != MaxSteps {
		t.Errorf("%d steps stored after three refusals, want %d", n, MaxSteps)
	}
}

// Gates 5.03 and 5.12: a tick shows who and when; ticking what somebody
// else ticked is not an error and keeps the first person.
func TestStepsHTTPTickShowsWhoAndSecondTickKeepsTheFirstPerson(t *testing.T) {
	s, newClient := newStepsHTTP(t)
	alex := newClient("Alex")
	id := s.add(s.client, "Book the hall")

	resp, _ := s.post(s.client, s.stepPath(id, "/tick"), url.Values{"done": {"1"}})
	wantRedirect(t, resp, s.path("#step-"+strconv.FormatInt(id, 10)))
	page := s.get(s.client, "")
	wantIn(t, page, `checked title="Untick Book the hall"`, `All 1 done`, `Sam, `)

	resp, body := s.post(alex, s.stepPath(id, "/tick"), url.Values{"done": {"1"}})
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("second tick = %d, want a plain redirect (%q)", resp.StatusCode, body)
	}
	page = s.get(alex, "")
	wantIn(t, page, `Sam, `)
	wantNotIn(t, page, `Alex, `)

	// Unticking: the form carries the state it asks for.
	s.post(alex, s.stepPath(id, "/tick"), url.Values{"done": {"0"}})
	page = s.get(s.client, "")
	wantIn(t, page, `title="Tick Book the hall"`, `0 of 1 done`)
	wantNotIn(t, page, `Sam, `)

	if resp, _ := s.post(s.client, s.stepPath(id, "/tick"), url.Values{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("a tick form with no state = %d, want 400", resp.StatusCode)
	}
}

// D-71: ticking is never recorded in History.
func TestStepsHTTPTicksAreNotInHistory(t *testing.T) {
	s, _ := newStepsHTTP(t)
	var ids []int64
	for i := 0; i < 10; i++ {
		ids = append(ids, s.add(s.client, "Step "+strconv.Itoa(i)))
	}
	before := s.count(`SELECT COUNT(*) FROM task_activity WHERE task_id = ?`, s.taskID)
	for _, id := range ids {
		s.post(s.client, s.stepPath(id, "/tick"), url.Values{"done": {"1"}})
	}
	for _, id := range ids[:5] {
		s.post(s.client, s.stepPath(id, "/tick"), url.Values{"done": {"0"}})
	}
	if after := s.count(`SELECT COUNT(*) FROM task_activity WHERE task_id = ?`, s.taskID); after != before {
		t.Errorf("History has %d rows after ticking ten steps, was %d: ticks must not be recorded", after, before)
	}
	wantIn(t, s.get(s.client, ""), `5 of 10 done`)
}

// Gate 5.05, without a script: the rename form is a page view, and a refused
// rename comes back in that form with the typed words.
func TestStepsHTTPRenameFormAndRefusal(t *testing.T) {
	s, _ := newStepsHTTP(t)
	id := s.add(s.client, "Book the hall")
	sid := strconv.FormatInt(id, 10)

	page := s.get(s.client, "?rename_step="+sid)
	wantIn(t, page, `id="step-text-`+sid+`"`, `value="Book the hall"`)
	wantNotIn(t, s.get(s.client, ""), `id="step-text-`)

	resp, _ := s.post(s.client, s.stepPath(id, ""), url.Values{"text": {"Book the big hall"}})
	wantRedirect(t, resp, s.path("#step-"+sid))
	wantIn(t, s.get(s.client, ""), `Book the big hall`)

	long := strings.Repeat("b", MaxStepChars+1)
	_, body := s.post(s.client, s.stepPath(id, ""), url.Values{"text": {long}})
	wantIn(t, body, `A step can be up to 200 characters.`, `value="`+long+`"`)
	wantIn(t, s.get(s.client, ""), `Book the big hall`)
}

// Gates 5.06, D-74: removing offers Undo straight away, which puts the step
// back where it was; the row is never deleted.
func TestStepsHTTPRemoveOffersUndoAndUndoPutsItBack(t *testing.T) {
	s, _ := newStepsHTTP(t)
	s.add(s.client, "One")
	two := s.add(s.client, "Two")
	s.add(s.client, "Three")
	sid := strconv.FormatInt(two, 10)

	resp, _ := s.post(s.client, s.stepPath(two, "/remove"), url.Values{})
	wantRedirect(t, resp, s.path("?undo_step="+sid+"#steps"))

	page := s.get(s.client, "?undo_step="+sid)
	wantIn(t, page, `steps-undo`, `Removed “Two”`, `/steps/`+sid+`/restore`)
	if strings.Contains(page, `id="step-`+sid+`"`) {
		t.Error("the removed step is still in the list")
	}
	wantNotIn(t, s.get(s.client, ""), `steps-undo`) // the next visit has no Undo
	if n := s.count(`SELECT COUNT(*) FROM task_checklist_items WHERE id = ?`, two); n != 1 {
		t.Errorf("removed step's row count = %d, want 1 (never deleted)", n)
	}

	resp, _ = s.post(s.client, s.stepPath(two, "/restore"), url.Values{})
	wantRedirect(t, resp, s.path("#step-"+sid))
	page = s.get(s.client, "")
	i1, i2, i3 := strings.Index(page, ">One<"), strings.Index(page, ">Two<"), strings.Index(page, ">Three<")
	if !(i1 >= 0 && i1 < i2 && i2 < i3) {
		t.Errorf("after Undo the order is wrong (positions %d, %d, %d)", i1, i2, i3)
	}

	// Someone crafting an Undo for a step that isn't removed gets no Undo.
	wantNotIn(t, s.get(s.client, "?undo_step="+sid), `steps-undo`)
}

// Gate 5.07, without a script.
func TestStepsHTTPMoveUpAndDown(t *testing.T) {
	s, _ := newStepsHTTP(t)
	s.add(s.client, "One")
	two := s.add(s.client, "Two")
	s.add(s.client, "Three")

	resp, _ := s.post(s.client, s.stepPath(two, "/move"), url.Values{"direction": {"up"}})
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("move = %d, want a redirect", resp.StatusCode)
	}
	page := s.get(s.client, "")
	if a, b := strings.Index(page, ">Two<"), strings.Index(page, ">One<"); !(a >= 0 && a < b) {
		t.Errorf("Two should now come before One (%d, %d)", a, b)
	}
	wantIn(t, page, `title="Move Two up"`, `aria-label="Move Two down"`)

	if resp, _ := s.post(s.client, s.stepPath(two, "/move"), url.Values{"direction": {"sideways"}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("a bad direction = %d, want 400", resp.StatusCode)
	}

	// Dragging sends an absolute place.
	s.post(s.client, s.stepPath(two, "/move"), url.Values{"position": {"3"}})
	page = s.get(s.client, "")
	if a, b := strings.Index(page, ">One<"), strings.Index(page, ">Two<"); !(a >= 0 && a < b) {
		t.Errorf("after moving Two to place 3, One should precede it (%d, %d)", a, b)
	}
}

// Gate 5.12: acting on a step somebody else removed says so plainly, in the
// page, and changes nothing.
func TestStepsHTTPActingOnAStepSomebodyRemovedSaysSo(t *testing.T) {
	s, newClient := newStepsHTTP(t)
	alex := newClient("Alex")
	id := s.add(s.client, "Book the hall")
	s.post(alex, s.stepPath(id, "/remove"), url.Values{})

	for name, act := range map[string]struct {
		suffix string
		form   url.Values
	}{
		"tick":   {"/tick", url.Values{"done": {"1"}}},
		"rename": {"", url.Values{"text": {"New words"}}},
		"remove": {"/remove", url.Values{}},
		"move":   {"/move", url.Values{"direction": {"up"}}},
	} {
		resp, body := s.post(s.client, s.stepPath(id, act.suffix), act.form)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s on a removed step = %d, want the page again (200), not a failure", name, resp.StatusCode)
		}
		wantIn(t, body, `Somebody else removed that step.`)
	}
	if n := s.count(`SELECT COUNT(*) FROM task_checklist_items WHERE id = ? AND text = 'Book the hall' AND done_at IS NULL`, id); n != 1 {
		t.Error("a refused action changed the removed step")
	}

	// A step id that isn't on this job at all reads plainly too.
	_, body := s.post(s.client, s.stepPath(999999, "/tick"), url.Values{"done": {"1"}})
	wantIn(t, body, `That step isn't there any more.`)
}

func TestStepsHTTPUnknownJobIs404(t *testing.T) {
	s, _ := newStepsHTTP(t)
	s.taskID = 424242
	resp, _ := s.post(s.client, "/steps", url.Values{"text": {"Orphan"}})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("add to a missing job = %d, want 404", resp.StatusCode)
	}
}

// The window's way in: fragment=1 answers with just the list, 200 when it
// worked and 422 with the notice when it was refused (B10.2).
func TestStepsHTTPFragmentAnswersTheListItself(t *testing.T) {
	s, _ := newStepsHTTP(t)

	resp, body := s.post(s.client, "/steps", url.Values{"text": {"Book the hall"}, "fragment": {"1"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fragment add = %d, want 200", resp.StatusCode)
	}
	wantIn(t, body, `class="task-steps"`, `Book the hall`, `name="fragment" value="1"`)
	wantNotIn(t, body, `<html`, `<nav`, `value="Book the hall"`) // the box comes back empty

	resp, body = s.post(s.client, "/steps", url.Values{"text": {""}, "fragment": {"1"}})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("refused fragment add = %d, want 422", resp.StatusCode)
	}
	wantIn(t, body, `Type what the step is first.`)

	var id int64
	s.sqlDB.QueryRow(`SELECT id FROM task_checklist_items WHERE task_id = ?`, s.taskID).Scan(&id)
	resp, body = s.post(s.client, s.stepPath(id, "/remove"), url.Values{"fragment": {"1"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fragment remove = %d, want 200", resp.StatusCode)
	}
	wantIn(t, body, `steps-undo`, `Removed “Book the hall”`)
}

// Every write bumps the change counter the other computers poll (gate 5.12).
func TestStepsHTTPWritesBumpTheChangeCounter(t *testing.T) {
	s, _ := newStepsHTTP(t)
	version := func() string {
		resp, err := s.client.Get(s.base + "/tasks/version")
		if err != nil {
			t.Fatal(err)
		}
		return readBody(t, resp)
	}
	v0 := version()
	id := s.add(s.client, "One")
	v1 := version()
	s.post(s.client, s.stepPath(id, "/tick"), url.Values{"done": {"1"}})
	v2 := version()
	if v0 == v1 || v1 == v2 {
		t.Errorf("counter did not move on add and tick: %s, %s, %s", v0, v1, v2)
	}
}
