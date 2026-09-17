package tasks

import "errors"

// ErrNeighborNotFound is returned when before_id/after_id doesn't name a
// task actually in the target stage (PLAN.md P2-02: "before or after a
// missing id -> error").
var ErrNeighborNotFound = errors.New("that task isn't there any more")

// insertTask returns the full ordered list of task ids for a stage after
// inserting taskID among others (which must not already contain taskID)
// at the position implied by exactly one of beforeID, afterID or
// toBottom (SPEC B4: "one of before_id, after_id or to_bottom").
//
// "Before/after" means the task's new neighbour, not a direction: moving
// a card up one slot is "insert before the card that's now two above
// it" (before its old upstairs neighbour), and moving down one slot is
// "insert after the card that's now one below it" — the handler computes
// which id that is from the current board order.
func insertTask(others []int64, taskID, beforeID, afterID int64, toBottom bool) ([]int64, error) {
	if toBottom {
		result := make([]int64, 0, len(others)+1)
		result = append(result, others...)
		return append(result, taskID), nil
	}

	result := make([]int64, 0, len(others)+1)
	found := false
	for _, id := range others {
		if beforeID != 0 && id == beforeID {
			result = append(result, taskID)
			found = true
		}
		result = append(result, id)
		if afterID != 0 && id == afterID {
			result = append(result, taskID)
			found = true
		}
	}
	if !found {
		return nil, ErrNeighborNotFound
	}
	return result, nil
}
