package app

import "testing"

// SPEC gate 4.11 and B9.4: exactly four icon-only buttons, each with a
// title and spoken name that are one string naming the task.
func TestNewIconButtonLabelsNameTheTask(t *testing.T) {
	cases := []struct {
		kind, icon, label string
	}{
		{IconMoveUp, "arrow-up", "Move Print exam papers up"},
		{IconMoveDown, "arrow-down", "Move Print exam papers down"},
		{IconRemove, "trash", "Remove Print exam papers"},
		{IconGiveBack, "give-back", "Give back Print exam papers"},
	}
	for _, c := range cases {
		got := NewIconButton(c.kind, "Print exam papers", false)
		if got.Icon != c.icon || got.Label != c.label || got.Disabled {
			t.Errorf("NewIconButton(%q) = %+v, want icon %q label %q", c.kind, got, c.icon, c.label)
		}
	}
	if !NewIconButton(IconMoveUp, "x", true).Disabled {
		t.Error("the disabled flag is dropped")
	}
}

func TestNewIconButtonRefusesAFifthKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewIconButton(\"edit\") did not panic; a fifth icon-only button must not be possible")
		}
	}()
	NewIconButton("edit", "Print exam papers", false)
}
