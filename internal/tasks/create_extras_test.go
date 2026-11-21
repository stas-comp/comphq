package tasks

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func countRows(t *testing.T, f *refFixture, query string) int {
	t.Helper()
	var n int
	if err := f.store.DB.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// Gates 8.30, 8.31: a job is saved with its steps (in the order given), its
// articles and its places, and History says it in one line.
func TestGate831CreateSavesStepsArticlesAndPlacesTogether(t *testing.T) {
	f := newRefFixture(t)
	job, err := f.store.Create(f.ctx, CreateInput{
		Title:      "Quarterly mailing",
		Steps:      []string{"  Pick the list ", "", "Print labels", "   ", "Stuff envelopes", "Post them", "Count the returns"},
		ArticleIDs: []int64{3, 1},
		Places: []PlaceInput{
			{Name: "Lists", Path: " //NAS/COSMOS/Lists "},
			{Name: "", Path: `S:\COSMOS\Proofs`},
		},
	}, f.sam, fixedNow)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	steps, err := f.store.ListSteps(f.ctx, job.ID, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	var words []string
	for _, st := range steps {
		words = append(words, st.Text)
	}
	if want := []string{"Pick the list", "Print labels", "Stuff envelopes", "Post them", "Count the returns"}; !sameStrings(words, want) {
		t.Errorf("steps = %v, want %v", words, want)
	}
	refs, _ := f.store.ListReferences(f.ctx, job.ID)
	if len(refs.Articles) != 2 || refs.Articles[0].ArticleID != 3 || refs.Articles[1].ArticleID != 1 {
		t.Errorf("articles = %+v", refs.Articles)
	}
	if len(refs.Places) != 2 || refs.Places[0].Path != `\\NAS\COSMOS\Lists` || refs.Places[1].Label != "Proofs" {
		t.Errorf("places = %+v", refs.Places)
	}
	// One History line for the whole job, and no step_added, article or place rows.
	acts, _ := f.store.ListActivity(f.ctx, job.ID)
	if len(acts) != 1 || acts[0].Text != "Sam created this (with 5 steps)" {
		t.Errorf("History = %+v, want one line \"Sam created this (with 5 steps)\"", acts)
	}
	for _, action := range []string{"step_added", "article_linked", "place_added"} {
		if n := countActivity(t, f.store.DB, job.ID, action); n != 0 {
			t.Errorf("%d %s rows were written for a job made with them", n, action)
		}
	}
	// A job with no steps keeps the old wording.
	plain, _ := f.store.Create(f.ctx, CreateInput{Title: "Plain"}, f.sam, fixedNow)
	if acts, _ := f.store.ListActivity(f.ctx, plain.ID); len(acts) != 1 || acts[0].Text != "Sam created this" {
		t.Errorf("History = %+v", acts)
	}
	one, _ := f.store.Create(f.ctx, CreateInput{Title: "One", Steps: []string{"Only step"}}, f.sam, fixedNow)
	if acts, _ := f.store.ListActivity(f.ctx, one.ID); acts[0].Text != "Sam created this (with 1 step)" {
		t.Errorf("History = %+v", acts)
	}
}

// Gate 8.32: anything refused saves nothing at all.
func TestGate832EveryRefusalLeavesNothingBehind(t *testing.T) {
	f := newRefFixture(t)
	f.articles.set(ArticleRef{ID: 30, Title: "Old policy", Category: "HR", Archived: true})
	jobsBefore := countRows(t, f, `SELECT COUNT(*) FROM tasks`)

	fifty1 := make([]string, 51)
	for i := range fifty1 {
		fifty1[i] = fmt.Sprintf("Step %d", i+1)
	}
	var tooManyArticles []int64
	for i := int64(1); i <= 21; i++ {
		tooManyArticles = append(tooManyArticles, i)
	}
	var tooManyPlaces []PlaceInput
	for i := 1; i <= 21; i++ {
		tooManyPlaces = append(tooManyPlaces, PlaceInput{Path: fmt.Sprintf(`\\NAS\COSMOS\F%02d`, i)})
	}
	okSteps := []string{"One", "Two"}
	cases := []struct {
		name  string
		input CreateInput
		want  error
	}{
		{"no title", CreateInput{Title: "  ", Steps: okSteps}, ErrEmptyTitle},
		{"a step over 200 characters", CreateInput{Title: "T", Steps: []string{"ok", strings.Repeat("x", 201)}}, ErrStepTooLong},
		{"51 steps", CreateInput{Title: "T", Steps: fifty1}, ErrTooManySteps},
		{"a bad folder path", CreateInput{Title: "T", Steps: okSteps, Places: []PlaceInput{{Path: `\\NAS\COSMOS\Fine`}, {Path: "no way"}}}, ErrPlacePath},
		{"a long place name", CreateInput{Title: "T", Places: []PlaceInput{{Name: strings.Repeat("n", 81), Path: `\\NAS\COSMOS\Fine`}}}, ErrPlaceNameLong},
		{"21 places", CreateInput{Title: "T", Steps: okSteps, Places: tooManyPlaces}, ErrTooManyPlaces},
		{"21 articles", CreateInput{Title: "T", Steps: okSteps, ArticleIDs: tooManyArticles}, ErrTooManyArticles},
		{"the same article twice", CreateInput{Title: "T", ArticleIDs: []int64{1, 2, 1}}, ErrArticleLinked},
		{"an article that isn't there", CreateInput{Title: "T", ArticleIDs: []int64{1, 999}}, ErrArticleGone},
		{"an archived article", CreateInput{Title: "T", ArticleIDs: []int64{30}}, ErrArticleArchived},
		// The third place is the same folder as the first: only the transaction
		// that saves the places can see that, so the job, its steps and its
		// links must all be rolled back with it.
		{"the same folder as the first, third", CreateInput{
			Title: "T", Steps: okSteps, ArticleIDs: []int64{1, 2},
			Places: []PlaceInput{{Path: `\\NAS\COSMOS\A`}, {Path: `\\NAS\COSMOS\B`}, {Path: `\\nas\cosmos\a`}},
		}, ErrPlaceDuplicate},
	}
	for _, c := range cases {
		if _, err := f.store.Create(f.ctx, c.input, f.sam, fixedNow); !errors.Is(err, c.want) {
			t.Errorf("%s: Create = %v, want %v", c.name, err, c.want)
		}
	}
	for _, q := range []struct{ what, sql string }{
		{"jobs", `SELECT COUNT(*) FROM tasks`},
		{"steps", `SELECT COUNT(*) FROM task_checklist_items`},
		{"article links", `SELECT COUNT(*) FROM task_articles`},
		{"places", `SELECT COUNT(*) FROM task_places`},
	} {
		want := 0
		if q.what == "jobs" {
			want = jobsBefore
		}
		if n := countRows(t, f, q.sql); n != want {
			t.Errorf("%s after the refusals = %d, want %d", q.what, n, want)
		}
	}
	// Nothing was written to History either: the one row is the fixture job's.
	if n := countRows(t, f, `SELECT COUNT(*) FROM task_activity`); n != 1 {
		t.Errorf("%d History rows, want 1 (the fixture job's)", n)
	}
}

func TestCreateWithoutExtrasIsUnchanged(t *testing.T) {
	f := newRefFixture(t)
	job, err := f.store.Create(f.ctx, CreateInput{Title: "Just a job", Steps: []string{" ", ""}}, f.sam, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, f, fmt.Sprintf(`SELECT COUNT(*) FROM task_checklist_items WHERE task_id = %d`, job.ID)); n != 0 {
		t.Errorf("%d steps from blank lines", n)
	}
}
