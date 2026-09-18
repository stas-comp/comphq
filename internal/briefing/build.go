// Package briefing is the Saturday Briefing (SPEC A8): what must be
// done by Saturday, what is happening this week, and what is coming up.
// Build holds every rule as a pure function of its inputs; the data
// arrives through small interfaces declared here and wired at
// registration (SPEC B2, D-53).
package briefing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app/format"
)

const dateLayout = "2006-01-02"

// Person is one assignee, as a task card shows them.
type Person struct {
	ID      int64
	Name    string
	Removed bool
}

// Task is an unfinished task with a due date, as the Briefing's own
// small shape (not tasks.Task itself).
type Task struct {
	ID       int64
	Title    string
	Notes    string
	DueDate  string // "YYYY-MM-DD"
	Position int
	People   []Person
}

// Occurrence is one concrete event date, as the Briefing's own shape.
type Occurrence struct {
	EventID      int64
	Title        string
	Notes        string
	StartDate    string // "YYYY-MM-DD"
	EndDate      string // "YYYY-MM-DD"; "" means the start date
	StartTime    string // "HH:MM"; "" means all day
	EndTime      string
	OriginalDate string
	// NoticeDays is the series' "show ahead" time; only Coming up uses it.
	NoticeDays int
}

// Input is what Build needs, already fetched: the unfinished tasks with
// due dates, the occurrences overlapping today..Friday, and each
// series' candidate next occurrence for Coming up. Build re-applies the
// date rules to all of it, so a source that fetches a little more than
// it must can't put anything on the page it shouldn't.
type Input struct {
	Today    time.Time
	PersonID int64
	Tasks    []Task
	ThisWeek []Occurrence
	Upcoming []Occurrence
}

// TaskCard is one "Must be done" card.
type TaskCard struct {
	ID      int64
	Title   string
	Notes   string
	Stamp   string // TODAY, OVERDUE or a weekday date such as WED 23 SEP
	Overdue bool
	Yours   bool
	People  []Person
}

// EventCard is one "This week" or "Coming up" card.
type EventCard struct {
	EventID      int64
	Title        string
	Notes        string
	Stamp        string // TODAY, MON 21 SEP, IN 6 WEEKS, …
	DateLabel    string // "Mon 21 Sep 2026"
	TimeLabel    string // "18:00–20:00", "" for all day
	UntilLabel   string // "until Tue 22 Sep" for multi-day events; "" otherwise
	OriginalDate string
}

// Briefing is the finished page.
type Briefing struct {
	Saturday time.Time
	Friday   time.Time
	IsToday  bool // today is the Saturday itself
	MustDo   []TaskCard
	ThisWeek []EventCard
	ComingUp []EventCard
}

// SaturdayFor returns the Saturday a briefing dated `today` is for:
// today if it's a Saturday, otherwise the coming one (SPEC A8).
func SaturdayFor(today time.Time) time.Time {
	days := (int(time.Saturday) - int(today.Weekday()) + 7) % 7
	return today.AddDate(0, 0, days)
}

