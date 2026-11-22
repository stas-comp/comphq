package questions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// SPEC gates 8.75, 8.76, 8.78, 8.81 (B14.6, D-110): editing, removing, restoring,
// searching and exporting questions.

// Gate 8.75: anyone can change the words; it then shows "edited by Sam, Sat 21 Nov".
func TestGate875EditingAQuestionAndAnAnswerMarksThem(t *testing.T) {
	f := newFixture(t)
	id, _ := f.store.Ask(f.ctx, "Original question", "Original details", f.sam, at("2026-11-01", 9))
	aid, _ := f.store.AddAnswer(f.ctx, id, "Original answer", f.sam, at("2026-11-01", 10))

	if err := f.store.EditQuestion(f.ctx, id, "  Better question ", "New\r\ndetails", f.jane, at("2026-11-21", 9)); err != nil {
		t.Fatal(err)
	}
	if err := f.store.EditAnswer(f.ctx, id, aid, "Better answer", f.jane, at("2026-11-21", 10)); err != nil {
		t.Fatal(err)
	}
	q, _ := f.store.Get(f.ctx, id, today)
	if q.Text != "Better question" || q.Details != "New\ndetails" || !q.Edited || q.EditedBy != "Jane" || q.EditedLabel != "Sat 21 Nov" {
		t.Errorf("question = %+v", q)
	}
	if a := q.Answers[0]; a.Text != "Better answer" || !a.Edited || a.EditedBy != "Jane" || a.EditedLabel != "Sat 21 Nov" || a.By != "Sam" {
		t.Errorf("answer = %+v (still Sam's, edited by Jane)", a)
	}
	// The same limits as asking, and a refusal changes nothing.
	if err := f.store.EditQuestion(f.ctx, id, "", "x", f.sam, at("2026-11-21", 11)); !errors.Is(err, ErrEmptyQuestion) {
		t.Errorf("empty: %v", err)
	}
	if err := f.store.EditQuestion(f.ctx, id, strings.Repeat("q", 301), "", f.sam, at("2026-11-21", 11)); !errors.Is(err, ErrQuestionTooLong) {
		t.Errorf("long: %v", err)
	}
	if err := f.store.EditAnswer(f.ctx, id, aid, strings.Repeat("a", 2001), f.sam, at("2026-11-21", 11)); !errors.Is(err, ErrAnswerTooLong) {
		t.Errorf("long answer: %v", err)
	}
	if err := f.store.EditAnswer(f.ctx, id, 9999, "x", f.sam, at("2026-11-21", 11)); !errors.Is(err, ErrAnswerNotFound) {
		t.Errorf("unknown answer: %v", err)
	}
	if q2, _ := f.store.Get(f.ctx, id, today); q2.Text != "Better question" || q2.EditedBy != "Jane" {
		t.Errorf("a refusal changed the question: %+v", q2)
	}
	// An answer of another question can't be edited through this one.
	other, _ := f.store.Ask(f.ctx, "Another", "", f.sam, at("2026-11-02", 9))
	if err := f.store.EditAnswer(f.ctx, other, aid, "hijack", f.sam, at("2026-11-21", 12)); !errors.Is(err, ErrAnswerNotFound) {
		t.Errorf("an answer of another question: %v", err)
	}
	// Editing doesn't change the answered state.
	if err := f.store.SetAnswered(f.ctx, id, f.sam, at("2026-11-21", 13)); err != nil {
		t.Fatal(err)
	}
	if err := f.store.EditQuestion(f.ctx, id, "Edited again", "", f.sam, at("2026-11-21", 14)); err != nil {
		t.Fatal(err)
	}
	if q3, _ := f.store.Get(f.ctx, id, today); !q3.Answered {
		t.Errorf("editing un-answered the question")
	}
}

