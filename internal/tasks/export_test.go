package tasks

import (
	"context"
	"reflect"
	"testing"
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
