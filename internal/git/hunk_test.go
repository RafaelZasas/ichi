package git

import "testing"

func TestParseDiffRoundTrip(t *testing.T) {
	diff := "diff --git a/f.go b/f.go\n" +
		"--- a/f.go\n+++ b/f.go\n" +
		"@@ -1,3 +1,3 @@ func F() {\n" +
		" ctx\n-old\n+new\n"
	files, err := ParseDiff(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(files[0].Hunks) != 1 {
		t.Fatalf("unexpected structure: %+v", files)
	}
	h := files[0].Hunks[0]
	if len(h.Lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(h.Lines))
	}
	// Arena-allocated lines must remain distinct and correctly typed.
	if h.Lines[0].Type != LineContext || h.Lines[1].Type != LineRemoved || h.Lines[2].Type != LineAdded {
		t.Errorf("line types wrong: %+v %+v %+v", h.Lines[0], h.Lines[1], h.Lines[2])
	}
	if h.Lines[2].Content != "new" || h.Lines[1].Content != "old" {
		t.Errorf("content wrong: %q %q", h.Lines[1].Content, h.Lines[2].Content)
	}
	// computeLineNumbers must have run.
	if h.Lines[0].OldLineNo != 1 || h.Lines[0].NewLineNo != 1 {
		t.Errorf("line numbers wrong: %+v", h.Lines[0])
	}
}
