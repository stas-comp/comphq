package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// A fake Knowledge Base, so these tests need none (D-105): Tasks only sees the
// ArticleFinder interface.
type fakeArticles struct {
	mu   sync.Mutex
	refs map[int64]ArticleRef
}

func newFakeArticles(refs ...ArticleRef) *fakeArticles {
	f := &fakeArticles{refs: map[int64]ArticleRef{}}
	for _, r := range refs {
		f.refs[r.ID] = r
	}
	return f
}

func (f *fakeArticles) FindByTitle(ctx context.Context, q string, limit int) []ArticleRef {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ArticleRef
	for id := int64(1); id <= int64(len(f.refs))+100; id++ {
		r, ok := f.refs[id]
		if !ok || r.Archived {
			continue
		}
		if strings.Contains(strings.ToLower(r.Title), strings.ToLower(q)) {
			out = append(out, r)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (f *fakeArticles) Articles(ctx context.Context, ids []int64) map[int64]ArticleRef {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int64]ArticleRef{}
	for _, id := range ids {
		if r, ok := f.refs[id]; ok {
			out[id] = r
		}
	}
	return out
}

func (f *fakeArticles) set(r ArticleRef) {
	f.mu.Lock()
	f.refs[r.ID] = r
	f.mu.Unlock()
}

type refFixture struct {
	store    *Store
	ctx      context.Context
	sam      int64
	job      Task
	articles *fakeArticles
}

func newRefFixture(t *testing.T) *refFixture {
	t.Helper()
	sqlDB := openTestDB(t)
	f := &refFixture{ctx: context.Background(), articles: newFakeArticles()}
	f.store = &Store{DB: sqlDB, Articles: f.articles}
	f.sam = testPerson(t, sqlDB, "Sam")
	f.job = createNamed(t, f.store, f.ctx, f.sam, "Send the mailing", StageTodo)
	for i := int64(1); i <= 25; i++ {
		f.articles.set(ArticleRef{ID: i, Title: fmt.Sprintf("Article %02d", i), Category: "Mailing"})
	}
	return f
}

func (f *refFixture) activity(t *testing.T) []string {
	t.Helper()
	acts, err := f.store.ListActivity(f.ctx, f.job.ID)
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	var out []string
	for _, a := range acts {
		out = append(out, a.Text)
	}
	return out
}

// Gate 8.15: every path case.
func TestGate815NormalisePath(t *testing.T) {
	good := []struct{ in, want string }{
		{`\\NAS\COSMOS\Mailings\2026`, `\\NAS\COSMOS\Mailings\2026`},
		{`//NAS/COSMOS/Mailings/2026`, `\\NAS\COSMOS\Mailings\2026`},
		{`  \\NAS\COSMOS  `, `\\NAS\COSMOS`},
		{"\t S:/COSMOS/Lists \n", `S:\COSMOS\Lists`},
		{`s:\x`, `s:\x`},
		{`C:\`, `C:\`},
		{`\\server\share`, `\\server\share`},
		{`\\NAS\`, `\\NAS\`},
	}
	for _, c := range good {
		got, err := NormalisePath(c.in)
		if err != nil {
			t.Errorf("NormalisePath(%q): unexpected refusal %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalisePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	bad := []string{
		"",
		"   ",
		`COSMOS\Mailings`,
		`\NAS\COSMOS`,
		`\\NAS`,
		`\\\COSMOS`,
		`C:COSMOS`,
		`1:\COSMOS`,
		`https://example.com/x`,
		`S:\` + strings.Repeat("a", 399), // 402 characters
	}
	for _, in := range bad {
		if _, err := NormalisePath(in); !errors.Is(err, ErrPlacePath) {
			t.Errorf("NormalisePath(%q) = %v, want the folder-path refusal", in, err)
		}
	}
	// 400 characters exactly is fine.
	if _, err := NormalisePath(`S:\` + strings.Repeat("a", 397)); err != nil {
		t.Errorf("a 400-character path was refused: %v", err)
	}
	if got := ErrPlacePath.Error(); got != `That doesn't look like a folder path. It should start with \\ (like \\NAS\COSMOS) or a drive letter (like S:\COSMOS).` {
		t.Errorf("refusal message = %q", got)
	}
}

// Gate 8.13: the display name.
func TestGate813PlaceDisplayName(t *testing.T) {
	for _, c := range []struct{ name, path, want string }{
		{"Mailing lists 2026", `\\NAS\COSMOS\Mailings\2026`, "Mailing lists 2026"},
		{"", `\\NAS\COSMOS\Mailings\2026`, "2026"},
		{"  ", `\\NAS\COSMOS\Mailings\2026\`, "2026"},
		{"", `S:\`, "S:"},
		{"", `\\NAS\COSMOS`, "COSMOS"},
	} {
		if got := PlaceDisplayName(c.name, c.path); got != c.want {
			t.Errorf("PlaceDisplayName(%q, %q) = %q, want %q", c.name, c.path, got, c.want)
		}
	}
}

func TestAddPlaceStoresNormalisedPathAndRecordsHistory(t *testing.T) {
	f := newRefFixture(t)
	if err := f.store.AddPlace(f.ctx, f.job.ID, "  Lists  ", " //NAS/COSMOS/Mailings/2026 ", f.sam, fixedNow); err != nil {
		t.Fatalf("AddPlace: %v", err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `S:\COSMOS\Proofs`, f.sam, fixedNow); err != nil {
		t.Fatalf("AddPlace (no name): %v", err)
	}
	refs, err := f.store.ListReferences(f.ctx, f.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs.Places) != 2 {
		t.Fatalf("places = %+v", refs.Places)
	}
	if p := refs.Places[0]; p.Name != "Lists" || p.Path != `\\NAS\COSMOS\Mailings\2026` || p.Label != "Lists" {
		t.Errorf("first place = %+v", p)
	}
	if p := refs.Places[1]; p.Name != "" || p.Label != "Proofs" {
		t.Errorf("second place = %+v", p)
	}
	hist := f.activity(t)
	if len(hist) < 3 || hist[0] != "Sam added the COSMOS place “Proofs”" || hist[1] != "Sam added the COSMOS place “Lists”" {
		t.Errorf("history = %v", hist)
	}
}

// Gates 8.15, 8.16: every refusal changes nothing.
func TestGate816PlaceRefusalsChangeNothing(t *testing.T) {
	f := newRefFixture(t)
	if err := f.store.AddPlace(f.ctx, f.job.ID, "Lists", `\\NAS\COSMOS\Mailings`, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	beforeVersion, _ := f.store.Version(f.ctx)
	beforeHistory := len(f.activity(t))

	cases := []struct {
		name, path string
		want       error
	}{
		{"", "", ErrPlacePath},
		{"", "NAS/COSMOS", ErrPlacePath},
		{strings.Repeat("n", 81), `\\NAS\COSMOS\X`, ErrPlaceNameLong},
		{"Again", `\\nas\cosmos\MAILINGS`, ErrPlaceDuplicate},   // case-insensitive duplicate
		{"Again", "//NAS/COSMOS/Mailings  ", ErrPlaceDuplicate}, // the same path typed another way
	}
	for _, c := range cases {
		if err := f.store.AddPlace(f.ctx, f.job.ID, c.name, c.path, f.sam, fixedNow); !errors.Is(err, c.want) {
			t.Errorf("AddPlace(%q, %q) = %v, want %v", c.name, c.path, err, c.want)
		}
	}
	if v, _ := f.store.Version(f.ctx); v != beforeVersion {
		t.Errorf("a refusal bumped the change counter: %d -> %d", beforeVersion, v)
	}
	if n := len(f.activity(t)); n != beforeHistory {
		t.Errorf("a refusal wrote History: %d -> %d rows", beforeHistory, n)
	}
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if len(refs.Places) != 1 {
		t.Errorf("places = %+v, want just the first", refs.Places)
	}
}

func TestGate816TwentyPlacesThenRefused(t *testing.T) {
	f := newRefFixture(t)
	for i := 1; i <= 20; i++ {
		if err := f.store.AddPlace(f.ctx, f.job.ID, "", fmt.Sprintf(`\\NAS\COSMOS\Folder%02d`, i), f.sam, fixedNow); err != nil {
			t.Fatalf("place %d: %v", i, err)
		}
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `\\NAS\COSMOS\Folder21`, f.sam, fixedNow); !errors.Is(err, ErrTooManyPlaces) {
		t.Fatalf("21st place = %v, want ErrTooManyPlaces", err)
	}
	// Removing one makes room again, and removed places don't count.
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if err := f.store.RemovePlace(f.ctx, f.job.ID, refs.Places[0].ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `\\NAS\COSMOS\Folder21`, f.sam, fixedNow); err != nil {
		t.Fatalf("after a removal: %v", err)
	}
	// A removed folder can be added again.
	if err := f.store.RemovePlace(f.ctx, f.job.ID, refs.Places[1].ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `\\NAS\COSMOS\Folder02`, f.sam, fixedNow); err != nil {
		t.Fatalf("re-adding a removed folder: %v", err)
	}
}

func TestGate816ArticleRefusals(t *testing.T) {
	f := newRefFixture(t)
	f.articles.set(ArticleRef{ID: 30, Title: "Old policy", Category: "HR", Archived: true})

	if err := f.store.AddArticle(f.ctx, f.job.ID, 1, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	beforeVersion, _ := f.store.Version(f.ctx)
	for _, c := range []struct {
		article int64
		want    error
	}{
		{1, ErrArticleLinked},
		{999, ErrArticleGone},
		{30, ErrArticleArchived},
	} {
		if err := f.store.AddArticle(f.ctx, f.job.ID, c.article, f.sam, fixedNow); !errors.Is(err, c.want) {
			t.Errorf("AddArticle(%d) = %v, want %v", c.article, err, c.want)
		}
	}
	if v, _ := f.store.Version(f.ctx); v != beforeVersion {
		t.Errorf("a refusal bumped the change counter")
	}

	// A store with no Knowledge Base finds no articles.
	bare := &Store{DB: f.store.DB}
	if err := bare.AddArticle(f.ctx, f.job.ID, 2, f.sam, fixedNow); !errors.Is(err, ErrArticleGone) {
		t.Errorf("AddArticle with no finder = %v, want ErrArticleGone", err)
	}
}

func TestGate816TwentyArticlesThenRefused(t *testing.T) {
	f := newRefFixture(t)
	for i := int64(1); i <= 20; i++ {
		if err := f.store.AddArticle(f.ctx, f.job.ID, i, f.sam, fixedNow); err != nil {
			t.Fatalf("article %d: %v", i, err)
		}
	}
	if err := f.store.AddArticle(f.ctx, f.job.ID, 21, f.sam, fixedNow); !errors.Is(err, ErrTooManyArticles) {
		t.Fatalf("21st article = %v, want ErrTooManyArticles", err)
	}
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, refs.Articles[0].LinkID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddArticle(f.ctx, f.job.ID, 21, f.sam, fixedNow); err != nil {
		t.Fatalf("after a removal: %v", err)
	}
	// The removed one can be linked again.
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, refs.Articles[1].LinkID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddArticle(f.ctx, f.job.ID, 2, f.sam, fixedNow); err != nil {
		t.Fatalf("re-linking a removed article: %v", err)
	}
}

// Gate 8.16: adding to a job somebody has just removed.
func TestGate816AddingToARemovedJob(t *testing.T) {
	f := newRefFixture(t)
	if err := f.store.Remove(f.ctx, f.job.ID, f.sam, fixedNow); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := f.store.AddArticle(f.ctx, f.job.ID, 1, f.sam, fixedNow); !errors.Is(err, ErrJobRemoved) {
		t.Errorf("AddArticle to a removed job = %v, want ErrJobRemoved", err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `\\NAS\COSMOS`, f.sam, fixedNow); !errors.Is(err, ErrJobRemoved) {
		t.Errorf("AddPlace to a removed job = %v, want ErrJobRemoved", err)
	}
	if err := f.store.AddArticle(f.ctx, 424242, 1, f.sam, fixedNow); !errors.Is(err, ErrTaskNotFound) {
		t.Errorf("AddArticle to a job that never existed = %v, want ErrTaskNotFound", err)
	}
	var n int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM task_articles`).Scan(&n)
	if n != 0 {
		t.Errorf("%d article links were written to a removed job", n)
	}
	if ErrJobRemoved.Error() != "That job has just been removed, so nothing can be added to it." {
		t.Errorf("message = %q", ErrJobRemoved.Error())
	}
}

// Two people racing for the 20th slot: exactly one gets it.
func TestGate816TwoPeopleRaceForTheTwentiethSlot(t *testing.T) {
	f := newRefFixture(t)
	for i := int64(1); i <= 19; i++ {
		if err := f.store.AddArticle(f.ctx, f.job.ID, i, f.sam, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 19; i++ {
		if err := f.store.AddPlace(f.ctx, f.job.ID, "", fmt.Sprintf(`\\NAS\COSMOS\F%02d`, i), f.sam, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for _, article := range []int64{20, 21} {
		wg.Add(1)
		go func(a int64) {
			defer wg.Done()
			results <- f.store.AddArticle(f.ctx, f.job.ID, a, f.sam, fixedNow)
		}(article)
	}
	for _, p := range []string{`\\NAS\COSMOS\Race1`, `\\NAS\COSMOS\Race2`} {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			results <- f.store.AddPlace(f.ctx, f.job.ID, "", p, f.sam, fixedNow)
		}(p)
	}
	wg.Wait()
	close(results)
	ok, full := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrTooManyArticles), errors.Is(err, ErrTooManyPlaces):
			full++
		default:
			t.Errorf("unexpected error %v", err)
		}
	}
	if ok != 2 || full != 2 {
		t.Errorf("%d succeeded and %d were refused as full, want 2 and 2", ok, full)
	}
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if len(refs.Articles) != 20 || len(refs.Places) != 20 {
		t.Errorf("articles %d, places %d, want 20 and 20", len(refs.Articles), len(refs.Places))
	}
}

// Gate 8.17: History keeps the title as it was then, not as it is now.
func TestGate817HistoryKeepsTheTitleAsItWasThen(t *testing.T) {
	f := newRefFixture(t)
	f.articles.set(ArticleRef{ID: 1, Title: "Mailing SOP", Category: "Mailing"})
	if err := f.store.AddArticle(f.ctx, f.job.ID, 1, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.articles.set(ArticleRef{ID: 1, Title: "Mailing SOP (2027)", Category: "Mailing"})
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if refs.Articles[0].Title != "Mailing SOP (2027)" {
		t.Errorf("the job shows %q, want the current title", refs.Articles[0].Title)
	}
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, refs.Articles[0].LinkID, f.sam, fixedNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	f.articles.set(ArticleRef{ID: 1, Title: "Something else entirely", Category: "Mailing"})
	hist := f.activity(t)
	want := []string{"Sam removed the article “Mailing SOP (2027)”", "Sam linked the article “Mailing SOP”"}
	if len(hist) < 2 || hist[0] != want[0] || hist[1] != want[1] {
		t.Errorf("history = %v, want it to start with %v", hist, want)
	}

	if err := f.store.AddPlace(f.ctx, f.job.ID, "Mailing lists 2026", `\\NAS\COSMOS\Mailings\2026`, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	refs, _ = f.store.ListReferences(f.ctx, f.job.ID)
	if err := f.store.RemovePlace(f.ctx, f.job.ID, refs.Places[0].ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	hist = f.activity(t)
	if hist[0] != "Sam removed the COSMOS place “Mailing lists 2026”" || hist[1] != "Sam added the COSMOS place “Mailing lists 2026”" {
		t.Errorf("history = %v", hist)
	}
}

func TestRemovingTwiceAndRemovingSomeoneElsesLinkIsHarmless(t *testing.T) {
	f := newRefFixture(t)
	other := createNamed(t, f.store, f.ctx, f.sam, "Another job", StageTodo)
	if err := f.store.AddArticle(f.ctx, f.job.ID, 1, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	link := refs.Articles[0].LinkID
	if err := f.store.RemoveArticle(f.ctx, other.ID, link, f.sam, fixedNow); !errors.Is(err, ErrReferenceGone) {
		t.Errorf("removing another job's link = %v, want ErrReferenceGone", err)
	}
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, link, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, link, f.sam, fixedNow); err != nil {
		t.Errorf("removing twice = %v, want no error", err)
	}
	if n := countActivity(t, f.store.DB, f.job.ID, "article_unlinked"); n != 1 {
		t.Errorf("%d unlinked rows, want 1", n)
	}
	// Nothing is deleted (A3).
	var removedBy int64
	if err := f.store.DB.QueryRow(`SELECT removed_by FROM task_articles WHERE id = ?`, link).Scan(&removedBy); err != nil || removedBy != f.sam {
		t.Errorf("removed_by = %d, %v", removedBy, err)
	}
}

// Gate 8.11: the picker leaves out linked and archived articles, at most ten.
func TestGate811PickerLeavesOutLinkedAndArchived(t *testing.T) {
	f := newRefFixture(t)
	f.articles.set(ArticleRef{ID: 30, Title: "Article old", Category: "HR", Archived: true})
	if err := f.store.AddArticle(f.ctx, f.job.ID, 1, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	got, err := f.store.SearchArticles(f.ctx, "article", f.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("picker returned %d, want 10", len(got))
	}
	for _, a := range got {
		if a.ID == 1 || a.ID == 30 {
			t.Errorf("picker offered article %d", a.ID)
		}
	}
	// Without a job (the Add task window) nothing is left out for being linked.
	got, _ = f.store.SearchArticles(f.ctx, "article 01", 0)
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("picker with no job = %+v", got)
	}
}

func TestGate812ArchivedArticlesStayOnTheJobMarked(t *testing.T) {
	f := newRefFixture(t)
	if err := f.store.AddArticle(f.ctx, f.job.ID, 3, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.articles.set(ArticleRef{ID: 3, Title: "Article 03", Category: "Mailing", Archived: true})
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if len(refs.Articles) != 1 || !refs.Articles[0].Archived || refs.Articles[0].Title != "Article 03" || refs.Articles[0].Category != "Mailing" {
		t.Errorf("articles = %+v", refs.Articles)
	}
}

// Gate 8.18: what the article's page lists.
func TestGate818JobsUsingAnArticle(t *testing.T) {
	f := newRefFixture(t)
	done := createNamed(t, f.store, f.ctx, f.sam, "Old mailing", StageDone)
	doing := createNamed(t, f.store, f.ctx, f.sam, "Zebra mailing", StageDoing)
	removed := createNamed(t, f.store, f.ctx, f.sam, "Removed mailing", StageTodo)
	unlinked := createNamed(t, f.store, f.ctx, f.sam, "Unlinked mailing", StageTodo)
	for _, j := range []Task{f.job, done, doing, removed, unlinked} {
		if err := f.store.AddArticle(f.ctx, j.ID, 5, f.sam, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.store.AddArticle(f.ctx, doing.ID, 6, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	refs, _ := f.store.ListReferences(f.ctx, unlinked.ID)
	if err := f.store.RemoveArticle(f.ctx, unlinked.ID, refs.Articles[0].LinkID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Remove(f.ctx, removed.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	jobs, err := f.store.JobsUsingArticle(f.ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, j := range jobs {
		got = append(got, fmt.Sprintf("%s|%s|%v", j.Title, j.StageLabel, j.Finished))
	}
	want := []string{"Zebra mailing|In progress|false", "Send the mailing|To do|false", "Old mailing|Done|true"}
	if !sameStrings(got, want) {
		t.Errorf("jobs = %v, want %v", got, want)
	}
	if jobs, _ := f.store.JobsUsingArticle(f.ctx, 7); len(jobs) != 0 {
		t.Errorf("an article no job uses lists %v", jobs)
	}
}

// Gate 8.20: the two export files, removed rows and archived titles included.
func TestGate820Exports(t *testing.T) {
	f := newRefFixture(t)
	f.articles.set(ArticleRef{ID: 40, Title: "Retired SOP", Category: "HR", Archived: true})
	if err := f.store.AddArticle(f.ctx, f.job.ID, 4, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.articles.set(ArticleRef{ID: 40, Title: "Retired SOP", Category: "HR"})
	if err := f.store.AddArticle(f.ctx, f.job.ID, 40, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	f.articles.set(ArticleRef{ID: 40, Title: "Retired SOP", Category: "HR", Archived: true})
	refs, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if err := f.store.RemoveArticle(f.ctx, f.job.ID, refs.Articles[0].LinkID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "Lists", `\\NAS\COSMOS\Lists`, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AddPlace(f.ctx, f.job.ID, "", `S:\Proofs`, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	places, _ := f.store.ListReferences(f.ctx, f.job.ID)
	if err := f.store.RemovePlace(f.ctx, f.job.ID, places.Places[1].ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}

	rows, err := ArticlesCSV{DB: f.store.DB, Articles: f.articles}.ExportRows(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(rows[0], []string{"Job", "Article", "Linked by", "Linked", "Removed"}) {
		t.Errorf("job-articles.csv header = %v", rows[0])
	}
	if len(rows) != 3 || rows[1][1] != "Article 04" || rows[1][4] == "" || rows[2][1] != "Retired SOP" || rows[2][4] != "" {
		t.Errorf("job-articles.csv rows = %v", rows)
	}
	if rows[1][0] != "Send the mailing" || rows[1][2] != "Sam" {
		t.Errorf("job-articles.csv row = %v", rows[1])
	}

	prow, err := PlacesCSV{DB: f.store.DB}.ExportRows(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !sameStrings(prow[0], []string{"Job", "Name", "Folder path", "Added by", "Added", "Removed"}) {
		t.Errorf("job-places.csv header = %v", prow[0])
	}
	if len(prow) != 3 || prow[1][1] != "Lists" || prow[1][2] != `\\NAS\COSMOS\Lists` || prow[1][5] != "" || prow[2][2] != `S:\Proofs` || prow[2][5] == "" {
		t.Errorf("job-places.csv rows = %v", prow)
	}
}

// The migration is add-only: tables a 1.5.0 database doesn't have.
func TestReferencesMigrationIsAddOnly(t *testing.T) {
	f := newRefFixture(t)
	for _, table := range []string{"tasks", "task_assignees", "task_activity", "task_links", "task_checklist_items"} {
		var n int
		if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?)`, table).Scan(&n); err != nil || n == 0 {
			t.Errorf("table %s: %d columns, %v", table, n, err)
		}
	}
	var cols int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('tasks')`).Scan(&cols)
	if cols != 15 {
		t.Logf("tasks has %d columns", cols)
	}
}