// Build applies SPEC A8's rules to what it's given. It never fetches
// anything itself.
func Build(in Input) Briefing {
	today := in.Today
	saturday := SaturdayFor(today)
	friday := saturday.AddDate(0, 0, 6)
	todayStr, fridayStr := today.Format(dateLayout), friday.Format(dateLayout)

	b := Briefing{Saturday: saturday, Friday: friday, IsToday: saturday.Equal(today)}

	tasks := make([]Task, 0, len(in.Tasks))
	for _, t := range in.Tasks {
		if t.DueDate != "" && t.DueDate <= fridayStr {
			tasks = append(tasks, t)
		}
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].DueDate != tasks[j].DueDate {
			return tasks[i].DueDate < tasks[j].DueDate
		}
		return tasks[i].Position < tasks[j].Position
	})
	for _, t := range tasks {
		card := TaskCard{ID: t.ID, Title: t.Title, Notes: t.Notes, People: t.People, Overdue: t.DueDate < todayStr}
		for _, p := range t.People {
			if p.ID == in.PersonID {
				card.Yours = true
			}
		}
		card.Stamp = taskStamp(t.DueDate, today)
		b.MustDo = append(b.MustDo, card)
	}

	var week []Occurrence
	for _, o := range in.ThisWeek {
		if endOf(o) >= todayStr && o.StartDate <= fridayStr {
			week = append(week, o)
		}
	}
	sortOccurrences(week)
	for _, o := range week {
		b.ThisWeek = append(b.ThisWeek, weekCard(o, today))
	}

	// Coming up: only each series' earliest occurrence after Friday whose
	// show-ahead time has started (start − notice ≤ Saturday).
	nextOf := map[int64]Occurrence{}
	for _, o := range in.Upcoming {
		if o.StartDate <= fridayStr || o.NoticeDays <= 0 {
			continue
		}
		start, err := time.ParseInLocation(dateLayout, o.StartDate, today.Location())
		if err != nil || start.AddDate(0, 0, -o.NoticeDays).After(saturday) {
			continue
		}
		if cur, ok := nextOf[o.EventID]; !ok || o.StartDate < cur.StartDate {
			nextOf[o.EventID] = o
		}
	}
	var coming []Occurrence
	for _, o := range nextOf {
		coming = append(coming, o)
	}
	sortOccurrences(coming)
	for _, o := range coming {
		start, _ := time.ParseInLocation(dateLayout, o.StartDate, today.Location())
		card := baseEventCard(o, start)
		card.Stamp = strings.ToUpper(distance(saturday, start))
		b.ComingUp = append(b.ComingUp, card)
	}
	return b
}

func endOf(o Occurrence) string {
	if o.EndDate != "" {
		return o.EndDate
	}
	return o.StartDate
}

func sortOccurrences(occs []Occurrence) {
	sort.SliceStable(occs, func(i, j int) bool {
		a, b := occs[i], occs[j]
		if a.StartDate != b.StartDate {
			return a.StartDate < b.StartDate
		}
		if a.StartTime != b.StartTime {
			return a.StartTime < b.StartTime
		}
		return a.EventID < b.EventID
	})
}

// taskStamp is TODAY, OVERDUE, or a weekday date like "WED 23 SEP".
func taskStamp(due string, today time.Time) string {
	todayStr := today.Format(dateLayout)
	switch {
	case due < todayStr:
		return "OVERDUE"
	case due == todayStr:
		return "TODAY"
	}
	d, err := time.Parse(dateLayout, due)
	if err != nil {
		return ""
	}
	return strings.ToUpper(d.Format("Mon 2 Jan"))
}

func weekCard(o Occurrence, today time.Time) EventCard {
	start, _ := time.Parse(dateLayout, o.StartDate)
	card := baseEventCard(o, start)
	todayStr := today.Format(dateLayout)
	if o.StartDate <= todayStr {
		card.Stamp = "TODAY"
	} else {
		card.Stamp = strings.ToUpper(start.Format("Mon 2 Jan"))
	}
	// An event already under way (started before today) shows how long
	// it still runs — SPEC gate 3.09.
	if o.StartDate < todayStr && endOf(o) > o.StartDate {
		if end, err := time.Parse(dateLayout, endOf(o)); err == nil {
			card.UntilLabel = "until " + end.Format("Mon 2 Jan")
		}
	}
	return card
}

func baseEventCard(o Occurrence, start time.Time) EventCard {
	card := EventCard{
		EventID:      o.EventID,
		Title:        o.Title,
		Notes:        o.Notes,
		DateLabel:    format.Date(start),
		OriginalDate: o.OriginalDate,
	}
	switch {
	case o.StartTime != "" && o.EndTime != "":
		card.TimeLabel = o.StartTime + "–" + o.EndTime
	case o.StartTime != "":
		card.TimeLabel = o.StartTime
	}
	return card
}

// distance is "in N weeks" when the gap from Saturday is a whole number
// of weeks, else "in N days" (SPEC gate 3.07).
func distance(saturday, start time.Time) string {
	days := int(start.Sub(saturday).Hours()/24 + 0.5)
	if days%7 == 0 {
		return plural(days/7, "in %d week", "in %d weeks")
	}
	return plural(days, "in %d day", "in %d days")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf(one, n)
	}
	return fmt.Sprintf(many, n)
}
