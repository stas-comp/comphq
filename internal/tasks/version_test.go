package tasks

import (
	"context"
	"testing"
)

// SPEC B4/D-15's change counter: every write bumps it, so a client
// polling GET /tasks/version can cheaply notice something changed
// elsewhere without re-fetching the board on every tick.
func TestVersionStartsAtZeroAndBumpsOnEveryWrite(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	v, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != 0 {
		t.Errorf("Version on a fresh install = %d, want 0", v)
	}

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	v1, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v1 <= v {
		t.Errorf("Version after Create = %d, want > %d", v1, v)
	}

	if err := store.Update(ctx, task.ID, UpdateInput{Title: "Renamed", Size: task.Size, Stage: task.Stage}, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}
	v2, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v2 <= v1 {
		t.Errorf("Version after Update = %d, want > %d", v2, v1)
	}

	if err := store.Move(ctx, task.ID, MoveInput{Stage: StageTodo, ToBottom: true}, creator, fixedNow); err != nil {
		t.Fatalf("Move: %v", err)
	}
	v3, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v3 <= v2 {
		t.Errorf("Version after Move = %d, want > %d", v3, v2)
	}
}

func TestVersionUnchangedByAFailedWrite(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "Task", StageIdea)
	before, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}

	if _, err := store.Create(ctx, CreateInput{Title: "  "}, creator, fixedNow); err != ErrEmptyTitle {
		t.Fatalf("Create with empty title error = %v, want ErrEmptyTitle", err)
	}

	after, err := store.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if after != before {
		t.Errorf("Version after a rejected Create = %d, want unchanged %d", after, before)
	}
}
