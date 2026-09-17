package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/people"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	for _, section := range []string{"people", "tasks"} {
		migrations, err := db.LoadMigrations(comphq.Migrations, section)
		if err != nil {
			t.Fatalf("LoadMigrations(%s): %v", section, err)
		}
		if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
			t.Fatalf("RunMigrations(%s): %v", section, err)
		}
	}
	return sqlDB
}

func testPerson(t *testing.T, sqlDB *sql.DB, name string) int64 {
	t.Helper()
	p, err := (&people.Store{DB: sqlDB}).Create(name)
	if err != nil {
		t.Fatalf("create person %q: %v", name, err)
	}
	return p.ID
}

var fixedNow = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestCreateAssignsPositionAtBottomOfStage(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	first, err := store.Create(ctx, CreateInput{Title: "First"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create(first): %v", err)
	}
	if first.Position != 1 {
		t.Errorf("first.Position = %d, want 1", first.Position)
	}
	if first.Stage != StageIdea {
		t.Errorf("first.Stage = %q, want %q (default)", first.Stage, StageIdea)
	}
	if first.Size != "M" {
		t.Errorf("first.Size = %q, want M (default)", first.Size)
	}

	second, err := store.Create(ctx, CreateInput{Title: "Second"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create(second): %v", err)
	}
	if second.Position != 2 {
		t.Errorf("second.Position = %d, want 2 (bottom of the same stage)", second.Position)
	}

	// A different stage starts its own position sequence at 1.
	thirdInTodo, err := store.Create(ctx, CreateInput{Title: "Third", Stage: StageTodo}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create(third): %v", err)
	}
	if thirdInTodo.Position != 1 {
		t.Errorf("thirdInTodo.Position = %d, want 1 (a different stage's own sequence)", thirdInTodo.Position)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")

	_, err := store.Create(context.Background(), CreateInput{Title: "   "}, creator, fixedNow)
	if err != ErrEmptyTitle {
		t.Fatalf("Create(blank title) error = %v, want ErrEmptyTitle", err)
	}
}

func TestCreateRecordsCreatorAndCreatedActivity(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")

	task, err := store.Create(context.Background(), CreateInput{Title: "Task"}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var action string
	var personID int64
	if err := sqlDB.QueryRow(
		`SELECT action, person_id FROM task_activity WHERE task_id = ?`, task.ID,
	).Scan(&action, &personID); err != nil {
		t.Fatalf("query task_activity: %v", err)
	}
	if action != "created" {
		t.Errorf("activity action = %q, want created", action)
	}
	if personID != creator {
		t.Errorf("activity person_id = %d, want %d (the creator)", personID, creator)
	}

	var createdBy int64
	if err := sqlDB.QueryRow(`SELECT created_by FROM tasks WHERE id = ?`, task.ID).Scan(&createdBy); err != nil {
		t.Fatalf("query tasks.created_by: %v", err)
	}
	if createdBy != creator {
		t.Errorf("tasks.created_by = %d, want %d", createdBy, creator)
	}
}

func TestCreateWithAssigneesAndDueDate(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	other := testPerson(t, sqlDB, "Alex")

	task, err := store.Create(context.Background(), CreateInput{
		Title:     "Task with people",
		DueDate:   "2026-09-01",
		PersonIDs: []int64{creator, other},
	}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.DueDate != "2026-09-01" {
		t.Errorf("DueDate = %q, want 2026-09-01", task.DueDate)
	}
	if len(task.Assignees) != 2 {
		t.Fatalf("len(Assignees) = %d, want 2", len(task.Assignees))
	}
}

func TestCreateRejectsInvalidSizeAndStage(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "T", Size: "XL"}, creator, fixedNow); err != ErrInvalidSize {
		t.Errorf("Create with invalid size error = %v, want ErrInvalidSize", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "T", Stage: "backlog"}, creator, fixedNow); err != ErrInvalidStage {
		t.Errorf("Create with invalid stage error = %v, want ErrInvalidStage", err)
	}
}

func TestListBoardOrdersByStageThenPosition(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	mustCreate := func(title, stage string) {
		if _, err := store.Create(ctx, CreateInput{Title: title, Stage: stage}, creator, fixedNow); err != nil {
			t.Fatalf("Create(%q): %v", title, err)
		}
	}
	mustCreate("Done 1", StageDone)
	mustCreate("Idea 1", StageIdea)
	mustCreate("Todo 1", StageTodo)
	mustCreate("Idea 2", StageIdea)

	tasks, err := store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	var titles []string
	for _, t := range tasks {
		titles = append(titles, t.Title)
	}
	want := []string{"Idea 1", "Idea 2", "Todo 1", "Done 1"}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v, want %v", titles, want)
	}
	for i, w := range want {
		if titles[i] != w {
			t.Errorf("titles[%d] = %q, want %q (stage order idea,todo,doing,done, then position)", i, titles[i], w)
		}
	}
}

// SPEC gate 2.07: "My tasks"/a chosen person narrows the board to
// exactly the cards assigned to that one person.
func TestListBoardFilterByPersonShowsOnlyTheirTasks(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Sam's task", PersonIDs: []int64{creator}}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Alex's task", PersonIDs: []int64{alex}}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Unassigned task"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tasks, err := store.ListBoard(ctx, fixedNow, BoardFilter{PersonID: alex})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Alex's task" {
		t.Errorf("filtered by Alex = %+v, want just Alex's task", tasks)
	}
}

// SPEC gate 2.07: typing a word shows only cards with that word in the
// title, case-insensitive.
func TestListBoardFilterByQueryMatchesTitleCaseInsensitively(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Fix the PRINTER"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Order paper"}, creator, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tasks, err := store.ListBoard(ctx, fixedNow, BoardFilter{Query: "printer"})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Fix the PRINTER" {
		t.Errorf("filtered by %q = %+v, want just the printer task", "printer", tasks)
	}
}

