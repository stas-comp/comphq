package tasks

import (
	"context"
	"testing"
	"time"
)

// SPEC gate 2.19: unfinished tasks with due dates show on their date;
// finished and removed tasks don't.
func TestDueTasksExcludesDoneAndRemoved(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	todo, err := store.Create(ctx, CreateInput{Title: "Due soon", Stage: StageTodo, DueDate: "2026-09-20"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Finished", Stage: StageDone, DueDate: "2026-09-21"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	removed, err := store.Create(ctx, CreateInput{Title: "Removed", Stage: StageTodo, DueDate: "2026-09-22"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Remove(ctx, removed.ID, creator, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "No due date", Stage: StageTodo}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	due, err := store.DueTasks(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("DueTasks: %v", err)
	}
	if len(due) != 1 || due[0].ID != todo.ID {
		t.Errorf("DueTasks = %+v, want just %q", due, "Due soon")
	}
}

// SPEC gate 2.19's own date window: a due date outside [from, to] doesn't show.
func TestDueTasksRespectsTheDateWindow(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Too early", Stage: StageTodo, DueDate: "2026-08-31"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	inWindow, err := store.Create(ctx, CreateInput{Title: "In window", Stage: StageTodo, DueDate: "2026-09-15"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Too late", Stage: StageTodo, DueDate: "2026-10-01"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	due, err := store.DueTasks(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("DueTasks: %v", err)
	}
	if len(due) != 1 || due[0].ID != inWindow.ID {
		t.Errorf("DueTasks = %+v, want just %q", due, "In window")
	}
}

func TestDueTasksOrdersByDueDate(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Later", Stage: StageTodo, DueDate: "2026-09-20"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Earlier", Stage: StageTodo, DueDate: "2026-09-10"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	due, err := store.DueTasks(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("DueTasks: %v", err)
	}
	if len(due) != 2 || due[0].Title != "Earlier" || due[1].Title != "Later" {
		t.Errorf("DueTasks order = %+v, want Earlier then Later", due)
	}
}

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}
