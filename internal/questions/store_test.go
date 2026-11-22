package questions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	comphq "github.com/stas-comp/comphq"
	"github.com/stas-comp/comphq/internal/db"
	"github.com/stas-comp/comphq/internal/people"
)

// SPEC gates 8.70-8.74 (B14.6, D-110): the Questions store.

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	for _, section := range []string{"app", "people", "questions"} {
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

type fixture struct {
	store *Store
	ctx   context.Context
	sam   int64
	jane  int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	sqlDB := openTestDB(t)
	ps := &people.Store{DB: sqlDB}
	sam, err := ps.Create("Sam")
	if err != nil {
		t.Fatal(err)
	}
	jane, err := ps.Create("Jane")
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{store: &Store{DB: sqlDB}, ctx: context.Background(), sam: sam.ID, jane: jane.ID}
}

func at(day string, hour int) time.Time {
	d, err := time.ParseInLocation("2006-01-02", day, time.UTC)
	if err != nil {
		panic(err)
	}
	return d.Add(time.Duration(hour) * time.Hour)
}

var today = at("2026-11-21", 0)

func texts(rows []Row) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Text)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Gate 8.71: asking, with the limits and the typed text kept on refusal.
func TestGate871AskStoresTheQuestionWithWhoAndWhen(t *testing.T) {
	f := newFixture(t)
	id, err := f.store.Ask(f.ctx, "  Where do the Christmas card lists live?  ", "Line one.\r\nLine two.\n\nLine four.", f.sam, at("2026-11-20", 9))
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	q, err := f.store.Get(f.ctx, id, today)
	if err != nil {
		t.Fatal(err)
	}
	if q.Text != "Where do the Christmas card lists live?" || q.AskedBy != "Sam" || q.AskedLabel != "Fri 20 Nov" || q.Answered {
		t.Errorf("question = %+v", q)
	}
	if q.Details != "Line one.\nLine two.\n\nLine four." {
		t.Errorf("details = %q (line breaks kept, CRLF made LF)", q.Details)
	}
	if v, _ := f.store.Version(f.ctx); v < 1 {
		t.Errorf("the questions change counter didn't move: %d", v)
	}
}

