package calendar

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/people"
)

const dateLayout = "2006-01-02"
const monthLayout = "2006-01"

// recurrenceOption is one entry in the repeats select (SPEC gate 2.13:
// "Never / Weekly / Monthly / Yearly").
type recurrenceOption struct {
	Value string
	Label string
}

func recurrenceOptions() []recurrenceOption {
	return []recurrenceOption{
		{Value: RecurrenceNone, Label: "Never"},
		{Value: RecurrenceWeekly, Label: "Weekly"},
		{Value: RecurrenceMonthly, Label: "Monthly"},
		{Value: RecurrenceYearly, Label: "Yearly"},
	}
}

// noticeAmountUnit splits notice_days back into the form's own
// amount+unit pair (SPEC B3: "weeks stored as days x7, displayed in
// weeks when divisible").
func noticeAmountUnit(noticeDays int) (int, string) {
	if noticeDays == 0 {
		return 0, "weeks"
	}
	if noticeDays%7 == 0 {
		return noticeDays / 7, "weeks"
	}
	return noticeDays, "days"
}

// noticeDaysFrom combines the form's amount+unit pair back into
// notice_days (0 = "not early", SPEC gate 2.13).
func noticeDaysFrom(amount int, unit string) int {
	if amount <= 0 {
		return 0
	}
	if unit == "weeks" {
		return amount * 7
	}
	return amount
}

// eventFormData backs both the Add event page and the event details
// page's own editable form — the two are close enough (SPEC gate
// 2.13's field list) that keeping one shape avoids maintaining the
// same field list twice.
type eventFormData struct {
	CurrentView   string
	Message       string
	FormAction    string
	EventID       int64 // 0 for a new, not-yet-created event
	Title         string
	Notes         string
	StartDate     string
	EndDate       string
	StartTime     string
	EndTime       string
	Recurrence    string
	UntilDate     string
	NoticeAmount  int
	NoticeUnit    string
	Recurrences   []recurrenceOption
	UpdatedByName string
	UpdatedAt     string
}

func inputFromForm(r *http.Request) Input {
	amount, _ := strconv.Atoi(r.FormValue("notice_amount"))
	return Input{
		Title:      r.FormValue("title"),
		Notes:      r.FormValue("notes"),
		StartDate:  r.FormValue("start_date"),
		EndDate:    r.FormValue("end_date"),
		StartTime:  r.FormValue("start_time"),
		EndTime:    r.FormValue("end_time"),
		Recurrence: r.FormValue("recurrence"),
		UntilDate:  r.FormValue("until_date"),
		NoticeDays: noticeDaysFrom(amount, r.FormValue("notice_unit")),
	}
}

// friendlyMessage renders a validation error the way the Board's own
// add-task refusal does — a plain sentence, not the internal error text.
func friendlyMessage(err error) string {
	switch err {
	case ErrEmptyTitle:
		return "Please type a title."
	case ErrEndDateBeforeStart:
		return "The end date can't be before the start date."
	case ErrEndTimeNeedsStartTime:
		return "Add a start time before an end time."
	default:
		return err.Error()
	}
}

func (h *Handlers) renderNewEventForm(w http.ResponseWriter, r *http.Request, status int, input Input, message string) {
	amount, unit := noticeAmountUnit(input.NoticeDays)
	h.srv.RenderFrame(w, r, status, "calendar-new.html", "Add event", eventFormData{
		Message:      message,
		FormAction:   "/calendar/events",
		Title:        input.Title,
		Notes:        input.Notes,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
		Recurrence:   input.Recurrence,
		UntilDate:    input.UntilDate,
		NoticeAmount: amount,
		NoticeUnit:   unit,
		Recurrences:  recurrenceOptions(),
	})
}

// handleNewEventForm serves gate 2.13's add-event form. A date carried
// over from clicking a day on the month grid pre-fills the start date
// (SPEC A7: "Clicking a day starts a new event on that date").
func (h *Handlers) handleNewEventForm(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = app.Today(h.srv.TestMode).Format(dateLayout)
	}
	h.renderNewEventForm(w, r, http.StatusOK, Input{StartDate: date, NoticeDays: 7}, "")
}

