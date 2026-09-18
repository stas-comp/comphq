package calendar

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

	for _, section := range []string{"app", "people", "calendar"} {
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

func TestCreateStoresEveryField(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{
		Title:      "  Christmas concert  ",
		Notes:      "Bring programmes",
		StartDate:  "2026-12-12",
		EndDate:    "2026-12-13",
		StartTime:  "18:00",
		EndTime:    "20:00",
		Recurrence: RecurrenceYearly,
		UntilDate:  "2030-12-31",
		NoticeDays: 14,
	}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if event.Title != "Christmas concert" {
		t.Errorf("Title = %q, want trimmed %q", event.Title, "Christmas concert")
	}
	if event.Notes != "Bring programmes" || event.StartDate != "2026-12-12" || event.EndDate != "2026-12-13" ||
		event.StartTime != "18:00" || event.EndTime != "20:00" || event.Recurrence != RecurrenceYearly ||
		event.UntilDate != "2030-12-31" || event.NoticeDays != 14 {
		t.Errorf("Create round-trip = %+v, a field didn't survive", event)
	}
	if event.UpdatedByName != "Sam" {
		t.Errorf("UpdatedByName = %q, want Sam", event.UpdatedByName)
	}
	if event.UpdatedAt != "Thu 17 Sep 2026, 12:00" {
		t.Errorf("UpdatedAt = %q, want the pre-formatted date/time", event.UpdatedAt)
	}
}

func TestCreateDefaultsRecurrenceToNone(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	event, err := store.Create(context.Background(), Input{Title: "One-off", StartDate: "2026-10-01"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if event.Recurrence != RecurrenceNone {
		t.Errorf("Recurrence = %q, want %q", event.Recurrence, RecurrenceNone)
	}
}

func TestCreateBlankTitleIsRejected(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	_, err := store.Create(context.Background(), Input{Title: "   ", StartDate: "2026-10-01"}, sam, fixedNow)
	if err != ErrEmptyTitle {
		t.Errorf("err = %v, want ErrEmptyTitle", err)
	}
}

func TestCreateEndDateBeforeStartIsRejected(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	_, err := store.Create(context.Background(), Input{
		Title: "Backwards", StartDate: "2026-10-05", EndDate: "2026-10-01",
	}, sam, fixedNow)
	if err != ErrEndDateBeforeStart {
		t.Errorf("err = %v, want ErrEndDateBeforeStart", err)
	}
}

func TestCreateEndTimeWithoutStartTimeIsRejected(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	_, err := store.Create(context.Background(), Input{
		Title: "No start time", StartDate: "2026-10-05", EndTime: "10:00",
	}, sam, fixedNow)
	if err != ErrEndTimeNeedsStartTime {
		t.Errorf("err = %v, want ErrEndTimeNeedsStartTime", err)
	}
}

func TestCreateInvalidRecurrenceIsRejected(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	_, err := store.Create(context.Background(), Input{
		Title: "Bad repeat", StartDate: "2026-10-05", Recurrence: "daily",
	}, sam, fixedNow)
	if err != ErrInvalidRecurrence {
		t.Errorf("err = %v, want ErrInvalidRecurrence", err)
	}
}

// SPEC gate 2.20: an event's details show who last changed it and
// when — Update records a fresh actor/time even though Create already
// set them.
func TestUpdateRecordsLastChangedByAndWhen(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Original", StartDate: "2026-10-01"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	later := fixedNow.Add(24 * time.Hour)
	if err := store.Update(ctx, event.ID, Input{Title: "Edited", StartDate: "2026-10-02"}, alex, later); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.Get(ctx, event.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Edited" || got.StartDate != "2026-10-02" {
		t.Errorf("Get after Update = %+v, want the edited fields", got)
	}
	if got.UpdatedByName != "Alex" {
		t.Errorf("UpdatedByName = %q, want Alex", got.UpdatedByName)
	}
	if got.UpdatedAt != "Fri 18 Sep 2026, 12:00" {
		t.Errorf("UpdatedAt = %q, want the later date/time", got.UpdatedAt)
	}
}

func TestUpdateMissingEventIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	err := store.Update(context.Background(), 9999, Input{Title: "Ghost", StartDate: "2026-10-01"}, sam, fixedNow)
	if err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
}

func TestGetMissingEventIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}

	_, err := store.Get(context.Background(), 9999)
	if err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
}

