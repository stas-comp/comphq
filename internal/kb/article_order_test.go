package kb

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// SPEC gates 7.55-7.59 (B13.8, D-98): articles in their own order.

type orderFixture struct {
	t      *testing.T
	db     *sql.DB
	store  *ArticleStore
	person int64
}

func newOrderFixture(t *testing.T) *orderFixture {
	t.Helper()
	db := openTestDB(t)
	return &orderFixture{t: t, db: db, store: &ArticleStore{DB: db}, person: testPerson(t, db)}
}

func (f *orderFixture) category(name string) int64 {
	f.t.Helper()
	c, err := (&CategoryStore{DB: f.db}).Create(name)
	if err != nil {
		f.t.Fatal(err)
	}
	return c.ID
}

func (f *orderFixture) publish(cat int64, title string) Article {
	f.t.Helper()
	a, err := f.store.Publish(context.Background(), ArticleInput{CategoryID: cat, Title: title, BodyHTML: "<p>" + title + "</p>"}, f.person)
	if err != nil {
		f.t.Fatal(err)
	}
	return a
}

func (f *orderFixture) order(cat int64) []string {
	f.t.Helper()
	list, err := f.store.ListByCategory(cat)
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, a := range list {
		out = append(out, a.Title)
	}
	return out
}

func (f *orderFixture) positions(cat int64) []int {
	f.t.Helper()
	rows, err := f.db.Query(`SELECT position FROM kb_articles WHERE category_id = ? AND status = 'published' ORDER BY position`, cat)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var p sql.NullInt64
		if err := rows.Scan(&p); err != nil {
			f.t.Fatal(err)
		}
		out = append(out, int(p.Int64))
	}
	return out
}

func TestNewArticleGoesToTheBottomOfItsCategory(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	other := f.category("Phones")
	f.publish(cat, "A")
	f.publish(cat, "B")
	f.publish(other, "X")
	f.publish(cat, "C")
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("order = %v, want creation order, new at the bottom", got)
	}
	if got := f.positions(cat); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("positions = %v", got)
	}
	if got := f.order(other); !reflect.DeepEqual(got, []string{"X"}) {
		t.Errorf("other category = %v", got)
	}
}

func TestEditingAnArticleDoesNotMoveIt(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	a := f.publish(cat, "A")
	f.publish(cat, "B")
	f.publish(cat, "C")
	time.Sleep(1100 * time.Millisecond) // a later updated_at than the others
	if _, err := f.store.Publish(context.Background(), ArticleInput{
		ID: a.ID, CategoryID: cat, Title: "A edited", BodyHTML: "<p>new</p>", ExpectedVersion: a.VersionNo,
	}, f.person); err != nil {
		t.Fatal(err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"A edited", "B", "C"}) {
		t.Errorf("order after an edit = %v, want it still first", got)
	}
}

func TestMoveReordersWithoutAnEditHistoryOrUpdatedChange(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	a, b, c, d := f.publish(cat, "A"), f.publish(cat, "B"), f.publish(cat, "C"), f.publish(cat, "D")
	ctx := context.Background()

	var updatedBefore string
	var versionBefore int
	var historyBefore int
	if err := f.db.QueryRow(`SELECT updated_at, version_no FROM kb_articles WHERE id = ?`, c.ID).Scan(&updatedBefore, &versionBefore); err != nil {
		t.Fatal(err)
	}
	f.db.QueryRow(`SELECT COUNT(*) FROM kb_article_versions`).Scan(&historyBefore)
	recentBefore, _ := recentArticles(f.db, 10)

	steps := []struct {
		move MoveInput
		id   int64
		want []string
	}{
		{MoveInput{Direction: "up"}, c.ID, []string{"A", "C", "B", "D"}},
		{MoveInput{Direction: "up"}, c.ID, []string{"C", "A", "B", "D"}},
		{MoveInput{Direction: "up"}, c.ID, []string{"C", "A", "B", "D"}}, // already first: nothing
		{MoveInput{Direction: "down"}, d.ID, []string{"C", "A", "B", "D"}},
		{MoveInput{BeforeID: a.ID}, d.ID, []string{"C", "D", "A", "B"}},
		{MoveInput{AfterID: b.ID}, c.ID, []string{"D", "A", "B", "C"}},
		{MoveInput{AfterID: d.ID}, b.ID, []string{"D", "B", "A", "C"}},
	}
	for i, s := range steps {
		if err := f.store.Move(ctx, s.id, s.move); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if got := f.order(cat); !reflect.DeepEqual(got, s.want) {
			t.Errorf("step %d: order = %v, want %v", i, got, s.want)
		}
		if got := f.positions(cat); !reflect.DeepEqual(got, []int{1, 2, 3, 4}) {
			t.Errorf("step %d: positions = %v, want 1..4", i, got)
		}
	}

	var updatedAfter string
	var versionAfter, historyAfter int
	f.db.QueryRow(`SELECT updated_at, version_no FROM kb_articles WHERE id = ?`, c.ID).Scan(&updatedAfter, &versionAfter)
	f.db.QueryRow(`SELECT COUNT(*) FROM kb_article_versions`).Scan(&historyAfter)
	if updatedAfter != updatedBefore || versionAfter != versionBefore || historyAfter != historyBefore {
		t.Errorf("a move changed updated_at/version/history: %s→%s, v%d→v%d, history %d→%d",
			updatedBefore, updatedAfter, versionBefore, versionAfter, historyBefore, historyAfter)
	}
	recentAfter, _ := recentArticles(f.db, 10)
	if !reflect.DeepEqual(recentBefore, recentAfter) {
		t.Errorf("Recently updated changed after a move")
	}
}

