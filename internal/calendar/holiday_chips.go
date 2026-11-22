package calendar

import (
	"time"

	"github.com/stas-comp/comphq/internal/calendar/holidays"
)

// HolidaysBetween is the US federal holidays (and the days observed for them) from
// from to to, both included (SPEC B14.5, gates 8.60-8.65). The month grid and the
// 12-week list draw them, and cmd/comphq hands the same answer to the Briefing
// through its own small interface (SPEC B2). They are worked out, never stored.
func HolidaysBetween(from, to time.Time) []holidays.Day {
	return holidays.Days(from, to)
}

// holidayChip is a holiday as the month grid draws it (gates 8.61-8.64): not a
// link and not a button, so it can't open the event editor, but focusable, with
// the sentence for a no-mail or a mail-as-normal day as its tooltip and as hidden
// text for a screen reader.
func holidayChip(d holidays.Day) chipView {
	return chipView{
		Kind:        "holiday",
		Title:       d.Label(),
		Explanation: d.Explanation(),
		MailKind:    string(d.Kind),
	}
}

// holidayRow is a holiday as a row of the 12-week list.
func holidayRow(d holidays.Day) listRowView {
	chip := holidayChip(d)
	return listRowView{Holiday: &chip, DateLabel: dateLabel(d.Date.Format(dateLayout)), Title: d.Label()}
}
