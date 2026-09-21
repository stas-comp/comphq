package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/stas-comp/comphq/internal/app"
	"github.com/stas-comp/comphq/internal/app/format"
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
	// DueDate is the ISO due date ("" for none); DueLabel is the same as a
	// card shows it ("Wed 23 Sep"), filled in by Lane.Decorate. AlsoOn names
	// the other people on the job, for "Also on it: Sam" (gate 4.31).
	DueDate  string
	DueLabel string
	AlsoOn   string
	// StepsTotal and StepsDone are the job's live steps, for the small 3/7.
	StepsTotal int
	StepsDone  int
}

// MoveUp and MoveDown are the two icon-only move buttons of an Up next
// card (gate 4.11), disabled at the top and bottom of the lane's own order.
func (t LaneTask) MoveUp() app.IconButton {
	return app.NewIconButton(app.IconMoveUp, t.Title, t.PrevID == 0)
}

func (t LaneTask) MoveDown() app.IconButton {
	return app.NewIconButton(app.IconMoveDown, t.Title, t.NextID == 0)
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
	// Initials and AvatarClass draw the person's circle in the lane head;
	// IsMe marks the lane of the person using the app (gate 4.31). All three
	// are filled in by Decorate.
	Initials     string
	AvatarClass  string
	IsMe         bool
	Workload     []WorkloadGroup
	WorkloadLine string
	// WorkloadLabel is the blocks' spoken equivalent (SPEC gate 4.10):
	// "Workload: 7 blocks — large, medium, small".
	WorkloadLabel string
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

// JobCount is how many jobs the lane holds (Working on now and Up next),
// for the Unassigned lane's "3 jobs waiting for someone".
func (l Lane) JobCount() int {
	return len(l.WorkingOnNow) + len(l.UpNext)
}

// Decorate fills in what a lane needs to be drawn and BuildLanes, being a
// pure grouping, doesn't know: the person's circle, whether this lane is the
// person using the app (meID), and every card's short due date.
func (l *Lane) Decorate(meID int64, today time.Time) {
	if l.PersonID != 0 {
		l.Initials = format.Initials(l.PersonName)
		l.AvatarClass = AvatarClass(l.PersonID, meID)
		l.IsMe = meID != 0 && l.PersonID == meID
	}
	for _, group := range [][]LaneTask{l.WorkingOnNow, l.UpNext, l.Ideas} {
		for i := range group {
			group[i].DueLabel = shortDueLabel(group[i].DueDate, today)
		}
	}
}

// shortDueLabel renders an ISO due date the way a card shows it, or "".
func shortDueLabel(iso string, today time.Time) string {
	due, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return format.DateShort(due, today)
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
			lane.WorkingOnNow = append(lane.WorkingOnNow, laneTask(t, personID))
		case StageTodo:
			number++
			lt := laneTask(t, personID)
			lt.Number = number
			lane.UpNext = append(lane.UpNext, lt)
		case StageIdea:
			if personID != 0 {
				lane.Ideas = append(lane.Ideas, laneTask(t, personID))
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
		lane.WorkloadLabel = WorkloadLabel(lane.Workload)
	}
	return lane
}

// laneTask is one job as a lane's card shows it; AlsoOn is everyone on it but
// the lane's own person.
func laneTask(t Task, personID int64) LaneTask {
	var others []string
	for _, a := range t.Assignees {
		if a.PersonID != personID {
			others = append(others, a.Name)
		}
	}
	return LaneTask{
		ID: t.ID, Title: t.Title, SizeLabel: sizeLabels[t.Size], DueDate: t.DueDate, AlsoOn: strings.Join(others, ", "),
		StepsTotal: t.StepsTotal, StepsDone: t.StepsDone,
	}
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

// blocksWord names a task's size from its block count: the same 1/2/4 the
// blocks are drawn with (sizeWeight).
func blocksWord(blocks int) string {
	for size, weight := range sizeWeight {
		if weight == blocks {
			return sizeWordLower(size)
		}
	}
	return ""
}

// WorkloadLabel is what a screen reader says for a row of workload blocks
// (SPEC gate 4.10): the total, then each job's size in the order the jobs
// are drawn — "Workload: 7 blocks — large, medium, small". It is empty
// when there are no blocks, since then nothing is drawn.
func WorkloadLabel(groups []WorkloadGroup) string {
	total := 0
	words := make([]string, 0, len(groups))
	for _, g := range groups {
		total += len(g.Blocks)
		words = append(words, blocksWord(len(g.Blocks)))
	}
	if total == 0 {
		return ""
	}
	noun := "blocks"
	if total == 1 {
		noun = "block"
	}
	return fmt.Sprintf("Workload: %d %s — %s", total, noun, strings.Join(words, ", "))
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
