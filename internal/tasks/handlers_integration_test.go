//go:build integration

package tasks

import (
	"database/sql"
	"encoding/json"
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
	if !strings.Contains(body, "Wed 1 Jan 2020") { // the card's short date (D-61b)
		t.Errorf("board missing the due date; got:\n%s", body)
	}
	if !strings.Contains(body, "OVERDUE") {
		t.Errorf("board missing the OVERDUE stamp for a task due in 2020; got:\n%s", body)
	}
	if !strings.Contains(body, "Sam") || !strings.Contains(body, "Alex") {
		t.Errorf("board missing both assignees' names (as avatar title attributes); got:\n%s", body)
	}
}

// TestMyJobsHTTPShowsOwnLaneAndUpForGrabs covers gate 2.32/2.33's My
// jobs home screen end to end over HTTP: it renders directly (no
// redirect), showing the signed-in person's own lane next to jobs and
// ideas nobody is on, but not another person's job or a finished one.
func TestMyJobsHTTPShowsOwnLaneAndUpForGrabs(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{
		Jar:           mustCookieJar(t),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	signIn(t, client, ts, "Sam")
	signIn(t, client, ts, "Alex")
	sam := personID(t, sqlDB, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Mine"}, "stage": {StageTodo}, "person_id": {itoa(sam)}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Alex's"}, "stage": {StageTodo}, "person_id": {itoa(personID(t, sqlDB, "Alex"))}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Grabbable"}, "stage": {StageTodo}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Someday"}, "stage": {StageIdea}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Finished already"}, "stage": {StageDone}}).Body.Close()

	// Re-select Sam: signing in Alex above replaced the session cookie.
	postForm(t, client, ts, "/who/select", url.Values{"person_id": {itoa(sam)}, "next": {"/"}}).Body.Close()

	resp, err := client.Get(ts.URL + "/tasks")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (My jobs renders directly, no redirect)", resp.StatusCode)
	}
	if !strings.Contains(body, "Mine") {
		t.Errorf("My jobs missing the signed-in person's own task; got:\n%s", body)
	}
	if !strings.Contains(body, "Grabbable") || !strings.Contains(body, "Someday") {
		t.Errorf("Up for grabs missing the unassigned task or idea; got:\n%s", body)
	}
	if strings.Contains(body, "Alex's") {
		t.Errorf("My jobs shows a job belonging only to another person; got:\n%s", body)
	}
	if strings.Contains(body, "Finished already") {
		t.Errorf("My jobs shows a finished task; got:\n%s", body)
	}
}

