package tasks

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
)

// stepFixture is a job and two people to act on it.
type stepFixture struct {
	sqlDB *sql.DB
	store *Store
	task  Task
	sam   int64
	alex  int64
}

func newStepFixture(t *testing.T) stepFixture {
	t.Helper()
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	task := createNamed(t, store, context.Background(), sam, "Concert", StageTodo)
	return stepFixture{sqlDB: sqlDB, store: store, task: task, sam: sam, alex: alex}
}

func (f stepFixture) add(t *testing.T, text string) Step {
	t.Helper()
	st, err := f.store.AddStep(context.Background(), f.task.ID, text, f.sam, fixedNow)
	if err != nil {
		t.Fatalf("AddStep(%q): %v", text, err)
	}
	return st
}

// order lists the live steps' words in display order.
func (f stepFixture) order(t *testing.T) []string {
	t.Helper()
	steps, err := f.store.ListSteps(context.Background(), f.task.ID, fixedNow)
	if err != nil {
		t.Fatalf("ListSteps: %v", err)
	}
	var words []string
	for i, st := range steps {
		if st.Position != i+1 {
			t.Errorf("step %q has position %d, want %d (positions must stay 1..n)", st.Text, st.Position, i+1)
		}
		words = append(words, st.Text)
	}
	return words
}

func sameWords(a, b []string) bool {
	return strings.Join(a, "|") == strings.Join(b, "|")
}