// Gate 8.76: removing an answer can be undone; nothing is deleted.
func TestGate876RemoveAndRestoreAnAnswer(t *testing.T) {
	f := newFixture(t)
	id, _ := f.store.Ask(f.ctx, "Q", "", f.sam, at("2026-11-01", 9))
	a1, _ := f.store.AddAnswer(f.ctx, id, "First", f.sam, at("2026-11-01", 10))
	f.store.AddAnswer(f.ctx, id, "Second", f.sam, at("2026-11-01", 11))
	text, err := f.store.RemoveAnswer(f.ctx, id, a1, f.jane, at("2026-11-21", 9))
	if err != nil || text != "First" {
		t.Fatalf("RemoveAnswer = %q, %v", text, err)
	}
	q, _ := f.store.Get(f.ctx, id, today)
	if len(q.Answers) != 1 || q.Answers[0].Text != "Second" {
		t.Errorf("answers after removing = %+v", q.Answers)
	}
	list, _ := f.store.List(f.ctx, today, false)
	if list.Waiting[0].AnswerCount != 1 {
		t.Errorf("the count still includes a removed answer: %d", list.Waiting[0].AnswerCount)
	}
	// Nothing is deleted: the row is there, with who removed it.
	var by int64
	if err := f.store.DB.QueryRow(`SELECT removed_by FROM question_answers WHERE id = ?`, a1).Scan(&by); err != nil || by != f.jane {
		t.Errorf("removed_by = %d, %v", by, err)
	}
	if err := f.store.RestoreAnswer(f.ctx, id, a1, f.sam, at("2026-11-21", 9)); err != nil {
		t.Fatal(err)
	}
	q, _ = f.store.Get(f.ctx, id, today)
	if len(q.Answers) != 2 || q.Answers[0].Text != "First" {
		t.Errorf("after Undo it is back in its place: %+v", q.Answers)
	}
	// Undo twice, or after somebody else did, is not an error; removing twice neither.
	if err := f.store.RestoreAnswer(f.ctx, id, a1, f.sam, at("2026-11-21", 10)); err != nil {
		t.Errorf("a second restore: %v", err)
	}
	f.store.RemoveAnswer(f.ctx, id, a1, f.sam, at("2026-11-21", 11))
	if _, err := f.store.RemoveAnswer(f.ctx, id, a1, f.sam, at("2026-11-21", 12)); err != nil {
		t.Errorf("a second removal: %v", err)
	}
	if _, err := f.store.RemoveAnswer(f.ctx, id, 9999, f.sam, at("2026-11-21", 12)); !errors.Is(err, ErrAnswerNotFound) {
		t.Errorf("unknown answer: %v", err)
	}
}

// Gate 8.76: removing a question moves it to Removed questions, where it can be restored.
func TestGate876RemoveAndRestoreAQuestion(t *testing.T) {
	f := newFixture(t)
	keep, _ := f.store.Ask(f.ctx, "Keep me", "", f.sam, at("2026-11-01", 9))
	gone, _ := f.store.Ask(f.ctx, "Remove me", "", f.sam, at("2026-11-02", 9))
	f.store.AddAnswer(f.ctx, gone, "An answer that goes with it", f.sam, at("2026-11-02", 10))
	if err := f.store.RemoveQuestion(f.ctx, gone, f.jane, at("2026-11-21", 9)); err != nil {
		t.Fatal(err)
	}
	list, _ := f.store.List(f.ctx, today, false)
	if got := texts(list.Waiting); !equal(got, []string{"Keep me"}) {
		t.Errorf("Waiting = %v", got)
	}
	removed, err := f.store.ListRemoved(f.ctx, today)
	if err != nil || len(removed) != 1 || removed[0].Text != "Remove me" || removed[0].RemovedBy != "Jane" || removed[0].RemovedLabel != "Sat 21 Nov" {
		t.Fatalf("Removed questions = %+v, %v", removed, err)
	}
	// A removed question takes no more answers, marks or edits.
	if _, err := f.store.AddAnswer(f.ctx, gone, "late", f.sam, at("2026-11-21", 10)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("answering a removed question: %v", err)
	}
	if err := f.store.SetAnswered(f.ctx, gone, f.sam, at("2026-11-21", 10)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("marking a removed question: %v", err)
	}
	if err := f.store.EditQuestion(f.ctx, gone, "x", "", f.sam, at("2026-11-21", 10)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("editing a removed question: %v", err)
	}
	// Restoring brings it back with its answer, still waiting.
	if err := f.store.RestoreQuestion(f.ctx, gone, f.sam, at("2026-11-21", 11)); err != nil {
		t.Fatal(err)
	}
	list, _ = f.store.List(f.ctx, today, false)
	if got := texts(list.Waiting); !equal(got, []string{"Remove me", "Keep me"}) {
		t.Errorf("after restoring = %v", got)
	}
	if q, _ := f.store.Get(f.ctx, gone, today); len(q.Answers) != 1 || q.Removed {
		t.Errorf("restored = %+v", q)
	}
	if removed, _ := f.store.ListRemoved(f.ctx, today); len(removed) != 0 {
		t.Errorf("Removed questions still lists %+v", removed)
	}
	// Removing twice, and restoring one that isn't removed, are not errors.
	f.store.RemoveQuestion(f.ctx, keep, f.sam, at("2026-11-21", 12))
	if err := f.store.RemoveQuestion(f.ctx, keep, f.sam, at("2026-11-21", 13)); err != nil {
		t.Errorf("a second removal: %v", err)
	}
	if err := f.store.RestoreQuestion(f.ctx, gone, f.sam, at("2026-11-21", 14)); err != nil {
		t.Errorf("restoring one that isn't removed: %v", err)
	}
	if err := f.store.RestoreQuestion(f.ctx, 9999, f.sam, at("2026-11-21", 14)); !errors.Is(err, ErrQuestionNotFound) {
		t.Errorf("unknown: %v", err)
	}
}

