package tasks

import (
	"context"
	"testing"
)

func assigneeIDsOf(task Task) []int64 {
	ids := make([]int64, len(task.Assignees))
	for i, a := range task.Assignees {
		ids[i] = a.PersonID
	}
	return ids
}

func containsID(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// SPEC gate 2.28: dragging from Unassigned onto a person assigns it to
// them.
func TestAssignFromUnassignedToPersonRecordsAssignedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageTodo)

	if err := store.Assign(ctx, task.ID, 0, alex, creator, fixedNow); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ids := assigneeIDsOf(got); len(ids) != 1 || ids[0] != alex {
		t.Errorf("Assignees = %v, want just [%d]", ids, alex)
	}
	if count := countActivity(t, sqlDB, task.ID, "assigned"); count != 1 {
		t.Errorf("assigned activity rows = %d, want 1", count)
	}
	if count := countActivity(t, sqlDB, task.ID, "unassigned"); count != 0 {
		t.Errorf("unassigned activity rows = %d, want 0", count)
	}
}

// SPEC gate 2.28: dragging from one person to another takes the first
// off and puts the second on; anyone else on the job stays.
func TestAssignFromPersonToPersonKeepsThirdAssignee(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	jo := testPerson(t, sqlDB, "Jo")
	kim := testPerson(t, sqlDB, "Kim")
	ctx := context.Background()

	task, err := store.Create(ctx, CreateInput{Title: "Task", Stage: StageTodo, PersonIDs: []int64{alex, kim}}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Assign(ctx, task.ID, alex, jo, creator, fixedNow); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	ids := assigneeIDsOf(got)
	if containsID(ids, alex) {
		t.Errorf("Assignees = %v, Alex should have been removed", ids)
	}
	if !containsID(ids, jo) {
		t.Errorf("Assignees = %v, Jo should have been added", ids)
	}
	if !containsID(ids, kim) {
		t.Errorf("Assignees = %v, Kim (the third assignee) should still be there", ids)
	}
	// Create's own initial assignees (Alex, Kim) are covered by the
	// single "created" activity row, not a separate "assigned" one each
	// — only this Assign call's own change gets its own row.
	if count := countActivity(t, sqlDB, task.ID, "assigned"); count != 1 {
		t.Errorf("assigned activity rows = %d, want 1 (Jo, from this Assign)", count)
	}
	if count := countActivity(t, sqlDB, task.ID, "unassigned"); count != 1 {
		t.Errorf("unassigned activity rows = %d, want 1 (Alex)", count)
	}
}

// "Give back" (P2-10) reuses this same endpoint with only from_person
// set — this proves that half of the contract now, since P2-09 already
// builds the shared Assign method.
func TestAssignWithOnlyFromPersonRemovesWithoutAddingAnyone(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	task, err := store.Create(ctx, CreateInput{Title: "Task", Stage: StageTodo, PersonIDs: []int64{alex}}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Assign(ctx, task.ID, alex, 0, creator, fixedNow); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Assignees) != 0 {
		t.Errorf("Assignees = %v, want none", got.Assignees)
	}
}

func TestAssignDoesNotChangeStageOrPosition(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageDoing)
	before, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if err := store.Assign(ctx, task.ID, 0, alex, creator, fixedNow); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	after, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if after.Stage != before.Stage {
		t.Errorf("Stage changed from %q to %q, want unchanged", before.Stage, after.Stage)
	}
	if after.Position != before.Position {
		t.Errorf("Position changed from %d to %d, want unchanged", before.Position, after.Position)
	}
}

// Assigning to someone already on the job is a harmless no-op, not a
// duplicate-key failure or a redundant activity row.
func TestAssignToAlreadyAssignedPersonIsANoOp(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	task, err := store.Create(ctx, CreateInput{Title: "Task", Stage: StageTodo, PersonIDs: []int64{alex}}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Assign(ctx, task.ID, 0, alex, creator, fixedNow); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	got, err := store.Get(ctx, task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Assignees) != 1 {
		t.Errorf("Assignees = %v, want just Alex (no duplicate)", got.Assignees)
	}
	// Create's own initial assignee (Alex) is covered by the "created"
	// row, not a separate "assigned" one — and this no-op Assign call
	// shouldn't add a redundant one either.
	if count := countActivity(t, sqlDB, task.ID, "assigned"); count != 0 {
		t.Errorf("assigned activity rows = %d, want 0", count)
	}
}

func TestAssignRejectsNeitherFromNorTo(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task := createNamed(t, store, ctx, creator, "Task", StageTodo)
	if err := store.Assign(ctx, task.ID, 0, 0, creator, fixedNow); err != ErrInvalidAssign {
		t.Errorf("err = %v, want ErrInvalidAssign", err)
	}
}

func TestAssignMissingTaskIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")

	if err := store.Assign(context.Background(), 9999, 0, alex, creator, fixedNow); err != ErrTaskNotFound {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}
}