// handleCreateEvent serves gate 2.13's add-event submission.
func (h *Handlers) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	input := inputFromForm(r)
	today := app.Today(h.srv.TestMode)
	event, err := h.store.Create(r.Context(), input, person.ID, today)
	switch err {
	case nil:
		http.Redirect(w, r, fmt.Sprintf("/calendar/events/%d", event.ID), http.StatusFound)
	case ErrEmptyTitle, ErrEndDateBeforeStart, ErrEndTimeNeedsStartTime:
		h.renderNewEventForm(w, r, http.StatusOK, input, friendlyMessage(err))
	case ErrInvalidRecurrence:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderEventDetails(w http.ResponseWriter, r *http.Request, status int, event Event, input Input, message string) {
	amount, unit := noticeAmountUnit(input.NoticeDays)
	h.srv.RenderFrame(w, r, status, "calendar-details.html", event.Title, eventFormData{
		Message:       message,
		FormAction:    fmt.Sprintf("/calendar/events/%d", event.ID),
		EventID:       event.ID,
		Title:         input.Title,
		Notes:         input.Notes,
		StartDate:     input.StartDate,
		EndDate:       input.EndDate,
		StartTime:     input.StartTime,
		EndTime:       input.EndTime,
		Recurrence:    input.Recurrence,
		UntilDate:     input.UntilDate,
		NoticeAmount:  amount,
		NoticeUnit:    unit,
		Recurrences:   recurrenceOptions(),
		UpdatedByName: event.UpdatedByName,
		UpdatedAt:     event.UpdatedAt,
	})
}

// inputFromEvent lets the details page redisplay exactly what's
// stored, the same shape POST /calendar/events/{id} itself takes.
func inputFromEvent(event Event) Input {
	return Input{
		Title: event.Title, Notes: event.Notes, StartDate: event.StartDate, EndDate: event.EndDate,
		StartTime: event.StartTime, EndTime: event.EndTime, Recurrence: event.Recurrence,
		UntilDate: event.UntilDate, NoticeDays: event.NoticeDays,
	}
}

// handleEventDetails serves gate 2.20's details page: every field
// redisplayed in an editable form, plus who last changed it and when.
func (h *Handlers) handleEventDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	event, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.renderEventDetails(w, r, http.StatusOK, event, inputFromEvent(event), "")
}

