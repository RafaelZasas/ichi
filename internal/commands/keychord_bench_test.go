package commands

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func BenchmarkParseKeyChord(b *testing.B) {
	inputs := []string{"x", "ctrl+g", "f5", "alt+x", "ctrl+shift+p", "enter", "space"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, in := range inputs {
			if _, err := parseKeyChord(in); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkKeyChordMatches measures the per-event match hot path that runs on
// every keystroke against the command table.
func BenchmarkKeyChordMatches(b *testing.B) {
	chord, err := parseKeyChord("ctrl+g")
	if err != nil {
		b.Fatal(err)
	}
	ev := tcell.NewEventKey(tcell.KeyCtrlG, 0, tcell.ModCtrl)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chord.matches(ev)
	}
}
