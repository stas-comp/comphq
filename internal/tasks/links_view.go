package tasks

import (
	"context"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app"
)

// What the Linked jobs section needs to be drawn (SPEC B13.6, gates 7.40,
// 7.41, 7.44, 7.47). One partial, "tasks-links", is drawn by the job's own page
// and by the task window alike, the way the Steps list is.

// linkLine is one line of a group: the other job, and its remove icon. It
// carries the job it is on and whether it is drawn in the window, which the
// line's own forms need (templates here have no way to pass extra arguments).
type linkLine struct {
	LinkView
	TaskID   int64
	InWindow bool
}

// Remove is the line's icon-only button (gate 4.11's remove icon: the count of
// icon-only buttons stays at four, and unlinking is the one that already
// means "take this away").
func (l linkLine) Remove() app.IconButton {
	return app.NewIconButton(app.IconRemove, "link to "+l.Title, false)
}

// linksData is the section.
type linksData struct {
	TaskID   int64
	InWindow bool
	DoFirst  []linkLine
	Then     []linkLine
	Related  []linkLine
	Count    int
	// Query and Matches are the no-script picker (?link_q=): the jobs that
	// match what was typed, each with its three buttons.
	Query   string
	Matches []JobHit
	Notice  string
}

func lines(views []LinkView, taskID int64, inWindow bool) []linkLine {
	out := make([]linkLine, len(views))
	for i, v := range views {
		out[i] = linkLine{LinkView: v, TaskID: taskID, InWindow: inWindow}
	}
	return out
}

// buildLinksData reads a job's links, and, for the no-script picker, the jobs
// matching query (never the job itself, never a removed job, and not one that
// is already linked).
func (h *Handlers) buildLinksData(ctx context.Context, taskID int64, query string, inWindow bool, notice string, today time.Time) (linksData, error) {
	links, err := h.tasks.ListLinks(ctx, taskID)
	if err != nil {
		return linksData{}, err
	}
	data := linksData{
		TaskID: taskID, InWindow: inWindow, Notice: notice, Query: query,
		DoFirst: lines(links.DoFirst, taskID, inWindow), Then: lines(links.Then, taskID, inWindow), Related: lines(links.Related, taskID, inWindow), Count: links.Count(),
	}
	if strings.TrimSpace(query) != "" {
		hits, err := h.tasks.SearchJobs(ctx, query, taskID, today)
		if err != nil {
			return linksData{}, err
		}
		linked := map[int64]bool{}
		for _, group := range [][]LinkView{links.DoFirst, links.Then, links.Related} {
			for _, v := range group {
				linked[v.OtherID] = true
			}
		}
		for _, hit := range hits {
			if !linked[hit.ID] {
				data.Matches = append(data.Matches, hit)
			}
		}
	}
	return data, nil
}

// waitingText is the stamp's tooltip and spoken name: "Waiting on: Order toner".
func waitingText(titles []string) string {
	if len(titles) == 0 {
		return ""
	}
	return "Waiting on: " + strings.Join(titles, ", ")
}

// waitingFor is the WAITING text for each job in ids that is waiting
// (gate 7.42), from one query for the whole list.
func (h *Handlers) waitingFor(ctx context.Context, ids []int64) map[int64]string {
	waiting, err := h.tasks.WaitingOn(ctx, ids)
	if err != nil {
		return nil
	}
	out := make(map[int64]string, len(waiting))
	for id, titles := range waiting {
		out[id] = waitingText(titles)
	}
	return out
}

// ApplyWaiting sets the WAITING text on a lane's cards (Working on now and Up
// next; an idea is not a job anybody is waiting to start).
func (l *Lane) ApplyWaiting(waiting map[int64]string) {
	for _, group := range [][]LaneTask{l.WorkingOnNow, l.UpNext} {
		for i := range group {
			group[i].Waiting = waiting[group[i].ID]
		}
	}
}