// Gate 8.78: every word in the text, details or live answers; the last may be half
// typed; accents and case fold; waiting first, then question-text matches before
// details or answer matches, newest first; removed ones aren't found.
func TestGate878SearchFindsAndOrders(t *testing.T) {
	f := newFixture(t)
	ask := func(text, details string, day string) int64 {
		id, err := f.store.Ask(f.ctx, text, details, f.sam, at(day, 9))
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	ask("Where are the Christmas card lists", "", "2026-11-01")
	ask("Christmas card budget?", "", "2026-11-05")
	ask("Something else", "It is about the christmas card printer", "2026-11-06")
	answerMatch := ask("Another thing", "", "2026-11-07")
	f.store.AddAnswer(f.ctx, answerMatch, "Try the Christmas cards folder", f.jane, at("2026-11-07", 10))
	answeredTextMatch := ask("Old Christmas card question", "", "2026-10-01")
	f.store.SetAnswered(f.ctx, answeredTextMatch, f.jane, at("2026-11-08", 9))
	removed := ask("Removed christmas card thing", "", "2026-11-09")
	f.store.RemoveQuestion(f.ctx, removed, f.sam, at("2026-11-09", 10))
	ask("Nothing relevant", "", "2026-11-10")

	hits, err := f.store.Search(f.ctx, "christmas card", today)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, h := range hits {
		got = append(got, h.Text)
	}
	// Waiting ones first: those with the words in the question (newest first), then
	// those with them in the details or an answer (newest first); answered last.
	want := []string{"Christmas card budget?", "Where are the Christmas card lists", "Another thing", "Something else", "Old Christmas card question"}
	if !equal(got, want) {
		t.Errorf("Search = %v\nwant     %v", got, want)
	}
	// Half-typed last word, and every word needed.
	if hits, _ := f.store.Search(f.ctx, "christ bud", today); len(hits) != 1 || hits[0].Text != "Christmas card budget?" {
		t.Errorf("half-typed: %+v", hits)
	}
	if hits, _ := f.store.Search(f.ctx, "christmas zebra", today); len(hits) != 0 {
		t.Errorf("a word that isn't there: %+v", hits)
	}
	// Folding.
	f.store.Ask(f.ctx, "Où est l'électricité du bureau", "", f.sam, at("2026-11-11", 9))
	if hits, _ := f.store.Search(f.ctx, "ÉLECTRICITÉ bur", today); len(hits) != 1 {
		t.Errorf("folding: %+v", hits)
	}
	// Fewer than two letters finds nothing, and the answer text that matched is marked.
	if hits, _ := f.store.Search(f.ctx, "c", today); len(hits) != 0 {
		t.Errorf("one letter: %+v", hits)
	}
	hits, _ = f.store.Search(f.ctx, "christmas cards folder", today)
	if len(hits) != 1 || !strings.Contains(string(hits[0].Passage), "<mark>Christmas</mark>") {
		t.Errorf("passage = %+v", hits)
	}
	// What was typed is escaped in the marked title.
	f.store.Ask(f.ctx, "<b>tag</b> question", "", f.sam, at("2026-11-12", 9))
	hits, _ = f.store.Search(f.ctx, "tag question", today)
	if len(hits) != 1 || strings.Contains(string(hits[0].Title), "<b>") || !strings.Contains(string(hits[0].Title), "&lt;b&gt;") {
		t.Errorf("title = %+v", hits)
	}
}

func TestSearchOfALargeLibraryStaysFast(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 1000; i++ {
		id, err := f.store.Ask(f.ctx, fmt.Sprintf("Question number %d about the mailing", i), "Some details about printing", f.sam, at("2026-10-01", 9))
		if err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 3; j++ {
			f.store.AddAnswer(f.ctx, id, fmt.Sprintf("Answer %d about stamps and envelopes", j), f.jane, at("2026-10-02", 9))
		}
	}
	start := time.Now()
	hits, err := f.store.Search(f.ctx, "envel stam", today)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("a search of 1,000 questions and 3,000 answers took %v, want well inside a second", d)
	}
	if len(hits) == 0 || len(hits) > SearchLimit {
		t.Errorf("%d hits", len(hits))
	}
}

