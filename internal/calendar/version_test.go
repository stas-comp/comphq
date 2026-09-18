package calendar

import (
	"context"
	"testing"
)

// SPEC gate 3.12 / D-15: every Calendar write bumps calendar_version in
// its own transaction, so the Briefing can cheaply notice a change made
// elsewhere; a write that fails changes nothing, the counter included.
func TestVersionStartsAtZeroAndBumpsOnEveryWrite(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	last := func() int64 {
		t.Helper()
		v, err := store.Version(ctx)
		if err != nil {
			t.Fatalf("Version: %v", err)
		}
		return v
	}
	if v := last(); v != 0 {
		t.Fatalf("Version on a fresh install = %d, want 0", v)
	}

	bumped := func(what string, before int64) int64 {
		t.Helper()
		after := last()
		if after <= before {
			t.Errorf("Version after %s = %d, want > %d", what, after, before)
		}
		return after
	}

	event, err := store.Create(ctx, Input{Title: "Concert", StartDate: "2026-12-12", Recurrence: RecurrenceYearly}, sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	v := bumped("Create", 0)

	if err := store.Update(ctx, event.ID, Input{Title: "Concert!", StartDate: "2026-12-12", Recurrence: RecurrenceYearly}, sam, fixedNow); err != nil {
		t.Fatalf("Update: %v", err)
	}
	v = bumped("Update", v)

	if err := store.SetMovedException(ctx, event.ID, "2026-12-12", "2026-12-19", "", "", "", sam, fixedNow); err != nil {
		t.Fatalf("SetMovedException: %v", err)
	}
	v = bumped("SetMovedException", v)

	if err := store.SetCancelledException(ctx, event.ID, "2027-12-12", sam, fixedNow); err != nil {
		t.Fatalf("SetCancelledException: %v", err)
	}
	v = bumped("SetCancelledException", v)

	if err := store.Remove(ctx, event.ID, sam, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	v = bumped("Remove", v)

	if err := store.Restore(ctx, event.ID, sam, fixedNow); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	v = bumped("Restore", v)

	// Failed writes leave the counter alone.
	if err := store.Update(ctx, 9999, Input{Title: "Nope", StartDate: "2026-12-12"}, sam, fixedNow); err != ErrEventNotFound {
		t.Fatalf("Update of a missing event = %v, want ErrEventNotFound", err)
	}
	if err := store.SetCancelledException(ctx, 9999, "2026-12-12", sam, fixedNow); err != ErrEventNotFound {
		t.Fatalf("SetCancelledException on a missing event = %v, want ErrEventNotFound", err)
	}
	if err := store.Remove(ctx, 9999, sam, fixedNow); err != ErrEventNotFound {
		t.Fatalf("Remove of a missing event = %v, want ErrEventNotFound", err)
	}
	if after := last(); after != v {
		t.Errorf("Version after failed writes = %d, want unchanged at %d", after, v)
	}
}
