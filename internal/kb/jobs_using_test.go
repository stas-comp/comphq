package kb

import (
	"context"
	"testing"
)

func publishTitled(t *testing.T, store *ArticleStore, cat int64, person int64, title string) Article {
	t.Helper()
	a, err := store.Publish(context.Background(), ArticleInput{CategoryID: cat, Title: title, BodyHTML: "<p>Body.</p>"}, person)
	if err != nil {
		t.Fatalf("Publish(%q): %v", title, err)
	}
	return a
}

// Gate 8.11: the picker's title match, best match first.
func TestGate811FindByTitleOrdersPrefixMatchesFirstThenAlphabetical(t *testing.T) {
	sqlDB := openTestDB(t)
	cat, _ := (&CategoryStore{DB: sqlDB}).Create("Mailing")
	person := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}
	publishTitled(t, store, cat.ID, person, "Printing the mailing labels")
	publishTitled(t, store, cat.ID, person, "Mailing SOP")
	publishTitled(t, store, cat.ID, person, "Mailing lists 2026")
	publishTitled(t, store, cat.ID, person, "Électricité du bureau")
	publishTitled(t, store, cat.ID, person, "Changing the toner")
	archived := publishTitled(t, store, cat.ID, person, "Mailing archive")
	if err := store.Archive(context.Background(), archived.ID, person); err != nil {
		t.Fatal(err)
	}

	titles := func(q string, limit int) []string {
		var out []string
		for _, r := range store.FindByTitle(context.Background(), q, limit) {
			out = append(out, r.Title)
		}
		return out
	}
	if got, want := titles("mail", 10), []string{"Mailing lists 2026", "Mailing SOP", "Printing the mailing labels"}; !sameStrs(got, want) {
		t.Errorf("mail = %v, want %v (title-prefix first, then alphabetical, archived left out)", got, want)
	}
	if got, want := titles("lists mail", 10), []string{"Mailing lists 2026"}; !sameStrs(got, want) {
		t.Errorf("every word has to appear: %v, want %v", got, want)
	}
	if got, want := titles("MAILING SO", 10), []string{"Mailing SOP"}; !sameStrs(got, want) {
		t.Errorf("a half-typed last word: %v, want %v", got, want)
	}
	if got := titles("électricité", 10); len(got) != 1 {
		t.Errorf("accented match: %v", got)
	}
	if got := titles("ÉLECTRICITÉ", 10); len(got) != 1 {
		t.Errorf("Unicode case folding: %v", got)
	}
	if got := titles("m", 10); len(got) != 0 {
		t.Errorf("a one-letter query finds %v, want nothing", got)
	}
	if got := titles("mail", 2); len(got) != 2 {
		t.Errorf("limit 2 returned %v", got)
	}
	if got := titles("nothing like this", 10); len(got) != 0 {
		t.Errorf("no match returned %v", got)
	}
	// The category comes with it.
	if r := store.FindByTitle(context.Background(), "toner", 5); len(r) != 1 || r[0].Category != "Mailing" || r[0].Archived {
		t.Errorf("toner = %+v", r)
	}
}

func TestArticlesReturnsArchivedOnesMarked(t *testing.T) {
	sqlDB := openTestDB(t)
	cat, _ := (&CategoryStore{DB: sqlDB}).Create("HR")
	person := testPerson(t, sqlDB)
	store := &ArticleStore{DB: sqlDB}
	live := publishTitled(t, store, cat.ID, person, "Leave policy")
	old := publishTitled(t, store, cat.ID, person, "Old policy")
	if err := store.Archive(context.Background(), old.ID, person); err != nil {
		t.Fatal(err)
	}
	got := store.Articles(context.Background(), []int64{live.ID, old.ID, 9999})
	if len(got) != 2 {
		t.Fatalf("Articles = %+v", got)
	}
	if r := got[live.ID]; r.Title != "Leave policy" || r.Category != "HR" || r.Archived {
		t.Errorf("live = %+v", r)
	}
	if r := got[old.ID]; r.Title != "Old policy" || !r.Archived {
		t.Errorf("old = %+v", r)
	}
	if got := store.Articles(context.Background(), nil); len(got) != 0 {
		t.Errorf("no ids = %+v", got)
	}
}

func sameStrs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