// Gate 8.81: every question and answer, removed rows included.
func TestGate881ExportRowsListEverythingIncludingRemoved(t *testing.T) {
	f := newFixture(t)
	id, _ := f.store.Ask(f.ctx, "Exported, with a comma", "Details", f.sam, at("2026-11-01", 9))
	a, _ := f.store.AddAnswer(f.ctx, id, "An answer", f.jane, at("2026-11-02", 9))
	gone, _ := f.store.AddAnswer(f.ctx, id, "A removed answer", f.jane, at("2026-11-03", 9))
	f.store.RemoveAnswer(f.ctx, id, gone, f.sam, at("2026-11-04", 9))
	f.store.SetAnswered(f.ctx, id, f.jane, at("2026-11-05", 9))
	f.store.EditAnswer(f.ctx, id, a, "An answer, edited", f.sam, at("2026-11-06", 9))
	dead, _ := f.store.Ask(f.ctx, "A removed question", "", f.jane, at("2026-11-07", 9))
	f.store.RemoveQuestion(f.ctx, dead, f.sam, at("2026-11-08", 9))

	qrows, err := QuestionsCSV{DB: f.store.DB}.ExportRows(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !equal(qrows[0], []string{"Question", "Details", "Asked by", "Asked", "Answered by", "Answered", "Edited by", "Edited", "Removed by", "Removed"}) {
		t.Errorf("questions.csv header = %v", qrows[0])
	}
	if len(qrows) != 3 || qrows[1][0] != "Exported, with a comma" || qrows[1][2] != "Sam" || qrows[1][4] != "Jane" || qrows[1][8] != "" ||
		qrows[2][0] != "A removed question" || qrows[2][8] != "Sam" || qrows[2][9] == "" {
		t.Errorf("questions.csv = %v", qrows)
	}
	arows, err := AnswersCSV{DB: f.store.DB}.ExportRows(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !equal(arows[0], []string{"Question", "Answer", "Answered by", "Answered", "Edited by", "Edited", "Removed by", "Removed"}) {
		t.Errorf("answers.csv header = %v", arows[0])
	}
	if len(arows) != 3 || arows[1][1] != "An answer, edited" || arows[1][4] != "Sam" || arows[2][1] != "A removed answer" || arows[2][6] != "Sam" {
		t.Errorf("answers.csv = %v", arows)
	}
}

// Gate 8.77: the Briefing lists waiting questions, oldest first, with how many wait
// in all; answered and removed ones never appear.
func TestGate877WaitingOldestIsOldestFirstAndCounted(t *testing.T) {
	f := newFixture(t)
	ids := make([]int64, 7)
	for i := range ids {
		ids[i], _ = f.store.Ask(f.ctx, fmt.Sprintf("Question %d", i), "", f.sam, at("2026-11-01", 9+i))
	}
	f.store.AddAnswer(f.ctx, ids[0], "An answer alone does not answer it", f.jane, at("2026-11-02", 9))
	f.store.SetAnswered(f.ctx, ids[1], f.jane, at("2026-11-02", 10))
	f.store.RemoveQuestion(f.ctx, ids[2], f.jane, at("2026-11-02", 11))

	rows, total, err := f.store.WaitingOldest(f.ctx, 3, today)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5 (7 asked, 1 answered, 1 removed)", total)
	}
	var got []string
	for _, r := range rows {
		got = append(got, r.Text)
	}
	if want := "Question 0,Question 3,Question 4"; strings.Join(got, ",") != want {
		t.Errorf("got %v, want %s (oldest first)", got, want)
	}
	if rows[0].AnswerCount != 1 || rows[0].AskedBy != "Sam" || rows[0].AskedLabel == "" {
		t.Errorf("row = %+v", rows[0])
	}
}
