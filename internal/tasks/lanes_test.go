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

// Done never appears in the Team view; Ideas do (gates 4.48, 4.49, D-59) —
// enforced by ListForTeamView's own query, not BuildLanes (which only
// groups whatever it's given).
func TestListForTeamViewIncludesIdeasButNotDone(t *testing.T) {
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
	want := map[string]bool{"An idea": true, "To do": true, "Doing": true}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v, want exactly %v", titles, want)
	}
	for _, title := range titles {
		if !want[title] {
			t.Errorf("unexpected task %q in Team view (Done must be excluded)", title)
		}
	}
}

// blockTotal sums a lane's workload blocks.
func blockTotal(lane Lane) int {
	total := 0
	for _, g := range lane.Workload {
		total += len(g.Blocks)
	}
	return total
}

// SPEC gates 4.48, 4.49: an idea with someone on it shows in that
// person's My jobs "Ideas I'm on" group and Team lane, is absent from Up
// for grabs (somebody is on it), and adds nothing to their workload (it
// isn't work yet). Written failing-first (SPEC B9.8): before the fix an
// assigned idea fell through every list.
func TestAssignedIdeaShowsInLaneNotUpForGrabsAndAddsNoWorkload(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	ctx := context.Background()
	samPerson := people.Person{ID: sam, Name: "Sam"}

	if _, err := store.Create(ctx, CreateInput{Title: "Real work", Size: "M", Stage: StageTodo, PersonIDs: []int64{sam}}, sam, fixedNow); err != nil {
		t.Fatalf("Create(todo): %v", err)
	}
	teamTasks, err := store.ListForTeamView(ctx)
	if err != nil {
		t.Fatalf("ListForTeamView: %v", err)
	}
	blocksBefore := blockTotal(LaneForPerson(teamTasks, samPerson))
	if blocksBefore != 2 {
		t.Fatalf("workload before the idea = %d blocks, want 2 (one medium)", blocksBefore)
	}

	if _, err := store.Create(ctx, CreateInput{Title: "Big idea", Size: "L", Stage: StageIdea, PersonIDs: []int64{sam}}, sam, fixedNow); err != nil {
		t.Fatalf("Create(idea): %v", err)
	}
	teamTasks, err = store.ListForTeamView(ctx)
	if err != nil {
		t.Fatalf("ListForTeamView: %v", err)
	}
	lanes := BuildLanes(teamTasks, []people.Person{samPerson})
	lane := laneFor(t, lanes, sam)

	if got := laneTitles(lane.Ideas); len(got) != 1 || got[0] != "Big idea" {
		t.Errorf("Sam's Ideas group = %v, want [Big idea]", got)
	}
	if got := laneTitles(lane.UpNext); len(got) != 1 || got[0] != "Real work" {
		t.Errorf("Sam's Up next = %v, want only [Real work] (an idea is not mixed into Up next)", got)
	}
	if got := laneTitles(lane.WorkingOnNow); len(got) != 0 {
		t.Errorf("Sam's Working on now = %v, want none", got)
	}
	if got := blockTotal(lane); got != blocksBefore {
		t.Errorf("workload = %d blocks, want unchanged at %d (an idea must not move anybody's blocks)", got, blocksBefore)
	}
	if lane.WorkloadLine != "1 medium" {
		t.Errorf("WorkloadLine = %q, want %q", lane.WorkloadLine, "1 medium")
	}
	if got := laneTitles(LaneForPerson(teamTasks, samPerson).Ideas); len(got) != 1 || got[0] != "Big idea" {
		t.Errorf("LaneForPerson Ideas = %v, want [Big idea] (My jobs uses the same lane)", got)
	}

	grabs, err := store.ListUpForGrabs(ctx)
	if err != nil {
		t.Fatalf("ListUpForGrabs: %v", err)
	}
	for _, g := range grabs {
		if g.Title == "Big idea" {
			t.Errorf("Up for grabs contains %q, but Sam is on it", g.Title)
		}
	}
}

