//go:build integration

package tasks

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/db"
)

func mustCookieJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return jar
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"app", "people", "tasks"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatalf("LoadMigrations(%s): %v", section, err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatalf("RunMigrations(%s): %v", section, err)
		}
	}

	srv, err := app.NewServer(sqlDB, "test", t.TempDir(), false)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	srv.Registry().Add(Section(srv))

	ts := httptest.NewServer(srv.Routes())
	t.Cleanup(ts.Close)
	return ts, sqlDB
}

func postForm(t *testing.T, client *http.Client, ts *httptest.Server, path string, form url.Values) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func signIn(t *testing.T, client *http.Client, ts *httptest.Server, name string) {
	t.Helper()
	resp := postForm(t, client, ts, "/who/add", url.Values{"name": {name}, "next": {"/"}})
	resp.Body.Close()
}

func personID(t *testing.T, sqlDB *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := sqlDB.QueryRow(`SELECT id FROM people WHERE name = ?`, name).Scan(&id); err != nil {
		t.Fatalf("find person %q: %v", name, err)
	}
	return id
}

// TestCreateTaskHTTPAppearsOnBoard covers SPEC gate 2.02's title-only
// path over HTTP.
func TestCreateTaskHTTPAppearsOnBoard(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/tasks", url.Values{"title": {"Fix the printer"}})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect to the board)", resp.StatusCode)
	}
	if !strings.Contains(body, "Fix the printer") {
		t.Errorf("board page missing the new task; got:\n%s", body)
	}
}

// TestCreateTaskHTTPBlankTitleShowsMessage covers the empty-title refusal
// path.
func TestCreateTaskHTTPBlankTitleShowsMessage(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/tasks", url.Values{"title": {"   "}})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (refused with a message, not an error page)", resp.StatusCode)
	}
	if !strings.Contains(body, "Please type a title.") {
		t.Errorf("response body missing the refusal message; got:\n%s", body)
	}
}

// TestCreateTaskHTTPWithStageDueDateAndPeople covers gate 2.03's card
// content (people, due date) end to end over HTTP.
func TestCreateTaskHTTPWithStageDueDateAndPeople(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")
	signIn(t, client, ts, "Alex")
	sam := personID(t, sqlDB, "Sam")
	alex := personID(t, sqlDB, "Alex")

	resp := postForm(t, client, ts, "/tasks", url.Values{
		"title":     {"Team task"},
		"stage":     {StageTodo},
		"due_date":  {"2020-01-01"},
		"person_id": {itoa(sam), itoa(alex)},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "Team task") {
		t.Errorf("board missing the new task; got:\n%s", body)
	}
	if !strings.Contains(body, "2020-01-01") {
		t.Errorf("board missing the due date; got:\n%s", body)
	}
	if !strings.Contains(body, "OVERDUE") {
		t.Errorf("board missing the OVERDUE stamp for a task due in 2020; got:\n%s", body)
	}
	if !strings.Contains(body, "Sam") || !strings.Contains(body, "Alex") {
		t.Errorf("board missing both assignees' names (as avatar title attributes); got:\n%s", body)
	}
}

// TestTasksRedirectsToBoard covers PLAN.md P2-01's note that GET /tasks
// temporarily redirects to the board until My jobs exists (P2-10).
func TestTasksRedirectsToBoard(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{
		Jar:           mustCookieJar(t),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	signIn(t, client, ts, "Sam")

	resp, err := client.Get(ts.URL + "/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	if loc := resp.Header.Get("Location"); loc != "/tasks/board" {
		t.Errorf("Location = %q, want /tasks/board", loc)
	}
}

func taskID(t *testing.T, sqlDB *sql.DB, title string) int64 {
	t.Helper()
	var id int64
	if err := sqlDB.QueryRow(`SELECT id FROM tasks WHERE title = ?`, title).Scan(&id); err != nil {
		t.Fatalf("find task %q: %v", title, err)
	}
	return id
}

// TestMoveTaskHTTPButtonReorders covers gate 2.04's button path over
// HTTP: Move up/down send the neighbour id a real board render would
// compute.
func TestMoveTaskHTTPButtonReorders(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	for _, title := range []string{"A", "B", "C"} {
		postForm(t, client, ts, "/tasks", url.Values{"title": {title}, "stage": {StageTodo}}).Body.Close()
	}
	b, c := taskID(t, sqlDB, "B"), taskID(t, sqlDB, "C")

	// [A, B, C], move C up (before its current upstairs neighbour, B) -> [A, C, B]
	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/move", c), url.Values{
		"stage": {StageTodo}, "before_id": {itoa(b)},
	})
	resp.Body.Close()

	var gotOrder []string
	rows, err := sqlDB.Query(`SELECT title FROM tasks WHERE stage = ? ORDER BY position`, StageTodo)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatal(err)
		}
		gotOrder = append(gotOrder, title)
	}
	want := []string{"A", "C", "B"}
	if len(gotOrder) != len(want) {
		t.Fatalf("order = %v, want %v", gotOrder, want)
	}
	for i := range want {
		if gotOrder[i] != want[i] {
			t.Fatalf("order = %v, want %v", gotOrder, want)
		}
	}
}

// TestMoveTaskHTTPToBottomOfDifferentStage covers gate 2.05: a card
// moved into a different column by button (here, "Move to…") lands at
// the bottom of that column.
func TestMoveTaskHTTPToBottomOfDifferentStage(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Existing"}, "stage": {StageTodo}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Moving"}, "stage": {StageIdea}}).Body.Close()
	moving := taskID(t, sqlDB, "Moving")

	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/move", moving), url.Values{
		"stage": {StageTodo}, "to_bottom": {"1"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var stage string
	var position int
	if err := sqlDB.QueryRow(`SELECT stage, position FROM tasks WHERE id = ?`, moving).Scan(&stage, &position); err != nil {
		t.Fatal(err)
	}
	if stage != StageTodo {
		t.Errorf("stage = %q, want %q", stage, StageTodo)
	}
	if position != 2 {
		t.Errorf("position = %d, want 2 (the bottom, after Existing)", position)
	}
	if !strings.Contains(body, "Moving") {
		t.Errorf("board response missing the moved task; got:\n%s", body)
	}
}

// TestMoveTaskHTTPMissingNeighborReRendersBoard covers the "board
// changed under us" race path: rather than an error page, the current
// board is shown.
func TestMoveTaskHTTPMissingNeighborReRendersBoard(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Task"}, "stage": {StageTodo}}).Body.Close()
	task := taskID(t, sqlDB, "Task")

	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/move", task), url.Values{
		"stage": {StageTodo}, "before_id": {"999999"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered board, not an error page)", resp.StatusCode)
	}
	if !strings.Contains(body, "Task") {
		t.Errorf("board response missing the existing task; got:\n%s", body)
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
