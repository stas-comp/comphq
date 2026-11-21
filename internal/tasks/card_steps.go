package tasks

import (
	"context"
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

// Steps under the cards on My jobs (SPEC B14.3, D-107, gates 8.35-8.39). A card
// with steps shows them as a compact tick-only list: the box, the words, and who
// ticked it and when. Adding, rewording, moving and removing steps stay in the
// job's window, which an "Edit steps" link opens.
//
// Jobs in Working on now show their steps open; Up next and Up for grabs cards
// start closed, with "Show steps · 3/7". Without script, ?open=<id> (repeatable)
// opens a card from the server, ?close=<id> closes one that starts open, and
// "Show steps" is a link to that. With script, the choice is kept per job in
// the browser (myjobs.js).

// CardSteps is what a card on My jobs needs to draw its steps. It is embedded in
// LaneTask and simpleCardView, so the one partial draws both.
type CardSteps struct {
	// Steps are the job's live steps, in order.
	Steps []Step
	// StepsOpen is whether the list is drawn open, as the server decides it.
	StepsOpen bool
	// StepsDefaultOpen is whether the job opens by default (Working on now).
	StepsDefaultOpen bool
	// StepsToggle is the no-script link that opens or closes this card.
	StepsToggle string
}

// stepListQueries counts calls to StepsForTasks, so a test can prove a page
// loads every card's steps with one query and not one per card (gate 8.39).
var stepListQueries atomic.Int64

// StepsForTasks returns the live steps of every job in ids, keyed by job, in one
// query (SPEC B14.3: never one per card). today only chooses the short form of
// a tick's date.
func (s *Store) StepsForTasks(ctx context.Context, ids []int64, today time.Time) (map[int64][]Step, error) {
	stepListQueries.Add(1)
	out := make(map[int64][]Step, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.task_id, c.text, c.position, COALESCE(p.name, ''), c.done_at
		FROM task_checklist_items c
		LEFT JOIN people p ON p.id = c.done_by
		WHERE c.removed_at IS NULL AND c.task_id IN (`+placeholders+`)
		ORDER BY c.task_id, c.position, c.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st Step
		var doneAt sql.NullString
		if err := rows.Scan(&st.ID, &st.TaskID, &st.Text, &st.Position, &st.DoneByName, &doneAt); err != nil {
			return nil, err
		}
		if doneAt.Valid {
			t, err := time.Parse(time.RFC3339, doneAt.String)
			if err != nil {
				return nil, err
			}
			t = t.In(today.Location())
			st.Done, st.DoneAt, st.DoneDate = true, t, format.DateShort(t, today)
		}
		out[st.TaskID] = append(out[st.TaskID], st)
	}
	return out, rows.Err()
}

// cardStepState is which cards the request asks to open or close (?open= and
// ?close=, both repeatable), for the no-script path.
type cardStepState struct {
	open  map[int64]bool
	close map[int64]bool
}

func cardStepStateFromQuery(q url.Values) cardStepState {
	st := cardStepState{open: map[int64]bool{}, close: map[int64]bool{}}
	for _, raw := range q["open"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			st.open[id] = true
		}
	}
	for _, raw := range q["close"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			st.close[id] = true
		}
	}
	return st
}

// isOpen: a card is open when it starts open and wasn't closed, or it was asked
// to open.
func (st cardStepState) isOpen(id int64, startsOpen bool) bool {
	if st.open[id] {
		return true
	}
	return startsOpen && !st.close[id]
}

// toggleHref is the link that flips one card and leaves the others as asked.
func (st cardStepState) toggleHref(id int64, nowOpen bool) string {
	v := url.Values{}
	open, closed := map[int64]bool{}, map[int64]bool{}
	for k := range st.open {
		open[k] = true
	}
	for k := range st.close {
		closed[k] = true
	}
	if nowOpen { // it will be closed
		delete(open, id)
		closed[id] = true
	} else {
		delete(closed, id)
		open[id] = true
	}
	for k := range open {
		v.Add("open", strconv.FormatInt(k, 10))
	}
	for k := range closed {
		v.Add("close", strconv.FormatInt(k, 10))
	}
	if len(v) == 0 {
		return "/tasks"
	}
	return "/tasks?" + v.Encode() + "#job-" + strconv.FormatInt(id, 10)
}

// cardStepsFor fills one card's steps, with its open state and toggle link.
func cardStepsFor(id int64, total int, steps []Step, startsOpen bool, st cardStepState) CardSteps {
	if total == 0 {
		return CardSteps{}
	}
	open := st.isOpen(id, startsOpen)
	return CardSteps{Steps: steps, StepsOpen: open, StepsDefaultOpen: startsOpen, StepsToggle: st.toggleHref(id, open)}
}

// applyCardSteps puts steps under the cards of Working on now and Up next, and
// of Up for grabs. (Ideas I'm on have none: an idea isn't work being done yet.)
func applyCardSteps(lane *Lane, grabs []simpleCardView, steps map[int64][]Step, st cardStepState) {
	for i := range lane.WorkingOnNow {
		t := &lane.WorkingOnNow[i]
		t.CardSteps = cardStepsFor(t.ID, t.StepsTotal, steps[t.ID], true, st)
	}
	for i := range lane.UpNext {
		t := &lane.UpNext[i]
		t.CardSteps = cardStepsFor(t.ID, t.StepsTotal, steps[t.ID], false, st)
	}
	for i := range grabs {
		g := &grabs[i]
		g.CardSteps = cardStepsFor(g.ID, g.StepsTotal, steps[g.ID], false, st)
	}
}
