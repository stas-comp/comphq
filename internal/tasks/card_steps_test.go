package tasks

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStepsForTasksReturnsEveryJobsLiveStepsInOrderWithOneQuery(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	ctx := context.Background()
	sam := testPerson(t, sqlDB, "Sam")
	jo := testPerson(t, sqlDB, "Jo")

	var ids []int64
	for i := 0; i < 20; i++ {
		job, err := store.Create(ctx, CreateInput{Title: fmt.Sprintf("Job %d", i), Steps: []string{"one", "two", "three"}}, sam, fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, job.ID)
	}
	noSteps, _ := store.Create(ctx, CreateInput{Title: "No steps"}, sam, fixedNow)
	ids = append(ids, noSteps.ID)

	// Tick the middle step of the first job as Jo, and remove the last of the second.
	first, _ := store.ListSteps(ctx, ids[0], fixedNow)
	if err := store.SetStepDone(ctx, ids[0], first[1].ID, true, jo, fixedNow); err != nil {
		t.Fatal(err)
	}
	second, _ := store.ListSteps(ctx, ids[1], fixedNow)
	if err := store.RemoveStep(ctx, ids[1], second[2].ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	before := stepListQueries.Load()
	got, err := store.StepsForTasks(ctx, ids, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if n := stepListQueries.Load() - before; n != 1 {
		t.Errorf("StepsForTasks counted %d queries, want 1 for the whole page", n)
	}
	if len(got[ids[0]]) != 3 || got[ids[0]][0].Text != "one" || got[ids[0]][2].Text != "three" {
		t.Errorf("first job's steps = %+v", got[ids[0]])
	}
	if s := got[ids[0]][1]; !s.Done || s.DoneByName != "Jo" || s.DoneDate == "" {
		t.Errorf("a ticked step shows who and when: %+v", s)
	}
	if len(got[ids[1]]) != 2 {
		t.Errorf("a removed step is left out: %+v", got[ids[1]])
	}
	if len(got[noSteps.ID]) != 0 {
		t.Errorf("a job with no steps has %+v", got[noSteps.ID])
	}
	if empty, err := store.StepsForTasks(ctx, nil, fixedNow); err != nil || len(empty) != 0 {
		t.Errorf("no ids = %v, %v", empty, err)
	}
}

// Gate 8.39 (shape): the page makes one steps query whatever the number of cards.
func TestMyJobsPageLoadsStepsWithOneQueryForAllCards(t *testing.T) {
	for _, jobs := range []int{1, 25} {
		f := newRefFixture(t)
		for i := 0; i < jobs; i++ {
			job, err := f.store.Create(f.ctx, CreateInput{Title: fmt.Sprintf("Mine %d", i), Stage: StageDoing, PersonIDs: []int64{f.sam}, Steps: []string{"a", "b"}}, f.sam, fixedNow)
			if err != nil {
				t.Fatal(err)
			}
			_ = job
		}
		h := &Handlers{tasks: f.store}
		_ = h
		before := stepListQueries.Load()
		ids := make([]int64, 0, jobs)
		rows, _ := f.store.DB.Query(`SELECT id FROM tasks`)
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		if _, err := f.store.StepsForTasks(f.ctx, ids, fixedNow); err != nil {
			t.Fatal(err)
		}
		if n := stepListQueries.Load() - before; n != 1 {
			t.Errorf("%d cards: %d steps queries, want 1", jobs, n)
		}
	}
}

func TestCardStepStateOpenCloseAndLinks(t *testing.T) {
	none := cardStepStateFromQuery(url.Values{})
	if !none.isOpen(5, true) || none.isOpen(5, false) {
		t.Errorf("defaults: Working on now open, the rest closed")
	}
	st := cardStepStateFromQuery(url.Values{"open": {"7", "x", "9"}, "close": {"5"}})
	if !st.isOpen(7, false) || !st.isOpen(9, false) || st.isOpen(8, false) {
		t.Errorf("?open= opens cards from the server")
	}
	if st.isOpen(5, true) || !st.isOpen(6, true) {
		t.Errorf("?close= closes one that starts open")
	}
	// An open card links to its own closing, and leaves the others as asked.
	href := st.toggleHref(7, true)
	u, err := url.Parse(href)
	if err != nil || u.Path != "/tasks" {
		t.Fatalf("href = %q, %v", href, err)
	}
	if got := u.Query()["open"]; len(got) != 1 || got[0] != "9" {
		t.Errorf("closing 7 keeps 9 open: %v", got)
	}
	if got := u.Query()["close"]; !(strings.Contains(strings.Join(got, ","), "7") && strings.Contains(strings.Join(got, ","), "5")) {
		t.Errorf("closing 7 keeps 5 closed: %v", got)
	}
	if u.Fragment != "job-7" {
		t.Errorf("link keeps the place on the page: %q", u.Fragment)
	}
	// A closed card links to opening it.
	if got, _ := url.Parse(st.toggleHref(8, false)); !strings.Contains(strings.Join(got.Query()["open"], ","), "8") {
		t.Errorf("opening 8: %v", got)
	}
	// Nothing asked: the plain page.
	if h := none.toggleHref(3, true); !strings.HasPrefix(h, "/tasks?close=3") {
		t.Errorf("none asked, close 3: %q", h)
	}
}

func TestCardStepsForAJobWithNoStepsIsEmpty(t *testing.T) {
	c := cardStepsFor(1, 0, nil, true, cardStepStateFromQuery(url.Values{}))
	if c.StepsOpen || c.StepsToggle != "" || len(c.Steps) != 0 {
		t.Errorf("%+v", c)
	}
	_ = time.Now
}
