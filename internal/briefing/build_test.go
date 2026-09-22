package briefing

import (
	"reflect"
	"testing"
	"time"
)

func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation(dateLayout, s, time.Local)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

// SPEC gate 6.20, D-83: the Calendar's month grid moving to Sunday-first
// (P6-04) changes nothing about the Briefing, which was always built
// around Saturday itself, not around where a week starts ("this week"
// still runs from today to Friday). Pinned to the exact date PLAN-v1.3.md
// names: the Briefing for Sat 26 Sep 2026 is exactly what v1.2 produced.
func TestBriefingForSat26Sep2026UnchangedByTheSundayFirstCalendar(t *testing.T) {
	got := Build(Input{Today: day(t, "2026-09-26")})
	want := Briefing{
		Saturday: day(t, "2026-09-26"),
		Friday:   day(t, "2026-10-02"),
		IsToday:  true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Build(Sat 26 Sep 2026) = %+v, want %+v", got, want)
	}
}

// SPEC A8: the briefing is for today if it's Saturday, else the coming
// Saturday; the Friday after it closes the window.
func TestSaturdayAndFridayForEveryWeekday(t *testing.T) {
	// Sun 13 Sep … Sat 19 Sep 2026.
	cases := map[string][2]string{
		"2026-09-13": {"2026-09-19", "2026-09-25"},
		"2026-09-14": {"2026-09-19", "2026-09-25"},
		"2026-09-15": {"2026-09-19", "2026-09-25"},
		"2026-09-16": {"2026-09-19", "2026-09-25"},
		"2026-09-17": {"2026-09-19", "2026-09-25"},
		"2026-09-18": {"2026-09-19", "2026-09-25"},
		"2026-09-19": {"2026-09-19", "2026-09-25"},
		"2026-09-20": {"2026-09-26", "2026-10-02"},
	}
	for today, want := range cases {
		b := Build(Input{Today: day(t, today)})
		if got := [2]string{b.Saturday.Format(dateLayout), b.Friday.Format(dateLayout)}; got != want {
			t.Errorf("today %s: Saturday/Friday = %v, want %v", today, got, want)
		}
		if isSat := today == "2026-09-19"; b.IsToday != isSat {
			t.Errorf("today %s: IsToday = %v, want %v", today, b.IsToday, isSat)
		}
	}
}

// SPEC gate 3.03: due on or before Friday 25 Sep, overdue included and
// stamped, earliest first; Wed 23 Sep in, Sat 26 Sep out.
func TestMustBeDoneRules(t *testing.T) {
	alex := Person{ID: 1, Name: "Alex"}
	b := Build(Input{
		Today:    day(t, "2026-09-19"),
		PersonID: 1,
		Tasks: []Task{
			{ID: 1, Title: "Saturday next", DueDate: "2026-09-26"},
			{ID: 2, Title: "Wednesday", DueDate: "2026-09-23", Position: 2, People: []Person{alex}},
			{ID: 3, Title: "Overdue", DueDate: "2026-09-10"},
			{ID: 4, Title: "Today", DueDate: "2026-09-19"},
			{ID: 5, Title: "Wednesday first in column", DueDate: "2026-09-23", Position: 1},
			{ID: 6, Title: "Friday", DueDate: "2026-09-25"},
			{ID: 7, Title: "No date"},
		},
	})
	var got []string
	for _, c := range b.MustDo {
		got = append(got, c.Title+"|"+c.Stamp)
	}
	want := []string{
		"Overdue|OVERDUE",
		"Today|TODAY",
		"Wednesday first in column|WED 23 SEP",
		"Wednesday|WED 23 SEP",
		"Friday|FRI 25 SEP",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MustDo = %v\nwant     %v", got, want)
	}
	if !b.MustDo[0].Overdue || b.MustDo[1].Overdue {
		t.Errorf("Overdue flags wrong: %+v", b.MustDo[:2])
	}
}

// SPEC gate 3.04: on a Wednesday the same rule applies to the coming
// Saturday's Friday.
func TestMustBeDoneOnAWednesdayUsesTheComingSaturdayFriday(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-16"),
		Tasks: []Task{
			{ID: 1, Title: "Fri 25", DueDate: "2026-09-25"},
			{ID: 2, Title: "Sat 26", DueDate: "2026-09-26"},
		},
	})
	if len(b.MustDo) != 1 || b.MustDo[0].Title != "Fri 25" || b.MustDo[0].Stamp != "FRI 25 SEP" {
		t.Errorf("MustDo = %+v, want just the Fri 25 task", b.MustDo)
	}
	if b.IsToday {
		t.Error("IsToday true on a Wednesday")
	}
}