// SPEC gates 2.12/2.14: Occurrences expands a repeating series and
// respects the query window, wiring straight onto recur.Occurrences.
func TestOccurrencesExpandsARepeatingSeries(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{
		Title: "Weekly sync", StartDate: "2026-09-07", Recurrence: RecurrenceWeekly,
	}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// A second event outside the query window shouldn't appear at all.
	if _, err := store.Create(ctx, Input{Title: "Next year", StartDate: "2027-01-01"}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}

	occs, err := store.Occurrences(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	want := []string{"2026-09-07", "2026-09-14", "2026-09-21", "2026-09-28"}
	if len(occs) != len(want) {
		t.Fatalf("occurrences = %v, want %d dates", occs, len(want))
	}
	for i, w := range want {
		if occs[i].StartDate != w {
			t.Errorf("occs[%d].StartDate = %q, want %q", i, occs[i].StartDate, w)
		}
		if occs[i].EventID != event.ID {
			t.Errorf("occs[%d].EventID = %d, want %d", i, occs[i].EventID, event.ID)
		}
		if occs[i].Title != "Weekly sync" {
			t.Errorf("occs[%d].Title = %q, want %q", i, occs[i].Title, "Weekly sync")
		}
	}
}

func TestOccurrencesExcludesRemovedEvents(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Soon removed", StartDate: "2026-09-10"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := sqlDB.Exec(`UPDATE events SET removed_at = ? WHERE id = ?`, fixedNow.UTC().Format(time.RFC3339), event.ID); err != nil {
		t.Fatalf("mark removed: %v", err)
	}

	occs, err := store.Occurrences(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	if len(occs) != 0 {
		t.Errorf("occurrences = %v, want none (event is removed)", occs)
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

// SPEC gate 2.15: the exact 2.15 script at the store layer — a yearly
// Christmas concert moved just once shows on its new date, and the
// other year is unaffected.
func TestSetMovedExceptionAppliesOnlyToThatOccurrence(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{
		Title: "Christmas concert", StartDate: "2026-12-12", Recurrence: RecurrenceYearly,
	}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.SetMovedException(ctx, event.ID, "2026-12-12", "2026-12-19", "", "", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException: %v", err)
	}

	occs, err := store.Occurrences(ctx, date(t, "2026-01-01"), date(t, "2027-12-31"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	want := []string{"2026-12-19", "2027-12-12"}
	if len(occs) != len(want) {
		t.Fatalf("occurrences = %v, want dates %v", occs, want)
	}
	for i, w := range want {
		if occs[i].StartDate != w {
			t.Errorf("occs[%d].StartDate = %q, want %q", i, occs[i].StartDate, w)
		}
		if occs[i].Title != "Christmas concert" {
			t.Errorf("occs[%d].Title = %q, want the series title even though moved", i, occs[i].Title)
		}
	}
}

// Editing an already-moved occurrence again replaces its exception
// rather than failing on the composite primary key.
func TestSetMovedExceptionCanBeEditedAgain(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Weekly sync", StartDate: "2026-03-02", Recurrence: RecurrenceWeekly}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "2026-03-10", "", "", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException (first): %v", err)
	}
	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "2026-03-11", "", "", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException (second): %v", err)
	}

	occs, err := store.Occurrences(ctx, date(t, "2026-03-01"), date(t, "2026-03-31"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	var found bool
	for _, o := range occs {
		if o.StartDate == "2026-03-10" {
			t.Errorf("occurrences still show the first move (2026-03-10); want only the latest edit")
		}
		if o.StartDate == "2026-03-11" {
			found = true
		}
	}
	if !found {
		t.Errorf("occurrences = %v, want the latest move (2026-03-11)", occs)
	}
}

// SPEC gate 2.16: "Cancel just this one" hides only that occurrence.
func TestSetCancelledExceptionRemovesOnlyThatOccurrence(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Weekly sync", StartDate: "2026-03-02", Recurrence: RecurrenceWeekly}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SetCancelledException(ctx, event.ID, "2026-03-09", sam, fixedNow); err != nil {
		t.Fatalf("SetCancelledException: %v", err)
	}

	occs, err := store.Occurrences(ctx, date(t, "2026-03-01"), date(t, "2026-03-31"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	want := []string{"2026-03-02", "2026-03-16", "2026-03-23", "2026-03-30"}
	if len(occs) != len(want) {
		t.Fatalf("occurrences = %v, want %v", occs, want)
	}
	for i, w := range want {
		if occs[i].StartDate != w {
			t.Errorf("occs[%d].StartDate = %q, want %q", i, occs[i].StartDate, w)
		}
	}
}

func TestSetMovedExceptionValidation(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Weekly sync", StartDate: "2026-03-02", Recurrence: RecurrenceWeekly}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "", "", "", "", sam, fixedNow); err != ErrEmptyStartDate {
		t.Errorf("empty new start date: err = %v, want ErrEmptyStartDate", err)
	}
	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "2026-03-10", "2026-03-09", "", "", sam, fixedNow); err != ErrEndDateBeforeStart {
		t.Errorf("end date before start: err = %v, want ErrEndDateBeforeStart", err)
	}
	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "2026-03-10", "", "", "10:00", sam, fixedNow); err != ErrEndTimeNeedsStartTime {
		t.Errorf("end time without start time: err = %v, want ErrEndTimeNeedsStartTime", err)
	}
}

func TestSetExceptionMissingEventIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	if err := store.SetMovedException(context.Background(), 9999, "2026-03-09", "2026-03-10", "", "", "", sam, fixedNow); err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
	if err := store.SetCancelledException(context.Background(), 9999, "2026-03-09", sam, fixedNow); err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
}

func TestOccurrenceForEditPrefillsFromSeriesWhenNoException(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{
		Title: "Offsite", StartDate: "2026-01-30", EndDate: "2026-02-02",
		StartTime: "09:00", EndTime: "17:00", Recurrence: RecurrenceYearly,
	}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.OccurrenceForEdit(ctx, event.ID, "2026-01-30")
	if err != nil {
		t.Fatalf("OccurrenceForEdit: %v", err)
	}
	if got.StartDate != "2026-01-30" || got.EndDate != "2026-02-02" || got.StartTime != "09:00" || got.EndTime != "17:00" {
		t.Errorf("OccurrenceForEdit = %+v, want the series' own dates/times with length preserved", got)
	}
}

func TestOccurrenceForEditPrefillsFromExistingException(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Weekly sync", StartDate: "2026-03-02", Recurrence: RecurrenceWeekly}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.SetMovedException(ctx, event.ID, "2026-03-09", "2026-03-10", "", "14:00", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException: %v", err)
	}

	got, err := store.OccurrenceForEdit(ctx, event.ID, "2026-03-09")
	if err != nil {
		t.Fatalf("OccurrenceForEdit: %v", err)
	}
	if got.StartDate != "2026-03-10" || got.StartTime != "14:00" {
		t.Errorf("OccurrenceForEdit = %+v, want the existing exception's own dates/times", got)
	}
}

