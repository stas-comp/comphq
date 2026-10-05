package tasks

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// SPEC gates 7.31-7.37, 7.39 (B13.5, D-95): weekly jobs and the Saturday reset.
// Dates: 2026-10-07 is a Wednesday, so Saturday is 10 Oct, then 17, then 24.

func day(s string) time.Time {
	t, err := time.ParseInLocation(dateLayout, s, time.UTC)
	if err != nil {
		panic(err)
	}
	return t
}

type weeklyFixture struct {
	t     *testing.T
	store *Store
	ctx   context.Context
	owner int64
}

func newWeeklyFixture(t *testing.T) *weeklyFixture {
	t.Helper()
	db := openTestDB(t)
	return &weeklyFixture{t: t, store: &Store{DB: db}, ctx: context.Background(), owner: testPerson(t, db, "Sam")}
}

// weekly makes a weekly job on `created`, in `stage`.
func (f *weeklyFixture) weekly(title, stage, created string) Task {
	f.t.Helper()
	task, err := f.store.Create(f.ctx, CreateInput{Title: title, Stage: stage, Weekly: true}, f.owner, day(created))
	if err != nil {
		f.t.Fatal(err)
	}
	return task
}

func (f *weeklyFixture) row(id int64) (stage string, position int, due, resetOn, doneAt string) {
	f.t.Helper()
	var d, r, da *string
	if err := f.store.DB.QueryRow(`SELECT stage, position, due_date, repeat_reset_on, done_at FROM tasks WHERE id = ?`, id).Scan(&stage, &position, &d, &r, &da); err != nil {
		f.t.Fatal(err)
	}
	return stage, position, deref(d), deref(r), deref(da)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (f *weeklyFixture) resets(id int64) int {
	var n int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM task_activity WHERE task_id = ? AND action = 'weekly_reset'`, id).Scan(&n)
	return n
}

func (f *weeklyFixture) version() int64 {
	v, _ := f.store.Version(f.ctx)
	return v
}

// Gate 7.31: ticking it on a job with no due date sets This Saturday (today, on a Saturday).
func TestWeeklyOnSetsDueDateToThisSaturday(t *testing.T) {
	f := newWeeklyFixture(t)
	for _, c := range []struct{ created, wantDue string }{
		{"2026-10-07", "2026-10-10"}, // a Wednesday: the coming Saturday
		{"2026-10-10", "2026-10-10"}, // a Saturday: today
		{"2026-10-11", "2026-10-17"}, // a Sunday
	} {
		task := f.weekly("Clean the kitchen "+c.created, StageTodo, c.created)
		_, _, due, resetOn, _ := f.row(task.ID)
		if due != c.wantDue {
			t.Errorf("created %s: due = %q, want %q", c.created, due, c.wantDue)
		}
		// It starts at the Saturday on or before today, so it doesn't reset straight away.
		if want := saturdayOnOrBefore(day(c.created)).Format(dateLayout); resetOn != want {
			t.Errorf("created %s: repeat_reset_on = %q, want %q", c.created, resetOn, want)
		}
	}
	// A job that already has a due date keeps it.
	keep, err := f.store.Create(f.ctx, CreateInput{Title: "Has a date", Stage: StageTodo, DueDate: "2026-11-01", Weekly: true}, f.owner, day("2026-10-07"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, due, _, _ := f.row(keep.ID); due != "2026-11-01" {
		t.Errorf("due = %q, want the date it already had", due)
	}
}

// Gate 7.32: at the start of Saturday a weekly job in Done goes back to the
// bottom of To do, due that Saturday, same people, steps unticked.
func TestResetWeeklyPutsADoneJobBackInTodo(t *testing.T) {
	f := newWeeklyFixture(t)
	alex := testPerson(t, f.store.DB, "Alex")
	other := createNamed(t, f.store, f.ctx, f.owner, "Another job", StageTodo)
	job, err := f.store.Create(f.ctx, CreateInput{Title: "Clean", Stage: StageTodo, Weekly: true, PersonIDs: []int64{alex}}, f.owner, day("2026-10-07"))
	if err != nil {
		t.Fatal(err)
	}
	step, _ := f.store.AddStep(f.ctx, job.ID, "Wipe the bench", f.owner, day("2026-10-07"))
	if err := f.store.SetStepDone(f.ctx, job.ID, step.ID, true, f.owner, day("2026-10-08")); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.owner, day("2026-10-09")); err != nil {
		t.Fatal(err)
	}
	later := createNamed(t, f.store, f.ctx, f.owner, "Added after", StageTodo)
	_ = other

	// Friday: nothing yet.
	if n, err := f.store.ResetWeekly(f.ctx, day("2026-10-09")); err != nil || n != 0 {
		t.Fatalf("Friday: ResetWeekly = %d, %v, want 0", n, err)
	}
	if stage, _, _, _, _ := f.row(job.ID); stage != StageDone {
		t.Fatalf("Friday: stage = %q, still Done", stage)
	}

	before := f.version()
	n, err := f.store.ResetWeekly(f.ctx, day("2026-10-10"))
	if err != nil || n != 1 {
		t.Fatalf("Saturday: ResetWeekly = %d, %v, want 1", n, err)
	}
	stage, pos, due, resetOn, doneAt := f.row(job.ID)
	if stage != StageTodo || due != "2026-10-10" || resetOn != "2026-10-10" || doneAt != "" {
		t.Errorf("after the reset: stage=%q due=%q resetOn=%q doneAt=%q", stage, due, resetOn, doneAt)
	}
	if _, laterPos, _, _, _ := f.row(later.ID); pos != laterPos+1 {
		t.Errorf("position %d, want directly after the last job in To do (%d)", pos, laterPos)
	}
	got, _ := f.store.Get(f.ctx, job.ID, day("2026-10-10"))
	if len(got.Assignees) != 1 || got.Assignees[0].PersonID != alex {
		t.Errorf("assignees = %+v, want Alex still on it", got.Assignees)
	}
	steps, _ := f.store.ListSteps(f.ctx, job.ID, day("2026-10-10"))
	if len(steps) != 1 || steps[0].Done {
		t.Errorf("steps = %+v, want the step still there but unticked", steps)
	}
	if f.version() != before+1 {
		t.Errorf("change counter %d -> %d, want one bump", before, f.version())
	}

	// Gate 7.36: History reads "Comp HQ put this back…", under the creator's id, not a made-up person.
	acts, _ := f.store.ListActivity(f.ctx, job.ID)
	var texts []string
	for _, a := range acts {
		texts = append(texts, a.Text)
	}
	if all := strings.Join(texts, "\n"); !strings.Contains(all, "Comp HQ put this back in To do for Sat 10 Oct (weekly)") {
		t.Errorf("History = %q", all)
	}
	var people int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM people WHERE name LIKE '%Comp HQ%'`).Scan(&people)
	if people != 0 {
		t.Errorf("a person called Comp HQ exists")
	}
}

