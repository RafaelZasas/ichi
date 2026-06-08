package commands

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseKeyChordMatches(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ev    *tcell.EventKey
		match bool
	}{
		{"plain rune", "x", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone), true},
		{"plain rune miss", "x", tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModNone), false},
		{"ctrl letter", "ctrl+g", tcell.NewEventKey(tcell.KeyCtrlG, 0, tcell.ModCtrl), true},
		{"function key", "f5", tcell.NewEventKey(tcell.KeyF5, 0, tcell.ModNone), true},
		{"alt rune", "alt+x", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt), true},
		{"alt rune needs mod", "alt+x", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone), false},
		{"plain rune rejects alt", "x", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt), false},
		{"enter named", "enter", tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), true},
		{"shift folded", "shift+a", tcell.NewEventKey(tcell.KeyRune, 'A', tcell.ModNone), true},
		{"space", "space", tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chord, err := parseKeyChord(tt.input)
			if err != nil {
				t.Fatalf("parseKeyChord(%q) error: %v", tt.input, err)
			}
			if got := chord.matches(tt.ev); got != tt.match {
				t.Errorf("matches = %v, want %v", got, tt.match)
			}
		})
	}
}

func TestParseKeyChordErrors(t *testing.T) {
	for _, in := range []string{"", "ctrl+", "ctrl+1", "alt+enter", "nope"} {
		if _, err := parseKeyChord(in); err == nil {
			t.Errorf("parseKeyChord(%q): expected error", in)
		}
	}
}