// SPEC gate 4.49: an idea shared by two people shows in both lanes, and
// an idea nobody is on stays out of the Team view's Unassigned lane (it
// belongs in Up for grabs — gate 4.50).
func TestIdeasSharedAcrossLanesAndUnassignedIdeaStaysOut(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	if _, err := store.Create(ctx, CreateInput{Title: "Shared idea", Stage: StageIdea, PersonIDs: []int64{sam, alex}}, sam, fixedNow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	createNamed(t, store, ctx, sam, "Nobody's idea", StageIdea)

	teamTasks, err := store.ListForTeamView(ctx)
	if err != nil {
		t.Fatalf("ListForTeamView: %v", err)
	}
	lanes := BuildLanes(teamTasks, []people.Person{{ID: sam, Name: "Sam"}, {ID: alex, Name: "Alex"}})
	for _, id := range []int64{sam, alex} {
		if got := laneTitles(laneFor(t, lanes, id).Ideas); len(got) != 1 || got[0] != "Shared idea" {
			t.Errorf("person %d Ideas = %v, want [Shared idea]", id, got)
		}
	}
	unassigned := laneFor(t, lanes, 0)
	if len(unassigned.Ideas) != 0 || len(unassigned.UpNext) != 0 || len(unassigned.WorkingOnNow) != 0 {
		t.Errorf("Unassigned lane = %+v, want empty (an unassigned idea belongs in Up for grabs)", unassigned)
	}
}

// SPEC gate 4.50: an idea nobody is on still appears in Up for grabs,
// and taking it still moves it to To do (gate 2.34, unchanged).
func TestUnassignedIdeaStillInUpForGrabsAndTakeMovesToTodo(t *testing.T) {
	sqlDB := openTestDB(t)
	store := &Store{DB: sqlDB}
	sam := testPerson(t, sqlDB, "Sam")
	alex := testPerson(t, sqlDB, "Alex")
	ctx := context.Background()

	idea := createNamed(t, store, ctx, sam, "Loose idea", StageIdea)
	grabs, err := store.ListUpForGrabs(ctx)
	if err != nil {
		t.Fatalf("ListUpForGrabs: %v", err)
	}
	found := false
	for _, g := range grabs {
		found = found || g.ID == idea.ID
	}
	if !found {
		t.Fatalf("an idea nobody is on is missing from Up for grabs: %+v", grabs)
	}

	if err := store.Take(ctx, idea.ID, 0, 0, alex, fixedNow); err != nil {
		t.Fatalf("Take: %v", err)
	}
	got, err := store.Get(ctx, idea.ID, fixedNow)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Stage != StageTodo {
		t.Errorf("stage after Take = %q, want %q", got.Stage, StageTodo)
	}
}

// SPEC gate 4.10: one block for a small job, two for medium, four for
// large, and a spoken equivalent that names the total and each job's
// size in drawing order.
func TestWorkloadBlocksAndSpokenLabel(t *testing.T) {
	alex := people.Person{ID: 1, Name: "Alex"}
	mine := []Assignee{{PersonID: 1, Name: "Alex"}}
	tasksList := []Task{
		{ID: 1, Size: "L", Stage: StageDoing, Assignees: mine},
		{ID: 2, Size: "M", Stage: StageTodo, Assignees: mine},
		{ID: 3, Size: "S", Stage: StageTodo, Assignees: mine},
		{ID: 4, Size: "L", Stage: StageIdea, Assignees: mine}, // an idea isn't work yet
	}
	lane := laneFor(t, BuildLanes(tasksList, []people.Person{alex}), 1)

	var sizes []int
	for _, g := range lane.Workload {
		sizes = append(sizes, len(g.Blocks))
	}
	if want := []int{4, 2, 1}; len(sizes) != 3 || sizes[0] != want[0] || sizes[1] != want[1] || sizes[2] != want[2] {
		t.Errorf("blocks per job = %v, want %v (large 4, medium 2, small 1)", sizes, want)
	}
	if want := "Workload: 7 blocks — large, medium, small"; lane.WorkloadLabel != want {
		t.Errorf("WorkloadLabel = %q, want %q", lane.WorkloadLabel, want)
	}
}

func TestWorkloadLabelSingularAndEmpty(t *testing.T) {
	if got, want := WorkloadLabel([]WorkloadGroup{{Blocks: make([]int, 1)}}), "Workload: 1 block — small"; got != want {
		t.Errorf("one small job = %q, want %q", got, want)
	}
	if got := WorkloadLabel(nil); got != "" {
		t.Errorf("no jobs = %q, want empty (nothing is drawn)", got)
	}
	if got, want := WorkloadLabel([]WorkloadGroup{{Blocks: make([]int, 2)}, {Blocks: make([]int, 2)}}), "Workload: 4 blocks — medium, medium"; got != want {
		t.Errorf("two medium jobs = %q, want %q", got, want)
	}
}
