package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/stas-comp/comphq/internal/people"
)

// SPEC gate 2.33: Up for grabs holds unassigned doing/todo (priority
// order), then unassigned ideas; assigned and done tasks never appear.
func TestListUpForGrabsOrderAndExclusions(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "An idea", StageIdea)
	createNamed(t, store, ctx, creator, "A todo", StageTodo)
	createNamed(t, store, ctx, creator, "A doing", StageDoing)
	createNamed(t, store, ctx, creator, "Done already", StageDone)
	if _, err := store.Create(ctx, CreateInput{Title: "Someone's todo", Stage: StageTodo, PersonIDs: []int64{alex}}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	grabs, err := store.ListUpForGrabs(ctx)
	if err != nil {
		t.Fatalf("ListUpForGrabs: %v", err)
	}
	var titles []string
	for _, t := range grabs {
		titles = append(titles, t.Title)
	}
	want := []string{"A doing", "A todo", "An idea"}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v, want %v", titles, want)
	}
	for i, w := range want {
		if titles[i] != w {
			t.Errorf("titles[%d] = %q, want %q (doing, then todo, then idea)", i, titles[i], w)
		}
	}
}

// SPEC B4: My jobs' left side is the person's own lane, built the same
// way as their Team view column.
func TestLaneForPersonMatchesTeamLane(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Mine", Stage: StageTodo, PersonIDs: []int64{creator}}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tasks, err := store.ListForTeamView(ctx)
	if err != nil {
		t.Fatalf("ListForTeamView: %v", err)
	}
	lane := LaneForPerson(tasks, people.Person{ID: creator, Name: "Sam"})
	if len(lane.UpNext) != 1 || lane.UpNext[0].Title != "Mine" {
		t.Errorf("lane.UpNext = %+v, want just [Mine]", lane.UpNext)
	}
}

// SPEC gate 2.34: taking a todo/doing job without a drop position keeps
// its own position.
func TestTakeUnassignedTodoWithoutPositionKeepsPosition(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "First", StageTodo)
	task := createNamed(t, store, ctx, creator, "Second", StageTodo)
	createNamed(t, store, ctx, creator, "Third", StageTodo)

	if err := store.Take(ctx, task.ID, 0, 0, alex, fixedNow); err != nil {
		t.Fatalf("Take: %v", err)
	}

	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"First", "Second", "Third"}
	if len(titles) != 3 || titles[0] != want[0] || titles[1] != want[1] || titles[2] != want[2] {
		t.Errorf("todo order after Take (no drop position) = %v, want unchanged %v", titles, want)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Assignees) != 1 || got.Assignees[0].PersonID != alex {
		t.Errorf("Assignees = %+v, want just Alex", got.Assignees)
	}
}

// SPEC gate 2.34: taking with a drop position lands the task there.
func TestTakeWithDropPositionInsertsThere(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	a := createNamed(t, store, ctx, creator, "A", StageTodo)
	task := createNamed(t, store, ctx, creator, "Taken", StageTodo)
	createNamed(t, store, ctx, creator, "C", StageTodo)

	if err := store.Take(ctx, task.ID, a.ID, 0, alex, fixedNow); err != nil {
		t.Fatalf("Take: %v", err)
	}

	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"Taken", "A", "C"}
	if len(titles) != 3 || titles[0] != want[0] || titles[1] != want[1] || titles[2] != want[2] {
		t.Errorf("todo order after Take (before A) = %v, want %v", titles, want)
	}
}

// SPEC gate 2.35: taking an idea moves it to To do, at the drop
// position or the bottom, and records both changes.
func TestTakeIdeaMovesToTodoAtBottomAndRecordsActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "Existing in todo", StageTodo)
	idea := createNamed(t, store, ctx, creator, "An idea", StageIdea)

	if err := store.Take(ctx, idea.ID, 0, 0, alex, fixedNow); err != nil {
		t.Fatalf("Take: %v", err)
	}

	got, err := store.Get(ctx, idea.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Stage != StageTodo {
		t.Errorf("Stage = %q, want %q", got.Stage, StageTodo)
	}
	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	if len(titles) != 2 || titles[0] != "Existing in todo" || titles[1] != "An idea" {
		t.Errorf("todo order = %v, want [Existing in todo, An idea] (taken idea at the bottom)", titles)
	}
	if count := countActivity(t, sqlDB, idea.ID, "assigned"); count != 1 {
		t.Errorf("assigned activity rows = %d, want 1", count)
	}
	if count := countActivity(t, sqlDB, idea.ID, "moved"); count != 1 {
		t.Errorf("moved activity rows = %d, want 1", count)
	}
}

func TestTakeIdeaWithDropPositionLandsInTodoThere(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	first := createNamed(t, store, ctx, creator, "First", StageTodo)
	createNamed(t, store, ctx, creator, "Second", StageTodo)
	idea := createNamed(t, store, ctx, creator, "An idea", StageIdea)

	if err := store.Take(ctx, idea.ID, first.ID, 0, alex, fixedNow); err != nil {
		t.Fatalf("Take: %v", err)
	}

	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	want := []string{"An idea", "First", "Second"}
	if len(titles) != 3 || titles[0] != want[0] || titles[1] != want[1] || titles[2] != want[2] {
		t.Errorf("todo order = %v, want %v", titles, want)
	}
}

func TestTakeAlreadyAssignedTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	jo := testPerson(t, sqlDB, "Jo")
	ctx := context.Background()

	task, err := store.Create(ctx, CreateInput{Title: "Task", Stage: StageTodo, PersonIDs: []int64{alex}}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var taken *TakenError
	err = store.Take(ctx, task.ID, 0, 0, jo, fixedNow)
	if !errors.As(err, &taken) {
		t.Fatalf("err = %v, want a *TakenError", err)
	}
	if len(taken.Names) != 1 || taken.Names[0] != "Alex" {
		t.Errorf("taken.Names = %v, want [Alex]", taken.Names)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Assignees) != 1 || got.Assignees[0].PersonID != alex {
		t.Errorf("Assignees changed to %+v, want unchanged [Alex]", got.Assignees)
	}
}

func TestTakeMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	alex := testPerson(t, sqlDB, "Alex")

	if err := store.Take(context.Background(), 9999, 0, 0, alex, fixedNow); err != ErrTaskNotFound {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}
}
