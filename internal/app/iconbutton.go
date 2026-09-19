package app

import "fmt"

// IconButton is one of the four icon-only buttons (SPEC gate 4.11, B9.4):
// move up, move down, remove and give back. They are the only buttons
// without words, so the kind is fixed here rather than free text, and the
// title and spoken name are one string that names the task: "Move Print
// exam papers up". The part-icon-button partial draws it.
type IconButton struct {
	Icon     string // the icon's name in web/static/theme/icons/
	Label    string // both the hover title and the aria-label
	Disabled bool
}

// The four things an icon-only button may do.
const (
	IconMoveUp   = "move-up"
	IconMoveDown = "move-down"
	IconRemove   = "remove"
	IconGiveBack = "give-back"
)

// NewIconButton builds the icon-only button for kind and task. It panics
// on any other kind: a fifth icon-only button is a design decision, not a
// call site's choice (gate 4.11).
func NewIconButton(kind, taskTitle string, disabled bool) IconButton {
	switch kind {
	case IconMoveUp:
		return IconButton{Icon: "arrow-up", Label: fmt.Sprintf("Move %s up", taskTitle), Disabled: disabled}
	case IconMoveDown:
		return IconButton{Icon: "arrow-down", Label: fmt.Sprintf("Move %s down", taskTitle), Disabled: disabled}
	case IconRemove:
		return IconButton{Icon: "trash", Label: fmt.Sprintf("Remove %s", taskTitle), Disabled: disabled}
	case IconGiveBack:
		return IconButton{Icon: "give-back", Label: fmt.Sprintf("Give back %s", taskTitle), Disabled: disabled}
	}
	panic("app: no icon-only button for " + kind + " (gate 4.11 allows exactly four)")
}
