package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
)

// SPEC gates 8.40-8.47, 8.51, 8.53 (B14.4, D-108): yearly jobs, waiting, and the
// yearly pass. Dates are fixed; "today" is whatever each test passes, as
// COMPHQ_TEST_TODAY does in the app.

type yearlyFixture struct {
	t     *testing.T
	store *Store
	ctx   context.Context
	owner int64
}

func newYearlyFixture(t *testing.T) *yearlyFixture {
	t.Helper()
	db := openTestDB(t)
	return &yearlyFixture{t: t, store: &Store{DB: db}, ctx: context.Background(), owner: testPerson(t, db, "Sam")}
}

// yearly makes a yearly job on day `made`, due `due`, with `weeks` weeks' notice.
func (f *yearlyFixture) yearly(title, due string, weeks int, made string) Task {
	f.t.Helper()
	task, err := f.store.Create(f.ctx, CreateInput{Title: title, Repeat: "yearly", NoticeWeeks: weeks, DueDate: due}, f.owner, day(made))
	if err != nil {
		f.t.Fatalf("Create yearly %q: %v", title, err)
	}
	return task
}

type yearlyRow struct {
	stage, due, anchor, showsFrom, resetOn, doneAt, repeat string
	notice                                                 int
	position                                               int
}

func (f *yearlyFixture) row(id int64) yearlyRow {
	f.t.Helper()
	var r yearlyRow
	var due, anchor, shows, reset, doneAt *string
	var notice *int
	if err := f.store.DB.QueryRow(
		`SELECT stage, position, due_date, repeat, repeat_anchor, notice_days, shows_from, repeat_reset_on, done_at FROM tasks WHERE id = ?`, id,
	).Scan(&r.stage, &r.position, &due, &r.repeat, &anchor, &notice, &shows, &reset, &doneAt); err != nil {
		f.t.Fatal(err)
	}
	r.due, r.anchor, r.showsFrom, r.resetOn, r.doneAt = deref(due), deref(anchor), deref(shows), deref(reset), deref(doneAt)
	if notice != nil {
		r.notice = *notice
	}
	return r
}

func (f *yearlyFixture) board(today string) []string {
	f.t.Helper()
	tasks, err := f.store.ListBoard(f.ctx, day(today), BoardFilter{})
	if err != nil {
		f.t.Fatal(err)
	}
	var titles []string
	for _, tk := range tasks {
		titles = append(titles, tk.Title)
	}
	return titles
}

