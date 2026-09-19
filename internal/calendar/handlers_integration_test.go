//go:build integration

package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/db"
)

// fakeDueTasksSource is P2-16's own "a fake DueTasks in calendar
// handler tests" (PLAN.md) — Calendar's tests never need a real
// internal/tasks store, only something satisfying its own small
// interface (SPEC B2).
type fakeDueTasksSource struct {
	tasks []DueTask
}

func (f fakeDueTasksSource) DueTasks(ctx context.Context, from, to time.Time) ([]DueTask, error) {
	return f.tasks, nil
}

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
	return newTestServerWithDueTasks(t, fakeDueTasksSource{})
}

func newTestServerWithDueTasks(t *testing.T, dueTasks DueTasksSource) (*httptest.Server, *sql.DB) {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"app", "people", "calendar"} {
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
	srv.Registry().Add(Section(srv, dueTasks))

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

func eventID(t *testing.T, sqlDB *sql.DB, title string) int64 {
	t.Helper()
	var id int64
	if err := sqlDB.QueryRow(`SELECT id FROM events WHERE title = ?`, title).Scan(&id); err != nil {
		t.Fatalf("find event %q: %v", title, err)
	}
	return id
}

// TestCreateEventHTTPRedisplaysEveryField covers gate 2.13's add-event
// path over HTTP: every field saved and redisplayed on the details page.
func TestCreateEventHTTPRedisplaysEveryField(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Christmas concert"}, "notes": {"Bring programmes"},
		"start_date": {"2026-12-12"}, "end_date": {"2026-12-13"},
		"start_time": {"18:00"}, "end_time": {"20:00"},
		"recurrence": {"yearly"}, "until_date": {"2030-12-31"},
		"notice_amount": {"2"}, "notice_unit": {"weeks"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect to the details page)", resp.StatusCode)
	}
	for _, want := range []string{"Christmas concert", "Bring programmes", "12/12/2026", "13/12/2026", "18:00", "20:00", "31/12/2030"} { // the date fields show day first (gate 4.05)
		if !strings.Contains(body, want) {
			t.Errorf("details page missing %q; got:\n%s", want, body)
		}
	}
}

// TestCreateEventHTTPAcceptsDayFirstDates covers gate 4.05: the date fields
// take UK order. The same event typed day first lands on the right days of
// the month grid, and is shown back day first.
func TestCreateEventHTTPAcceptsDayFirstDates(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Day first"}, "start_date": {"5/12/2026"}, "end_date": {"06.12.2026"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	for _, want := range []string{"Day first", "05/12/2026", "06/12/2026"} {
		if !strings.Contains(body, want) {
			t.Errorf("details page missing %q", want)
		}
	}

	gridResp, err := client.Get(ts.URL + "/calendar?month=2026-12")
	if err != nil {
		t.Fatal(err)
	}
	grid := readBody(t, gridResp)
	i5, i6 := strings.Index(grid, `data-date="2026-12-05"`), strings.Index(grid, `data-date="2026-12-07"`)
	if i5 < 0 || i6 < 0 || !strings.Contains(grid[i5:i6], "Day first") {
		t.Errorf("the event typed as 5/12/2026 to 06.12.2026 is not on 5 and 6 December in the month grid")
	}
}

// TestCreateEventHTTPBlankTitleShowsMessage covers the empty-title refusal.
func TestCreateEventHTTPBlankTitleShowsMessage(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/calendar/events", url.Values{"title": {"  "}, "start_date": {"2026-10-01"}})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (refused with a message, not an error page)", resp.StatusCode)
	}
	if !strings.Contains(body, "Please type a title.") {
		t.Errorf("response missing the refusal message; got:\n%s", body)
	}
}

// TestCreateEventHTTPEndDateBeforeStartShowsMessage covers gate 2.13's
// end-date-not-before-start validation over HTTP.
func TestCreateEventHTTPEndDateBeforeStartShowsMessage(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Backwards"}, "start_date": {"2026-10-05"}, "end_date": {"2026-10-01"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "end date can") || !strings.Contains(body, "before the start date") {
		t.Errorf("response missing the refusal message; got:\n%s", body)
	}
}

// TestCreateEventHTTPEndTimeWithoutStartTimeShowsMessage covers gate
// 2.13's end-time-needs-start-time validation over HTTP.
func TestCreateEventHTTPEndTimeWithoutStartTimeShowsMessage(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp := postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"No start time"}, "start_date": {"2026-10-05"}, "end_time": {"10:00"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "Add a start time before an end time.") {
		t.Errorf("response missing the refusal message; got:\n%s", body)
	}
}