// TestTakeTaskHTTPUnassignedTodoKeepsPosition covers gate 2.34's Take
// it button path (no drop position) end to end over HTTP.
func TestTakeTaskHTTPUnassignedTodoKeepsPosition(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Grabbable"}, "stage": {StageTodo}}).Body.Close()
	task := taskID(t, sqlDB, "Grabbable")

	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/take", task), nil)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect to My jobs)", resp.StatusCode)
	}
	if !strings.Contains(body, "Grabbable") {
		t.Errorf("My jobs missing the taken task; got:\n%s", body)
	}

	var assigneeCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_assignees WHERE task_id = ?`, task).Scan(&assigneeCount); err != nil {
		t.Fatal(err)
	}
	if assigneeCount != 1 {
		t.Errorf("task_assignees rows = %d, want 1", assigneeCount)
	}
}

// TestTakeTaskHTTPIdeaMovesToTodo covers gate 2.35's idea-to-todo Take
// path end to end over HTTP.
func TestTakeTaskHTTPIdeaMovesToTodo(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Someday"}, "stage": {StageIdea}}).Body.Close()
	idea := taskID(t, sqlDB, "Someday")

	postForm(t, client, ts, fmt.Sprintf("/tasks/%d/take", idea), nil).Body.Close()

	var stage string
	if err := sqlDB.QueryRow(`SELECT stage FROM tasks WHERE id = ?`, idea).Scan(&stage); err != nil {
		t.Fatal(err)
	}
	if stage != StageTodo {
		t.Errorf("stage after Take = %q, want %q", stage, StageTodo)
	}
}

// TestTakeTaskHTTPConflictShowsMessageAndChangesNothing covers gate
// 2.36's conflict end to end over HTTP: once someone is already on a
// job, a second Take refuses with the exact message and adds nobody.
func TestTakeTaskHTTPConflictShowsMessageAndChangesNothing(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Alex")
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Contested"}, "stage": {StageTodo}}).Body.Close()
	task := taskID(t, sqlDB, "Contested")

	// Alex takes it first (Sam is currently signed in, so switch back
	// to Alex for that request, then back to Sam for the conflicting one).
	alex := personID(t, sqlDB, "Alex")
	postForm(t, client, ts, "/who/select", url.Values{"person_id": {itoa(alex)}, "next": {"/"}}).Body.Close()
	postForm(t, client, ts, fmt.Sprintf("/tasks/%d/take", task), nil).Body.Close()

	sam := personID(t, sqlDB, "Sam")
	postForm(t, client, ts, "/who/select", url.Values{"person_id": {itoa(sam)}, "next": {"/"}}).Body.Close()
	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/take", task), nil)
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	if !strings.Contains(body, "Alex has just taken this job.") {
		t.Errorf("response missing the exact conflict message; got:\n%s", body)
	}

	var assigneeCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_assignees WHERE task_id = ?`, task).Scan(&assigneeCount); err != nil {
		t.Fatal(err)
	}
	if assigneeCount != 1 {
		t.Errorf("task_assignees rows = %d, want 1 (Sam's conflicting take added nobody)", assigneeCount)
	}
}

