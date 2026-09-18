package tasks

import (
	"context"
	"reflect"
	"testing"
)

// SPEC A8 / gate 3.03: BriefingTasks returns every unfinished,
// unremoved task due on or before the cutoff — overdue included — in
// due-date order, with its people; done, removed, later and undated
// tasks never appear.
func TestBriefingTasks(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	create := func(title, stage, due string, people ...int64) Task {
		task, err := store.Create(ctx, CreateInput{Title: title, Stage: stage, DueDate: due, PersonIDs: people}, creator, fixedNow)
		if err != nil {
			t.Fatalf("Create %q: %v", title, err)
		}
		return task
	}
	create("Later", StageTodo, "2026-09-26")
	wed := create("Wednesday", StageDoing, "2026-09-23", alex)
	overdue := create("Overdue idea", StageIdea, "2026-09-10")
	create("Finished", StageDone, "2026-09-21")
	create("No date", StageTodo, "")
	gone := create("Removed", StageTodo, "2026-09-20")
	if err := store.Remove(ctx, gone.ID, creator, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := store.BriefingTasks(ctx, date(t, "2026-09-25"))
	if err != nil {
		t.Fatalf("BriefingTasks: %v", err)
	}
	var ids []int64
	for _, bt := range got {
		ids = append(ids, bt.ID)
	}
	if want := []int64{overdue.ID, wed.ID}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("BriefingTasks ids = %v, want %v (overdue first, then Wednesday)", ids, want)
	}
	if len(got[0].Assignees) != 0 {
		t.Errorf("Overdue idea assignees = %+v, want none", got[0].Assignees)
	}
	if len(got[1].Assignees) != 1 || got[1].Assignees[0].PersonID != alex {
		t.Errorf("Wednesday assignees = %+v, want just Alex", got[1].Assignees)
	}
}