func (f stepFixture) expectOrder(t *testing.T, want ...string) {
	t.Helper()
	if got := f.order(t); !sameWords(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func (f stepFixture) version(t *testing.T) int64 {
	t.Helper()
	v, err := f.store.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	return v
}

func (f stepFixture) activityActions(t *testing.T) []string {
	t.Helper()
	rows, err := f.sqlDB.Query(`SELECT action FROM task_activity WHERE task_id = ? ORDER BY id`, f.task.ID)
	if err != nil {
		t.Fatalf("activity: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, a)
	}
	return actions
}

func TestAddStepAppendsAtTheEndAndTidiesTheWords(t *testing.T) {
	f := newStepFixture(t)
	first := f.add(t, "Print exam papers")
	f.add(t, "  Book the hall \n")
	f.add(t, "Email the\tparents")

	f.expectOrder(t, "Print exam papers", "Book the hall", "Email the parents")
	if first.Position != 1 || first.Done || first.Removed {
		t.Errorf("first step = %+v, want position 1, not done, not removed", first)
	}
}

func TestAddStepRefusesEmptyWordsAndAJobThatIsNotThere(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	for _, blank := range []string{"", "   ", "\n\t"} {
		if _, err := f.store.AddStep(ctx, f.task.ID, blank, f.sam, fixedNow); err != ErrEmptyStep {
			t.Errorf("AddStep(%q) err = %v, want ErrEmptyStep", blank, err)
		}
	}
	if _, err := f.store.AddStep(ctx, 9999, "Orphan", f.sam, fixedNow); err != ErrTaskNotFound {
		t.Errorf("AddStep on a missing job err = %v, want ErrTaskNotFound", err)
	}
	if err := f.store.Remove(ctx, f.task.ID, f.sam, fixedNow); err != nil {
		t.Fatalf("Remove task: %v", err)
	}
	if _, err := f.store.AddStep(ctx, f.task.ID, "Too late", f.sam, fixedNow); err != ErrTaskNotFound {
		t.Errorf("AddStep on a removed job err = %v, want ErrTaskNotFound", err)
	}
}

// Gate 5.13: a step holds up to 200 characters — counted as characters, not
// bytes, so accented and emoji text isn't cut short.
func TestStepLengthLimitIsInCharactersAndIsEnforcedOnAddAndRename(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()

	exactly := strings.Repeat("é", MaxStepChars)
	st, err := f.store.AddStep(ctx, f.task.ID, exactly, f.sam, fixedNow)
	if err != nil {
		t.Fatalf("a step of exactly %d characters was refused: %v", MaxStepChars, err)
	}
	tooLong := exactly + "x"
	if _, err := f.store.AddStep(ctx, f.task.ID, tooLong, f.sam, fixedNow); err != ErrStepTooLong {
		t.Errorf("AddStep(201 chars) err = %v, want ErrStepTooLong", err)
	}
	if err := f.store.RenameStep(ctx, f.task.ID, st.ID, tooLong, f.sam, fixedNow); err != ErrStepTooLong {
		t.Errorf("RenameStep(201 chars) err = %v, want ErrStepTooLong", err)
	}
	f.expectOrder(t, exactly)
}

// Gate 5.13: a job holds up to 50 steps. Removed steps don't count, so a
// removal makes room again.
func TestJobStepLimitCountsOnlyLiveSteps(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	var first Step
	for i := 0; i < MaxSteps; i++ {
		st := f.add(t, "Step "+string(rune('A'+i%26))+strings.Repeat("!", i/26))
		if i == 0 {
			first = st
		}
	}
	if _, err := f.store.AddStep(ctx, f.task.ID, "One too many", f.sam, fixedNow); err != ErrTooManySteps {
		t.Fatalf("AddStep #51 err = %v, want ErrTooManySteps", err)
	}
	if got := len(f.order(t)); got != MaxSteps {
		t.Fatalf("after a refused add there are %d steps, want %d", got, MaxSteps)
	}

	if err := f.store.RemoveStep(ctx, f.task.ID, first.ID, f.sam, fixedNow); err != nil {
		t.Fatalf("RemoveStep: %v", err)
	}
	f.add(t, "Fits now")

	// Restoring into a full list is refused too — the limit can't be got
	// round by removing, adding, and then undoing.
	if err := f.store.RestoreStep(ctx, f.task.ID, first.ID, f.sam, fixedNow); err != ErrTooManySteps {
		t.Errorf("RestoreStep into a full job err = %v, want ErrTooManySteps", err)
	}
}

func TestStepLimitIsPerJob(t *testing.T) {
	f := newStepFixture(t)
	other := createNamed(t, f.store, context.Background(), f.sam, "Sports day", StageTodo)
	for i := 0; i < MaxSteps; i++ {
		f.add(t, "Step")
	}
	if _, err := f.store.AddStep(context.Background(), other.ID, "Mine", f.sam, fixedNow); err != nil {
		t.Errorf("a second job can't take its first step because the first is full: %v", err)
	}
}

func TestTickCarriesWhoAndWhenAndUntickClearsBoth(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	st := f.add(t, "Book the hall")

	tickedAt := time.Date(2026, 9, 19, 9, 30, 0, 0, time.UTC)
	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, true, f.alex, tickedAt); err != nil {
		t.Fatalf("tick: %v", err)
	}
	steps, _ := f.store.ListSteps(ctx, f.task.ID, fixedNow)
	if len(steps) != 1 || !steps[0].Done {
		t.Fatalf("steps after tick = %+v, want one done step", steps)
	}
	if steps[0].DoneByName != "Alex" {
		t.Errorf("DoneByName = %q, want Alex", steps[0].DoneByName)
	}
	if want := "Sat 19 Sep"; steps[0].DoneDate != want {
		t.Errorf("DoneDate = %q, want %q (the short form, gate 5.03)", steps[0].DoneDate, want)
	}

	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, false, f.sam, fixedNow); err != nil {
		t.Fatalf("untick: %v", err)
	}
	steps, _ = f.store.ListSteps(ctx, f.task.ID, fixedNow)
	if steps[0].Done || steps[0].DoneByName != "" || steps[0].DoneDate != "" {
		t.Errorf("step after untick = %+v, want who and when cleared", steps[0])
	}
}

// Gate 5.12: ticking what somebody else just ticked is not an error, and the
// first person's name and time stay.
func TestTickingAnAlreadyTickedStepKeepsTheFirstPerson(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	st := f.add(t, "Book the hall")

	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, true, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	before := f.version(t)
	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, true, f.sam, fixedNow.Add(time.Hour)); err != nil {
		t.Fatalf("second tick returned an error: %v", err)
	}
	steps, _ := f.store.ListSteps(ctx, f.task.ID, fixedNow)
	if steps[0].DoneByName != "Alex" || !steps[0].DoneAt.Equal(fixedNow) {
		t.Errorf("after a second tick: by %q at %v, want Alex at %v", steps[0].DoneByName, steps[0].DoneAt, fixedNow)
	}
	if f.version(t) != before {
		t.Error("a tick that changed nothing still bumped the change counter")
	}
	// And unticking what is not ticked is equally quiet.
	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, false, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, false, f.sam, fixedNow); err != nil {
		t.Errorf("unticking an unticked step err = %v, want nil", err)
	}
}