func TestMoveRefusesAnArticleOrNeighbourThatIsNotInTheList(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	other := f.category("Phones")
	a, b := f.publish(cat, "A"), f.publish(cat, "B")
	x := f.publish(other, "X")
	ctx := context.Background()
	if err := f.store.Move(ctx, a.ID, MoveInput{BeforeID: x.ID}); err != ErrArticleNotInOrder {
		t.Errorf("neighbour in another category: %v, want ErrArticleNotInOrder", err)
	}
	if err := f.store.Move(ctx, 9999, MoveInput{Direction: "up"}); err != ErrArticleNotInOrder {
		t.Errorf("missing article: %v", err)
	}
	if err := f.store.Archive(ctx, b.ID, f.person); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Move(ctx, b.ID, MoveInput{Direction: "up"}); err != ErrArticleNotInOrder {
		t.Errorf("archived article: %v", err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("order = %v", got)
	}
}

func TestMovingCategoryRestoringAndUnarchivingGoToTheBottom(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	other := f.category("Phones")
	a, b, c := f.publish(cat, "A"), f.publish(cat, "B"), f.publish(cat, "C")
	f.publish(other, "X")
	f.publish(other, "Y")
	ctx := context.Background()

	// Moved to another category on save: the bottom of that one; the old one closes up.
	if _, err := f.store.Publish(ctx, ArticleInput{ID: a.ID, CategoryID: other, Title: "A", BodyHTML: "<p>A</p>", ExpectedVersion: a.VersionNo}, f.person); err != nil {
		t.Fatal(err)
	}
	if got := f.order(other); !reflect.DeepEqual(got, []string{"X", "Y", "A"}) {
		t.Errorf("new category = %v, want A at the bottom", got)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Errorf("old category = %v", got)
	}
	if got := f.positions(cat); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Errorf("old category positions = %v, want closed up", got)
	}

	// Archived and brought back: the bottom of its category.
	if err := f.store.Archive(ctx, b.ID, f.person); err != nil {
		t.Fatal(err)
	}
	if got := f.positions(cat); !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("after archiving: positions = %v", got)
	}
	if err := f.store.Unarchive(ctx, b.ID, f.person); err != nil {
		t.Fatal(err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"C", "B"}) {
		t.Errorf("after unarchive = %v, want B at the bottom", got)
	}

	// Restoring an old version that was in another category: the bottom there.
	if _, err := f.store.Restore(ctx, c.ID, 1, f.person); err != nil {
		t.Fatal(err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"C", "B"}) {
		t.Errorf("a restore into the same category keeps the place: %v", got)
	}
	if _, err := f.store.Publish(ctx, ArticleInput{ID: c.ID, CategoryID: other, Title: "C", BodyHTML: "<p>C2</p>", ExpectedVersion: 2}, f.person); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Restore(ctx, c.ID, 1, f.person); err != nil { // version 1 was in cat
		t.Fatal(err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Errorf("restored into its old category = %v, want C at the bottom", got)
	}
}

// Gate 7.58: on upgrade day each category shows what 1.3.0 showed (most
// recently edited first), by running the migration's own backfill.
func TestBackfillKeepsTheOrderOfVersion13(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	other := f.category("Phones")
	ids := map[string]int64{}
	for i, title := range []string{"Old", "Middle", "New"} {
		a := f.publish(cat, title)
		ids[title] = a.ID
		// Distinct edit times, oldest first.
		f.db.Exec(`UPDATE kb_articles SET updated_at = ? WHERE id = ?`, time.Date(2026, 9, 1+i, 12, 0, 0, 0, time.UTC).Format(time.RFC3339), a.ID)
	}
	x := f.publish(other, "X")
	arch := f.publish(cat, "Archived one")
	if err := f.store.Archive(context.Background(), arch.ID, f.person); err != nil {
		t.Fatal(err)
	}
	_ = x
	// Put the database back to how 1.3.0 left it, then run the migration's UPDATE.
	f.db.Exec(`UPDATE kb_articles SET position = NULL`)
	raw, err := os.ReadFile("../../migrations/kb/0006_article_order.sql")
	if err != nil {
		t.Fatal(err)
	}
	update := string(raw)[strings.Index(string(raw), "UPDATE kb_articles"):]
	if _, err := f.db.Exec(update); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if got := f.order(cat); !reflect.DeepEqual(got, []string{"New", "Middle", "Old"}) {
		t.Errorf("order after the backfill = %v, want most recently edited first", got)
	}
	if got := f.positions(cat); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("positions = %v", got)
	}
}

