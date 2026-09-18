package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/stas-comp/comphq/internal/people"
)

// ListForTeamView returns every unremoved task in To do or In progress
// (SPEC B4's Team view: "Ideas and Done are not shown"), for BuildLanes
// to group. Ordered by stage then position so a lane's Working on
// now/Up next tasks come out already in shared-priority order.
func (s *Store) ListForTeamView(ctx context.Context) ([]Task, error) {
	return s.listTasks(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL AND stage IN ('doing', 'todo')
		ORDER BY stage, position
	`)
}

// sizeWeight is each size's contribution to workload (SPEC B4: "load is
// the sum of weights S=1, M=2, L=4").
var sizeWeight = map[string]int{"S": 1, "M": 2, "L": 4}

// LaneTask is one card within a lane's Working on now or Up next group.
// Number is Up next's 1-based priority number; 0 for Working on now,
// which isn't numbered (SPEC B4).
type LaneTask struct {
	ID        int64
	Title     string
	SizeLabel string
	Number    int
}

// WorkloadGroup is one task's share of a lane's workload, kept apart
// from every other task's rather than one flat row of blocks (SPEC B4:
// "drawn as blocks grouped per task"). Blocks has no meaningful values,
// only a length (1, 2 or 4) — a template ranges over it purely to
// repeat one block that many times.
type WorkloadGroup struct {
	TaskID int64
	Blocks []int
}

// Lane is one Team view column: Unassigned (PersonID 0, no workload row
// — SPEC gate 2.27 is specifically "each person's column") or one
// active person, in the order BuildLanes returns them (SPEC B4:
// Unassigned first, then alphabetical).
type Lane struct {
	PersonID     int64
	PersonName   string
	WorkingOnNow []LaneTask
	UpNext       []LaneTask
	Workload     []WorkloadGroup
	WorkloadLine string
}

// BuildLanes groups tasks — already filtered by the caller to
// non-removed To do/In progress; Ideas and Done never appear here
// (SPEC B4) — into an Unassigned lane followed by one lane per active
// person. A task with more than one assignee appears in every one of
// their lanes and counts toward each of their workloads (gate 2.30). A
// pure function: no DB access, so it's exactly as reusable by My jobs
// (P2-10, one person's lane plus an Unassigned-style "Up for grabs"
// list) as it is here.
func BuildLanes(tasksList []Task, peopleList []people.Person) []Lane {
	sorted := append([]people.Person(nil), peopleList...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	lanes := make([]Lane, 0, len(sorted)+1)
	lanes = append(lanes, buildLane(0, "Unassigned", tasksList, false))
	for _, p := range sorted {
		lanes = append(lanes, buildLane(p.ID, p.Name, tasksList, true))
	}
	return lanes
}

func buildLane(personID int64, name string, tasksList []Task, showWorkload bool) Lane {
	lane := Lane{PersonID: personID, PersonName: name}

	var laneTasks []Task
	for _, t := range tasksList {
		if belongsToLane(t, personID) {
			laneTasks = append(laneTasks, t)
		}
	}

	number := 0
	for _, t := range laneTasks {
		switch t.Stage {
		case StageDoing:
			lane.WorkingOnNow = append(lane.WorkingOnNow, LaneTask{ID: t.ID, Title: t.Title, SizeLabel: sizeLabels[t.Size]})
		case StageTodo:
			number++
			lane.UpNext = append(lane.UpNext, LaneTask{ID: t.ID, Title: t.Title, SizeLabel: sizeLabels[t.Size], Number: number})
		}
	}

	if showWorkload {
		lane.Workload, lane.WorkloadLine = workloadFor(laneTasks)
	}
	return lane
}

// belongsToLane reports whether a task shows in personID's lane (0 for
// Unassigned, which holds every task with no assignees at all).
func belongsToLane(t Task, personID int64) bool {
	if personID == 0 {
		return len(t.Assignees) == 0
	}
	for _, a := range t.Assignees {
		if a.PersonID == personID {
			return true
		}
	}
	return false
}

func workloadFor(tasksList []Task) ([]WorkloadGroup, string) {
	var groups []WorkloadGroup
	counts := map[string]int{}
	for _, t := range tasksList {
		groups = append(groups, WorkloadGroup{TaskID: t.ID, Blocks: make([]int, sizeWeight[t.Size])})
		counts[t.Size]++
	}
	return groups, workloadLine(counts)
}

// workloadLine renders SPEC B4's example exactly: "1 large · 1 medium ·
// 1 small", largest first, omitting any size with nothing on the board.
func workloadLine(counts map[string]int) string {
	var parts []string
	for _, size := range []string{"L", "M", "S"} {
		if n := counts[size]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, sizeWordLower(size)))
		}
	}
	return strings.Join(parts, " · ")
}

func sizeWordLower(size string) string {
	return strings.ToLower(sizeLabels[size])
}