// SPEC gate 3.05: tasks assigned to the current person are YOURS.
func TestYoursMarksOnlyTheCurrentPersonsTasks(t *testing.T) {
	b := Build(Input{
		Today:    day(t, "2026-09-19"),
		PersonID: 2,
		Tasks: []Task{
			{ID: 1, Title: "Mine", DueDate: "2026-09-20", People: []Person{{ID: 1, Name: "Alex"}, {ID: 2, Name: "Kim"}}},
			{ID: 2, Title: "Theirs", DueDate: "2026-09-21", People: []Person{{ID: 1, Name: "Alex"}}},
			{ID: 3, Title: "Nobody", DueDate: "2026-09-22"},
		},
	})
	if !b.MustDo[0].Yours || b.MustDo[1].Yours || b.MustDo[2].Yours {
		t.Errorf("Yours = %v %v %v, want true false false", b.MustDo[0].Yours, b.MustDo[1].Yours, b.MustDo[2].Yours)
	}
}

// SPEC gate 3.06: This week lists events from today to Friday, in date
// order, with notes.
func TestThisWeekRules(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-19"),
		ThisWeek: []Occurrence{
			{EventID: 3, Title: "Later", StartDate: "2026-09-25", StartTime: "10:00", EndTime: "11:30"},
			{EventID: 1, Title: "Exams", Notes: "Print exam papers today", StartDate: "2026-09-21", OriginalDate: "2026-09-21"},
			{EventID: 4, Title: "After Friday", StartDate: "2026-09-26"},
			{EventID: 5, Title: "Yesterday", StartDate: "2026-09-18"},
			{EventID: 2, Title: "Today's", StartDate: "2026-09-19"},
		},
	})
	var got []string
	for _, c := range b.ThisWeek {
		got = append(got, c.Title+"|"+c.Stamp)
	}
	want := []string{"Today's|TODAY", "Exams|MON 21 SEP", "Later|FRI 25 SEP"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ThisWeek = %v, want %v", got, want)
	}
	if b.ThisWeek[1].Notes != "Print exam papers today" {
		t.Errorf("Exams notes = %q", b.ThisWeek[1].Notes)
	}
	if b.ThisWeek[2].TimeLabel != "10:00–11:30" {
		t.Errorf("time label = %q, want 10:00–11:30", b.ThisWeek[2].TimeLabel)
	}
}

// SPEC gate 3.09: a multi-day event already under way shows "until date".
func TestMultiDayEventAlreadyStartedShowsUntil(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-19"),
		ThisWeek: []Occurrence{
			{EventID: 1, Title: "Camp", StartDate: "2026-09-17", EndDate: "2026-09-22"},
			{EventID: 2, Title: "Starts Monday", StartDate: "2026-09-21", EndDate: "2026-09-23"},
		},
	})
	if len(b.ThisWeek) != 2 {
		t.Fatalf("ThisWeek = %+v, want 2 cards", b.ThisWeek)
	}
	if got := b.ThisWeek[0]; got.Title != "Camp" || got.UntilLabel != "until Tue 22 Sep" || got.Stamp != "TODAY" {
		t.Errorf("Camp card = %+v, want stamp TODAY and 'until Tue 22 Sep'", got)
	}
	if got := b.ThisWeek[1].UntilLabel; got != "" {
		t.Errorf("not-yet-started event has UntilLabel %q, want none", got)
	}
}

// An event that ended before today isn't in This week even if a source
// returned it.
func TestThisWeekDropsEventsThatAlreadyEnded(t *testing.T) {
	b := Build(Input{
		Today:    day(t, "2026-09-19"),
		ThisWeek: []Occurrence{{EventID: 1, Title: "Over", StartDate: "2026-09-14", EndDate: "2026-09-18"}},
	})
	if len(b.ThisWeek) != 0 {
		t.Errorf("ThisWeek = %+v, want none", b.ThisWeek)
	}
}