// SPEC gate 2.18: Remove hides the whole event; Restore brings it back.
func TestRemoveAndRestore(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Removable", StartDate: "2026-09-10"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Remove(ctx, event.ID, sam, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	occs, err := store.Occurrences(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	if len(occs) != 0 {
		t.Errorf("occurrences after Remove = %v, want none", occs)
	}
	removed, err := store.ListRemoved(ctx)
	if err != nil {
		t.Fatalf("ListRemoved: %v", err)
	}
	if len(removed) != 1 || removed[0].ID != event.ID {
		t.Errorf("ListRemoved = %+v, want just the removed event", removed)
	}

	if err := store.Restore(ctx, event.ID, sam, fixedNow); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	occs, err = store.Occurrences(ctx, date(t, "2026-09-01"), date(t, "2026-09-30"))
	if err != nil {
		t.Fatalf("Occurrences: %v", err)
	}
	if len(occs) != 1 {
		t.Errorf("occurrences after Restore = %v, want the event back", occs)
	}
	removed, err = store.ListRemoved(ctx)
	if err != nil {
		t.Fatalf("ListRemoved: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("ListRemoved after Restore = %v, want none", removed)
	}
}

func TestRemoveMissingEventIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")

	if err := store.Remove(context.Background(), 9999, sam, fixedNow); err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
}

func TestRestoreNotRemovedEventIsError(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	event, err := store.Create(ctx, Input{Title: "Not removed", StartDate: "2026-09-10"}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Restore(ctx, event.ID, sam, fixedNow); err != ErrEventNotFound {
		t.Errorf("err = %v, want ErrEventNotFound", err)
	}
}
