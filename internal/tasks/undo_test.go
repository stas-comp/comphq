package tasks

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// SPEC gates 7.16-7.18 (B13.3, D-93): Undo puts a removed job back beside the
// job it sat next to.

func TestUndoPutsTheJobBackInTheSamePlace(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	createNamed(t, store, ctx, sam, "A", StageTodo)
	b, err := store.Create(ctx, CreateInput{Title: "B", Stage: StageTodo, PersonIDs: []int64{alex}}, sam, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	createNamed(t, store, ctx, sam, "C", StageTodo)
	step, err := store.AddStep(ctx, b.ID, "Buy toner", sam, fixedNow)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Remove(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "C"}) {
		t.Fatalf("after Remove: %v", got)
	}
	restored, err := store.Undo(ctx, b.ID, sam, fixedNow)
	if err != nil || !restored {
		t.Fatalf("Undo = %v, %v, want true, nil", restored, err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("after Undo: %v, want [A B C] (the same place)", got)
	}
	back, err := store.Get(ctx, b.ID, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Assignees) != 1 || back.Assignees[0].PersonID != alex {
		t.Errorf("assignees after Undo = %+v, want Alex still on it", back.Assignees)
	}
	steps, _ := store.ListSteps(ctx, b.ID, fixedNow)
	if len(steps) != 1 || steps[0].ID != step.ID {
		t.Errorf("steps after Undo = %+v, want the one step", steps)
	}
	acts, err := store.ListActivity(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, a := range acts {
		texts = append(texts, a.Text)
	}
	all := strings.Join(texts, "\n")
	if !strings.Contains(all, "Sam removed this") || !strings.Contains(all, "Sam undid the removal") {
		t.Errorf("History = %q, want both the removal and the undo", all)
	}
}

func TestUndoOfTheLastJobGoesBackAtTheBottom(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, sam, "A", StageTodo)
	createNamed(t, store, ctx, sam, "B", StageTodo)
	c := createNamed(t, store, ctx, sam, "C", StageTodo)
	if err := store.Remove(ctx, c.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Undo(ctx, c.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("after Undo: %v, want [A B C]", got)
	}
}

// If the order changed in the meantime the job goes back next to the one it
// used to sit beside, not to a stale number.
func TestUndoAfterTheNeighbourMoved(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, sam, "A", StageTodo)
	b := createNamed(t, store, ctx, sam, "B", StageTodo)
	c := createNamed(t, store, ctx, sam, "C", StageTodo)
	createNamed(t, store, ctx, sam, "D", StageTodo)
	if err := store.Remove(ctx, b.ID, sam, fixedNow); err != nil { // B sat before C
		t.Fatal(err)
	}
	// C goes to the bottom, and A is pushed behind it.
	if err := store.Move(ctx, c.ID, MoveInput{Stage: StageTodo, ToBottom: true}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "D", "C"}) {
		t.Fatalf("setup: %v", got)
	}
	if _, err := store.Undo(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "D", "B", "C"}) {
		t.Errorf("after Undo: %v, want B directly before C wherever C is", got)
	}
	_ = a
}

// If the neighbour left the column or was removed, the old position is used,
// held within the column's length.
func TestUndoFallsBackToTheOldPositionWhenTheNeighbourIsGone(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, sam, "A", StageTodo)
	b := createNamed(t, store, ctx, sam, "B", StageTodo)
	c := createNamed(t, store, ctx, sam, "C", StageTodo)
	createNamed(t, store, ctx, sam, "D", StageTodo)
	if err := store.Remove(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(ctx, c.ID, sam, fixedNow); err != nil { // B's neighbour, gone too
		t.Fatal(err)
	}
	if _, err := store.Undo(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "B", "D"}) {
		t.Errorf("after Undo: %v, want B back at its old second place", got)
	}

	// An older removal, with no neighbour remembered at all.
	d := createNamed(t, store, ctx, sam, "E", StageTodo)
	if err := store.Remove(ctx, d.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`UPDATE task_activity SET detail = '{}' WHERE task_id = ? AND action = 'removed'`, d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Undo(ctx, d.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "B", "D", "E"}) {
		t.Errorf("after an old-style Undo: %v, want E back at its old last place", got)
	}
}

// Somebody else got there first (from Removed tasks, or their own Undo): no error, no change.
func TestUndoOfAJobThatIsNotRemovedIsANoOp(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, sam, "A", StageTodo)
	b := createNamed(t, store, ctx, sam, "B", StageTodo)
	if err := store.Remove(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := store.Restore(ctx, b.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Version(ctx)
	restored, err := store.Undo(ctx, b.ID, sam, fixedNow)
	if err != nil || restored {
		t.Fatalf("Undo of a job that isn't removed = %v, %v, want false, nil", restored, err)
	}
	if after, _ := store.Version(ctx); after != before {
		t.Errorf("a no-op Undo bumped the change counter (%v -> %v)", before, after)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("tasks = %v, want [A B]", got)
	}
	if restored, err := store.Undo(ctx, 99999, sam, fixedNow); err != nil || restored {
		t.Errorf("Undo of an id that doesn't exist = %v, %v, want false, nil", restored, err)
	}
}

// Removed tasks (gate 7.18): Restore still goes to the bottom of its column.
func TestRestoreFromRemovedTasksStillGoesToTheBottom(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, sam, "A", StageTodo)
	createNamed(t, store, ctx, sam, "B", StageTodo)
	if err := store.Remove(ctx, a.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := store.Restore(ctx, a.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := boardTitlesByStage(t, store, ctx, StageTodo); !reflect.DeepEqual(got, []string{"B", "A"}) {
		t.Errorf("after Restore: %v, want [B A] (the bottom, as before)", got)
	}
}

func TestRemovedTitle(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, sam, "Order toner", StageTodo)
	if _, ok, _ := store.RemovedTitle(ctx, a.ID); ok {
		t.Error("RemovedTitle ok for a job that isn't removed")
	}
	if err := store.Remove(ctx, a.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if title, ok, err := store.RemovedTitle(ctx, a.ID); err != nil || !ok || title != "Order toner" {
		t.Errorf("RemovedTitle = %q, %v, %v", title, ok, err)
	}
}