// Saturday twice: the second is a no-op, and the change counter doesn't move.
func TestResetWeeklyOnTheSameSaturdayTwiceIsANoOp(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Clean", StageDone, "2026-10-07")
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-10")); n != 1 {
		t.Fatal("first run did not reset")
	}
	v := f.version()
	// Someone finishes it again during the day; a later request must not reset it.
	if err := f.store.Move(f.ctx, job.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.owner, day("2026-10-10")); err != nil {
		t.Fatal(err)
	}
	v = f.version()
	if n, err := f.store.ResetWeekly(f.ctx, day("2026-10-10")); err != nil || n != 0 {
		t.Fatalf("second run = %d, %v, want 0", n, err)
	}
	if stage, _, _, _, _ := f.row(job.ID); stage != StageDone {
		t.Errorf("stage = %q: a second request on the same Saturday reset it again", stage)
	}
	if f.version() != v {
		t.Errorf("a no-op bumped the change counter")
	}
	// Any day until the next Saturday is the same Saturday.
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-16")); n != 0 {
		t.Errorf("Friday after: reset %d", n)
	}
	if f.resets(job.ID) != 1 {
		t.Errorf("%d reset rows, want 1", f.resets(job.ID))
	}
}

// Gate 7.35: switched off at midnight, or off for weeks: one reset, due the latest Saturday.
func TestResetWeeklyAfterTwoMissedSaturdaysResetsOnceDueTheLatest(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Clean", StageDone, "2026-10-07")
	n, err := f.store.ResetWeekly(f.ctx, day("2026-10-24")) // 10 and 17 were missed
	if err != nil || n != 1 {
		t.Fatalf("ResetWeekly = %d, %v, want 1", n, err)
	}
	stage, _, due, resetOn, _ := f.row(job.ID)
	if stage != StageTodo || due != "2026-10-24" || resetOn != "2026-10-24" {
		t.Errorf("stage=%q due=%q resetOn=%q, want To do, due the latest Saturday", stage, due, resetOn)
	}
	if f.resets(job.ID) != 1 {
		t.Errorf("%d reset rows, want 1 however many Saturdays were missed", f.resets(job.ID))
	}
}