// TestUpdateEventHTTPSavesChanges covers gate 2.20's edit-from-details
// path over HTTP.
func TestUpdateEventHTTPSavesChanges(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{"title": {"Original"}, "start_date": {"2026-10-01"}}).Body.Close()
	id := eventID(t, sqlDB, "Original")

	resp := postForm(t, client, ts, fmt.Sprintf("/calendar/events/%d", id), url.Values{
		"title": {"Edited"}, "start_date": {"2026-10-02"}, "notes": {"Updated notes"},
	})
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect back to the details page)", resp.StatusCode)
	}
	if !strings.Contains(body, "Edited") || !strings.Contains(body, "02/10/2026") || !strings.Contains(body, "Updated notes") {
		t.Errorf("details page missing the edited fields; got:\n%s", body)
	}
	if !strings.Contains(body, "Last changed by Sam") {
		t.Errorf("details page missing the last-changed line; got:\n%s", body)
	}
}

// TestMonthViewHTTPShowsCreatedEvent covers gate 2.12's month view
// wiring over HTTP: a created event's chip appears on its own date.
func TestMonthViewHTTPShowsCreatedEvent(t *testing.T) {
	ts, _ := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{"title": {"October event"}, "start_date": {"2026-10-15"}}).Body.Close()

	resp, err := client.Get(ts.URL + "/calendar?month=2026-10")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "October event") {
		t.Errorf("month view missing the created event's chip; got:\n%s", body)
	}
	if !strings.Contains(body, `data-date="2026-10-15"`) {
		t.Errorf("month view missing the expected day cell; got:\n%s", body)
	}
	if !strings.Contains(body, `data-kind="event"`) {
		t.Errorf("event chip missing data-kind=\"event\"; got:\n%s", body)
	}
}

// TestMonthViewHTTPShowsTaskDeadlineChip covers gate 2.19 over HTTP,
// using a fake DueTasksSource (PLAN.md's own test guidance) so this
// test never needs a real internal/tasks store.
func TestMonthViewHTTPShowsTaskDeadlineChip(t *testing.T) {
	ts, _ := newTestServerWithDueTasks(t, fakeDueTasksSource{tasks: []DueTask{
		{ID: 42, Title: "File the report", DueDate: "2026-10-20"},
	}})
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	resp, err := client.Get(ts.URL + "/calendar?month=2026-10")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "File the report") {
		t.Errorf("month view missing the task deadline chip; got:\n%s", body)
	}
	if !strings.Contains(body, `data-kind="task"`) {
		t.Errorf("task chip missing data-kind=\"task\"; got:\n%s", body)
	}
	if !strings.Contains(body, `href="/tasks/42"`) {
		t.Errorf("task chip doesn't link to the task page; got:\n%s", body)
	}
}

