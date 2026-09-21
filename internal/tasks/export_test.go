package tasks

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// SPEC gate 2.22: tasks.csv has plain-English headers, readable dates,
// and every task — removed ones included, with their Removed date.
func TestExportRows(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{
		Title: "Fix printer", Stage: StageTodo, Size: "L", DueDate: "2026-09-19", PersonIDs: []int64{sam, alex},
	}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	removed, err := store.Create(ctx, CreateInput{Title: "Old idea"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Remove(ctx, removed.ID, sam, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := store.ExportRows(ctx)
	if err != nil {
		t.Fatalf("ExportRows: %v", err)
	}
	want := [][]string{
		{"Title", "Column", "Size", "People", "Due date", "Created by", "Created", "Finished", "Removed"},
		{"Fix printer", "To do", "Large", "Sam, Alex", "Sat 19 Sep 2026", "Sam", "Thu 17 Sep 2026, 12:00", "", ""},
		{"Old idea", "Ideas", "Medium", "", "", "Sam", "Thu 17 Sep 2026, 12:00", "", "Thu 17 Sep 2026, 12:00"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExportRows =\n%v\nwant\n%v", got, want)
	}
}

// SPEC gate 5.14: steps.csv lists every step of every job with plain-English
// headers and readable dates; a removed step is still there, marked with the
// date it was removed, and so are the steps of a removed job.
func TestStepsCSVRows(t *testing.T) {
	f := newStepFixture(t)
	ctx := context.Background()
	one := f.add(t, "Print exam papers")
	two := f.add(t, "Book the hall")
	f.add(t, "Email the parents, and \"the office\"")
	tickedAt := time.Date(2026, 9, 19, 9, 30, 0, 0, time.UTC)
	if err := f.store.SetStepDone(ctx, f.task.ID, one.ID, true, f.alex, tickedAt); err != nil {
		t.Fatal(err)
	}
	removedAt := time.Date(2026, 9, 20, 16, 5, 0, 0, time.UTC)
	if err := f.store.RemoveStep(ctx, f.task.ID, two.ID, f.sam, removedAt); err != nil {
		t.Fatal(err)
	}
	gone := createNamed(t, f.store, ctx, f.sam, "Old job", StageIdea)
	if _, err := f.store.AddStep(ctx, gone.ID, "Left behind", f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Remove(ctx, gone.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	got, err := StepsCSV{DB: f.sqlDB}.ExportRows(ctx)
	if err != nil {
		t.Fatalf("ExportRows: %v", err)
	}
	want := [][]string{
		{"Job", "Step", "Done", "Ticked by", "Ticked", "Removed"},
		{"Concert", "Print exam papers", "Yes", "Alex", "Sat 19 Sep 2026, 09:30", ""},
		{"Concert", "Email the parents, and \"the office\"", "No", "", "", ""},
		{"Concert", "Book the hall", "No", "", "", "Sun 20 Sep 2026, 16:05"},
		{"Old job", "Left behind", "No", "", "", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("StepsCSV rows =\n%v\nwant\n%v", got, want)
	}
}
