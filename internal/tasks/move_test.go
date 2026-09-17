package tasks

import (
	"context"
	"database/sql"
	"testing"
)

func createNamed(t *testing.T, store *Store, ctx context.Context, creator int64, title, stage string) Task {
	t.Helper()
	task, err := store.Create(ctx, CreateInput{Title: title, Stage: stage}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	return task
}

func boardTitlesByStage(t *testing.T, store *Store, ctx context.Context, stage string) []string {
	t.Helper()
	tasks, err := store.ListBoard(ctx, fixedNow)
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	var titles []string
	for _, tk := range tasks {
		if tk.Stage == stage {
			titles = append(titles, tk.Title)
		}
	}
	return titles
}

func TestMoveWithinStageBefore(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageTodo)
	createNamed(t, store, ctx, creator, "B", StageTodo)
	c := createNamed(t, store, ctx, creator, "C", StageTodo)
	// [A, B, C] -> move C before A -> [C, A, B]
	if err := store.Move(ctx, c.ID, MoveInput{Stage: StageTodo, BeforeID: a.ID}, creator, fixedNow); err != nil {
		t.Fatalf("Move: %v", err)
	}
	got := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"C", "A", "B"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestMoveAcrossStagesToBottom(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "Existing", StageTodo)
	moving := createNamed(t, store, ctx, creator, "Moving", StageIdea)

	if err := store.Move(ctx, moving.ID, MoveInput{Stage: StageTodo, ToBottom: true}, creator, fixedNow); err != nil {
		t.Fatalf("Move: %v", err)
	}

	gotIdea := boardTitlesByStage(t, store, ctx, StageIdea)
	if len(gotIdea) != 0 {
		t.Errorf("Idea column = %v, want empty (the task moved out)", gotIdea)
	}
	gotTodo := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"Existing", "Moving"}
	if len(gotTodo) != 2 || gotTodo[0] != want[0] || gotTodo[1] != want[1] {
		t.Errorf("To do column = %v, want %v", gotTodo, want)
	}
}

func TestMoveAcrossStagesRecordsMovedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	if err := store.Move(ctx, task.ID, MoveInput{Stage: StageTodo, ToBottom: true}, creator, fixedNow); err != nil {
		t.Fatalf("Move: %v", err)
	}

	var count int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM task_activity WHERE task_id = ? AND action = 'moved'`, task.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("moved activity rows = %d, want 1", count)
	}
}

func TestMoveWithinSameStageRecordsNoMovedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageTodo)
	b := createNamed(t, store, ctx, creator, "B", StageTodo)
	if err := store.Move(ctx, b.ID, MoveInput{Stage: StageTodo, BeforeID: a.ID}, creator, fixedNow); err != nil {
		t.Fatalf("Move: %v", err)
	}

	var count int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM task_activity WHERE task_id = ? AND action = 'moved'`, b.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("moved activity rows = %d, want 0 (same-stage reorder isn't a move)", count)
	}
}

func TestMoveToMissingNeighborIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageTodo)
	err := store.Move(ctx, task.ID, MoveInput{Stage: StageTodo, BeforeID: 999999}, creator, fixedNow)
	if err != ErrNeighborNotFound {
		t.Errorf("err = %v, want ErrNeighborNotFound", err)
	}
}

func TestMoveMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	err := store.Move(ctx, 999999, MoveInput{Stage: StageTodo, ToBottom: true}, creator, fixedNow)
	if err != ErrTaskNotFound {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}
}

func TestMoveRequiresExactlyOneTarget(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageTodo)
	other := createNamed(t, store, ctx, creator, "Other", StageTodo)

	if err := store.Move(ctx, task.ID, MoveInput{Stage: StageTodo}, creator, fixedNow); err != ErrInvalidMove {
		t.Errorf("no target: err = %v, want ErrInvalidMove", err)
	}
	if err := store.Move(ctx, task.ID, MoveInput{Stage: StageTodo, BeforeID: other.ID, ToBottom: true}, creator, fixedNow); err != ErrInvalidMove {
		t.Errorf("two targets: err = %v, want ErrInvalidMove", err)
	}
}

// TestMoveSequenceStaysConsistent runs several moves back to back (the
// closest a single-SQLite-connection app gets to "concurrency": two
// people's moves are always serialised, never truly parallel — PLAN.md
// P2-02's "concurrency via two transactions ends consistent") and checks
// positions in every touched stage stay a contiguous, unique 1..n
// sequence throughout.
func TestMoveSequenceStaysConsistent(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageIdea)
	b := createNamed(t, store, ctx, creator, "B", StageIdea)
	c := createNamed(t, store, ctx, creator, "C", StageIdea)

	moves := []struct {
		id    int64
		input MoveInput
	}{
		{c.ID, MoveInput{Stage: StageTodo, ToBottom: true}},
		{a.ID, MoveInput{Stage: StageTodo, BeforeID: c.ID}},
		{b.ID, MoveInput{Stage: StageDoing, ToBottom: true}},
		{c.ID, MoveInput{Stage: StageDoing, AfterID: b.ID}},
	}
	for i, m := range moves {
		if err := store.Move(ctx, m.id, m.input, creator, fixedNow); err != nil {
			t.Fatalf("move %d: %v", i, err)
		}
		assertPositionsContiguous(t, sqlDB)
	}
}

// assertPositionsContiguous checks that within every stage, the visible
// tasks' positions are exactly 1..n with no duplicate or missing value.
func assertPositionsContiguous(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	for _, stage := range Stages {
		rows, err := sqlDB.Query(
			`SELECT position FROM tasks WHERE stage = ? AND removed_at IS NULL ORDER BY position`, stage,
		)
		if err != nil {
			t.Fatalf("query positions for %s: %v", stage, err)
		}
		var positions []int
		for rows.Next() {
			var p int
			if err := rows.Scan(&p); err != nil {
				t.Fatalf("scan: %v", err)
			}
			positions = append(positions, p)
		}
		rows.Close()

		for i, p := range positions {
			if p != i+1 {
				t.Fatalf("stage %s positions = %v, want a contiguous 1..%d sequence", stage, positions, len(positions))
			}
		}
	}
}