func TestRenameChangesTheWordsInPlace(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	two := f.add(t, "Two")
	f.add(t, "Three")

	if err := f.store.RenameStep(ctx, f.task.ID, two.ID, "  Second  ", f.alex, fixedNow); err != nil {
		t.Fatalf("RenameStep: %v", err)
	}
	f.expectOrder(t, "One", "Second", "Three")
	if err := f.store.RenameStep(ctx, f.task.ID, two.ID, "   ", f.alex, fixedNow); err != ErrEmptyStep {
		t.Errorf("rename to nothing err = %v, want ErrEmptyStep", err)
	}
	f.expectOrder(t, "One", "Second", "Three")
}

func TestRemoveClosesTheGapAndKeepsTheRow(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	two := f.add(t, "Two")
	f.add(t, "Three")

	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.alex, fixedNow); err != nil {
		t.Fatalf("RemoveStep: %v", err)
	}
	f.expectOrder(t, "One", "Three")

	// SPEC A3: never permanently deleted. The row is still there, marked.
	var removedAt sql.NullString
	if err := f.sqlDB.QueryRow(`SELECT removed_at FROM task_checklist_items WHERE id = ?`, two.ID).Scan(&removedAt); err != nil {
		t.Fatalf("the removed step's row is gone: %v", err)
	}
	if !removedAt.Valid {
		t.Error("removed step has no removed_at")
	}
	all, err := f.store.ListStepsWithRemoved(ctx, f.task.ID, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || !all[2].Removed || all[2].Text != "Two" {
		t.Errorf("ListStepsWithRemoved = %+v, want the two live steps and then the removed one", all)
	}
}

func TestRestorePutsTheStepBackWhereItWas(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	two := f.add(t, "Two")
	f.add(t, "Three")

	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RestoreStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Fatalf("RestoreStep: %v", err)
	}
	f.expectOrder(t, "One", "Two", "Three")

	// Restoring a step that is already back is quiet, not an error, and
	// doesn't duplicate it.
	if err := f.store.RestoreStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Errorf("second restore err = %v, want nil", err)
	}
	f.expectOrder(t, "One", "Two", "Three")
}

// SPEC B4: undoing "pushes later steps down if that position has since been
// taken", and a position past the end of a shorter list just goes last.
func TestRestoreIntoATakenPositionPushesLaterStepsDown(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	two := f.add(t, "Two")
	f.add(t, "Three")

	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	// While it is away somebody adds a step, which takes the freed position 3.
	f.add(t, "Four")
	f.expectOrder(t, "One", "Three", "Four")

	if err := f.store.RestoreStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "One", "Two", "Three", "Four")
}

