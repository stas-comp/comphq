package people

import (
	"database/sql"
	"testing"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	migrations, err := db.LoadMigrations(comphq.Migrations, "people")
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	if err := db.RunMigrations(sqlDB, "test", migrations); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return sqlDB
}

func TestCreateAndList(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	if _, err := store.Create("Sam"); err != nil {
		t.Fatalf("Create(Sam): %v", err)
	}
	if _, err := store.Create("Alex"); err != nil {
		t.Fatalf("Create(Alex): %v", err)
	}

	people, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(people) != 2 || people[0].Name != "Alex" || people[1].Name != "Sam" {
		t.Errorf("List() = %+v, want [Alex Sam] alphabetically", people)
	}
}

func TestCreateRejectsCaseInsensitiveDuplicate(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	if _, err := store.Create("Sam"); err != nil {
		t.Fatalf("Create(Sam): %v", err)
	}
	if _, err := store.Create("sam"); err != ErrDuplicateName {
		t.Errorf("Create(sam) error = %v, want ErrDuplicateName", err)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	for _, name := range []string{"", "   "} {
		if _, err := store.Create(name); err != ErrEmptyName {
			t.Errorf("Create(%q) error = %v, want ErrEmptyName", name, err)
		}
	}
}

func TestDeactivateRemovesFromListButKeepsRow(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	p, err := store.Create("Sam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Deactivate(p.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	people, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(people) != 0 {
		t.Errorf("List() after deactivate = %+v, want empty", people)
	}

	got, active, err := store.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if active {
		t.Error("Get() active = true, want false after Deactivate")
	}
	if got.Name != "Sam" {
		t.Errorf("Get().Name = %q, want Sam (history keeps the name)", got.Name)
	}
}

func TestCreateAllowsNameReuseAfterDeactivate(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	p, err := store.Create("Sam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Deactivate(p.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	if _, err := store.Create("Sam"); err != nil {
		t.Errorf("Create(Sam) after deactivate: %v, want nil (name freed up)", err)
	}
}

func TestGetUnknownPerson(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	_, active, err := store.Get(999)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if active {
		t.Error("Get() of an unknown id: active = true, want false")
	}
}

func TestRenameChangesTheName(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	p, err := store.Create("Sam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Rename(p.ID, "Samantha"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	got, _, err := store.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Samantha" {
		t.Errorf("Get().Name = %q, want Samantha", got.Name)
	}
}

func TestRenameToOwnCurrentNameIsNotADuplicate(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	p, err := store.Create("Sam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Rename(p.ID, "sam"); err != nil {
		t.Errorf("Rename to own name (different case) = %v, want nil", err)
	}
}

func TestRenameRejectsDuplicateOfAnotherActivePerson(t *testing.T) {
	store := &Store{DB: openTestDB(t)}

	if _, err := store.Create("Sam"); err != nil {
		t.Fatalf("Create(Sam): %v", err)
	}
	alex, err := store.Create("Alex")
	if err != nil {
		t.Fatalf("Create(Alex): %v", err)
	}

	if err := store.Rename(alex.ID, "sam"); err != ErrDuplicateName {
		t.Errorf("Rename(Alex -> sam) error = %v, want ErrDuplicateName", err)
	}
}
