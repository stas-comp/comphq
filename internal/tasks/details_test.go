package tasks

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stas-comp/comphq/internal/people"
)

func TestUpdateChangesTitleAndNotesRecordsEditedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Old title", StageIdea)

	input := UpdateInput{Title: "New title", Notes: "Some notes", Size: task.Size, Stage: task.Stage}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "New title" || got.Notes != "Some notes" {
		t.Errorf("Title/Notes = %q/%q, want %q/%q", got.Title, got.Notes, "New title", "Some notes")
	}

	count := countActivity(t, sqlDB, task.ID, "edited")
	if count != 1 {
		t.Errorf("edited activity rows = %d, want 1", count)
	}
}

func TestUpdateWithNoChangesRecordsNoActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	input := UpdateInput{Title: task.Title, Notes: task.Notes, Size: task.Size, Stage: task.Stage, DueDate: task.DueDate}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var total int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_activity WHERE task_id = ? AND action != 'created'`, task.ID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("non-created activity rows after a no-op Update = %d, want 0", total)
	}
}

func TestUpdateDueDateRecordsDueChangedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	input := UpdateInput{Title: task.Title, Size: task.Size, Stage: task.Stage, DueDate: "2026-09-20"}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.DueDate != "2026-09-20" {
		t.Errorf("DueDate = %q, want 2026-09-20", got.DueDate)
	}
	if count := countActivity(t, sqlDB, task.ID, "due_changed"); count != 1 {
		t.Errorf("due_changed activity rows = %d, want 1", count)
	}

	// Clearing it again is also a change.
	input2 := UpdateInput{Title: task.Title, Size: task.Size, Stage: task.Stage, DueDate: ""}
	if err := store.Update(ctx, task.ID, input2, creator, fixedNow); err != nil {
		t.Fatalf("Update (clear): %v", err)
	}
	if count := countActivity(t, sqlDB, task.ID, "due_changed"); count != 2 {
		t.Errorf("due_changed activity rows after clearing = %d, want 2", count)
	}
}

func TestUpdateSizeRecordsSizeChangedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	input := UpdateInput{Title: task.Title, Size: "L", Stage: task.Stage}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Size != "L" {
		t.Errorf("Size = %q, want L", got.Size)
	}
	if count := countActivity(t, sqlDB, task.ID, "size_changed"); count != 1 {
		t.Errorf("size_changed activity rows = %d, want 1", count)
	}
}

func TestUpdateAssigneesRecordsAssignedAndUnassignedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	jo := testPerson(t, sqlDB, "Jo")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	input := UpdateInput{Title: task.Title, Size: task.Size, Stage: task.Stage, PersonIDs: []int64{alex}}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update (assign Alex): %v", err)
	}
	if count := countActivity(t, sqlDB, task.ID, "assigned"); count != 1 {
		t.Errorf("assigned activity rows = %d, want 1", count)
	}

	// Swap Alex for Jo: one unassigned, one assigned.
	input2 := UpdateInput{Title: task.Title, Size: task.Size, Stage: task.Stage, PersonIDs: []int64{jo}}
	if err := store.Update(ctx, task.ID, input2, creator, fixedNow); err != nil {
		t.Fatalf("Update (swap to Jo): %v", err)
	}
	if count := countActivity(t, sqlDB, task.ID, "assigned"); count != 2 {
		t.Errorf("assigned activity rows after swap = %d, want 2", count)
	}
	if count := countActivity(t, sqlDB, task.ID, "unassigned"); count != 1 {
		t.Errorf("unassigned activity rows after swap = %d, want 1", count)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Assignees) != 1 || got.Assignees[0].PersonID != jo {
		t.Errorf("Assignees = %+v, want just Jo", got.Assignees)
	}
}

func TestUpdateStageChangeMovesToBottomAndRecordsMovedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	existing := createNamed(t, store, ctx, creator, "Existing", StageTodo)
	task := createNamed(t, store, ctx, creator, "Task", StageIdea)

	input := UpdateInput{Title: task.Title, Size: task.Size, Stage: StageTodo}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	titles := boardTitlesByStage(t, store, ctx, StageTodo)
	if want := []string{existing.Title, task.Title}; len(titles) != 2 || titles[0] != want[0] || titles[1] != want[1] {
		t.Errorf("todo order = %v, want %v (moved task lands at the bottom)", titles, want)
	}
	if count := countActivity(t, sqlDB, task.ID, "moved"); count != 1 {
		t.Errorf("moved activity rows = %d, want 1", count)
	}
}

func TestUpdateMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")

	err := store.Update(context.Background(), 9999, UpdateInput{Title: "T", Size: "M", Stage: StageIdea}, creator, fixedNow)
	if err != ErrTaskNotFound {
		t.Errorf("Update on missing task error = %v, want ErrTaskNotFound", err)
	}
}

func TestUpdateRejectsEmptyTitleInvalidSizeAndStage(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()
	task := createNamed(t, store, ctx, creator, "Task", StageIdea)

	if err := store.Update(ctx, task.ID, UpdateInput{Title: "  ", Size: "M", Stage: StageIdea}, creator, fixedNow); err != ErrEmptyTitle {
		t.Errorf("empty title error = %v, want ErrEmptyTitle", err)
	}
	if err := store.Update(ctx, task.ID, UpdateInput{Title: "T", Size: "XL", Stage: StageIdea}, creator, fixedNow); err != ErrInvalidSize {
		t.Errorf("invalid size error = %v, want ErrInvalidSize", err)
	}
	if err := store.Update(ctx, task.ID, UpdateInput{Title: "T", Size: "M", Stage: "backlog"}, creator, fixedNow); err != ErrInvalidStage {
		t.Errorf("invalid stage error = %v, want ErrInvalidStage", err)
	}
}

func TestListActivityIsNewestFirstAndResolvesNamesByID(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	input := UpdateInput{Title: task.Title, Size: "L", Stage: task.Stage}
	if err := store.Update(ctx, task.ID, input, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	activity, err := store.ListActivity(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	if len(activity) != 2 {
		t.Fatalf("len(activity) = %d, want 2 (created, size_changed)", len(activity))
	}
	if activity[0].Text != "Sam changed the size to Large" {
		t.Errorf("activity[0].Text = %q, want %q", activity[0].Text, "Sam changed the size to Large")
	}
	if activity[1].Text != "Sam created this" {
		t.Errorf("activity[1].Text = %q, want %q", activity[1].Text, "Sam created this")
	}

	if err := (&people.Store{DB: sqlDB}).Rename(creator, "Samantha"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	activity, err = store.ListActivity(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListActivity after rename: %v", err)
	}
	if activity[0].Text != "Samantha changed the size to Large" {
		t.Errorf("activity[0].Text after rename = %q, want the new name", activity[0].Text)
	}
}

func TestListActivityRendersAssignedUnassignedAndDueDateMessages(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageIdea)
	if err := store.Update(ctx, task.ID, UpdateInput{Title: task.Title, Size: task.Size, Stage: task.Stage, PersonIDs: []int64{alex}, DueDate: "2026-09-20"}, creator, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}

	activity, err := store.ListActivity(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	texts := make([]string, len(activity))
	for i, a := range activity {
		texts[i] = a.Text
	}
	want := map[string]bool{
		"Sam created this":                   true,
		"Sam assigned Alex":                  true,
		"Sam changed the due date to Sun 20 Sep 2026": true,
	}
	if len(texts) != len(want) {
		t.Fatalf("activity texts = %v, want exactly %v", texts, want)
	}
	for _, text := range texts {
		if !want[text] {
			t.Errorf("unexpected activity text %q", text)
		}
	}
}

func countActivity(t *testing.T, sqlDB *sql.DB, taskID int64, action string) int {
	t.Helper()
	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM task_activity WHERE task_id = ? AND action = ?`, taskID, action).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