func TestListBoardExcludesTasksDoneOverTwoWeeksAgo(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	task, err := store.Create(ctx, CreateInput{Title: "Old done task", Stage: StageDone}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	longAgo := fixedNow.AddDate(0, 0, -20).UTC().Format(time.RFC3339)
	if _, err := sqlDB.Exec(`UPDATE tasks SET done_at = ? WHERE id = ?`, longAgo, task.ID); err != nil {
		t.Fatalf("set done_at: %v", err)
	}

	tasks, err := store.ListBoard(ctx, fixedNow, BoardFilter{})
	if err != nil {
		t.Fatalf("ListBoard: %v", err)
	}
	for _, tk := range tasks {
		if tk.ID == task.ID {
			t.Fatalf("ListBoard still includes a task done 20 days ago (gate 2.08: 14-day retention)")
		}
	}
}

func TestOverdueOnlyForUnfinishedPastDueTasks(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	today := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)

	overdue, err := store.Create(ctx, CreateInput{Title: "Overdue", DueDate: "2026-09-01"}, creator, today)
	if err != nil {
		t.Fatalf("Create(overdue): %v", err)
	}
	if !overdue.Overdue {
		t.Error("a task due in the past, not done, should be Overdue")
	}

	future, err := store.Create(ctx, CreateInput{Title: "Future", DueDate: "2026-12-01"}, creator, today)
	if err != nil {
		t.Fatalf("Create(future): %v", err)
	}
	if future.Overdue {
		t.Error("a task due in the future should not be Overdue")
	}

	doneAndPast, err := store.Create(ctx, CreateInput{Title: "Done", DueDate: "2026-09-01", Stage: StageDone}, creator, today)
	if err != nil {
		t.Fatalf("Create(done): %v", err)
	}
	if doneAndPast.Overdue {
		t.Error("a done task should never be Overdue, even with a past due date")
	}
}

func TestAssigneesForIncludesRemovedPeopleMarked(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	peopleStore := &people.Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	toRemove := testPerson(t, sqlDB, "Alex")

	task, err := store.Create(context.Background(), CreateInput{
		Title: "Task", PersonIDs: []int64{creator, toRemove},
	}, creator, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := peopleStore.Deactivate(toRemove); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	got, err := store.Get(context.Background(), task.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var foundRemoved bool
	for _, a := range got.Assignees {
		if a.PersonID == toRemove {
			foundRemoved = true
			if !a.Removed {
				t.Error("a deactivated assignee should have Removed = true")
			}
		}
	}
	if !foundRemoved {
		t.Fatal("removed person no longer appears among assignees at all — SPEC gate 2.11 requires it to still show, marked")
	}
}