// TestMyJobsStartDoneAndGiveBackHTTP covers gate 2.37 end to end over
// HTTP: Start/Done move the job, and Give back returns it to Up for
// grabs when nobody else is on it, but not when someone else still is.
func TestMyJobsStartDoneAndGiveBackHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")
	sam := personID(t, sqlDB, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Solo job"}, "stage": {StageTodo}, "person_id": {itoa(sam)}}).Body.Close()
	solo := taskID(t, sqlDB, "Solo job")

	// Start: todo -> doing.
	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/move", solo), url.Values{
		"stage": {StageDoing}, "to_bottom": {"1"}, "redirect_to": {"myjobs"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Start status = %d, want 200 (following the redirect to My jobs)", resp.StatusCode)
	}
	if !strings.Contains(body, "Solo job") {
		t.Errorf("My jobs missing the started job; got:\n%s", body)
	}
	var stage string
	if err := sqlDB.QueryRow(`SELECT stage FROM tasks WHERE id = ?`, solo).Scan(&stage); err != nil {
		t.Fatal(err)
	}
	if stage != StageDoing {
		t.Errorf("stage after Start = %q, want %q", stage, StageDoing)
	}

	// Done: doing -> done, leaves My jobs.
	resp = postForm(t, client, ts, fmt.Sprintf("/tasks/%d/move", solo), url.Values{
		"stage": {StageDone}, "to_bottom": {"1"}, "redirect_to": {"myjobs"},
	})
	body = readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Done status = %d, want 200", resp.StatusCode)
	}
	if strings.Contains(body, "Solo job") {
		t.Errorf("My jobs still shows a job marked Done; got:\n%s", body)
	}
	if err := sqlDB.QueryRow(`SELECT stage FROM tasks WHERE id = ?`, solo).Scan(&stage); err != nil {
		t.Fatal(err)
	}
	if stage != StageDone {
		t.Errorf("stage after Done = %q, want %q", stage, StageDone)
	}

	// Give back with no other assignee: returns to Up for grabs.
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Give back solo"}, "stage": {StageTodo}, "person_id": {itoa(sam)}}).Body.Close()
	soloGiveBack := taskID(t, sqlDB, "Give back solo")
	resp = postForm(t, client, ts, fmt.Sprintf("/tasks/%d/assign", soloGiveBack), url.Values{
		"from_person": {itoa(sam)}, "redirect_to": {"myjobs"},
	})
	body = readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Give back status = %d, want 200 (following the redirect to My jobs)", resp.StatusCode)
	}
	grabsIdx := strings.Index(body, "Up for grabs")
	if grabsIdx < 0 {
		t.Fatalf("My jobs missing the Up for grabs heading; got:\n%s", body)
	}
	if strings.Contains(body[:grabsIdx], "Give back solo") {
		t.Errorf("the given-back job still shows in the person's own lane; got:\n%s", body[:grabsIdx])
	}
	if !strings.Contains(body[grabsIdx:], "Give back solo") {
		t.Errorf("Up for grabs missing the given-back job; got:\n%s", body[grabsIdx:])
	}
	var assigneeCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_assignees WHERE task_id = ?`, soloGiveBack).Scan(&assigneeCount); err != nil {
		t.Fatal(err)
	}
	if assigneeCount != 0 {
		t.Errorf("task_assignees rows for the given-back job = %d, want 0", assigneeCount)
	}

	// Give back with another assignee still on it: stays off Up for grabs.
	signIn(t, client, ts, "Alex")
	alex := personID(t, sqlDB, "Alex")
	postForm(t, client, ts, "/who/select", url.Values{"person_id": {itoa(sam)}, "next": {"/"}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{
		"title": {"Shared job"}, "stage": {StageTodo}, "person_id": {itoa(sam), itoa(alex)},
	}).Body.Close()
	shared := taskID(t, sqlDB, "Shared job")
	postForm(t, client, ts, fmt.Sprintf("/tasks/%d/assign", shared), url.Values{
		"from_person": {itoa(sam)}, "redirect_to": {"myjobs"},
	}).Body.Close()

	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_assignees WHERE task_id = ?`, shared).Scan(&assigneeCount); err != nil {
		t.Fatal(err)
	}
	if assigneeCount != 1 {
		t.Errorf("task_assignees rows for the shared job = %d, want 1 (Alex still on it)", assigneeCount)
	}

	grabsResp, err := client.Get(ts.URL + "/tasks")
	if err != nil {
		t.Fatal(err)
	}
	grabsBody := readBody(t, grabsResp)
	if strings.Contains(grabsBody, "Shared job") {
		t.Errorf("Up for grabs shows a job that still has an assignee; got:\n%s", grabsBody)
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

// TestReopenTaskHTTPReturnsToBottomOfTodo covers gate 2.08's Reopen
// action end to end over HTTP.
func TestReopenTaskHTTPReturnsToBottomOfTodo(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Existing"}, "stage": {StageTodo}}).Body.Close()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Finished"}, "stage": {StageDone}}).Body.Close()
	finished := taskID(t, sqlDB, "Finished")

	// Reopen doesn't require the task to have actually aged off the
	// board first (Store.Reopen has no such gate — only ListFinished's
	// own query does), so this checks the wiring, not the 14-day rule
	// itself (already covered at the store level).
	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/reopen", finished), nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect to Finished tasks)", resp.StatusCode)
	}

	var stage string
	var doneAt sql.NullString
	if err := sqlDB.QueryRow(`SELECT stage, done_at FROM tasks WHERE id = ?`, finished).Scan(&stage, &doneAt); err != nil {
		t.Fatal(err)
	}
	if stage != StageTodo {
		t.Errorf("stage after Reopen = %q, want %q", stage, StageTodo)
	}
	if doneAt.Valid {
		t.Errorf("done_at after Reopen = %+v, want NULL", doneAt)
	}
}

