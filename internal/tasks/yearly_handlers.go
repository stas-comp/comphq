package tasks

import (
	"net/http"
	"strconv"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
)

// The Yearly jobs page (SPEC B14.4, gate 8.43): every yearly job, waiting or
// not, in order of due date, so the year ahead can be seen in one place. It is a
// Tasks page, so its top search box searches jobs, and a job opens in the window.

type yearlyLine struct {
	ID          int64
	Title       string
	DueLabel    string
	NoticeWeeks int
	Status      string // "Appears Sat 24 Oct" or "In To do now"
	Waiting     bool
	Assignees   []assigneeView
}

type yearlyPageData struct {
	Jobs []yearlyLine
	// Saved is the message after saving a yearly job that is still waiting: when it
	// will appear in To do (gate 8.42).
	Saved string
	// WindowClose is the task window's close icon.
	WindowClose app.IconButton
}

// handleYearly serves GET /tasks/yearly. ?saved=ID shows the plain message that
// saving a waiting yearly job leaves, since that job isn't on the Board.
func (h *Handlers) handleYearly(w http.ResponseWriter, r *http.Request) {
	today := app.Today(h.srv.TestMode)
	jobs, err := h.tasks.ListYearly(r.Context(), today)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	me := currentPersonID(r)
	lines := make([]yearlyLine, 0, len(jobs))
	for _, j := range jobs {
		line := yearlyLine{ID: j.ID, Title: j.Title, NoticeWeeks: j.NoticeWeeks, Waiting: j.Waiting}
		if d, err := time.Parse(dateLayout, j.DueDate); err == nil {
			line.DueLabel = format.Date(d)
		}
		if j.Waiting {
			if d, err := time.Parse(dateLayout, j.ShowsFrom); err == nil {
				line.Status = "Appears " + d.Format("Mon 2 Jan")
			}
		} else {
			line.Status = "In " + stageLabels[j.Stage] + " now"
		}
		for _, a := range j.Assignees {
			line.Assignees = append(line.Assignees, assigneeView{
				PersonID: a.PersonID, Name: a.Name, Initials: InitialsFor(a.Name), ColorClass: AvatarClass(a.PersonID, me), Removed: a.Removed,
			})
		}
		lines = append(lines, line)
	}
	data := yearlyPageData{Jobs: lines, WindowClose: app.NewIconButton(app.IconClose, "", false)}
	if id, err := strconv.ParseInt(r.URL.Query().Get("saved"), 10, 64); err == nil {
		if notice, err := h.tasks.SavedNotice(r.Context(), id, today); err == nil {
			data.Saved = notice
		}
	}
	h.srv.RenderFrame(w, r, http.StatusOK, "tasks-yearly.html", "Yearly jobs", data)
}