// Gate 7.33: not finished by Saturday: left alone, old due date, never doubled up.
func TestResetWeeklyLeavesAnUnfinishedJobAlone(t *testing.T) {
	f := newWeeklyFixture(t)
	todo := f.weekly("Still to do", StageTodo, "2026-10-07")
	doing := f.weekly("In progress", StageDoing, "2026-10-07")
	idea := f.weekly("An idea", StageIdea, "2026-10-07")
	v := f.version()
	if n, err := f.store.ResetWeekly(f.ctx, day("2026-10-10")); err != nil || n != 0 {
		t.Fatalf("ResetWeekly = %d, %v, want 0 (nothing was Done)", n, err)
	}
	for _, c := range []struct {
		job   Task
		stage string
	}{{todo, StageTodo}, {doing, StageDoing}, {idea, StageIdea}} {
		stage, _, due, resetOn, _ := f.row(c.job.ID)
		if stage != c.stage || due != "2026-10-10" {
			t.Errorf("%s: stage=%q due=%q, want unchanged", c.job.Title, stage, due)
		}
		if resetOn != "2026-10-10" {
			t.Errorf("%s: resetOn = %q, want marked as reset for this Saturday", c.job.Title, resetOn)
		}
		if f.resets(c.job.ID) != 0 {
			t.Errorf("%s has a reset row", c.job.Title)
		}
	}
	if f.version() != v {
		t.Errorf("the change counter moved though nothing visible changed")
	}
	// Next week it is overdue and still one card.
	got, _ := f.store.Get(f.ctx, todo.ID, day("2026-10-17"))
	if !got.Overdue {
		t.Errorf("a weekly job left unfinished should read OVERDUE the week after")
	}
	var copies int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE title = 'Still to do'`).Scan(&copies)
	if copies != 1 {
		t.Errorf("%d cards, want 1: nothing is ever copied", copies)
	}
}

// Gate 7.37: a removed weekly job doesn't reset; restored (or undone) it repeats again.
func TestRemovedWeeklyJobDoesNotResetUntilRestored(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Clean", StageDone, "2026-10-07")
	if err := f.store.Remove(f.ctx, job.ID, f.owner, day("2026-10-08")); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-10")); n != 0 {
		t.Fatalf("a removed job was reset")
	}
	if stage, _, _, resetOn, _ := f.row(job.ID); stage != StageDone || resetOn != "2026-10-03" {
		t.Errorf("removed job changed: stage=%q resetOn=%q", stage, resetOn)
	}
	if err := f.store.Restore(f.ctx, job.ID, f.owner, day("2026-10-11")); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-17")); n != 1 {
		t.Errorf("after Restore the job should repeat again, reset %d", n)
	}
}

// Gate 7.37: unticking it makes it an ordinary job again.
func TestTurningWeeklyOffMakesItOrdinary(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Clean", StageDone, "2026-10-07")
	off := false
	if err := f.store.Update(f.ctx, job.ID, UpdateInput{Title: "Clean", Size: "M", Stage: StageDone, DueDate: "2026-10-10", Weekly: &off}, f.owner, day("2026-10-08")); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-10")); n != 0 {
		t.Errorf("an ordinary job was reset")
	}
	// And a nil Weekly leaves things as they are.
	on := true
	if err := f.store.Update(f.ctx, job.ID, UpdateInput{Title: "Clean", Size: "M", Stage: StageDone, DueDate: "2026-10-10", Weekly: &on}, f.owner, day("2026-10-11")); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Update(f.ctx, job.ID, UpdateInput{Title: "Clean", Size: "M", Stage: StageDone, DueDate: "2026-10-10"}, f.owner, day("2026-10-12")); err != nil {
		t.Fatal(err)
	}
	var repeat string
	f.store.DB.QueryRow(`SELECT repeat FROM tasks WHERE id = ?`, job.ID).Scan(&repeat)
	if repeat != repeatWeekly {
		t.Errorf("repeat = %q, want weekly (a nil Weekly means unchanged)", repeat)
	}
	acts, _ := f.store.ListActivity(f.ctx, job.ID)
	var all []string
	for _, a := range acts {
		all = append(all, a.Text)
	}
	text := strings.Join(all, "\n")
	if !strings.Contains(text, "stopped this repeating") || !strings.Contains(text, "made this repeat every week") {
		t.Errorf("History = %q, want both the on and off rows", text)
	}
}

// Gate 7.34: a weekly job stays in Done however long it has been there, and never goes to Finished tasks.
func TestWeeklyJobNeverLeavesDoneOrReachesFinishedTasks(t *testing.T) {
	f := newWeeklyFixture(t)
	weeklyJob := f.weekly("Weekly", StageDone, "2026-09-01")
	plain := createNamed(t, f.store, f.ctx, f.owner, "Plain", StageDone)
	long := day("2026-10-10")
	setDoneAt(t, f.store, weeklyJob.ID, long.AddDate(0, 0, -40))
	setDoneAt(t, f.store, plain.ID, long.AddDate(0, 0, -40))

	board, _ := f.store.ListBoard(f.ctx, long, BoardFilter{})
	onBoard := map[string]bool{}
	for _, tk := range board {
		onBoard[tk.Title] = true
	}
	if !onBoard["Weekly"] || onBoard["Plain"] {
		t.Errorf("board has Weekly=%v Plain=%v, want the weekly job kept in Done and the ordinary one tucked away", onBoard["Weekly"], onBoard["Plain"])
	}
	finished, _ := f.store.ListFinished(f.ctx, long)
	for _, tk := range finished {
		if tk.Title == "Weekly" {
			t.Errorf("a weekly job is in Finished tasks")
		}
	}
	// A weekly job finished and tucked away while rolled back to 1.3.0 is still
	// stage done, so the first reset after upgrading brings it back (gate 7.39).
	if n, _ := f.store.ResetWeekly(f.ctx, day("2026-10-10")); n != 1 {
		t.Errorf("the long-finished weekly job was not brought back (%d)", n)
	}
}

// Two requests at once: one reset, one History row.
func TestResetWeeklyTwoCallsAtOnce(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Clean", StageDone, "2026-10-07")
	var wg sync.WaitGroup
	results := make([]int, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := f.store.ResetWeekly(f.ctx, day("2026-10-10"))
			if err != nil {
				t.Errorf("call %d: %v", i, err)
			}
			results[i] = n
		}(i)
	}
	wg.Wait()
	if results[0]+results[1] != 1 {
		t.Errorf("resets = %v, want exactly one between them", results)
	}
	if f.resets(job.ID) != 1 {
		t.Errorf("%d History rows, want 1", f.resets(job.ID))
	}
}

// A database as 1.3.0 leaves it: the columns exist (the migration ran) but the
// job was finished and tucked away while rolled back, 20 days ago.
func TestResetWeeklyOnA13ShapedDatabase(t *testing.T) {
	f := newWeeklyFixture(t)
	job := f.weekly("Weekly", StageTodo, "2026-09-16")
	// 1.3.0 finished it and, 20 days on, hid it from the board.
	if _, err := f.store.DB.Exec(`UPDATE tasks SET stage = 'done', done_at = ?, repeat_reset_on = '2026-09-12' WHERE id = ?`,
		day("2026-09-20").UTC().Format(time.RFC3339), job.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := f.store.ResetWeekly(f.ctx, day("2026-10-10")); err != nil || n != 1 {
		t.Fatalf("ResetWeekly = %d, %v, want the job brought back", n, err)
	}
	if stage, _, due, _, _ := f.row(job.ID); stage != StageTodo || due != "2026-10-10" {
		t.Errorf("stage=%q due=%q", stage, due)
	}
}