// handleUpdateEvent serves the details page's own edit submission
// (SPEC B4: "Change all edits the series row").
func (h *Handlers) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	person, ok := people.FromContext(r.Context())
	if !ok {
		http.Error(w, "not signed in", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	input := inputFromForm(r)
	today := app.Today(h.srv.TestMode)
	switch err := h.store.Update(r.Context(), id, input, person.ID, today); err {
	case nil:
		http.Redirect(w, r, fmt.Sprintf("/calendar/events/%d", id), http.StatusFound)
	case ErrEmptyTitle, ErrEndDateBeforeStart, ErrEndTimeNeedsStartTime:
		event, getErr := h.store.Get(r.Context(), id)
		if getErr != nil {
			http.NotFound(w, r)
			return
		}
		h.renderEventDetails(w, r, http.StatusOK, event, input, friendlyMessage(err))
	case ErrEventNotFound:
		http.NotFound(w, r)
	case ErrInvalidRecurrence:
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// occurrenceView is one chip on the month grid or row on the list view.
type occurrenceView struct {
	EventID   int64
	Title     string
	TimeLabel string // "" for all-day
}

func viewsFor(occs []CalendarOccurrence) []occurrenceView {
	views := make([]occurrenceView, 0, len(occs))
	for _, o := range occs {
		views = append(views, occurrenceView{EventID: o.EventID, Title: o.Title, TimeLabel: o.StartTime})
	}
	return views
}

type dayCell struct {
	Date        string
	Day         int
	InMonth     bool
	IsToday     bool
	IsSaturday  bool
	Occurrences []occurrenceView
}

type monthPageData struct {
	CurrentView string
	MonthLabel  string
	PrevMonth   string
	NextMonth   string
	TodayMonth  string
	Weeks       [][]dayCell
}

// mondayOnOrBefore/sundayOnOrAfter build the Monday-first grid (SPEC
// gate 2.12), which can spill into the adjacent month at either end.
func mondayOnOrBefore(d time.Time) time.Time {
	offset := (int(d.Weekday()) + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func sundayOnOrAfter(d time.Time) time.Time {
	offset := (7 - int(d.Weekday())) % 7
	return d.AddDate(0, 0, offset)
}

// handleMonth serves gate 2.12's month view.
func (h *Handlers) handleMonth(w http.ResponseWriter, r *http.Request) {
	today := app.Today(h.srv.TestMode)
	year, month := today.Year(), today.Month()

	if v := r.URL.Query().Get("month"); v != "" {
		if t, err := time.Parse(monthLayout, v); err == nil {
			year, month = t.Year(), t.Month()
		}
	}

	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	gridStart := mondayOnOrBefore(firstOfMonth)
	gridEnd := sundayOnOrAfter(lastOfMonth)

	occs, err := h.store.Occurrences(r.Context(), gridStart, gridEnd)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	byDate := map[string][]CalendarOccurrence{}
	for _, occ := range occs {
		start, err1 := time.Parse(dateLayout, occ.StartDate)
		end, err2 := time.Parse(dateLayout, occ.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			byDate[d.Format(dateLayout)] = append(byDate[d.Format(dateLayout)], occ)
		}
	}

	todayStr := today.Format(dateLayout)
	var weeks [][]dayCell
	for cursor := gridStart; !cursor.After(gridEnd); {
		week := make([]dayCell, 0, 7)
		for i := 0; i < 7; i++ {
			dateStr := cursor.Format(dateLayout)
			week = append(week, dayCell{
				Date:        dateStr,
				Day:         cursor.Day(),
				InMonth:     cursor.Month() == month && cursor.Year() == year,
				IsToday:     dateStr == todayStr,
				IsSaturday:  cursor.Weekday() == time.Saturday,
				Occurrences: viewsFor(byDate[dateStr]),
			})
			cursor = cursor.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	h.srv.RenderFrame(w, r, http.StatusOK, "calendar-month.html", "Calendar", monthPageData{
		CurrentView: "month",
		MonthLabel:  firstOfMonth.Format("January 2006"),
		PrevMonth:   firstOfMonth.AddDate(0, -1, 0).Format(monthLayout),
		NextMonth:   firstOfMonth.AddDate(0, 1, 0).Format(monthLayout),
		TodayMonth:  today.Format(monthLayout),
		Weeks:       weeks,
	})
}

// listRowView is one row of the 12-week list view.
type listRowView struct {
	EventID   int64
	DateLabel string
	Title     string
	Notes     string
	TimeLabel string
}

type listPageData struct {
	CurrentView string
	Rows        []listRowView
}

func dateLabel(s string) string {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return s
	}
	return format.Date(t)
}

// handleList serves gate 2.12's List view: the next 12 weeks, in date order.
func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	today := app.Today(h.srv.TestMode)
	to := today.AddDate(0, 0, 12*7-1)

	occs, err := h.store.Occurrences(r.Context(), today, to)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	rows := make([]listRowView, 0, len(occs))
	for _, o := range occs {
		rows = append(rows, listRowView{
			EventID:   o.EventID,
			DateLabel: dateLabel(o.StartDate),
			Title:     o.Title,
			Notes:     o.Notes,
			TimeLabel: o.StartTime,
		})
	}

	h.srv.RenderFrame(w, r, http.StatusOK, "calendar-list.html", "Calendar", listPageData{
		CurrentView: "list",
		Rows:        rows,
	})
}
