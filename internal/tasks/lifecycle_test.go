package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func setDoneAt(t *testing.T, store *Store, taskID int64, at time.Time) {
	t.Helper()
	if _, err := store.DB.Exec(`UPDATE tasks SET done_at = ? WHERE id = ?`, at.UTC().Format(time.RFC3339), taskID); err != nil {
		t.Fatal(err)
	}
}

// SPEC gate 2.08's 14-day boundary, exactly: a task done 14 days minus a
// minute ago still belongs on the board; one done 14 days plus a minute
// ago has aged off it and into Finished tasks.
func TestFourteenDayBoundaryBetweenBoardAndFinished(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	stillOnBoard := createNamed(t, store, ctx, creator, "Just under 14 days", StageDone)
	setDoneAt(t, store, stillOnBoard.ID, fixedNow.AddDate(0, 0, -14).Add(time.Minute))

	agedOff := createNamed(t, store, ctx, creator, "Just over 14 days", StageDone)
	setDoneAt(t, store, agedOff.ID, fixedNow.AddDate(0, 0, -14).Add(-time.Minute))

	board, err := store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	boardTitles := map[string]bool{}
	for _, tk := range board {
		boardTitles[tk.Title] = true
	}
	if !boardTitles["Just under 14 days"] {
		t.Error("a task done just under 14 days ago is missing from the board")
	}
	if boardTitles["Just over 14 days"] {
		t.Error("a task done just over 14 days ago is still on the board")
	}

	finished, err := store.ListFinished(ctx, fixedNow)
	if err != nil {
		t.Fatalf("ListFinished: %v", err)
	}
	finishedTitles := map[string]bool{}
	for _, tk := range finished {
		finishedTitles[tk.Title] = true
	}
	if finishedTitles["Just under 14 days"] {
		t.Error("a task done just under 14 days ago is already in Finished tasks")
	}
	if !finishedTitles["Just over 14 days"] {
		t.Error("a task done just over 14 days ago is missing from Finished tasks")
	}
}

func TestListFinishedIsNewestFirst(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	older := createNamed(t, store, ctx, creator, "Older", StageDone)
	setDoneAt(t, store, older.ID, fixedNow.AddDate(0, 0, -30))
	newer := createNamed(t, store, ctx, creator, "Newer", StageDone)
	setDoneAt(t, store, newer.ID, fixedNow.AddDate(0, 0, -20))

	finished, err := store.ListFinished(ctx, fixedNow)
	if err != nil {
		t.Fatalf("ListFinished: %v", err)
	}
	if len(finished) != 2 || finished[0].Title != "Newer" || finished[1].Title != "Older" {
		t.Errorf("ListFinished = %+v, want [Newer, Older]", finished)
	}
}

// SPEC gate 2.08: Reopen returns a finished task to the bottom of To do.
func TestReopenReturnsToBottomOfTodoClearsDoneAtAndRecordsActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "Existing in To do", StageTodo)
	finished := createNamed(t, store, ctx, creator, "Finished task", StageDone)
	setDoneAt(t, store, finished.ID, fixedNow.AddDate(0, 0, -30))

	if err := store.Reopen(ctx, finished.ID, creator, fixedNow); err != nil {
		t.Fatalf("Reopen: %v", err)
	}

	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"Existing in To do", "Finished task"}
	if len(titles) != 2 || titles[0] != want[0] || titles[1] != want[1] {
		t.Errorf("todo order after Reopen = %v, want %v", titles, want)
	}

	got, err := store.Get(ctx, finished.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Stage != StageTodo {
		t.Errorf("Stage after Reopen = %q, want %q", got.Stage, StageTodo)
	}

	var doneAt sql.NullString
	if err := sqlDB.QueryRow(`SELECT done_at FROM tasks WHERE id = ?`, finished.ID).Scan(&doneAt); err != nil {
		t.Fatal(err)
	}
	if doneAt.Valid {
		t.Errorf("done_at after Reopen = %+v, want NULL", doneAt)
	}

	if count := countActivity(t, sqlDB, finished.ID, "reopened"); count != 1 {
		t.Errorf("reopened activity rows = %d, want 1", count)
	}

	// It's gone from Finished tasks now that it's back on the board.
	finishedList, err := store.ListFinished(ctx, fixedNow)
	if err != nil {
		t.Fatalf("ListFinished: %v", err)
	}
	for _, tk := range finishedList {
		if tk.ID == finished.ID {
			t.Error("reopened task still appears in Finished tasks")
		}
	}
}

func TestReopenMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")

	if err := store.Reopen(context.Background(), 9999, creator, fixedNow); err != ErrTaskNotFound {
		t.Errorf("Reopen on missing task error = %v, want ErrTaskNotFound", err)
	}
}

// SPEC gate 2.09: Remove hides a task from the board and lists it in
// Removed tasks; nothing is permanently deleted, so Restore brings it
// back, landing at the bottom of its stage.
func TestRemoveHidesFromBoardListsInRemovedAndRestoreBringsItBack(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Removable", StageTodo)

	if err := store.Remove(ctx, task.ID, creator, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	board, err := store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	for _, tk := range board {
		if tk.ID == task.ID {
			t.Fatal("removed task still appears on the board")
		}
	}

	removed, err := store.ListRemoved(ctx)
	if err != nil {
		t.Fatalf("ListRemoved: %v", err)
	}
	if len(removed) != 1 || removed[0].ID != task.ID {
		t.Fatalf("ListRemoved = %+v, want just the removed task", removed)
	}
	if count := countActivity(t, sqlDB, task.ID, "removed"); count != 1 {
		t.Errorf("removed activity rows = %d, want 1", count)
	}

	if err := store.Restore(ctx, task.ID, creator, fixedNow); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	board, err = store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatalf("ListBoard after Restore: %v", err)
	}
	found := false
	for _, tk := range board {
		if tk.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Error("restored task doesn't appear back on the board")
	}

	removed, err = store.ListRemoved(ctx)
	if err != nil {
		t.Fatalf("ListRemoved after Restore: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("ListRemoved after Restore = %+v, want empty", removed)
	}
	if count := countActivity(t, sqlDB, task.ID, "restored"); count != 1 {
		t.Errorf("restored activity rows = %d, want 1", count)
	}
}

// Removing a task must not leave a gap in its old stage's positions.
func TestRemoveRenumbersRemainingStagePositions(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageTodo)
	b := createNamed(t, store, ctx, creator, "B", StageTodo)
	c := createNamed(t, store, ctx, creator, "C", StageTodo)

	if err := store.Remove(ctx, b.ID, creator, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	rows, err := sqlDB.Query(`SELECT id, position FROM tasks WHERE stage = ? AND removed_at IS NULL ORDER BY position`, StageTodo)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var positions []int
	var ids []int64
	for rows.Next() {
		var id int64
		var pos int
		if err := rows.Scan(&id, &pos); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		positions = append(positions, pos)
	}
	if len(positions) != 2 || positions[0] != 1 || positions[1] != 2 {
		t.Errorf("positions after removing B = %v, want [1, 2] (contiguous, no gap)", positions)
	}
	if ids[0] != a.ID || ids[1] != c.ID {
		t.Errorf("remaining ids = %v, want [A, C]", ids)
	}
}

func TestRemoveMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")

	if err := store.Remove(context.Background(), 9999, creator, fixedNow); err != ErrTaskNotFound {
		t.Errorf("Remove on missing task error = %v, want ErrTaskNotFound", err)
	}
}

func TestRestoreMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Not removed", StageTodo)
	if err := store.Restore(ctx, task.ID, creator, fixedNow); err != ErrTaskNotFound {
		t.Errorf("Restore on a not-removed task error = %v, want ErrTaskNotFound", err)
	}
	if err := store.Restore(ctx, 9999, creator, fixedNow); err != ErrTaskNotFound {
		t.Errorf("Restore on missing task error = %v, want ErrTaskNotFound", err)
	}
}