func TestGate871LimitsRefuseWithPlainWordsAndSaveNothing(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name, text, details string
		want                error
	}{
		{"empty", "", "", ErrEmptyQuestion},
		{"only spaces", "   \n ", "", ErrEmptyQuestion},
		{"301 characters", strings.Repeat("q", 301), "", ErrQuestionTooLong},
		{"details over 2000", "Fine", strings.Repeat("d", 2001), ErrDetailsTooLong},
	}
	for _, c := range cases {
		if _, err := f.store.Ask(f.ctx, c.text, c.details, f.sam, at("2026-11-20", 9)); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	var n int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM questions`).Scan(&n)
	if n != 0 {
		t.Errorf("%d questions saved by refusals", n)
	}
	// Exactly at the limits is fine, counted in characters not bytes.
	if _, err := f.store.Ask(f.ctx, strings.Repeat("é", 300), strings.Repeat("ü", 2000), f.sam, at("2026-11-20", 9)); err != nil {
		t.Errorf("a question of exactly 300 characters and details of 2,000: %v", err)
	}
	for err, want := range map[error]string{
		ErrEmptyQuestion:    "Type your question first.",
		ErrQuestionTooLong:  "A question can be up to 300 characters.",
		ErrDetailsTooLong:   "The details can be up to 2,000 characters.",
		ErrEmptyAnswer:      "Type your answer first.",
		ErrAnswerTooLong:    "An answer can be up to 2,000 characters.",
		ErrQuestionNotFound: "That question isn't there any more.",
	} {
		if err.Error() != want {
			t.Errorf("message %q, want %q", err.Error(), want)
		}
	}
}

// Gate 8.72: two groups, waiting newest first, answered by when they were answered;
// answered shows 50 then Show all.
func TestGate872TheListHasTwoGroupsInOrder(t *testing.T) {
	f := newFixture(t)
	ask := func(text string, day string, hour int) int64 {
		id, err := f.store.Ask(f.ctx, text, "", f.sam, at(day, hour))
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	a := ask("A (oldest)", "2026-11-01", 9)
	b := ask("B", "2026-11-02", 9)
	c := ask("C (newest)", "2026-11-03", 9)
	d := ask("D", "2026-11-04", 9)
	// D answered first, then A.
	if err := f.store.SetAnswered(f.ctx, d, f.jane, at("2026-11-10", 9)); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetAnswered(f.ctx, a, f.jane, at("2026-11-12", 9)); err != nil {
		t.Fatal(err)
	}
	_ = b
	_ = c
	list, err := f.store.List(f.ctx, today, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := texts(list.Waiting), []string{"C (newest)", "B"}; !equal(got, want) {
		t.Errorf("Waiting = %v, want %v", got, want)
	}
	if got, want := texts(list.Answered), []string{"A (oldest)", "D"}; !equal(got, want) {
		t.Errorf("Answered = %v, want %v (most recently answered first)", got, want)
	}
	if list.AnsweredTotal != 2 || list.MoreAnswered {
		t.Errorf("answered total %d, more %v", list.AnsweredTotal, list.MoreAnswered)
	}
	if r := list.Waiting[0]; r.AskedBy != "Sam" || r.AskedLabel != "Tue 3 Nov" || r.AnswerCount != 0 {
		t.Errorf("row = %+v", r)
	}
}

func TestGate872AnsweredShows50ThenShowAll(t *testing.T) {
	f := newFixture(t)
	for i := 1; i <= 60; i++ {
		id, err := f.store.Ask(f.ctx, fmt.Sprintf("Q%02d", i), "", f.sam, at("2026-10-01", 9))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.store.SetAnswered(f.ctx, id, f.jane, at("2026-11-01", 0).Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	list, _ := f.store.List(f.ctx, today, false)
	if len(list.Answered) != 50 || list.AnsweredTotal != 60 || !list.MoreAnswered {
		t.Errorf("%d answered shown of %d, more=%v", len(list.Answered), list.AnsweredTotal, list.MoreAnswered)
	}
	if list.Answered[0].Text != "Q60" || list.Answered[49].Text != "Q11" {
		t.Errorf("the 50 most recently answered: %s ... %s", list.Answered[0].Text, list.Answered[49].Text)
	}
	all, _ := f.store.List(f.ctx, today, true)
	if len(all.Answered) != 60 || all.MoreAnswered {
		t.Errorf("Show all: %d, more=%v", len(all.Answered), all.MoreAnswered)
	}
}

// Gate 8.73: answers oldest first, anyone, any number of times, counted on the list.
func TestGate873AnswersAreOldestFirstAndCounted(t *testing.T) {
	f := newFixture(t)
	id, _ := f.store.Ask(f.ctx, "How?", "", f.sam, at("2026-11-01", 9))
	for i, c := range []struct {
		by   int64
		text string
		hour int
	}{{f.jane, "First answer", 10}, {f.sam, "Second answer", 11}, {f.jane, "Third, by Jane again", 12}} {
		if _, err := f.store.AddAnswer(f.ctx, id, c.text, c.by, at("2026-11-02", c.hour)); err != nil {
			t.Fatalf("answer %d: %v", i, err)
		}
	}
	q, err := f.store.Get(f.ctx, id, today)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range q.Answers {
		got = append(got, a.Text+"|"+a.By)
	}
	want := []string{"First answer|Jane", "Second answer|Sam", "Third, by Jane again|Jane"}
	if !equal(got, want) {
		t.Errorf("answers = %v, want %v", got, want)
	}
	list, _ := f.store.List(f.ctx, today, false)
	if list.Waiting[0].AnswerCount != 3 {
		t.Errorf("answer count = %d", list.Waiting[0].AnswerCount)
	}
	// Adding an answer doesn't mark the question answered.
	if q.Answered {
		t.Errorf("an answer marked the question answered")
	}
	// Refusals.
	if _, err := f.store.AddAnswer(f.ctx, id, "  ", f.sam, at("2026-11-02", 13)); !errors.Is(err, ErrEmptyAnswer) {
		t.Errorf("empty answer: %v", err)
	}
	if _, err := f.store.AddAnswer(f.ctx, id, strings.Repeat("a", 2001), f.sam, at("2026-11-02", 13)); !errors.Is(err, ErrAnswerTooLong) {
		t.Errorf("long answer: %v", err)
	}
	if _, err := f.store.AddAnswer(f.ctx, 9999, "x", f.sam, at("2026-11-02", 13)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("unknown question: %v", err)
	}
}

// Gate 8.74: Mark answered with who and when, and Not answered yet puts it back.
func TestGate874MarkAndUnmarkAnswered(t *testing.T) {
	f := newFixture(t)
	id, _ := f.store.Ask(f.ctx, "Done yet?", "", f.sam, at("2026-11-01", 9))
	if err := f.store.SetAnswered(f.ctx, id, f.jane, at("2026-11-21", 10)); err != nil {
		t.Fatal(err)
	}
	q, _ := f.store.Get(f.ctx, id, today)
	if !q.Answered || q.AnsweredBy != "Jane" || q.AnsweredLabel != "Sat 21 Nov" {
		t.Errorf("after marking: %+v", q)
	}
	if q.AnsweredText() != "Answered · marked by Jane, Sat 21 Nov" {
		t.Errorf("AnsweredText = %q", q.AnsweredText())
	}
	// Marking again keeps the first person's name (nobody is overwritten).
	if err := f.store.SetAnswered(f.ctx, id, f.sam, at("2026-11-21", 11)); err != nil {
		t.Fatal(err)
	}
	if q, _ := f.store.Get(f.ctx, id, today); q.AnsweredBy != "Jane" {
		t.Errorf("a second marking changed who: %s", q.AnsweredBy)
	}
	if err := f.store.ClearAnswered(f.ctx, id, f.sam, at("2026-11-21", 12)); err != nil {
		t.Fatal(err)
	}
	q, _ = f.store.Get(f.ctx, id, today)
	if q.Answered || q.AnsweredBy != "" {
		t.Errorf("after Not answered yet: %+v", q)
	}
	list, _ := f.store.List(f.ctx, today, false)
	if len(list.Waiting) != 1 || len(list.Answered) != 0 {
		t.Errorf("it should be back among the waiting: %v / %v", texts(list.Waiting), texts(list.Answered))
	}
	if err := f.store.SetAnswered(f.ctx, 9999, f.sam, at("2026-11-21", 12)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("unknown question: %v", err)
	}
}

// Everything is plain text and nothing is ever stored as HTML.
func TestTextIsKeptAsTyped(t *testing.T) {
	f := newFixture(t)
	id, err := f.store.Ask(f.ctx, "Is <b>bold</b> allowed? & more", "<script>alert(1)</script>", f.sam, at("2026-11-01", 9))
	if err != nil {
		t.Fatal(err)
	}
	q, _ := f.store.Get(f.ctx, id, today)
	if q.Text != "Is <b>bold</b> allowed? & more" || q.Details != "<script>alert(1)</script>" {
		t.Errorf("stored as %q / %q", q.Text, q.Details)
	}
}

func TestMigrationIsAddOnly(t *testing.T) {
	f := newFixture(t)
	for _, table := range []string{"questions", "question_answers"} {
		var n int
		if err := f.store.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?)`, table).Scan(&n); err != nil || n == 0 {
			t.Errorf("table %s: %d columns, %v", table, n, err)
		}
	}
}