// TestNoDeleteRouteForCalendar covers SPEC's "nothing is permanently
// deleted" for the routes that exist so far — matching the Tasks
// section's own route-table precedent.
func TestNoDeleteRouteForCalendar(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{"title": {"Event"}, "start_date": {"2026-10-01"}}).Body.Close()
	id := eventID(t, sqlDB, "Event")

	paths := []string{
		"/calendar", "/calendar/list", "/calendar/new", "/calendar/removed",
		fmt.Sprintf("/calendar/events/%d", id),
		fmt.Sprintf("/calendar/events/%d/occurrence", id),
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
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM events WHERE id = ?`, id).Scan(&stillExists); err != nil {
		t.Fatal(err)
	}
	if stillExists != 1 {
		t.Error("event no longer exists after DELETE attempts")
	}
}

// TestEventDetailsHTTPShowsChoiceForARepeatingOccurrence covers gate
// 2.15's "Change just this one" / "Change all" choice over HTTP: it
// only appears for a repeating event reached via ?occurrence=, never
// for a plain visit or a non-repeating event.
func TestEventDetailsHTTPShowsChoiceForARepeatingOccurrence(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Christmas concert"}, "start_date": {"2026-12-12"}, "recurrence": {"yearly"},
	}).Body.Close()
	id := eventID(t, sqlDB, "Christmas concert")

	resp, err := client.Get(ts.URL + fmt.Sprintf("/calendar/events/%d?occurrence=2026-12-12", id))
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Change just this one") || !strings.Contains(body, "Change all") {
		t.Errorf("response missing the gate 2.15 choice; got:\n%s", body)
	}

	resp, err = client.Get(ts.URL + fmt.Sprintf("/calendar/events/%d", id))
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(t, resp)
	if strings.Contains(body, "Change just this one") {
		t.Errorf("a plain visit (no occurrence param) shows the choice; got:\n%s", body)
	}
}

// TestMoveJustThisOneHTTP covers the exact gate 2.15 script over HTTP:
// a yearly event moved just once shows on its new date, and the other
// year is unaffected.
func TestMoveJustThisOneHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Christmas concert"}, "start_date": {"2026-12-12"}, "recurrence": {"yearly"},
	}).Body.Close()
	id := eventID(t, sqlDB, "Christmas concert")

	resp := postForm(t, client, ts, fmt.Sprintf("/calendar/events/%d/occurrence", id), url.Values{
		"original_date": {"2026-12-12"}, "start_date": {"2026-12-19"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (following the redirect back to the event)", resp.StatusCode)
	}
	resp.Body.Close()

	dec2026, err := client.Get(ts.URL + "/calendar?month=2026-12")
	if err != nil {
		t.Fatal(err)
	}
	dec2026Body := readBody(t, dec2026)
	cell12 := dec2026Body[strings.Index(dec2026Body, `data-date="2026-12-12"`):strings.Index(dec2026Body, `data-date="2026-12-13"`)]
	if strings.Contains(cell12, "Christmas concert") {
		t.Errorf("the moved-away original date still shows a chip; cell:\n%s", cell12)
	}
	if !strings.Contains(dec2026Body, "Christmas concert") {
		t.Errorf("December 2026 missing the moved concert on its new date; got:\n%s", dec2026Body)
	}

	dec2027, err := client.Get(ts.URL + "/calendar?month=2027-12")
	if err != nil {
		t.Fatal(err)
	}
	dec2027Body := readBody(t, dec2027)
	if !strings.Contains(dec2027Body, `data-date="2027-12-12"`) || !strings.Contains(dec2027Body, "Christmas concert") {
		t.Errorf("December 2027 missing the unaffected concert on the 12th; got:\n%s", dec2027Body)
	}
}

// TestCancelJustThisOneHTTP covers gate 2.16 over HTTP.
func TestCancelJustThisOneHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{
		"title": {"Weekly sync"}, "start_date": {"2026-03-02"}, "recurrence": {"weekly"},
	}).Body.Close()
	id := eventID(t, sqlDB, "Weekly sync")

	resp := postForm(t, client, ts, fmt.Sprintf("/calendar/events/%d/occurrence/cancel", id), url.Values{
		"original_date": {"2026-03-09"},
	})
	resp.Body.Close()

	monthResp, err := client.Get(ts.URL + "/calendar?month=2026-03")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, monthResp)
	if !strings.Contains(body, `data-date="2026-03-02"`) {
		t.Fatalf("missing expected day cells; got:\n%s", body)
	}
	// The cancelled week's cell exists but has no chip; the others do.
	cell09 := body[strings.Index(body, `data-date="2026-03-09"`):strings.Index(body, `data-date="2026-03-10"`)]
	if strings.Contains(cell09, "Weekly sync") {
		t.Errorf("cancelled occurrence still shows a chip; cell:\n%s", cell09)
	}
	cell16 := body[strings.Index(body, `data-date="2026-03-16"`):strings.Index(body, `data-date="2026-03-17"`)]
	if !strings.Contains(cell16, "Weekly sync") {
		t.Errorf("an unrelated occurrence disappeared too; cell:\n%s", cell16)
	}
}

// TestRemoveAndRestoreEventHTTP covers gate 2.18 over HTTP.
func TestRemoveAndRestoreEventHTTP(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")

	postForm(t, client, ts, "/calendar/events", url.Values{"title": {"Removable"}, "start_date": {"2026-10-01"}}).Body.Close()
	id := eventID(t, sqlDB, "Removable")

	resp := postForm(t, client, ts, fmt.Sprintf("/calendar/events/%d/remove", id), nil)
	monthBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove status = %d, want 200 (following the redirect to the Calendar)", resp.StatusCode)
	}
	if strings.Contains(monthBody, "Removable") {
		t.Errorf("month view still shows a removed event; got:\n%s", monthBody)
	}

	removedResp, err := client.Get(ts.URL + "/calendar/removed")
	if err != nil {
		t.Fatal(err)
	}
	removedBody := readBody(t, removedResp)
	if !strings.Contains(removedBody, "Removable") {
		t.Errorf("Removed events page missing the removed event; got:\n%s", removedBody)
	}

	resp = postForm(t, client, ts, fmt.Sprintf("/calendar/events/%d/restore", id), nil)
	restoredListBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore status = %d, want 200 (following the redirect to Removed events)", resp.StatusCode)
	}
	if strings.Contains(restoredListBody, "Removable") {
		t.Errorf("Removed events still lists the restored event; got:\n%s", restoredListBody)
	}

	monthResp, err := client.Get(ts.URL + "/calendar?month=2026-10")
	if err != nil {
		t.Fatal(err)
	}
	monthBody = readBody(t, monthResp)
	if !strings.Contains(monthBody, "Removable") {
		t.Errorf("month view missing the restored event; got:\n%s", monthBody)
	}
}
