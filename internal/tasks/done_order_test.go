package tasks

import (
	"context"
	"testing"
	"time"
)

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// Gate 8.01: Done lists the job finished most recently at the top.
func TestGate801DoneIsInFinishingOrderNewestFirst(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageDoing)
	b := createNamed(t, store, ctx, creator, "B", StageDoing)
	c := createNamed(t, store, ctx, creator, "C", StageDoing)

	for i, task := range []Task{a, c, b} {
		at := fixedNow.Add(time.Duration(i) * time.Minute)
		if err := store.Move(ctx, task.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, at); err != nil {
			t.Fatalf("Move to done: %v", err)
		}
	}
	// Finished in the order A, C, B, so newest first is B, C, A.
	if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"B", "C", "A"}; !sameStrings(got, want) {
		t.Fatalf("Done = %v, want %v", got, want)
	}
}

// Gate 8.02/8.03: a drop that names a neighbour in Done is ignored, so the
// finished job is on top however it was dropped (this fails on 1.5.0's code,
// which kept the dropped position).
func TestGate802DroppedLowInDoneStillLandsOnTop(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageDoing)
	b := createNamed(t, store, ctx, creator, "B", StageDoing)
	c := createNamed(t, store, ctx, creator, "C", StageDoing)

	if err := store.Move(ctx, a.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := store.Move(ctx, b.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// C is dropped "after B", which is the bottom of the column the old way.
	if err := store.Move(ctx, c.ID, MoveInput{Stage: StageDone, AfterID: b.ID}, creator, fixedNow.Add(2*time.Minute)); err != nil {
		t.Fatalf("Move into Done after a neighbour: %v", err)
	}
	if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"C", "B", "A"}; !sameStrings(got, want) {
		t.Fatalf("Done = %v, want %v", got, want)
	}
	// A neighbour that has gone away must not make the move fail either: Done
	// ignores before_id and after_id altogether.
	d := createNamed(t, store, ctx, creator, "D", StageDoing)
	if err := store.Move(ctx, d.ID, MoveInput{Stage: StageDone, BeforeID: 999999}, creator, fixedNow.Add(3*time.Minute)); err != nil {
		t.Fatalf("Move into Done before a missing neighbour: %v", err)
	}
	if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"D", "C", "B", "A"}; !sameStrings(got, want) {
		t.Fatalf("Done = %v, want %v", got, want)
	}
}

// Gate 8.03: a move inside Done (what an old page's button or a stale drag
// would send) changes nothing at all.
func TestGate803MoveWithinDoneChangesNothing(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageDoing)
	b := createNamed(t, store, ctx, creator, "B", StageDoing)
	for i, task := range []Task{a, b} {
		if err := store.Move(ctx, task.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	// Done is [B, A]. Try to put B after A, and A before B.
	if err := store.Move(ctx, b.ID, MoveInput{Stage: StageDone, AfterID: a.ID}, creator, fixedNow.Add(10*time.Minute)); err != nil {
		t.Fatalf("Move B within Done: %v", err)
	}
	if err := store.Move(ctx, a.ID, MoveInput{Stage: StageDone, BeforeID: b.ID}, creator, fixedNow.Add(11*time.Minute)); err != nil {
		t.Fatalf("Move A within Done: %v", err)
	}
	if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"B", "A"}; !sameStrings(got, want) {
		t.Fatalf("Done = %v, want %v (a move within Done must change nothing)", got, want)
	}
	// And the finishing times are untouched.
	var doneAt string
	if err := sqlDB.QueryRow(`SELECT done_at FROM tasks WHERE id = ?`, a.ID).Scan(&doneAt); err != nil {
		t.Fatal(err)
	}
	if want := fixedNow.UTC().Format(time.RFC3339); doneAt != want {
		t.Errorf("done_at of A = %q, want %q (unchanged)", doneAt, want)
	}
}

// Gate 8.04: reopened and finished again goes back to the top with the new time.
func TestGate804ReopenedAndFinishedAgainGoesToTheTop(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageDoing)
	b := createNamed(t, store, ctx, creator, "B", StageDoing)
	for i, task := range []Task{a, b} {
		if err := store.Move(ctx, task.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Move(ctx, a.ID, MoveInput{Stage: StageDoing, ToBottom: true}, creator, fixedNow.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.Move(ctx, a.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"A", "B"}; !sameStrings(got, want) {
		t.Fatalf("Done = %v, want %v", got, want)
	}
}

// Two jobs finished in the same second stay in a stable order: the later id first.
func TestGate801SameSecondTiebreakIsNewestIDFirst(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageDoing)
	b := createNamed(t, store, ctx, creator, "B", StageDoing)
	for _, task := range []Task{a, b} {
		if err := store.Move(ctx, task.ID, MoveInput{Stage: StageDone, ToBottom: true}, creator, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if got, want := boardTitlesByStage(t, store, ctx, StageDone), []string{"B", "A"}; !sameStrings(got, want) {
			t.Fatalf("Done = %v, want %v", got, want)
		}
	}
}

// Other columns keep their hand-made order.
func TestGate803OtherColumnsStillOrderedByPosition(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageTodo)
	createNamed(t, store, ctx, creator, "B", StageTodo)
	c := createNamed(t, store, ctx, creator, "C", StageTodo)
	if err := store.Move(ctx, c.ID, MoveInput{Stage: StageTodo, BeforeID: a.ID}, creator, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got, want := boardTitlesByStage(t, store, ctx, StageTodo), []string{"C", "A", "B"}; !sameStrings(got, want) {
		t.Fatalf("To do = %v, want %v", got, want)
	}
}
