package tasks

import (
	"reflect"
	"testing"
)

func TestInsertTaskToBottom(t *testing.T) {
	got, err := insertTask([]int64{1, 2, 3}, 4, 0, 0, true)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1, 2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskToBottomWhenStageIsEmpty(t *testing.T) {
	got, err := insertTask(nil, 1, 0, 0, true)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskBefore(t *testing.T) {
	// [1, 2, 3], insert 4 before 2 -> [1, 4, 2, 3]
	got, err := insertTask([]int64{1, 2, 3}, 4, 2, 0, false)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1, 4, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskAfter(t *testing.T) {
	// [1, 2, 3], insert 4 after 2 -> [1, 2, 4, 3]
	got, err := insertTask([]int64{1, 2, 3}, 4, 0, 2, false)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1, 2, 4, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskMoveUpWithinStage(t *testing.T) {
	// [A, B, C], C moves up one slot: remove C -> others=[A,B], insert
	// before B (C's new upstairs neighbour) -> [A, C, B].
	got, err := insertTask([]int64{1, 2}, 3, 2, 0, false)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1, 3, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskMoveDownWithinStage(t *testing.T) {
	// [A, B, C], B moves down one slot: remove B -> others=[A,C], insert
	// after C (B's new downstairs neighbour) -> [A, C, B].
	got, err := insertTask([]int64{1, 3}, 2, 0, 3, false)
	if err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	want := []int64{1, 3, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInsertTaskBeforeMissingIDIsError(t *testing.T) {
	_, err := insertTask([]int64{1, 2, 3}, 4, 99, 0, false)
	if err != ErrNeighborNotFound {
		t.Errorf("err = %v, want ErrNeighborNotFound", err)
	}
}

func TestInsertTaskAfterMissingIDIsError(t *testing.T) {
	_, err := insertTask([]int64{1, 2, 3}, 4, 0, 99, false)
	if err != ErrNeighborNotFound {
		t.Errorf("err = %v, want ErrNeighborNotFound", err)
	}
}

// TestInsertTaskPositionsStayUnique proves the result is always a
// permutation of others plus taskID exactly once each — no duplicates,
// no gaps, regardless of where the insertion happens.
func TestInsertTaskPositionsStayUnique(t *testing.T) {
	others := []int64{10, 20, 30, 40}
	cases := []struct {
		name              string
		beforeID, afterID int64
		toBottom          bool
	}{
		{"to bottom", 0, 0, true},
		{"before first", 10, 0, false},
		{"before last", 40, 0, false},
		{"after first", 0, 10, false},
		{"after last", 0, 40, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := insertTask(others, 99, c.beforeID, c.afterID, c.toBottom)
			if err != nil {
				t.Fatalf("insertTask: %v", err)
			}
			if len(got) != len(others)+1 {
				t.Fatalf("len(got) = %d, want %d", len(got), len(others)+1)
			}
			seen := make(map[int64]bool)
			for _, id := range got {
				if seen[id] {
					t.Fatalf("duplicate id %d in result %v", id, got)
				}
				seen[id] = true
			}
			for _, id := range others {
				if !seen[id] {
					t.Fatalf("result %v is missing original id %d", got, id)
				}
			}
			if !seen[99] {
				t.Fatalf("result %v is missing the inserted id", got)
			}
		})
	}
}