func (f *yearlyFixture) history(id int64) []string {
	f.t.Helper()
	acts, err := f.store.ListActivity(f.ctx, id)
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, a := range acts {
		out = append(out, a.Text)
	}
	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// Gates 8.40, 8.41: saving a yearly job sets its anchor, notice and show date.
func TestGate841SavingAYearlyJobSetsAnchorNoticeAndShowDate(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew the licence", "2026-12-01", 6, "2026-09-01")
	r := f.row(job.ID)
	if r.repeat != "yearly" || r.anchor != "12-01" || r.notice != 42 || r.showsFrom != "2026-10-20" {
		t.Errorf("row = %+v, want yearly, anchor 12-01, 42 days' notice, shown from 2026-10-20", r)
	}
	// A yearly job always belongs in To do, whatever column the form sent.
	for _, stage := range []string{StageIdea, StageDoing, StageTodo} {
		j, err := f.store.Create(f.ctx, CreateInput{Title: "Starts in " + stage, Stage: stage, Repeat: "yearly", DueDate: "2027-03-01"}, f.owner, day("2026-09-01"))
		if err != nil {
			t.Fatal(err)
		}
		if got := f.row(j.ID).stage; got != StageTodo {
			t.Errorf("a yearly job started in %s is in %s, want todo", stage, got)
		}
	}
	// The notice defaults to 6 weeks.
	d, err := f.store.Create(f.ctx, CreateInput{Title: "Default notice", Repeat: "yearly", DueDate: "2027-03-01"}, f.owner, day("2026-09-01"))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.row(d.ID); got.notice != 42 || got.showsFrom != "2027-01-18" {
		t.Errorf("default notice = %+v", got)
	}
	// 1 to 52 weeks.
	for _, weeks := range []int{1, 52} {
		if _, err := f.store.Create(f.ctx, CreateInput{Title: fmt.Sprintf("%d weeks", weeks), Repeat: "yearly", NoticeWeeks: weeks, DueDate: "2027-03-01"}, f.owner, day("2026-09-01")); err != nil {
			t.Errorf("%d weeks refused: %v", weeks, err)
		}
	}
	for _, weeks := range []int{-1, 53, 100} {
		if _, err := f.store.Create(f.ctx, CreateInput{Title: "Bad notice", Repeat: "yearly", NoticeWeeks: weeks, DueDate: "2027-03-01"}, f.owner, day("2026-09-01")); !errors.Is(err, ErrNoticeWeeks) {
			t.Errorf("%d weeks = %v, want ErrNoticeWeeks", weeks, err)
		}
	}
	// A yearly job needs a due date, and nothing is saved without one.
	before := countRows(t, &refFixture{store: f.store}, `SELECT COUNT(*) FROM tasks`)
	if _, err := f.store.Create(f.ctx, CreateInput{Title: "No date", Repeat: "yearly"}, f.owner, day("2026-09-01")); !errors.Is(err, ErrYearlyNeedsDue) {
		t.Errorf("no due date = %v, want ErrYearlyNeedsDue", err)
	}
	if ErrYearlyNeedsDue.Error() != "A yearly job needs a due date." {
		t.Errorf("message = %q", ErrYearlyNeedsDue.Error())
	}
	if after := countRows(t, &refFixture{store: f.store}, `SELECT COUNT(*) FROM tasks`); after != before {
		t.Errorf("a refused yearly job was saved")
	}
}

// Gate 8.42: a job is waiting while its show date is still ahead.
func TestGate842AWaitingJobIsOnNoList(t *testing.T) {
	f := newYearlyFixture(t)
	f.yearly("Waiting job", "2026-12-01", 6, "2026-09-01") // shows from 2026-10-20
	f.yearly("Showing job", "2026-10-25", 6, "2026-09-01") // shows from 2026-09-13
	if got := f.board("2026-10-01"); !sameStrings(got, []string{"Showing job"}) {
		t.Errorf("Board on 1 Oct = %v, want only the job whose date has come", got)
	}
	if got := f.board("2026-10-19"); !sameStrings(got, []string{"Showing job"}) {
		t.Errorf("Board on 19 Oct = %v", got)
	}
	if got := f.board("2026-10-20"); len(got) != 2 || !contains(got, "Showing job") || !contains(got, "Waiting job") {
		t.Errorf("Board on its show date = %v, want both", got)
	}
}

// Gate 8.42 and SPEC B14.4: every list that shows jobs leaves a waiting job out,
// except job search and the Calendar, which must find it.
func TestGate842EveryListLeavesAWaitingJobOut(t *testing.T) {
	f := newYearlyFixture(t)
	today := day("2026-10-01")
	visible := f.mk("Visible job", StageTodo)
	waiting := f.yearly("Waiting yearly", "2026-12-01", 6, "2026-09-01")
	if err := f.store.Assign(f.ctx, waiting.ID, 0, f.owner, f.owner, today); err != nil {
		t.Fatal(err)
	}
	unassigned := f.yearly("Waiting and unassigned", "2026-12-02", 6, "2026-09-01")
	// A job that must finish first, which is the waiting yearly job.
	if err := f.store.AddLink(f.ctx, visible.ID, waiting.ID, LinkFirst, f.owner, today); err != nil {
		t.Fatal(err)
	}

	titlesOf := func(tasks []Task) []string {
		var out []string
		for _, t := range tasks {
			out = append(out, t.Title)
		}
		return out
	}
	lists := map[string]func() []string{
		"Board": func() []string { b, _ := f.store.ListBoard(f.ctx, today, BoardFilter{}); return titlesOf(b) },
		"Board filtered to a person": func() []string {
			b, _ := f.store.ListBoard(f.ctx, today, BoardFilter{PersonID: f.owner})
			return titlesOf(b)
		},
		"Board filtered by a word": func() []string {
			b, _ := f.store.ListBoard(f.ctx, today, BoardFilter{Query: "waiting"})
			return titlesOf(b)
		},
		"My jobs and Team lanes": func() []string { l, _ := f.store.ListForTeamView(f.ctx, today); return titlesOf(l) },
		"Up for grabs":           func() []string { g, _ := f.store.ListUpForGrabs(f.ctx, today); return titlesOf(g) },
		"Finished tasks":         func() []string { l, _ := f.store.ListFinished(f.ctx, today); return titlesOf(l) },
		"the Briefing": func() []string {
			rows, _ := f.store.BriefingTasks(f.ctx, day("2027-01-01"), today)
			var out []string
			for _, r := range rows {
				out = append(out, r.Title)
			}
			return out
		},
		"the WAITING stamps": func() []string {
			m, _ := f.store.WaitingOn(f.ctx, []int64{waiting.ID, unassigned.ID}, today)
			var out []string
			for id := range m {
				out = append(out, fmt.Sprint(id))
			}
			return out
		},
	}
	for name, list := range lists {
		for _, title := range list() {
			if strings.Contains(title, "Waiting") || title == fmt.Sprint(waiting.ID) || title == fmt.Sprint(unassigned.ID) {
				t.Errorf("%s lists the waiting job %q", name, title)
			}
		}
	}
	// The visible job is still on the lists that should have it.
	if got := lists["Board"](); !contains(got, "Visible job") {
		t.Errorf("Board = %v", got)
	}
	if got := lists["Up for grabs"](); !contains(got, "Visible job") {
		t.Errorf("Up for grabs = %v", got)
	}

	// Search and the Calendar must find it.
	hits, err := f.store.SearchJobs(f.ctx, "waiting yearly", 0, today)
	if err != nil || len(hits) != 1 || hits[0].ID != waiting.ID {
		t.Errorf("search = %+v, %v; want the waiting job found", hits, err)
	}
	due, err := f.store.DueTasks(f.ctx, day("2026-12-01"), day("2026-12-31"))
	if err != nil || len(due) != 2 {
		t.Errorf("Calendar = %+v, %v; want both waiting jobs on their dates", due, err)
	}
}

// mk is a plain job.
func (f *yearlyFixture) mk(title, stage string, people ...int64) Task {
	f.t.Helper()
	task, err := f.store.Create(f.ctx, CreateInput{Title: title, Stage: stage, PersonIDs: people}, f.owner, day("2026-09-01"))
	if err != nil {
		f.t.Fatal(err)
	}
	return task
}

// Gate 8.51: with waiting jobs interleaved in To do, every move gives exactly the
// result it gives with them absent.
func TestGate851WaitingJobsNeverChangeAMovesResult(t *testing.T) {
	scenario := func(withWaiting bool) []string {
		f := newYearlyFixture(t)
		today := day("2026-10-01")
		ids := map[string]int64{}
		for _, title := range []string{"A", "B", "C", "D", "E"} {
			ids[title] = f.mk(title, StageTodo).ID
			if withWaiting {
				// A waiting job after every visible one, so they sit between them.
				f.yearly("Wait after "+title, "2027-06-01", 6, "2026-09-01")
			}
		}
		other := f.mk("Doing one", StageDoing).ID
		move := func(id int64, in MoveInput) {
			if err := f.store.Move(f.ctx, id, in, f.owner, today); err != nil {
				t.Fatalf("Move: %v", err)
			}
		}
		move(ids["D"], MoveInput{Stage: StageTodo, BeforeID: ids["B"]}) // up
		move(ids["A"], MoveInput{Stage: StageTodo, AfterID: ids["C"]})  // down
		move(ids["E"], MoveInput{Stage: StageTodo, ToBottom: true})     // bottom (already there)
		move(other, MoveInput{Stage: StageTodo, BeforeID: ids["C"]})    // in from another column
		move(ids["B"], MoveInput{Stage: StageDoing, ToBottom: true})    // out to another column
		move(ids["B"], MoveInput{Stage: StageTodo, AfterID: ids["D"]})  // and back
		move(ids["C"], MoveInput{Stage: StageIdea, ToBottom: true})
		move(ids["C"], MoveInput{Stage: StageTodo, BeforeID: ids["D"]})
		if err := f.store.Take(f.ctx, ids["E"], ids["A"], 0, f.owner, today); err != nil {
			t.Fatalf("Take: %v", err)
		}
		// Remove one, and undo it (it goes back beside the one it sat beside).
		if err := f.store.Remove(f.ctx, ids["D"], f.owner, today); err != nil {
			t.Fatal(err)
		}
		if _, err := f.store.Undo(f.ctx, ids["D"], f.owner, today); err != nil {
			t.Fatal(err)
		}
		// A new job lands at the bottom of what people see.
		f.mk("New", StageTodo)
		board, err := f.store.ListBoard(f.ctx, today, BoardFilter{})
		if err != nil {
			t.Fatal(err)
		}
		var order []string
		for _, tk := range board {
			order = append(order, tk.Stage+":"+tk.Title)
		}
		return order
	}
	without, with := scenario(false), scenario(true)
	if !sameStrings(with, without) {
		t.Errorf("with waiting jobs interleaved:\n  %v\nwithout them:\n  %v", with, without)
	}
}

// Gates 8.44, 8.46: the show day puts the job at the bottom of To do, once.
func TestGate844TheShowDayBringsAJobToTheBottomOfToDo(t *testing.T) {
	f := newYearlyFixture(t)
	f.mk("Plain 1", StageTodo)
	f.mk("Plain 2", StageTodo)
	job := f.yearly("Renew", "2026-12-01", 6, "2026-09-01")
	if err := f.store.Assign(f.ctx, job.ID, 0, f.owner, f.owner, day("2026-09-01")); err != nil {
		t.Fatal(err)
	}
	f.mk("Plain 3", StageTodo) // made later, so it sits above the shown job once that arrives? no: below

	if n, err := f.store.ResetYearly(f.ctx, day("2026-10-19")); err != nil || n != 0 {
		t.Fatalf("the day before = %d, %v; want nothing", n, err)
	}
	if got := f.board("2026-10-19"); contains(got, "Renew") {
		t.Errorf("shown the day before: %v", got)
	}
	n, err := f.store.ResetYearly(f.ctx, day("2026-10-20"))
	if err != nil || n != 1 {
		t.Fatalf("show day = %d, %v; want 1", n, err)
	}
	got := f.board("2026-10-20")
	// To do order: the job is last.
	var todo []string
	tasks, _ := f.store.ListBoard(f.ctx, day("2026-10-20"), BoardFilter{})
	for _, tk := range tasks {
		if tk.Stage == StageTodo {
			todo = append(todo, tk.Title)
		}
	}
	if want := []string{"Plain 1", "Plain 2", "Plain 3", "Renew"}; !sameStrings(todo, want) {
		t.Errorf("To do = %v, want %v (board %v)", todo, want, got)
	}
	r := f.row(job.ID)
	if r.resetOn != "2026-10-20" || r.due != "2026-12-01" || r.stage != StageTodo {
		t.Errorf("row = %+v", r)
	}
	hist := f.history(job.ID)
	if !contains(hist, "Comp HQ put this in To do, due Tue 1 Dec 2026 (yearly)") {
		t.Errorf("History = %v", hist)
	}
	// Run again, and on later days: nothing more happens.
	for _, d := range []string{"2026-10-20", "2026-10-21", "2026-11-15"} {
		if n, err := f.store.ResetYearly(f.ctx, day(d)); err != nil || n != 0 {
			t.Errorf("second run on %s = %d, %v", d, n, err)
		}
	}
	if n := countActivity(t, f.store.DB, job.ID, "yearly_shown"); n != 1 {
		t.Errorf("%d yearly_shown rows, want 1", n)
	}
	// The people stay on it.
	if tk, _ := f.store.Get(f.ctx, job.ID, day("2026-10-20")); len(tk.Assignees) != 1 {
		t.Errorf("assignees = %+v", tk.Assignees)
	}
	// "Comp HQ" is never a name in any list (it is filed under the job's creator).
	var names int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM people WHERE name LIKE '%Comp HQ%'`).Scan(&names)
	if names != 0 {
		t.Errorf("a person called Comp HQ exists")
	}
}

// Gate 8.46: an unfinished yearly job is never moved on or doubled up.
func TestGate846AnUnfinishedYearlyJobStaysPut(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew", "2026-12-01", 6, "2026-09-01")
	f.store.ResetYearly(f.ctx, day("2026-10-20"))
	if err := f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDoing, ToBottom: true}, f.owner, day("2026-11-01")); err != nil {
		t.Fatal(err)
	}
	// A year and a bit later it is still the same job, overdue, where it was.
	for _, d := range []string{"2026-12-30", "2027-03-01", "2027-11-01", "2028-06-01"} {
		if n, err := f.store.ResetYearly(f.ctx, day(d)); err != nil || n != 0 {
			t.Errorf("%s: %d, %v", d, n, err)
		}
	}
	r := f.row(job.ID)
	if r.stage != StageDoing || r.due != "2026-12-01" || r.showsFrom != "2026-10-20" {
		t.Errorf("row = %+v, want untouched", r)
	}
	if n := countRows(t, &refFixture{store: f.store}, `SELECT COUNT(*) FROM tasks WHERE title = 'Renew'`); n != 1 {
		t.Errorf("%d jobs called Renew, want 1", n)
	}
}

// Gate 8.45: a finished yearly job stays in Done for 14 days, then is put away
// until next year, counted from its old due date.
func TestGate845PutAwayAfterFourteenDaysInDone(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew", "2026-12-01", 6, "2026-09-01")
	if err := f.store.Assign(f.ctx, job.ID, 0, f.owner, f.owner, day("2026-09-01")); err != nil {
		t.Fatal(err)
	}
	f.store.ResetYearly(f.ctx, day("2026-10-20"))
	step, err := f.store.AddStep(f.ctx, job.ID, "Send the form", f.owner, day("2026-10-21"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetStepDone(f.ctx, job.ID, step.ID, true, f.owner, day("2026-11-02")); err != nil {
		t.Fatal(err)
	}
	// Finished on 3 Nov 2026 (a week before it was due, so due and finished differ).
	if err := f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.owner, day("2026-11-03")); err != nil {
		t.Fatal(err)
	}
	// Still in Done through 17 Nov (14 days), put away on 18 Nov.
	for _, d := range []string{"2026-11-03", "2026-11-10", "2026-11-17"} {
		if n, _ := f.store.ResetYearly(f.ctx, day(d)); n != 0 {
			t.Errorf("%s: put away early", d)
		}
		if got := f.row(job.ID).stage; got != StageDone {
			t.Errorf("%s: stage %s, want done", d, got)
		}
		if !contains(f.board(d), "Renew") {
			t.Errorf("%s: a finished yearly job should still show in Done", d)
		}
	}
	n, err := f.store.ResetYearly(f.ctx, day("2026-11-18"))
	if err != nil || n != 1 {
		t.Fatalf("put-away day = %d, %v", n, err)
	}
	r := f.row(job.ID)
	// One year from the old due date (not from the finishing day).
	if r.stage != StageTodo || r.due != "2027-12-01" || r.showsFrom != "2027-10-20" || r.doneAt != "" || r.resetOn == "2027-10-20" {
		t.Errorf("row = %+v", r)
	}
	if contains(f.board("2026-11-18"), "Renew") {
		t.Errorf("a put-away job is waiting, so not on the Board")
	}
	// Never in Finished tasks.
	fin, _ := f.store.ListFinished(f.ctx, day("2026-12-31"))
	for _, tk := range fin {
		if tk.Title == "Renew" {
			t.Errorf("a yearly job is listed in Finished tasks")
		}
	}
	// Steps unticked, people kept.
	steps, _ := f.store.ListSteps(f.ctx, job.ID, day("2026-11-18"))
	if len(steps) != 1 || steps[0].Done {
		t.Errorf("steps = %+v, want unticked", steps)
	}
	if tk, _ := f.store.Get(f.ctx, job.ID, day("2026-11-18")); len(tk.Assignees) != 1 {
		t.Errorf("people = %+v", tk.Assignees)
	}
	hist := f.history(job.ID)
	if !contains(hist, "Comp HQ put this away until Wed 20 Oct 2027 (yearly)") {
		t.Errorf("History = %v", hist)
	}
	// And it shows again next autumn.
	if n, _ := f.store.ResetYearly(f.ctx, day("2027-10-19")); n != 0 {
		t.Errorf("shown a day early")
	}
	if n, _ := f.store.ResetYearly(f.ctx, day("2027-10-20")); n != 1 {
		t.Errorf("not shown on its day")
	}
}

// Gate 8.47: the NAS was off for more than a year: it catches up once.
func TestGate847CatchesUpOnceAfterTheNASWasOff(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew", "2026-12-01", 6, "2026-09-01")
	f.store.ResetYearly(f.ctx, day("2026-10-20"))
	if err := f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.owner, day("2026-11-03")); err != nil {
		t.Fatal(err)
	}
	// Nobody opens Comp HQ until February 2028.
	n, err := f.store.ResetYearly(f.ctx, day("2028-02-10"))
	if err != nil || n != 1 {
		t.Fatalf("catch-up = %d, %v", n, err)
	}
	r := f.row(job.ID)
	// Moved on one year only (to 2027), and since that date's show day is past it is back in To do.
	if r.due != "2027-12-01" || r.stage != StageTodo || r.resetOn != "2027-10-20" {
		t.Errorf("row = %+v", r)
	}
	if !contains(f.board("2028-02-10"), "Renew") {
		t.Errorf("it should be in To do, overdue")
	}
	if n, _ := f.store.ResetYearly(f.ctx, day("2028-02-11")); n != 0 {
		t.Errorf("a second run did more")
	}
	hist := f.history(job.ID)
	for _, want := range []string{"Comp HQ put this away until Wed 20 Oct 2027 (yearly)", "Comp HQ put this in To do, due Wed 1 Dec 2027 (yearly)"} {
		if !contains(hist, want) {
			t.Errorf("History %v lacks %q", hist, want)
		}
	}
}

// Gate 8.45: a notice so long that the next show day has already passed goes
// straight into To do.
func TestGate845ALongNoticeShowsAtOnce(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew", "2026-12-01", 52, "2026-09-01") // shows from 2025-12-02: at once
	if r := f.row(job.ID); r.showsFrom != "2025-12-02" || r.resetOn != "2025-12-02" {
		t.Errorf("a long notice is shown from the start: %+v", r)
	}
	if !contains(f.board("2026-09-01"), "Renew") {
		t.Errorf("not on the Board")
	}
	if n, _ := f.store.ResetYearly(f.ctx, day("2026-09-02")); n != 0 {
		t.Errorf("shown twice")
	}
	f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.owner, day("2026-12-01"))
	// 15 Dec: put away, new due 2027-12-01, which shows from 2026-12-02, already past: straight into To do.
	n, err := f.store.ResetYearly(f.ctx, day("2026-12-16"))
	if err != nil || n != 1 {
		t.Fatalf("= %d, %v", n, err)
	}
	if r := f.row(job.ID); r.stage != StageTodo || r.due != "2027-12-01" || r.resetOn != "2026-12-02" {
		t.Errorf("row = %+v", r)
	}
	hist := f.history(job.ID)
	if !contains(hist, "Comp HQ put this away until Wed 2 Dec 2026 (yearly)") || !contains(hist, "Comp HQ put this in To do, due Wed 1 Dec 2027 (yearly)") {
		t.Errorf("History = %v", hist)
	}
}

// The next due date: the anchor's month and day in the year after the old due
// date, 29 February clamped to 28 February in other years.
func TestNextYearlyDue(t *testing.T) {
	for _, c := range []struct{ old, anchor, want string }{
		{"2026-12-01", "12-01", "2027-12-01"},
		{"2028-02-29", "02-29", "2029-02-28"},
		{"2029-02-28", "02-29", "2030-02-28"},
		{"2031-02-28", "02-29", "2032-02-29"},
		{"2027-03-01", "03-01", "2028-03-01"},
		{"2026-12-31", "12-31", "2027-12-31"},
		{"2024-02-28", "02-28", "2025-02-28"},
	} {
		if got := nextYearlyDue(c.old, c.anchor); got != c.want {
			t.Errorf("nextYearlyDue(%s, %s) = %s, want %s", c.old, c.anchor, got, c.want)
		}
	}
}

// Unfinished, removed and restored.
func TestYearlyRemovedAndRestoredJobsStayWaiting(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.yearly("Renew", "2026-12-01", 6, "2026-09-01")
	if err := f.store.Remove(f.ctx, job.ID, f.owner, day("2026-09-02")); err != nil {
		t.Fatal(err)
	}
	// A removed job isn't shown on its show day.
	if n, _ := f.store.ResetYearly(f.ctx, day("2026-10-25")); n != 0 {
		t.Errorf("a removed job was shown")
	}
	if err := f.store.Restore(f.ctx, job.ID, f.owner, day("2026-09-03")); err != nil {
		t.Fatal(err)
	}
	if contains(f.board("2026-09-03"), "Renew") {
		t.Errorf("a restored yearly job should wait until its show date")
	}
	if n, _ := f.store.ResetYearly(f.ctx, day("2026-10-25")); n != 1 {
		t.Errorf("a restored job is shown on its day, got %d", n)
	}
}

// Gate 8.40 and B14.4 "Setting it": switching the repeat on an existing job.
func TestUpdateSwitchesBetweenRepeats(t *testing.T) {
	f := newYearlyFixture(t)
	job := f.mk("Plain", StageDoing)
	weeks := 8
	_ = weeks
	yearly := "yearly"
	in := UpdateInput{Title: "Plain", Size: "M", Stage: StageDoing, DueDate: "2026-12-01", Repeat: &yearly, NoticeWeeks: 8}
	if err := f.store.Update(f.ctx, job.ID, in, f.owner, day("2026-09-10")); err != nil {
		t.Fatal(err)
	}
	r := f.row(job.ID)
	if r.repeat != "yearly" || r.anchor != "12-01" || r.notice != 56 || r.showsFrom != "2026-10-06" || r.stage != StageTodo {
		t.Errorf("after switching to yearly: %+v (a yearly job is put in To do)", r)
	}
	if !contains(f.history(job.ID), "Sam made this repeat every year") {
		t.Errorf("History = %v", f.history(job.ID))
	}

	// Changing the due date moves the anchor and the show date.
	in.DueDate = "2027-01-15"
	if err := f.store.Update(f.ctx, job.ID, in, f.owner, day("2026-09-11")); err != nil {
		t.Fatal(err)
	}
	if r := f.row(job.ID); r.anchor != "01-15" || r.showsFrom != "2026-11-20" {
		t.Errorf("after a new due date: %+v", r)
	}
	// Taking the due date away is refused.
	in.DueDate = ""
	if err := f.store.Update(f.ctx, job.ID, in, f.owner, day("2026-09-12")); !errors.Is(err, ErrYearlyNeedsDue) {
		t.Errorf("no due date = %v", err)
	}
	// A long notice shows it at once, at the bottom of To do.
	in.DueDate = "2027-01-15"
	in.NoticeWeeks = 52
	if err := f.store.Update(f.ctx, job.ID, in, f.owner, day("2026-09-13")); err != nil {
		t.Fatal(err)
	}
	if !contains(f.board("2026-09-13"), "Plain") {
		t.Errorf("a long notice should show it at once")
	}
	// Back to none clears the yearly columns, and nothing is left waiting.
	none := "none"
	in.Repeat = &none
	if err := f.store.Update(f.ctx, job.ID, in, f.owner, day("2026-09-14")); err != nil {
		t.Fatal(err)
	}
	r = f.row(job.ID)
	if r.repeat != "none" || r.anchor != "" || r.notice != 0 || r.showsFrom != "" {
		t.Errorf("after switching off: %+v", r)
	}
	if !contains(f.history(job.ID), "Sam stopped this repeating") {
		t.Errorf("History = %v", f.history(job.ID))
	}
	// Weekly to yearly clears the weekly reset date, so a weekly job's last
	// Saturday can never be mistaken for a showing.
	w, err := f.store.Create(f.ctx, CreateInput{Title: "Weekly", Weekly: true}, f.owner, day("2026-10-07"))
	if err != nil {
		t.Fatal(err)
	}
	if f.row(w.ID).resetOn == "" {
		t.Fatalf("a weekly job has a reset date")
	}
	in2 := UpdateInput{Title: "Weekly", Size: "M", Stage: StageTodo, DueDate: "2027-05-01", Repeat: &yearly, NoticeWeeks: 6}
	if err := f.store.Update(f.ctx, w.ID, in2, f.owner, day("2026-10-08")); err != nil {
		t.Fatal(err)
	}
	if r := f.row(w.ID); r.repeat != "yearly" || r.resetOn != "" {
		t.Errorf("weekly -> yearly: %+v", r)
	}
}

// Gate 8.53 (pass half): a database as 1.5.0 left it has none of the new values,
// and every list and the yearly pass treat its jobs exactly as before.
func TestGate853AJobFrom150HasNoYearlyValuesAndIsNeverWaiting(t *testing.T) {
	f := newYearlyFixture(t)
	old := f.mk("Made by 1.5.0", StageTodo)
	// A yearly row with nothing in the new columns (as 1.5.0 can't write one, but a
	// row 1.6 made and a rollback then edited might be): never waiting.
	if _, err := f.store.DB.Exec(`UPDATE tasks SET repeat = 'yearly', shows_from = NULL WHERE id = ?`, old.ID); err != nil {
		t.Fatal(err)
	}
	if !contains(f.board("2026-09-02"), "Made by 1.5.0") {
		t.Errorf("a job with no show date must be shown")
	}
	if n, err := f.store.ResetYearly(f.ctx, day("2026-09-02")); err != nil || n != 0 {
		t.Errorf("the pass touched it: %d, %v", n, err)
	}
	// 1.5.0's own reading of any such job: it ignores the new columns, reads
	// repeat <> 'weekly' as an ordinary job, and so shows a waiting one in To do.
	waiting := f.yearly("Waiting", "2027-06-01", 6, "2026-09-01")
	var n int
	if err := f.store.DB.QueryRow(
		`SELECT COUNT(*) FROM tasks WHERE id = ? AND removed_at IS NULL AND repeat <> 'weekly' AND stage = 'todo'`, waiting.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("1.5.0's reading = %d, %v", n, err)
	}
	_ = time.Now
}

// Gate 8.53 (pass half): a database as 1.5.0 left it, migrated by 1.6.0, keeps
// every job exactly as it was, and none of them is waiting.
func TestGate853Migrating150DatabaseKeepsEveryJob(t *testing.T) {
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	for _, section := range []string{"app", "people"} {
		ms, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.RunMigrations(sqlDB, "1.5.0", ms); err != nil {
			t.Fatal(err)
		}
	}
	all, err := db.LoadMigrations(comphq.Migrations, "tasks")
	if err != nil {
		t.Fatal(err)
	}
	var upTo150 []db.Migration
	for _, m := range all {
		if m.Version <= 4 { // 1.5.0 had tasks migrations 0001 to 0004
			upTo150 = append(upTo150, m)
		}
	}
	if len(upTo150) != 4 {
		t.Fatalf("1.5.0's tasks migrations = %d, want 4", len(upTo150))
	}
	if err := db.RunMigrations(sqlDB, "1.5.0", upTo150); err != nil {
		t.Fatal(err)
	}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()
	// Jobs as 1.5.0 wrote them (its code can't run here, so its rows are inserted by hand).
	insert := func(title, stage, repeat string) int64 {
		res, err := sqlDB.Exec(`INSERT INTO tasks (title, notes, size, stage, position, created_by, created_at, updated_by, updated_at, repeat)
			VALUES (?, '', 'M', ?, 1, ?, '2026-09-01T00:00:00Z', ?, '2026-09-01T00:00:00Z', ?)`, title, stage, sam, sam, repeat)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	plainID := insert("Plain", StageTodo, "none")
	weeklyID := insert("Weekly", StageTodo, "weekly")
	insert("Done", StageDone, "none")

	// 1.6.0 starts up: the add-only migrations run, and nothing about the jobs changes.
	if err := db.RunMigrations(sqlDB, "1.6.0", all); err != nil {
		t.Fatalf("migrating: %v", err)
	}
	store := &Store{DB: sqlDB}
	board, err := store.ListBoard(ctx, day("2026-09-02"), BoardFilter{})
	if err != nil || len(board) != 3 {
		t.Fatalf("board after migrating = %+v, %v", board, err)
	}
	for _, id := range []int64{plainID, weeklyID} {
		var shows, anchor *string
		var notice *int
		if err := sqlDB.QueryRow(`SELECT shows_from, repeat_anchor, notice_days FROM tasks WHERE id = ?`, id).Scan(&shows, &anchor, &notice); err != nil {
			t.Fatal(err)
		}
		if shows != nil || anchor != nil || notice != nil {
			t.Errorf("job %d has yearly values after migrating: %v %v %v", id, shows, anchor, notice)
		}
	}
	if n, err := store.ResetYearly(ctx, day("2026-09-02")); err != nil || n != 0 {
		t.Errorf("the yearly pass changed something: %d, %v", n, err)
	}
	if _, err := store.ResetWeekly(ctx, day("2026-09-02")); err != nil {
		t.Fatal(err)
	}
	// Upgrading again, after the same migrations have run, changes nothing.
	if err := db.RunMigrations(sqlDB, "1.6.0", all); err != nil {
		t.Fatalf("running them again: %v", err)
	}
}
