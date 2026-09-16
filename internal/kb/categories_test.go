package kb

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

	// kb_articles.created_by/updated_by reference people(id), so people's
	// own migration needs to run too, in the same order the real registry
	// applies them (people before kb).
	for _, section := range []string{"people", "kb"} {
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

func TestCategoryCreateAndList(t *testing.T) {
	store := &CategoryStore{DB: openTestDB(t)}

	if _, err := store.Create("Printers"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := store.Create("Wi-Fi"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	categories, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(categories) != 2 || categories[0].Name != "Printers" || categories[1].Name != "Wi-Fi" {
		t.Errorf("List() = %+v, want [Printers Wi-Fi] in creation order", categories)
	}
}

func TestCategoryReorder(t *testing.T) {
	store := &CategoryStore{DB: openTestDB(t)}

	if _, err := store.Create("A"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, _ := store.Create("B")
	c, _ := store.Create("C")

	// Move B up: order becomes B, A, C.
	if err := store.MoveUp(b.ID); err != nil {
		t.Fatalf("MoveUp: %v", err)
	}
	categories, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := names(categories); got[0] != "B" || got[1] != "A" || got[2] != "C" {
		t.Errorf("after MoveUp(B) = %v, want [B A C]", got)
	}

	// Move C up twice: order becomes C, B, A.
	if err := store.MoveUp(c.ID); err != nil {
		t.Fatalf("MoveUp: %v", err)
	}
	if err := store.MoveUp(c.ID); err != nil {
		t.Fatalf("MoveUp: %v", err)
	}
	categories, _ = store.List()
	if got := names(categories); got[0] != "C" || got[1] != "B" || got[2] != "A" {
		t.Errorf("after MoveUp(C) x2 = %v, want [C B A]", got)
	}

	// Moving the first item up further is a no-op.
	if err := store.MoveUp(c.ID); err != nil {
		t.Fatalf("MoveUp (no-op): %v", err)
	}
	categories, _ = store.List()
	if got := names(categories); got[0] != "C" {
		t.Errorf("MoveUp at the top changed order: %v", got)
	}

	// MoveDown brings it back.
	if err := store.MoveDown(c.ID); err != nil {
		t.Fatalf("MoveDown: %v", err)
	}
	categories, _ = store.List()
	if got := names(categories); got[0] != "B" || got[1] != "C" {
		t.Errorf("after MoveDown(C) = %v, want [B C A]", got)
	}
}

func names(categories []Category) []string {
	out := make([]string, len(categories))
	for i, c := range categories {
		out[i] = c.Name
	}
	return out
}

func TestCategoryDeleteEmpty(t *testing.T) {
	store := &CategoryStore{DB: openTestDB(t)}

	c, err := store.Create("Empty")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if count, err := store.Delete(c.ID); err != nil {
		t.Errorf("Delete: %v (count %d)", err, count)
	}

	categories, _ := store.List()
	if len(categories) != 0 {
		t.Errorf("List() after delete = %+v, want empty", categories)
	}
}

func TestCategoryDeleteRefusedWithArticles(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &CategoryStore{DB: sqlDB}

	c, err := store.Create("Printers")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Both published and archived articles count (SPEC gate 1.12:
	// "Archived articles still count, because they can be restored").
	if _, err := sqlDB.Exec(
		`INSERT INTO kb_articles (category_id, title, status, created_at, updated_at) VALUES (?, 'A', 'published', '', ''), (?, 'B', 'archived', '', '')`,
		c.ID, c.ID,
	); err != nil {
		t.Fatalf("seed articles: %v", err)
	}

	count, err := store.Delete(c.ID)
	if err != ErrCategoryHasArticles {
		t.Errorf("Delete error = %v, want ErrCategoryHasArticles", err)
	}
	if count != 2 {
		t.Errorf("Delete article count = %d, want 2", count)
	}

	categories, _ := store.List()
	if len(categories) != 1 {
		t.Error("category was deleted despite having articles")
	}
}

func TestCategoryCreateRejectsEmptyName(t *testing.T) {
	store := &CategoryStore{DB: openTestDB(t)}
	if _, err := store.Create("   "); err != ErrEmptyName {
		t.Errorf("Create(blank) error = %v, want ErrEmptyName", err)
	}
}
