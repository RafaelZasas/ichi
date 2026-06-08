package commands

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// keyChord is a parsed keyboard shortcut that can be matched against a
// tcell.EventKey. Exactly one of key/ch identifies the base key.
type keyChord struct {
	key tcell.Key // non-KeyRune special key (e.g. F5, Enter), or a Ctrl+<letter> key
	ch  rune      // base rune for plain / Alt+<rune> chords; 0 when key is set
	alt bool      // Alt modifier required (only meaningful with ch)
}

// namedKeys maps chord names to their tcell key constants.
var namedKeys = map[string]tcell.Key{
	"enter":     tcell.KeyEnter,
	"return":    tcell.KeyEnter,
	"tab":       tcell.KeyTab,
	"backtab":   tcell.KeyBacktab,
	"esc":       tcell.KeyEscape,
	"escape":    tcell.KeyEscape,
	"space":     tcell.KeyRune, // handled specially below (rune ' ')
	"backspace": tcell.KeyBackspace2,
	"delete":    tcell.KeyDelete,
	"insert":    tcell.KeyInsert,
	"home":      tcell.KeyHome,
	"end":       tcell.KeyEnd,
	"pageup":    tcell.KeyPgUp,
	"pagedown":  tcell.KeyPgDn,
	"up":        tcell.KeyUp,
	"down":      tcell.KeyDown,
	"left":      tcell.KeyLeft,
	"right":     tcell.KeyRight,
	"f1":        tcell.KeyF1,
	"f2":        tcell.KeyF2,
	"f3":        tcell.KeyF3,
	"f4":        tcell.KeyF4,
	"f5":        tcell.KeyF5,
	"f6":        tcell.KeyF6,
	"f7":        tcell.KeyF7,
	"f8":        tcell.KeyF8,
	"f9":        tcell.KeyF9,
	"f10":       tcell.KeyF10,
	"f11":       tcell.KeyF11,
	"f12":       tcell.KeyF12,
}

// parseKeyChord parses a human-readable shortcut such as "ctrl+g", "alt+x",
// "f5", "ctrl+shift+p", or a single rune like "x". Modifiers are joined with
// '+'. Returns an error describing why a chord is unsupported.
func parseKeyChord(s string) (keyChord, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return keyChord{}, fmt.Errorf("empty key")
	}

	var ctrl, alt, shift bool
	var base string
	for p := range strings.SplitSeq(strings.ToLower(raw), "+") {
		switch p {
		case "ctrl", "control", "c":
			ctrl = true
		case "alt", "meta", "option", "m":
			alt = true
		case "shift":
			// Shift is folded into the rune itself (e.g. "shift+a" == "A").
			shift = true
		case "":
			return keyChord{}, fmt.Errorf("invalid key %q", raw)
		default:
			if base != "" {
				return keyChord{}, fmt.Errorf("invalid key %q: multiple base keys", raw)
			}
			base = p
		}
	}
	if base == "" {
		return keyChord{}, fmt.Errorf("invalid key %q: no base key", raw)
	}

	// Named special keys (function keys, arrows, enter, ...).
	if k, ok := namedKeys[base]; ok {
		if base == "space" {
			if ctrl {
				return keyChord{}, fmt.Errorf("unsupported key %q", raw)
			}
			return keyChord{ch: ' ', alt: alt}, nil
		}
		if alt {
			return keyChord{}, fmt.Errorf("alt+%s is not supported", base)
		}
		return keyChord{key: k}, nil
	}

	// Single-rune base.
	runes := []rune(base)
	if len(runes) != 1 {
		return keyChord{}, fmt.Errorf("unknown key %q", raw)
	}
	r := runes[0]

	if ctrl {
		// tcell encodes Ctrl+<letter> as a dedicated key value (KeyCtrlA..Z).
		if r < 'a' || r > 'z' {
			return keyChord{}, fmt.Errorf("unsupported ctrl chord %q", raw)
		}
		return keyChord{key: tcell.Key(r-'a') + tcell.KeyCtrlA}, nil
	}
	if shift && r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	return keyChord{ch: r, alt: alt}, nil
}

// matches reports whether ev triggers this chord.
func (c keyChord) matches(ev *tcell.EventKey) bool {
	if c.key != 0 {
		return ev.Key() == c.key
	}
	// Rune-based chord.
	if ev.Key() != tcell.KeyRune {
		return false
	}
	if ev.Rune() != c.ch {
		return false
	}
	hasAlt := ev.Modifiers()&tcell.ModAlt != 0
	return hasAlt == c.alt
}

// keyBinding pairs a parsed chord with the command it triggers.
type keyBinding struct {
	chord keyChord
	cmd   *Command
}

var keyBindings []keyBinding

// bindKey registers chord as a trigger for cmd, replacing any existing binding
// on the same chord.
func bindKey(chord keyChord, cmd *Command) {
	for i := range keyBindings {
		if keyBindings[i].chord == chord {
			keyBindings[i].cmd = cmd
			return
		}
	}
	keyBindings = append(keyBindings, keyBinding{chord: chord, cmd: cmd})
}

// MatchKey returns the custom command bound to ev, or nil if none matches.
func MatchKey(ev *tcell.EventKey) *Command {
	for _, b := range keyBindings {
		if b.chord.matches(ev) {
			return b.cmd
		}
	}
	return nil
}