func TestRestoreWhenTheListHasShrunkGoesToTheEnd(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	f.add(t, "Two")
	three := f.add(t, "Three")

	steps, _ := f.store.ListSteps(ctx, f.task.ID, fixedNow)
	if err := f.store.RemoveStep(ctx, f.task.ID, three.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	// Both earlier steps go too, so position 3 is beyond the end.
	for _, st := range steps[:2] {
		if err := f.store.RemoveStep(ctx, f.task.ID, st.ID, f.sam, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.store.RestoreStep(ctx, f.task.ID, three.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "Three")
}

func TestMoveUpAndDownSwapNeighboursAndStopAtTheEnds(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	one := f.add(t, "One")
	f.add(t, "Two")
	three := f.add(t, "Three")

	if err := f.store.MoveStep(ctx, f.task.ID, three.ID, MoveUp, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "One", "Three", "Two")
	if err := f.store.MoveStep(ctx, f.task.ID, one.ID, MoveDown, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "Three", "One", "Two")

	// At an end, moving further is a quiet no-op that bumps nothing.
	before := f.version(t)
	if err := f.store.MoveStep(ctx, f.task.ID, three.ID, MoveUp, f.sam, fixedNow); err != nil {
		t.Errorf("moving the first step up err = %v, want nil", err)
	}
	f.expectOrder(t, "Three", "One", "Two")
	if f.version(t) != before {
		t.Error("a move that changed nothing bumped the change counter")
	}
	if err := f.store.MoveStep(ctx, f.task.ID, one.ID, "sideways", f.sam, fixedNow); err != ErrInvalidDirection {
		t.Errorf("MoveStep(sideways) err = %v, want ErrInvalidDirection", err)
	}
}

func TestMoveStepToPlacesAtAPositionAndClamps(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	one := f.add(t, "One")
	f.add(t, "Two")
	f.add(t, "Three")
	f.add(t, "Four")

	if err := f.store.MoveStepTo(ctx, f.task.ID, one.ID, 3, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "Two", "Three", "One", "Four")
	if err := f.store.MoveStepTo(ctx, f.task.ID, one.ID, 99, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "Two", "Three", "Four", "One")
	if err := f.store.MoveStepTo(ctx, f.task.ID, one.ID, -5, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.expectOrder(t, "One", "Two", "Three", "Four")
}

// The ordering has to hold up after a mixed run of adds, moves, removals and
// restores: always 1..n, no duplicates, no holes.
func TestOrderingStaysContiguousAfterAMixedSequence(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	a := f.add(t, "A")
	b := f.add(t, "B")
	c := f.add(t, "C")
	d := f.add(t, "D")
	e := f.add(t, "E")

	steps := []func() error{
		func() error { return f.store.MoveStep(ctx, f.task.ID, e.ID, MoveUp, f.sam, fixedNow) },      // A B C E D
		func() error { return f.store.RemoveStep(ctx, f.task.ID, b.ID, f.sam, fixedNow) },            // A C E D
		func() error { return f.store.MoveStepTo(ctx, f.task.ID, a.ID, 3, f.sam, fixedNow) },         // C E A D
		func() error { return f.store.RemoveStep(ctx, f.task.ID, d.ID, f.sam, fixedNow) },            // C E A
		func() error { return f.store.RestoreStep(ctx, f.task.ID, b.ID, f.sam, fixedNow) },           // B was 2 → C B E A
		func() error { _, err := f.store.AddStep(ctx, f.task.ID, "F", f.sam, fixedNow); return err }, // C B E A F
		func() error { return f.store.MoveStep(ctx, f.task.ID, c.ID, MoveDown, f.sam, fixedNow) },    // B C E A F
	}
	for i, do := range steps {
		if err := do(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		f.order(t) // asserts positions are 1..n after every operation
	}
	f.expectOrder(t, "B", "C", "E", "A", "F")
}

// Gate 5.12: acting on a step somebody else removed says so and changes
// nothing — at every write that could reach it.
func TestActingOnARemovedStepIsRefusedAndChangesNothing(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	two := f.add(t, "Two")
	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	before := f.version(t)

	acts := map[string]error{
		"tick":   f.store.SetStepDone(ctx, f.task.ID, two.ID, true, f.sam, fixedNow),
		"rename": f.store.RenameStep(ctx, f.task.ID, two.ID, "New words", f.sam, fixedNow),
		"remove": f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, fixedNow),
		"up":     f.store.MoveStep(ctx, f.task.ID, two.ID, MoveUp, f.sam, fixedNow),
		"to":     f.store.MoveStepTo(ctx, f.task.ID, two.ID, 1, f.sam, fixedNow),
	}
	for name, err := range acts {
		if err != ErrStepRemoved {
			t.Errorf("%s on a removed step err = %v, want ErrStepRemoved", name, err)
		}
	}
	if f.version(t) != before {
		t.Error("refused actions bumped the change counter")
	}
	f.expectOrder(t, "One")
	all, _ := f.store.ListStepsWithRemoved(ctx, f.task.ID, fixedNow)
	if all[1].Text != "Two" || all[1].Done {
		t.Errorf("the removed step was changed: %+v", all[1])
	}
}

func TestAStepThatIsNotOnThisJobIsNotFound(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	other := createNamed(t, f.store, ctx, f.sam, "Sports day", StageTodo)
	st := f.add(t, "Mine")

	if err := f.store.SetStepDone(ctx, other.ID, st.ID, true, f.sam, fixedNow); err != ErrStepNotFound {
		t.Errorf("tick via another job err = %v, want ErrStepNotFound", err)
	}
	if err := f.store.RemoveStep(ctx, f.task.ID, 99999, f.sam, fixedNow); err != ErrStepNotFound {
		t.Errorf("remove of a missing step err = %v, want ErrStepNotFound", err)
	}
}

// Every write bumps the tasks change counter (gate 5.12's refresh).
func TestEveryStepWriteBumpsTheChangeCounter(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	st := f.add(t, "One")
	two := f.add(t, "Two")

	writes := map[string]func() error{
		"add":     func() error { _, err := f.store.AddStep(ctx, f.task.ID, "Three", f.sam, fixedNow); return err },
		"tick":    func() error { return f.store.SetStepDone(ctx, f.task.ID, st.ID, true, f.sam, fixedNow) },
		"untick":  func() error { return f.store.SetStepDone(ctx, f.task.ID, st.ID, false, f.sam, fixedNow) },
		"rename":  func() error { return f.store.RenameStep(ctx, f.task.ID, st.ID, "Uno", f.sam, fixedNow) },
		"move":    func() error { return f.store.MoveStep(ctx, f.task.ID, two.ID, MoveUp, f.sam, fixedNow) },
		"remove":  func() error { return f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, fixedNow) },
		"restore": func() error { return f.store.RestoreStep(ctx, f.task.ID, two.ID, f.sam, fixedNow) },
	}
	for _, name := range []string{"add", "tick", "untick", "rename", "move", "remove", "restore"} {
		before := f.version(t)
		if err := writes[name](); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if f.version(t) <= before {
			t.Errorf("%s did not bump the change counter", name)
		}
	}
}

// A refused write must not leave a half-done change behind, and the counter
// only moves when the transaction commits.
func TestARefusedWriteBumpsNothingAndRecordsNothing(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	f.add(t, "One")
	before := f.version(t)
	actionsBefore := len(f.activityActions(t))

	if _, err := f.store.AddStep(ctx, f.task.ID, "", f.sam, fixedNow); err == nil {
		t.Fatal("empty add succeeded")
	}
	if f.version(t) != before || len(f.activityActions(t)) != actionsBefore {
		t.Error("a refused add left a change counter bump or an activity row behind")
	}
}

// D-71: adding, renaming, removing and restoring go into History; ticking
// and unticking and moving do not.
func TestHistoryRecordsEditsToStepsButNotTicksOrMoves(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	one := f.add(t, "One")
	two := f.add(t, "Two")

	if err := f.store.SetStepDone(ctx, f.task.ID, one.ID, true, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetStepDone(ctx, f.task.ID, one.ID, false, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.MoveStep(ctx, f.task.ID, two.ID, MoveUp, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := f.activityActions(t); !sameWords(got, []string{"created", "step_added", "step_added"}) {
		t.Errorf("activity after ticks and a move = %v, want only created + two step_added", got)
	}

	if err := f.store.RenameStep(ctx, f.task.ID, one.ID, "Uno", f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RemoveStep(ctx, f.task.ID, one.ID, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RestoreStep(ctx, f.task.ID, one.ID, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	want := []string{"created", "step_added", "step_added", "step_renamed", "step_removed", "step_restored"}
	if got := f.activityActions(t); !sameWords(got, want) {
		t.Errorf("activity = %v, want %v", got, want)
	}
}

func TestHistoryReadsAsPlainSentences(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	one := f.add(t, "Book the hall")
	if err := f.store.RenameStep(ctx, f.task.ID, one.ID, "Book the big hall", f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RemoveStep(ctx, f.task.ID, one.ID, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RestoreStep(ctx, f.task.ID, one.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	activity, err := f.store.ListActivity(ctx, f.task.ID)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	var got []string
	for _, a := range activity {
		got = append(got, a.Text)
	}
	want := []string{
		"Sam restored the step “Book the big hall”",
		"Alex removed the step “Book the big hall”",
		"Alex renamed the step “Book the hall” to “Book the big hall”",
		"Sam added the step “Book the hall”",
		"Sam created this",
	}
	if !sameWords(got, want) {
		t.Errorf("history =\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// Gate 5.15's groundwork: the migration only adds a table. Everything a
// v1.1.0 install has — tasks, assignees, activity — is still there and still
// readable after it runs, and the new table has the shape SPEC B3 names.
func TestMigrationAddsTheTableAndLeavesTheOldSchemaReadable(t *testing.T) {
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	// Bring a database up to what v1.1.0 shipped: every section, but the
	// tasks section only as far as 0001.
	for _, section := range []string{"app", "people", "tasks"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatal(err)
		}
		if section == "tasks" {
			var upTo1 []db.Migration
			for _, m := range migrations {
				if m.Version <= 1 {
					upTo1 = append(upTo1, m)
				}
			}
			migrations = upTo1
		}
		if err := db.RunMigrations(sqlDB, "1.1.0", migrations); err != nil {
			t.Fatal(err)
		}
	}
	sam := testPerson(t, sqlDB, "Sam")
	store := &Store{DB: sqlDB}
	old := createNamed(t, store, context.Background(), sam, "Written under 1.1.0", StageDoing)

	var n int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'task_checklist_items'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("before 0002 the table exists (n=%d, err=%v)", n, err)
	}

	tasksMigrations, err := db.LoadMigrations(comphq.Migrations, "tasks")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RunMigrations(sqlDB, "1.2.0", tasksMigrations); err != nil {
		t.Fatalf("applying 0002 over a 1.1.0 database: %v", err)
	}

	got, err := store.Get(context.Background(), old.ID, fixedNow)
	if err != nil || got.Title != "Written under 1.1.0" || got.Stage != StageDoing {
		t.Errorf("a job written before the upgrade reads back as %+v, %v", got, err)
	}
	if acts, err := store.ListActivity(context.Background(), old.ID); err != nil || len(acts) != 1 {
		t.Errorf("that job's history = %v, %v; want its one 'created' row", acts, err)
	}

	cols := map[string]bool{}
	rows, err := sqlDB.Query(`SELECT name FROM pragma_table_info('task_checklist_items')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		cols[name] = true
	}
	for _, want := range []string{"id", "task_id", "text", "position", "done_by", "done_at", "removed_at",
		"created_by", "created_at", "updated_by", "updated_at"} {
		if !cols[want] {
			t.Errorf("task_checklist_items has no %q column", want)
		}
	}
	var idx int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'task_checklist_items_task'`).Scan(&idx); err != nil || idx != 1 {
		t.Errorf("index task_checklist_items_task missing (n=%d, err=%v)", idx, err)
	}
}

// Gate 5.10's groundwork: the Board, Team and My jobs queries carry each
// job's live step counts, and a job with no steps carries none.
func TestListsCarryLiveStepCounts(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	bare := createNamed(t, f.store, ctx, f.sam, "No steps here", StageTodo)

	one := f.add(t, "One")
	two := f.add(t, "Two")
	f.add(t, "Three")
	if err := f.store.SetStepDone(ctx, f.task.ID, one.ID, true, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetStepDone(ctx, f.task.ID, two.ID, true, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	// A removed step is not counted, whether it was ticked or not.
	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	byTitle := func(tasks []Task) map[string]Task {
		m := map[string]Task{}
		for _, tk := range tasks {
			m[tk.Title] = tk
		}
		return m
	}
	board, err := f.store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatal(err)
	}
	team, err := f.store.ListForTeamView(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for name, tasks := range map[string][]Task{"ListBoard": board, "ListForTeamView": team} {
		m := byTitle(tasks)
		if got := m["Concert"]; got.StepsTotal != 2 || got.StepsDone != 1 {
			t.Errorf("%s: Concert has %d/%d steps, want 1/2", name, got.StepsDone, got.StepsTotal)
		}
		if got := m[bare.Title]; got.StepsTotal != 0 || got.StepsDone != 0 {
			t.Errorf("%s: a job with no steps has %d/%d, want 0/0", name, got.StepsDone, got.StepsTotal)
		}
	}
}

// D-73: the Briefing does not read steps, so a job with unfinished steps
// briefs exactly as it did at v1.1.0 — the same rows, the same order.
func TestBriefingIsUnchangedByStepsOnAJob(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	due := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	if err := f.store.Update(ctx, f.task.ID, UpdateInput{
		Title: "Concert", Size: "M", Stage: StageTodo, DueDate: "2026-09-22", PersonIDs: []int64{f.sam},
	}, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	other := createNamed(t, f.store, ctx, f.sam, "Other job", StageTodo)
	if err := f.store.Update(ctx, other.ID, UpdateInput{Title: "Other job", Size: "M", Stage: StageTodo, DueDate: "2026-09-24"}, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	before, err := f.store.BriefingTasks(ctx, due)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 2 {
		t.Fatalf("briefing before has %d jobs, want 2", len(before))
	}

	for i := 0; i < 5; i++ {
		f.add(t, "A step that is not finished")
	}
	st := f.add(t, "A step that is")
	if err := f.store.SetStepDone(ctx, f.task.ID, st.ID, true, f.alex, fixedNow); err != nil {
		t.Fatal(err)
	}

	after, err := f.store.BriefingTasks(ctx, due)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Errorf("steps changed the Briefing:\nbefore %+v\nafter  %+v", before, after)
	}
}
