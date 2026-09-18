package tasks

import (
	"context"
	"testing"

	"github.com/stas-comp/comphq/internal/people"
)

func laneFor(t *testing.T, lanes []Lane, personID int64) Lane {
	t.Helper()
	for _, l := range lanes {
		if l.PersonID == personID {
			return l
		}
	}
	t.Fatalf("no lane for person %d", personID)
	return Lane{}
}

func laneTitles(tasksList []LaneTask) []string {
	titles := make([]string, len(tasksList))
	for i, t := range tasksList {
		titles[i] = t.Title
	}
	return titles
}

// SPEC gate 2.24: Unassigned first, then one lane per active person,
// alphabetically.
func TestBuildLanesOrdersUnassignedFirstThenAlphabetical(t *testing.T) {
	peopleList := []people.Person{{ID: 2, Name: "Zoe"}, {ID: 1, Name: "Alex"}}
	lanes := BuildLanes(nil, peopleList)
	if len(lanes) != 3 {
		t.Fatalf("len(lanes) = %d, want 3", len(lanes))
	}
	if lanes[0].PersonID != 0 || lanes[0].PersonName != "Unassigned" {
		t.Errorf("lanes[0] = %+v, want the Unassigned lane first", lanes[0])
	}
	if lanes[1].PersonName != "Alex" || lanes[2].PersonName != "Zoe" {
		t.Errorf("person lane order = [%q, %q], want [Alex, Zoe]", lanes[1].PersonName, lanes[2].PersonName)
	}
}

// SPEC gate 2.25: Unassigned lists To do/In progress jobs nobody is on;
// an assigned task never appears there.
func TestBuildLanesUnassignedOnlyHasTasksWithNoAssignees(t *testing.T) {
	alex := people.Person{ID: 1, Name: "Alex"}
	tasksList := []Task{
		{ID: 1, Title: "Nobody's job", Size: "M", Stage: StageTodo},
		{ID: 2, Title: "Alex's job", Size: "M", Stage: StageTodo, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
	}
	lanes := BuildLanes(tasksList, []people.Person{alex})

	unassigned := laneFor(t, lanes, 0)
	if got := laneTitles(unassigned.UpNext); len(got) != 1 || got[0] != "Nobody's job" {
		t.Errorf("Unassigned Up next = %v, want just [Nobody's job]", got)
	}
}

// SPEC gate 2.25: each lane groups its tasks into Working on now
// (doing) and numbered Up next (todo); gate 2.30: a shared task shows
// in every one of its assignees' lanes.
func TestBuildLanesGroupsAndSharesAcrossLanes(t *testing.T) {
	alex := people.Person{ID: 1, Name: "Alex"}
	jo := people.Person{ID: 2, Name: "Jo"}
	tasksList := []Task{
		{ID: 1, Title: "In progress", Size: "S", Stage: StageDoing, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
		{ID: 2, Title: "First up", Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
		{ID: 3, Title: "Second up", Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
		{
			ID: 4, Title: "Shared job", Size: "S", Stage: StageTodo,
			Assignees: []Assignee{{PersonID: 1, Name: "Alex"}, {PersonID: 2, Name: "Jo"}},
		},
	}
	lanes := BuildLanes(tasksList, []people.Person{alex, jo})

	alexLane := laneFor(t, lanes, 1)
	if got := laneTitles(alexLane.WorkingOnNow); len(got) != 1 || got[0] != "In progress" {
		t.Errorf("Alex's Working on now = %v, want [In progress]", got)
	}
	upNext := alexLane.UpNext
	if len(upNext) != 3 {
		t.Fatalf("Alex's Up next = %+v, want 3 tasks", upNext)
	}
	for i, want := range []string{"First up", "Second up", "Shared job"} {
		if upNext[i].Title != want || upNext[i].Number != i+1 {
			t.Errorf("Up next[%d] = %+v, want title %q numbered %d", i, upNext[i], want, i+1)
		}
	}

	joLane := laneFor(t, lanes, 2)
	if got := laneTitles(joLane.UpNext); len(got) != 1 || got[0] != "Shared job" {
		t.Errorf("Jo's Up next = %v, want just [Shared job] (gate 2.30)", got)
	}
}

// SPEC gate 2.27: workload is blocks summed by size (S=1, M=2, L=4);
// one large job shows the same block count as four small ones.
func TestBuildLanesWorkloadBlocksAndLine(t *testing.T) {
	alex := people.Person{ID: 1, Name: "Alex"}
	tasksList := []Task{
		{ID: 1, Title: "Big one", Size: "L", Stage: StageDoing, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
		{ID: 2, Title: "Medium one", Size: "M", Stage: StageTodo, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
	}
	lanes := BuildLanes(tasksList, []people.Person{alex})
	alexLane := laneFor(t, lanes, 1)

	totalBlocks := 0
	for _, g := range alexLane.Workload {
		totalBlocks += len(g.Blocks)
	}
	if totalBlocks != 6 {
		t.Errorf("total blocks = %d, want 6 (one L=4 plus one M=2)", totalBlocks)
	}
	if want := "1 large · 1 medium"; alexLane.WorkloadLine != want {
		t.Errorf("WorkloadLine = %q, want %q", alexLane.WorkloadLine, want)
	}
}

// A large job's blocks equal four small jobs' blocks (gate 2.27's own
// worked example).
func TestBuildLanesOneLargeEqualsFourSmallBlocks(t *testing.T) {
	alex := people.Person{ID: 1, Name: "Alex"}
	jo := people.Person{ID: 2, Name: "Jo"}
	tasksList := []Task{
		{ID: 1, Size: "L", Stage: StageTodo, Assignees: []Assignee{{PersonID: 1, Name: "Alex"}}},
		{ID: 2, Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 2, Name: "Jo"}}},
		{ID: 3, Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 2, Name: "Jo"}}},
		{ID: 4, Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 2, Name: "Jo"}}},
		{ID: 5, Size: "S", Stage: StageTodo, Assignees: []Assignee{{PersonID: 2, Name: "Jo"}}},
	}
	lanes := BuildLanes(tasksList, []people.Person{alex, jo})

	blockCount := func(lane Lane) int {
		total := 0
		for _, g := range lane.Workload {
			total += len(g.Blocks)
		}
		return total
	}
	alexBlocks := blockCount(laneFor(t, lanes, 1))
	joBlocks := blockCount(laneFor(t, lanes, 2))
	if alexBlocks != joBlocks {
		t.Errorf("one L (%d blocks) != four S (%d blocks), want equal", alexBlocks, joBlocks)
	}
}

