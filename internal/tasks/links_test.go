package tasks

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// SPEC gates 7.41-7.46, 7.48 (B13.6, D-96): linked jobs.

type linkFixture struct {
	t     *testing.T
	store *Store
	ctx   context.Context
	sam   int64
}

func newLinkFixture(t *testing.T) *linkFixture {
	t.Helper()
	db := openTestDB(t)
	return &linkFixture{t: t, store: &Store{DB: db}, ctx: context.Background(), sam: testPerson(t, db, "Sam")}
}

func (f *linkFixture) job(title string) Task { return createNamed(f.t, f.store, f.ctx, f.sam, title, StageTodo) }

func (f *linkFixture) link(a, b int64, kind string) error {
	return f.store.AddLink(f.ctx, a, b, kind, f.sam, fixedNow)
}

func titlesOf(views []LinkView) []string {
	var out []string
	for _, v := range views {
		out = append(out, v.Title)
	}
	return out
}

func (f *linkFixture) history(id int64) string {
	acts, err := f.store.ListActivity(f.ctx, id)
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, a := range acts {
		out = append(out, a.Text)
	}
	return strings.Join(out, "\n")
}

// Gate 7.41: three groups, and the other job shows the same link from its own side.
func TestLinksAreShownInThreeGroupsFromBothSides(t *testing.T) {
	f := newLinkFixture(t)
	me, toner, news, together := f.job("Me"), f.job("Order toner"), f.job("Print newsletter"), f.job("Same project")
	for _, c := range []struct {
		other int64
		kind  string
	}{{toner.ID, LinkFirst}, {news.ID, LinkThen}, {together.ID, LinkRelated}} {
		if err := f.link(me.ID, c.other, c.kind); err != nil {
			t.Fatalf("link %s: %v", c.kind, err)
		}
	}
	got, err := f.store.ListLinks(f.ctx, me.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(titlesOf(got.DoFirst), []string{"Order toner"}) ||
		!reflect.DeepEqual(titlesOf(got.Then), []string{"Print newsletter"}) ||
		!reflect.DeepEqual(titlesOf(got.Related), []string{"Same project"}) {
		t.Errorf("from Me: first=%v then=%v related=%v", titlesOf(got.DoFirst), titlesOf(got.Then), titlesOf(got.Related))
	}
	// From the other side the same links read the other way round.
	fromToner, _ := f.store.ListLinks(f.ctx, toner.ID)
	fromNews, _ := f.store.ListLinks(f.ctx, news.ID)
	fromTogether, _ := f.store.ListLinks(f.ctx, together.ID)
	if !reflect.DeepEqual(titlesOf(fromToner.Then), []string{"Me"}) || len(fromToner.DoFirst) != 0 {
		t.Errorf("Order toner sees %+v, want Me under Then", fromToner)
	}
	if !reflect.DeepEqual(titlesOf(fromNews.DoFirst), []string{"Me"}) {
		t.Errorf("Print newsletter sees %+v, want Me under Do first", fromNews)
	}
	if !reflect.DeepEqual(titlesOf(fromTogether.Related), []string{"Me"}) {
		t.Errorf("Same project sees %+v, want Me under Related", fromTogether)
	}
	// A related link is stored once, lower id first, whichever side made it.
	var n int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM task_links WHERE kind = 'related' AND task_id < other_task_id`).Scan(&n)
	if n != 1 {
		t.Errorf("related link stored %d times in the right order, want once", n)
	}
	// Column and Finished are carried for each line.
	if got.DoFirst[0].Column != "To do" || got.DoFirst[0].Finished {
		t.Errorf("line = %+v", got.DoFirst[0])
	}
}

// Gate 7.45: every refusal, with a plain message and no change.
func TestLinkRefusals(t *testing.T) {
	f := newLinkFixture(t)
	a, b, c := f.job("A"), f.job("B"), f.job("C")

	if err := f.link(a.ID, a.ID, LinkRelated); err != ErrLinkSelf {
		t.Errorf("self-link: %v", err)
	}
	if err := f.link(a.ID, b.ID, LinkFirst); err != nil {
		t.Fatal(err)
	}
	// A second live link between the same two jobs, of any kind, from either side.
	for _, kind := range []string{LinkFirst, LinkThen, LinkRelated} {
		if err := f.link(a.ID, b.ID, kind); err != ErrLinkExists {
			t.Errorf("duplicate %s: %v", kind, err)
		}
		if err := f.link(b.ID, a.ID, kind); err != ErrLinkExists {
			t.Errorf("duplicate from the other side %s: %v", kind, err)
		}
	}
	// A loop: B already has to wait for... A has B first; so B can't also have A first.
	var loop *LinkLoopError
	if err := f.link(b.ID, a.ID, LinkThen); !errors.Is(err, ErrLinkExists) {
		t.Errorf("reverse direct link = %v, want the 'already linked' refusal", err)
	}
	// Three-job circle: B before A (A has B first), C before B ... then A before C closes it.
	if err := f.link(b.ID, c.ID, LinkFirst); err != nil { // C before B
		t.Fatal(err)
	}
	// Now C before B before A. Making A come before C would close the circle.
	err := f.link(c.ID, a.ID, LinkFirst) // A before C
	if !errors.As(err, &loop) {
		t.Fatalf("three-job loop = %v, want a LinkLoopError", err)
	}
	if !strings.Contains(err.Error(), "“A”") {
		t.Errorf("loop message = %q, want it to name the other job", err.Error())
	}
	var count int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM task_links WHERE removed_at IS NULL`).Scan(&count)
	if count != 2 {
		t.Errorf("%d live links after the refusals, want the 2 that were made", count)
	}

	// A removed job can't be linked to.
	gone := f.job("Gone")
	if err := f.store.Remove(f.ctx, gone.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := f.link(a.ID, gone.ID, LinkRelated); err != ErrLinkOtherRemoved {
		t.Errorf("link to a removed job: %v", err)
	}
	if err := f.link(a.ID, 99999, LinkRelated); err != ErrLinkOtherRemoved {
		t.Errorf("link to a job that doesn't exist: %v", err)
	}
	if err := f.link(a.ID, c.ID, "sideways"); err != ErrLinkKind {
		t.Errorf("a made-up kind: %v", err)
	}
}

func TestLinkLoopOfTwoGoingTheOtherWay(t *testing.T) {
	f := newLinkFixture(t)
	a, b := f.job("Print newsletter"), f.job("Order toner")
	if err := f.link(a.ID, b.ID, LinkFirst); err != nil { // Order toner before Print newsletter
		t.Fatal(err)
	}
	// Remove the direct link's blocking by using a third job: b before c, c before a (via "then").
	c := f.job("Proof it")
	if err := f.link(b.ID, c.ID, LinkThen); err != nil { // Order toner before Proof it
		t.Fatal(err)
	}
	if err := f.link(c.ID, a.ID, LinkThen); err != nil { // Proof it before Print newsletter
		t.Fatal(err)
	}
	// Print newsletter before Order toner would now go round in a circle: refused with
	// the "already linked" rule first (a and b are directly linked), so use a fresh pair.
	d := f.job("Distribute")
	if err := f.link(a.ID, d.ID, LinkThen); err != nil { // Print newsletter before Distribute
		t.Fatal(err)
	}
	var loop *LinkLoopError
	if err := f.link(d.ID, b.ID, LinkThen); !errors.As(err, &loop) { // Distribute before Order toner: circle
		t.Errorf("circle through three links = %v, want a loop refusal", err)
	}
}

// Gate 7.45: more than 20 links on one job.
func TestTwentyLinksIsTheMost(t *testing.T) {
	f := newLinkFixture(t)
	hub := f.job("Hub")
	for i := 0; i < 20; i++ {
		if err := f.link(hub.ID, f.job("Spoke "+string(rune('A'+i))).ID, LinkRelated); err != nil {
			t.Fatalf("link %d: %v", i+1, err)
		}
	}
	if err := f.link(hub.ID, f.job("One too many").ID, LinkRelated); err != ErrLinkTooMany {
		t.Errorf("21st link = %v, want ErrLinkTooMany", err)
	}
	// The other job is full, too: it can't be linked from the other side either.
	if err := f.link(f.job("Another").ID, hub.ID, LinkFirst); err != ErrLinkTooMany {
		t.Errorf("link to a full job = %v", err)
	}
}

// Two people linking at once: the rules are checked inside the write transaction.
func TestTwoPeopleLinkingTheSamePairAtOnce(t *testing.T) {
	f := newLinkFixture(t)
	a, b := f.job("A"), f.job("B")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = f.link(a.ID, b.ID, LinkRelated)
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, e := range errs {
		if e == nil {
			ok++
		} else if e != ErrLinkExists {
			t.Errorf("unexpected error %v", e)
		}
	}
	if ok != 1 {
		t.Errorf("%d of 2 simultaneous links succeeded, want exactly 1", ok)
	}

	// And two people closing a circle at once: at most one side wins.
	x, y := f.job("X"), f.job("Y")
	z := f.job("Z")
	if err := f.link(y.ID, x.ID, LinkFirst); err != nil { // X before Y
		t.Fatal(err)
	}
	if err := f.link(z.ID, y.ID, LinkFirst); err != nil { // Y before Z
		t.Fatal(err)
	}
	var wg2 sync.WaitGroup
	results := make([]error, 2)
	wg2.Add(2)
	go func() { defer wg2.Done(); results[0] = f.link(x.ID, z.ID, LinkFirst) }() // Z before X: closes the circle
	go func() { defer wg2.Done(); results[1] = f.link(x.ID, z.ID, LinkFirst) }()
	wg2.Wait()
	for _, e := range results {
		var loop *LinkLoopError
		if e == nil {
			t.Errorf("a circle was allowed: %v", results)
		} else if !errors.As(e, &loop) && e != ErrLinkExists {
			t.Errorf("unexpected: %v", e)
		}
	}
}

// Gate 7.46: a removed job's links are hidden, and come back when it is restored; finished jobs keep theirs.
func TestRemovedJobsLinksAreHiddenAndComeBack(t *testing.T) {
	f := newLinkFixture(t)
	a, b := f.job("A"), f.job("B")
	if err := f.link(a.ID, b.ID, LinkFirst); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Remove(f.ctx, b.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	got, _ := f.store.ListLinks(f.ctx, a.ID)
	if got.Count() != 0 {
		t.Errorf("a removed job's link is still shown: %+v", got)
	}
	if err := f.store.Restore(f.ctx, b.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	got, _ = f.store.ListLinks(f.ctx, a.ID)
	if !reflect.DeepEqual(titlesOf(got.DoFirst), []string{"B"}) {
		t.Errorf("after the restore: %+v, want the link back", got)
	}
	// Undo brings it back too (gate 7.16): nothing about the link was touched.
	if err := f.store.Remove(f.ctx, b.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Undo(f.ctx, b.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	got, _ = f.store.ListLinks(f.ctx, a.ID)
	if got.Count() != 1 {
		t.Errorf("after Undo: %+v", got)
	}
	// A finished job keeps its link and shows Finished.
	if err := f.store.Move(f.ctx, b.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	got, _ = f.store.ListLinks(f.ctx, a.ID)
	if len(got.DoFirst) != 1 || !got.DoFirst[0].Finished {
		t.Errorf("finished job's line = %+v, want Finished", got.DoFirst)
	}
}

// Gate 7.44: removing a link takes it from both jobs; History on both, with the title kept after a rename.
func TestRemovingALinkAndItsHistoryOnBothJobs(t *testing.T) {
	f := newLinkFixture(t)
	a, b := f.job("Print newsletter"), f.job("Order toner")
	if err := f.link(a.ID, b.ID, LinkFirst); err != nil {
		t.Fatal(err)
	}
	if h := f.history(a.ID); !strings.Contains(h, "Sam linked this to “Order toner” (do first)") {
		t.Errorf("History of A = %q", h)
	}
	if h := f.history(b.ID); !strings.Contains(h, "Sam linked this to “Print newsletter” (then)") {
		t.Errorf("History of B = %q (from its own side it is the other way round)", h)
	}
	// Renaming the other job later can't rewrite what History said.
	if err := f.store.Update(f.ctx, b.ID, UpdateInput{Title: "Order cartridges", Size: "M", Stage: StageTodo}, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if h := f.history(a.ID); !strings.Contains(h, "“Order toner”") || strings.Contains(h, "Order cartridges") {
		t.Errorf("History of A after a rename = %q, want the title as it was", h)
	}

	links, _ := f.store.ListLinks(f.ctx, a.ID)
	if err := f.store.RemoveLink(f.ctx, a.ID, links.DoFirst[0].LinkID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{a.ID, b.ID} {
		got, _ := f.store.ListLinks(f.ctx, id)
		if got.Count() != 0 {
			t.Errorf("job %d still shows the removed link: %+v", id, got)
		}
	}
	// The removal row records the title as it is when the link is removed; the earlier row
	// still says “Order toner” (checked above), so history is never rewritten.
	if h := f.history(a.ID); !strings.Contains(h, "Sam removed the link to “Order cartridges” (do first)") || !strings.Contains(h, "linked this to “Order toner”") {
		t.Errorf("History of A = %q", h)
	}
	if h := f.history(b.ID); !strings.Contains(h, "Sam removed the link to “Print newsletter” (then)") {
		t.Errorf("History of B = %q", h)
	}
	// It was never deleted (A3), and removing it twice is quiet.
	var removed int
	f.store.DB.QueryRow(`SELECT COUNT(*) FROM task_links WHERE removed_at IS NOT NULL AND removed_by IS NOT NULL`).Scan(&removed)
	if removed != 1 {
		t.Errorf("%d removed links kept in the table, want 1", removed)
	}
	if err := f.store.RemoveLink(f.ctx, a.ID, links.DoFirst[0].LinkID, f.sam, fixedNow); err != nil {
		t.Errorf("removing it again = %v, want no error", err)
	}
	if err := f.store.RemoveLink(f.ctx, f.job("Stranger").ID, links.DoFirst[0].LinkID, f.sam, fixedNow); err != ErrLinkNotFound {
		t.Errorf("removing another job's link = %v, want ErrLinkNotFound", err)
	}
	// And the pair can be linked again afterwards.
	if err := f.link(a.ID, b.ID, LinkRelated); err != nil {
		t.Errorf("re-linking after a removal: %v", err)
	}
}

// Gate 7.42 (the query): WAITING comes and goes with the first job's state.
func TestWaitingOnFollowsTheFirstJobsState(t *testing.T) {
	f := newLinkFixture(t)
	toner, news := f.job("Order toner"), f.job("Print newsletter")
	if err := f.link(news.ID, toner.ID, LinkFirst); err != nil {
		t.Fatal(err)
	}
	waiting := func() map[int64][]string {
		m, err := f.store.WaitingOn(f.ctx, []int64{toner.ID, news.ID})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	if got := waiting(); !reflect.DeepEqual(got, map[int64][]string{news.ID: {"Order toner"}}) {
		t.Errorf("waiting = %v, want Print newsletter waiting on Order toner", got)
	}
	// Done clears it; Reopen brings it back (gate 7.42).
	if err := f.store.Move(f.ctx, toner.ID, MoveInput{Stage: StageDone, ToBottom: true}, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := waiting(); len(got) != 0 {
		t.Errorf("waiting after Done = %v, want none", got)
	}
	if err := f.store.Reopen(f.ctx, toner.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := waiting(); len(got[news.ID]) != 1 {
		t.Errorf("waiting after Reopen = %v", got)
	}
	// A removed first job isn't waited on; related links never wait.
	if err := f.store.Remove(f.ctx, toner.ID, f.sam, fixedNow); err != nil {
		t.Fatal(err)
	}
	if got := waiting(); len(got) != 0 {
		t.Errorf("waiting on a removed job = %v", got)
	}
	other := f.job("Other")
	if err := f.link(news.ID, other.ID, LinkRelated); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.store.WaitingOn(f.ctx, []int64{news.ID, other.ID}); len(got) != 0 {
		t.Errorf("a related link made a job wait: %v", got)
	}
	// Several jobs to wait on are all listed, in the order linked.
	a, b := f.job("Alpha"), f.job("Bravo")
	target := f.job("Target")
	f.link(target.ID, a.ID, LinkFirst)
	f.link(target.ID, b.ID, LinkFirst)
	if got, _ := f.store.WaitingOn(f.ctx, []int64{target.ID}); !reflect.DeepEqual(got[target.ID], []string{"Alpha", "Bravo"}) {
		t.Errorf("waiting on %v, want [Alpha Bravo]", got[target.ID])
	}
	// No ids: no query, no entries.
	if got, err := f.store.WaitingOn(f.ctx, nil); err != nil || len(got) != 0 {
		t.Errorf("WaitingOn(nil) = %v, %v", got, err)
	}
}

// Gate 7.48: links.csv.
func TestLinksCSV(t *testing.T) {
	f := newLinkFixture(t)
	a, b, c := f.job("Print newsletter"), f.job("Order toner"), f.job("Same project")
	f.link(a.ID, b.ID, LinkFirst) // Order toner (b) before Print newsletter (a)
	f.link(a.ID, c.ID, LinkRelated)
	links, _ := f.store.ListLinks(f.ctx, a.ID)
	f.store.RemoveLink(f.ctx, a.ID, links.Related[0].LinkID, f.sam, fixedNow)

	rows, err := (LinksCSV{DB: f.store.DB}).ExportRows(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"Job", "Linked job", "Link", "Linked by", "Linked", "Removed"},
		{"Order toner", "Print newsletter", "do first", "Sam", "Thu 17 Sep 2026, 12:00", ""},
		{"Print newsletter", "Same project", "related", "Sam", "Thu 17 Sep 2026, 12:00", "Thu 17 Sep 2026, 12:00"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("links.csv rows =\n%v\nwant\n%v", rows, want)
	}
}
