package git

import "testing"

func TestParseTrack(t *testing.T) {
	cases := []struct {
		in            string
		ahead, behind int
	}{
		{"", 0, 0},
		{"gone", 0, 0},
		{"ahead 2", 2, 0},
		{"behind 4", 0, 4},
		{"ahead 2, behind 1", 2, 1},
		{"ahead 13, behind 27", 13, 27},
	}
	for _, c := range cases {
		a, b := parseTrack(c.in)
		if a != c.ahead || b != c.behind {
			t.Errorf("parseTrack(%q) = (%d,%d), want (%d,%d)", c.in, a, b, c.ahead, c.behind)
		}
	}
}