// Gate 7.59: at every start-up each category is tidied to 1..n, with articles
// written while rolled back (no number) at the bottom, and none twice or lost.
func TestRenumberAllTidiesNullsStaleNumbersAndArchived(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	other := f.category("Phones")
	a, b, c := f.publish(cat, "A"), f.publish(cat, "B"), f.publish(cat, "C")
	x := f.publish(other, "X")
	arch := f.publish(cat, "Archived")
	if err := f.store.Archive(context.Background(), arch.ID, f.person); err != nil {
		t.Fatal(err)
	}
	// What a rolled-back 1.3.0 leaves behind: new articles with no number (two of
	// them, created at different times), a gap, a duplicate, and an article moved
	// to another category that kept its old number.
	f.db.Exec(`UPDATE kb_articles SET position = 5 WHERE id = ?`, a.ID)
	f.db.Exec(`UPDATE kb_articles SET position = 5 WHERE id = ?`, b.ID)
	f.db.Exec(`UPDATE kb_articles SET position = 2 WHERE id = ?`, c.ID)
	f.db.Exec(`UPDATE kb_articles SET category_id = ? WHERE id = ?`, other, c.ID) // moved, stale number 2
	f.db.Exec(`UPDATE kb_articles SET position = 1 WHERE id = ?`, x.ID)
	f.db.Exec(`UPDATE kb_articles SET position = 9 WHERE id = ?`, arch.ID)
	n1 := f.publish(cat, "New on 1.3.0 first")
	n2 := f.publish(cat, "New on 1.3.0 second")
	f.db.Exec(`UPDATE kb_articles SET position = NULL, created_at = '2026-09-02T00:00:00Z' WHERE id = ?`, n1.ID)
	f.db.Exec(`UPDATE kb_articles SET position = NULL, created_at = '2026-09-03T00:00:00Z' WHERE id = ?`, n2.ID)

	for pass := 0; pass < 2; pass++ { // and it is idempotent
		if err := RenumberAll(context.Background(), f.db); err != nil {
			t.Fatal(err)
		}
		if got := f.order(cat); !reflect.DeepEqual(got, []string{"A", "B", "New on 1.3.0 first", "New on 1.3.0 second"}) {
			t.Errorf("pass %d: Printers = %v (the tie between A and B falls to creation order; nulls last)", pass, got)
		}
		if got := f.positions(cat); !reflect.DeepEqual(got, []int{1, 2, 3, 4}) {
			t.Errorf("pass %d: Printers positions = %v", pass, got)
		}
		if got := f.order(other); !reflect.DeepEqual(got, []string{"X", "C"}) {
			t.Errorf("pass %d: Phones = %v (X had 1, the moved C had a stale 2)", pass, got)
		}
		var archivedPos sql.NullInt64
		f.db.QueryRow(`SELECT position FROM kb_articles WHERE id = ?`, arch.ID).Scan(&archivedPos)
		if archivedPos.Valid {
			t.Errorf("pass %d: an archived article kept a position (%d)", pass, archivedPos.Int64)
		}
		var total, shown int
		f.db.QueryRow(`SELECT COUNT(*) FROM kb_articles WHERE status = 'published'`).Scan(&total)
		shown = len(f.order(cat)) + len(f.order(other))
		if total != shown {
			t.Errorf("pass %d: %d published, %d shown: one is missing or twice", pass, total, shown)
		}
	}
}

func TestExportListsArticlesInTheCategorysOrder(t *testing.T) {
	f := newOrderFixture(t)
	cat := f.category("Printers")
	a, _, c := f.publish(cat, "Apple"), f.publish(cat, "Banana"), f.publish(cat, "Cherry")
	if err := f.store.Move(context.Background(), c.ID, MoveInput{BeforeID: a.ID}); err != nil {
		t.Fatal(err)
	}
	cats, _, err := f.store.ListForExport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, c := range cats {
		for _, a := range c.Articles {
			titles = append(titles, a.Title)
		}
	}
	if !reflect.DeepEqual(titles, []string{"Cherry", "Apple", "Banana"}) {
		t.Errorf("export order = %v, want the category's own order, not by title", titles)
	}
}
