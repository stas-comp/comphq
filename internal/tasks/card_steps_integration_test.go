//go:build integration

package tasks

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Gates 8.35-8.39: My jobs draws every card's steps from one query, however many
// cards there are, and the tick form comes back to My jobs with the card open.
func TestMyJobsPageDrawsStepsFromOneQueryAndTicksComeBack(t *testing.T) {
	ts, sqlDB := newTestServer(t)
	client := &http.Client{Jar: mustCookieJar(t)}
	signIn(t, client, ts, "Sam")
	sam := personID(t, sqlDB, "Sam")

	makeJobs := func(prefix string, n int, stage string) {
		for i := 0; i < n; i++ {
			form := url.Values{"title": {fmt.Sprintf("%s %d", prefix, i)}, "stage": {stage}, "person_id": {fmt.Sprint(sam)}}
			for s := 0; s < 5; s++ {
				form.Add("step", fmt.Sprintf("step %d", s))
			}
			postForm(t, client, ts, "/tasks", form).Body.Close()
		}
	}
	queriesFor := func() (int64, string) {
		before := stepListQueries.Load()
		resp, err := client.Get(ts.URL + "/tasks")
		if err != nil {
			t.Fatal(err)
		}
		body := readBody(t, resp)
		return stepListQueries.Load() - before, body
	}

	makeJobs("Working", 1, "doing")
	one, _ := queriesFor()
	makeJobs("Working", 9, "doing")
	makeJobs("Next", 10, "todo")
	many, body := queriesFor()
	if one != 1 || many != 1 {
		t.Errorf("steps queries per page: %d with one card, %d with twenty; want 1 each", one, many)
	}
	if got := strings.Count(body, `data-card-steps`); got != 20 {
		t.Errorf("%d cards draw steps, want 20", got)
	}
	// Working on now cards come open, Up next ones closed.
	if !strings.Contains(body, `aria-expanded="true"`) || !strings.Contains(body, `aria-expanded="false"`) {
		t.Errorf("both open and closed cards are drawn")
	}
	if !strings.Contains(body, "Show steps") || !strings.Contains(body, "Hide steps") {
		t.Errorf("the toggle says Show steps on closed cards and Hide steps on open ones")
	}

	// A tick from a card comes back to My jobs with that card open (gate 8.38).
	var jobID, stepID int64
	if err := sqlDB.QueryRow(`SELECT t.id, c.id FROM tasks t JOIN task_checklist_items c ON c.task_id = t.id WHERE t.stage = 'todo' ORDER BY t.id, c.position LIMIT 1`).Scan(&jobID, &stepID); err != nil {
		t.Fatal(err)
	}
	noFollow := &http.Client{Jar: client.Jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tasks/%d/steps/%d/tick", ts.URL, jobID, stepID), strings.NewReader(url.Values{"done": {"0", "1"}, "redirect_to": {"myjobs"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := noFollow.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != fmt.Sprintf("/tasks?open=%d#job-%d", jobID, jobID) {
		t.Errorf("tick answered %d, Location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	var done int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM task_checklist_items WHERE id = ? AND done_at IS NOT NULL`, stepID).Scan(&done)
	if done != 1 {
		t.Errorf("the step wasn't ticked")
	}
	// ...and that card is drawn open, with the step ticked and Sam's name on it.
	resp2, _ := client.Get(fmt.Sprintf("%s/tasks?open=%d", ts.URL, jobID))
	page := readBody(t, resp2)
	card := page[strings.Index(page, fmt.Sprintf(`id="job-%d"`, jobID)):]
	card = card[:strings.Index(card, "</li>")]
	if !strings.Contains(card, `aria-expanded="true"`) || !strings.Contains(card, "Hide steps") || !strings.Contains(card, "1/5") || !strings.Contains(card, "Sam, ") {
		t.Errorf("the ticked card = %s", card)
	}
}