func TestUnassignedLaneHasNoWorkload(t *testing.T) {
	tasksList := []Task{{ID: 1, Size: "L", Stage: StageTodo}}
	lanes := BuildLanes(tasksList, nil)
	unassigned := laneFor(t, lanes, 0)
	if unassigned.Workload != nil || unassigned.WorkloadLine != "" {
		t.Errorf("Unassigned lane workload = %+v / %q, want empty (SPEC gate 2.27 is per-person)", unassigned.Workload, unassigned.WorkloadLine)
	}
}

// SPEC B4: Ideas and Done never appear in the Team view — enforced by
// ListForTeamView's own query, not BuildLanes (which only groups
// whatever it's given).
func TestListForTeamViewExcludesIdeasAndDone(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	creator := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()

	createNamed(t, store, ctx, creator, "An idea", StageIdea)
	createNamed(t, store, ctx, creator, "Done already", StageDone)
	createNamed(t, store, ctx, creator, "To do", StageTodo)
	createNamed(t, store, ctx, creator, "Doing", StageDoing)

	tasksList, err := store.ListForTeamView(ctx)
	if err != nil {
		t.Fatalf("ListForTeamView: %v", err)
	}
	var titles []string
	for _, tk := range tasksList {
		titles = append(titles, tk.Title)
	}
	want := map[string]bool{"To do": true, "Doing": true}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v, want exactly %v", titles, want)
	}
	for _, title := range titles {
		if !want[title] {
			t.Errorf("unexpected task %q in Team view (Ideas/Done must be excluded)", title)
		}
	}
}
