package tasks

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// SPEC gates 7.21-7.22, 7.25 (B13.4, D-94): job search.

func TestSearchWords(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"toner", []string{"toner"}},
		{"  Order   toner ", []string{"Order", "toner"}},
		{"a", nil},                         // fewer than 2 characters in total
		{"a b", []string{"a", "b"}},        // 2 in total is enough
		{"50%_off", []string{"50", "off"}}, // symbols just separate words
		{"", nil},
		{"!!!", nil},
		{"électricité", []string{"électricité"}},
	}
	for _, c := range cases {
		if got := SearchWords(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("SearchWords(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func searchTitles(t *testing.T, store *Store, query string) []string {
	t.Helper()
	hits, err := store.SearchJobs(context.Background(), query, 0, fixedNow)
	if err != nil {
		t.Fatalf("SearchJobs(%q): %v", query, err)
	}
	var out []string
	for _, h := range hits {
		out = append(out, strings.NewReplacer("<mark>", "", "</mark>", "").Replace(string(h.Title)))
	}
	return out
}

func TestSearchJobsMatchesEveryWordInTitleOrNotesCaseInsensitively(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	mk := func(title, notes, stage string) Task {
		task, err := store.Create(ctx, CreateInput{Title: title, Notes: notes, Stage: stage}, sam, fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		return task
	}
	mk("Fix the printer", "", StageTodo)
	mk("Order toner", "For the PRINTER on level two", StageTodo)
	mk("Water the plants", "nothing to see", StageTodo)
	mk("Électricité check", "", StageTodo)

	if got := searchTitles(t, store, "printer"); !reflect.DeepEqual(got, []string{"Fix the printer", "Order toner"}) {
		t.Errorf("printer = %v (title match first, then notes-only)", got)
	}
	if got := searchTitles(t, store, "toner printer"); !reflect.DeepEqual(got, []string{"Order toner"}) {
		t.Errorf("toner printer = %v, want just the one with both (one in the title, one in the notes)", got)
	}
	if got := searchTitles(t, store, "prin"); len(got) != 2 {
		t.Errorf("prin = %v, want the half-typed word to find printer twice", got)
	}
	if got := searchTitles(t, store, "ÉLECTRICITÉ"); !reflect.DeepEqual(got, []string{"Électricité check"}) {
		t.Errorf("accented upper case = %v", got)
	}
	if got := searchTitles(t, store, "p"); len(got) != 0 {
		t.Errorf("one character = %v, want nothing", got)
	}
	if got := searchTitles(t, store, "zebra"); len(got) != 0 {
		t.Errorf("zebra = %v, want nothing", got)
	}
}

func TestSearchJobsSymbolsAreLiteralAndNeverBreakIt(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()
	if _, err := store.Create(ctx, CreateInput{Title: "100% done_ish", Stage: StageTodo}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "1000 chairs", Stage: StageTodo}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`%`, `_`, `"`, `'`, `\`, `NEAR AND OR`, `100%`, `)(`, "\x00ab"} {
		if _, err := store.SearchJobs(ctx, q, 0, fixedNow); err != nil {
			t.Errorf("SearchJobs(%q) errored: %v", q, err)
		}
	}
	if got := searchTitles(t, store, "100% done"); !reflect.DeepEqual(got, []string{"100% done_ish"}) {
		t.Errorf("100%% done = %v", got)
	}
	if got := searchTitles(t, store, "%%"); len(got) != 0 {
		t.Errorf("%%%% matched everything: %v", got)
	}
}

func TestSearchJobsFinishedAreFoundAndMarkedRemovedAreNot(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	todo := createNamed(t, store, ctx, sam, "Toner todo", StageTodo)
	idea := createNamed(t, store, ctx, sam, "Toner idea", StageIdea)
	doing := createNamed(t, store, ctx, sam, "Toner doing", StageDoing)
	recent := createNamed(t, store, ctx, sam, "Toner recently done", StageDone)
	old := createNamed(t, store, ctx, sam, "Toner done long ago", StageDone)
	setDoneAt(t, store, old.ID, fixedNow.AddDate(0, 0, -60)) // tucked away into Finished tasks
	gone := createNamed(t, store, ctx, sam, "Toner removed", StageTodo)
	if err := store.Remove(ctx, gone.ID, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	_, _, _, _ = todo, idea, doing, recent

	hits, err := store.SearchJobs(ctx, "toner", 0, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	finished := map[string]bool{}
	for _, h := range hits {
		name := strings.NewReplacer("<mark>", "", "</mark>", "").Replace(string(h.Title))
		got = append(got, name)
		finished[name] = h.Finished
	}
	// Unfinished first (In progress, To do, Ideas), then finished; removed never.
	want := []string{"Toner doing", "Toner todo", "Toner idea", "Toner recently done", "Toner done long ago"}
	sortedFinished := append([]string(nil), got[3:]...)
	if !reflect.DeepEqual(got[:3], want[:3]) || len(got) != 5 {
		t.Fatalf("results = %v, want unfinished first %v then the two finished ones", got, want)
	}
	for _, n := range sortedFinished {
		if !finished[n] {
			t.Errorf("%q should be marked Finished", n)
		}
	}
	for _, n := range got[:3] {
		if finished[n] {
			t.Errorf("%q should not be marked Finished", n)
		}
	}
	for _, n := range got {
		if n == "Toner removed" {
			t.Error("a removed job was found")
		}
	}
}

func TestSearchJobsOrderLimitAndExclude(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Plain", Notes: "mentions widget here", Stage: StageDoing}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, CreateInput{Title: "Widget in title", Stage: StageIdea}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	// Same unfinished state: title match beats notes-only, whatever the column.
	if got := searchTitles(t, store, "widget"); !reflect.DeepEqual(got, []string{"Widget in title", "Plain"}) {
		t.Errorf("order = %v, want the title match first", got)
	}

	for i := 0; i < 30; i++ {
		createNamed(t, store, ctx, sam, fmt.Sprintf("Gadget %02d", i), StageTodo)
	}
	hits, _ := store.SearchJobs(ctx, "gadget", 0, fixedNow)
	if len(hits) != 20 {
		t.Errorf("got %d results, want the limit of 20", len(hits))
	}
	// Priority order within a column.
	if got := searchTitles(t, store, "gadget"); got[0] != "Gadget 00" || got[19] != "Gadget 19" {
		t.Errorf("order within To do = %v ... %v", got[0], got[19])
	}
	first := hits[0].ID
	after, _ := store.SearchJobs(ctx, "gadget", first, fixedNow)
	for _, h := range after {
		if h.ID == first {
			t.Error("exclude did not leave the job out")
		}
	}
}

func TestSearchJobsHighlightsAndEscapes(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	notes := strings.Repeat("filler words go here. ", 12) + "Then replace the Toner cartridge <b>today</b> & check. " + strings.Repeat("more filler. ", 12)
	if _, err := store.Create(ctx, CreateInput{
		Title: "Order <script>toner</script>", Notes: notes, Stage: StageTodo, DueDate: "2026-09-19", PersonIDs: []int64{sam, alex},
	}, sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	hits, err := store.SearchJobs(ctx, "toner", 0, fixedNow)
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits = %v, err = %v", hits, err)
	}
	h := hits[0]
	if got := string(h.Title); got != "Order &lt;script&gt;<mark>toner</mark>&lt;/script&gt;" {
		t.Errorf("Title = %q (escaped first, then marked)", got)
	}
	p := string(h.Passage)
	if !strings.Contains(p, "<mark>Toner</mark> cartridge &lt;b&gt;today&lt;/b&gt; &amp; check") {
		t.Errorf("Passage = %q, want the notes around the match, escaped and marked", p)
	}
	if !strings.HasPrefix(p, "…") || !strings.HasSuffix(p, "…") {
		t.Errorf("Passage = %q, want ellipses at a cut start and end", p)
	}
	if n := len([]rune(p)); n > 200 {
		t.Errorf("Passage is %d characters, want about 120 of notes", n)
	}
	if h.Column != "To do" || len(h.People) != 2 || h.Due == "" || h.Finished {
		t.Errorf("hit = %+v", h)
	}
	if h.URL != fmt.Sprintf("/tasks/%d", h.ID) {
		t.Errorf("URL = %q", h.URL)
	}
	_ = time.Now
}
