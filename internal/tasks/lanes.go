package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/stas-comp/comphq/internal/people"
)

// ListForTeamView returns every unremoved task in To do, In progress or
// Ideas (Done never appears), for BuildLanes to group. Ideas are here so
// an idea somebody is on shows in their lane (SPEC gates 4.48, 4.49); they
// get their own group and never count towards workload. Ordered by stage
// then position so a lane's Working on now/Up next tasks come out already
// in shared-priority order.
func (s *Store) ListForTeamView(ctx context.Context) ([]Task, error) {
	return s.listTasks(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL AND stage IN ('doing', 'todo', 'idea')
		ORDER BY stage, position
	`)
}

// ListUpForGrabs returns every unremoved, unassigned task nobody is on
// (SPEC B4's My jobs "Up for grabs"): In progress and To do first, in
// priority order, then Ideas. An idea somebody is on is not here — it
// shows in that person's lanes instead (gates 4.48, 4.49). Done tasks never appear
// here (SPEC gate 2.33: "finished jobs don't appear").
func (s *Store) ListUpForGrabs(ctx context.Context) ([]Task, error) {
	return s.listTasks(ctx, `
		SELECT id, title, notes, size, stage, position, due_date, done_at
		FROM tasks
		WHERE removed_at IS NULL
		  AND stage IN ('doing', 'todo', 'idea')
		  AND id NOT IN (SELECT DISTINCT task_id FROM task_assignees)
		ORDER BY
			CASE stage WHEN 'doing' THEN 0 WHEN 'todo' THEN 1 WHEN 'idea' THEN 2 END,
			position
	`)
}

// LaneForPerson builds one person's own lane exactly the way BuildLanes
// does (SPEC B4: My jobs' left side is "the current person's lane,
// built by the same code as their Team lane").
func LaneForPerson(tasksList []Task, person people.Person) Lane {
	return BuildLanes(tasksList, []people.Person{person})[1] // [0] is always Unassigned
}

// sizeWeight is each size's contribution to workload (SPEC B4: "load is
// the sum of weights S=1, M=2, L=4").
var sizeWeight = map[string]int{"S": 1, "M": 2, "L": 4}

// LaneTask is one card within a lane's Working on now, Up next or Ideas group.
// Number is Up next's 1-based priority number; 0 for Working on now,
// which isn't numbered (SPEC B4). PrevID/NextID are the neighbouring
// task's id within this lane's own Up next order (0 if there isn't
// one), the same "computed once per render" pattern the Board's own
// cardView uses for its Move up/down buttons — reordering still posts
// to the shared move endpoint (SPEC B4: "before_id/after_id is the
// neighbouring card in that lane"), just with a lane-scoped neighbour
// instead of a board-column-scoped one.
type LaneTask struct {
	ID        int64
	Title     string
	SizeLabel string
	Number    int
	PrevID    int64
	NextID    int64
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
	Ideas        []LaneTask
	Workload     []WorkloadGroup
	WorkloadLine string
	// AssignOptions is every active person this lane's tasks can be
	// assigned to by drag or "Assign to…" (SPEC gate 2.28) — every
	// active person except this lane's own (reassigning to the person
	// who already has it is meaningless); Unassigned excludes no one.
	AssignOptions []personOption
}

// BuildLanes groups tasks — already filtered by the caller to
// non-removed To do/In progress/Ideas; Done never appears here — into an
// Unassigned lane followed by one lane per active person. Ideas go in
// their own group in a person's lane; an unassigned idea is left out of
// the Unassigned lane (it lives in My jobs' Up for grabs, gate 4.50). A
// task with more than one assignee appears in every one of their lanes and
// counts toward each of their workloads (gate 2.30). A pure function: no DB access, so it's exactly as reusable by My jobs
// (P2-10, one person's lane plus an Unassigned-style "Up for grabs"
// list) as it is here.
func BuildLanes(tasksList []Task, peopleList []people.Person) []Lane {
	sorted := append([]people.Person(nil), peopleList...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	lanes := make([]Lane, 0, len(sorted)+1)
	lanes = append(lanes, buildLane(0, "Unassigned", tasksList, sorted, false))
	for _, p := range sorted {
		lanes = append(lanes, buildLane(p.ID, p.Name, tasksList, sorted, true))
	}
	return lanes
}

func buildLane(personID int64, name string, tasksList []Task, everyone []people.Person, showWorkload bool) Lane {
	lane := Lane{PersonID: personID, PersonName: name}
	for _, p := range everyone {
		if p.ID != personID {
			lane.AssignOptions = append(lane.AssignOptions, personOption{ID: p.ID, Name: p.Name})
		}
	}

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
		case StageIdea:
			if personID != 0 {
				lane.Ideas = append(lane.Ideas, LaneTask{ID: t.ID, Title: t.Title, SizeLabel: sizeLabels[t.Size]})
			}
		}
	}
	for i := range lane.UpNext {
		if i > 0 {
			lane.UpNext[i].PrevID = lane.UpNext[i-1].ID
		}
		if i < len(lane.UpNext)-1 {
			lane.UpNext[i].NextID = lane.UpNext[i+1].ID
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
		if t.Stage != StageTodo && t.Stage != StageDoing {
			continue // an idea isn't work yet (SPEC gate 4.48)
		}
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