// TestRemoveAndRestoreTaskHTTP covers gate 2.09 end to end over HTTP:
// Remove hides a task from the board and lists it in Removed tasks;
// Restore brings it back.
func TestRemoveAndRestoreTaskHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Removable"}, "stage": {StageTodo}}).Body.Close()
	task := taskID(t, sqlDB, "Removable")

	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/remove", task), nil)
	boardBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove status = %d, want 200 (following the redirect to the board)", resp.StatusCode)
	}
	if strings.Contains(boardBody, "Removable") {
		t.Errorf("board still shows a removed task; got:\n%s", boardBody)
	}

	removedResp, err := client.Get(ts.URL + "/tasks/removed")
	if err != nil {
		t.Fatal(err)
	}
	removedBody := readBody(t, removedResp)
	if !strings.Contains(removedBody, "Removable") {
		t.Errorf("Removed tasks page missing the removed task; got:\n%s", removedBody)
	}

	resp = postForm(t, client, ts, fmt.Sprintf("/tasks/%d/restore", task), nil)
	restoredListBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore status = %d, want 200 (following the redirect to Removed tasks)", resp.StatusCode)
	}
	if strings.Contains(restoredListBody, "Removable") {
		t.Errorf("Removed tasks page still lists the restored task; got:\n%s", restoredListBody)
	}

	boardResp, err := client.Get(ts.URL + "/tasks/board")
	if err != nil {
		t.Fatal(err)
	}
	boardBody = readBody(t, boardResp)
	if !strings.Contains(boardBody, "Removable") {
		t.Errorf("board missing the restored task; got:\n%s", boardBody)
	}
}

// TestNoDeleteRouteForTasks covers SPEC gate 2.09's "nothing can be
// permanently deleted": every task-related path registers only the
// methods this section actually needs (SPEC B4's enhanced ServeMux
// automatically answers 405 for a registered path given an
// unregistered method), so a DELETE to any of them must never succeed.
func TestNoDeleteRouteForTasks(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Task"}, "stage": {StageTodo}}).Body.Close()
	id := taskID(t, sqlDB, "Task")
	idStr := itoa(id)

	paths := []string{
		"/tasks",
		"/tasks/board",
		"/tasks/finished",
		"/tasks/removed",
		"/tasks/" + idStr,
		"/tasks/" + idStr + "/move",
		"/tasks/" + idStr + "/take",
		"/tasks/" + idStr + "/reopen",
		"/tasks/" + idStr + "/remove",
		"/tasks/" + idStr + "/restore",
	}
	for _, path := range paths {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", ts.URL)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("DELETE %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			t.Errorf("DELETE %s = %d, want a non-success status (no delete route exists)", path, resp.StatusCode)
		}
	}

	var stillExists int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = ?`, id).Scan(&stillExists); err != nil {
		t.Fatal(err)
	}
	if stillExists != 1 {
		t.Error("task no longer exists after DELETE attempts")
	}
}

// TestVersionEndpointReturnsIncreasingCounter covers the refresh
// mechanism's polling endpoint (SPEC B4/D-15) end to end over HTTP.
func TestVersionEndpointReturnsIncreasingCounter(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	getVersion := func() int64 {
		t.Helper()
		resp, err := client.Get(ts.URL + "/tasks/version")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body struct {
			Version int64 `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body.Version
	}

	before := getVersion()
	postForm(t, client, ts, "/tasks", url.Values{"title": {"Task"}}).Body.Close()
	after := getVersion()
	if after <= before {
		t.Errorf("version after creating a task = %d, want > %d", after, before)
	}
}

// TestAssignTaskHTTPFromUnassignedToPerson covers gate 2.28's assign
// endpoint end to end over HTTP: dragging (or the button equivalent)
// from Unassigned onto a person.
func TestAssignTaskHTTPFromUnassignedToPerson(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")
	signIn(t, client, ts, "Alex")
	alex := personID(t, sqlDB, "Alex")

	postForm(t, client, ts, "/tasks", url.Values{"title": {"Task"}, "stage": {StageTodo}}).Body.Close()
	task := taskID(t, sqlDB, "Task")

	resp := postForm(t, client, ts, fmt.Sprintf("/tasks/%d/assign", task), url.Values{
		"to_person": {itoa(alex)},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect to Team)", resp.StatusCode)
	}
	if !strings.Contains(body, "Alex") {
		t.Errorf("Team page missing Alex's lane with the assigned task; got:\n%s", body)
	}

	var assigneeCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_assignees WHERE task_id = ? AND person_id = ?`, task, alex).Scan(&assigneeCount); err != nil {
		t.Fatal(err)
	}
	if assigneeCount != 1 {
		t.Errorf("task_assignees rows for Alex = %d, want 1", assigneeCount)
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