// SPEC gate 3.07: a card campaign on Sat 31 Oct with 42 days' notice is
// "in 6 weeks" on 19 Sep and absent on 12 Sep.
func TestComingUpCardCampaign(t *testing.T) {
	campaign := Occurrence{EventID: 1, Title: "Card campaign", StartDate: "2026-10-31", NoticeDays: 42}

	b := Build(Input{Today: day(t, "2026-09-19"), Upcoming: []Occurrence{campaign}})
	if len(b.ComingUp) != 1 || b.ComingUp[0].Stamp != "IN 6 WEEKS" || b.ComingUp[0].DateLabel != "Sat 31 Oct 2026" {
		t.Errorf("for 19 Sep ComingUp = %+v, want one card stamped IN 6 WEEKS", b.ComingUp)
	}

	b = Build(Input{Today: day(t, "2026-09-12"), Upcoming: []Occurrence{campaign}})
	if len(b.ComingUp) != 0 {
		t.Errorf("for 12 Sep ComingUp = %+v, want none", b.ComingUp)
	}

	// Counted from Saturday, so a Wednesday view gives the same label.
	b = Build(Input{Today: day(t, "2026-09-16"), Upcoming: []Occurrence{campaign}})
	if len(b.ComingUp) != 1 || b.ComingUp[0].Stamp != "IN 6 WEEKS" {
		t.Errorf("for Wed 16 Sep ComingUp = %+v, want IN 6 WEEKS", b.ComingUp)
	}
}

func TestComingUpLabelsDaysWhenNotWholeWeeks(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-19"),
		Upcoming: []Occurrence{
			{EventID: 1, Title: "Ten days", StartDate: "2026-09-29", NoticeDays: 14},
			{EventID: 2, Title: "One week", StartDate: "2026-09-26", NoticeDays: 14},
		},
	})
	got := map[string]string{}
	for _, c := range b.ComingUp {
		got[c.Title] = c.Stamp
	}
	want := map[string]string{"Ten days": "IN 10 DAYS", "One week": "IN 1 WEEK"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("stamps = %v, want %v", got, want)
	}
}

// Coming up excludes events already in This week's window, and events
// with no show-ahead time.
func TestComingUpExcludesThisWeekAndNoNotice(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-19"),
		Upcoming: []Occurrence{
			{EventID: 1, Title: "Inside the week", StartDate: "2026-09-25", NoticeDays: 30},
			{EventID: 2, Title: "No notice", StartDate: "2026-10-05"},
		},
	})
	if len(b.ComingUp) != 0 {
		t.Errorf("ComingUp = %+v, want none", b.ComingUp)
	}
}

// A repeating series shows only its next qualifying occurrence.
func TestComingUpShowsOneOccurrencePerSeries(t *testing.T) {
	b := Build(Input{
		Today: day(t, "2026-09-19"),
		Upcoming: []Occurrence{
			{EventID: 1, Title: "Meeting", StartDate: "2026-10-09", NoticeDays: 30},
			{EventID: 1, Title: "Meeting", StartDate: "2026-10-02", NoticeDays: 30},
			{EventID: 1, Title: "Meeting", StartDate: "2026-10-16", NoticeDays: 30},
		},
	})
	if len(b.ComingUp) != 1 || b.ComingUp[0].DateLabel != "Fri 2 Oct 2026" {
		t.Errorf("ComingUp = %+v, want just Fri 2 Oct", b.ComingUp)
	}
}

func TestEmptyInputGivesEmptySections(t *testing.T) {
	b := Build(Input{Today: day(t, "2026-09-19")})
	if len(b.MustDo)+len(b.ThisWeek)+len(b.ComingUp) != 0 {
		t.Errorf("Build(empty) = %+v, want empty sections", b)
	}
}

// SPEC gate 3.05: Just mine keeps only the current person's tasks, and
// leaves events (which belong to nobody) alone.
func TestJustMineKeepsOnlyYoursAndAllEvents(t *testing.T) {
	b := Build(Input{
		Today:    day(t, "2026-09-19"),
		PersonID: 2,
		Tasks: []Task{
			{ID: 1, Title: "Mine", DueDate: "2026-09-20", People: []Person{{ID: 2, Name: "Kim"}}},
			{ID: 2, Title: "Theirs", DueDate: "2026-09-21", People: []Person{{ID: 1, Name: "Alex"}}},
		},
		ThisWeek: []Occurrence{{EventID: 1, Title: "Exams", StartDate: "2026-09-21"}},
	}).JustMine()
	if len(b.MustDo) != 1 || b.MustDo[0].Title != "Mine" {
		t.Errorf("MustDo = %+v, want just Mine", b.MustDo)
	}
	if len(b.ThisWeek) != 1 {
		t.Errorf("ThisWeek = %+v, want the event kept", b.ThisWeek)
	}
}
