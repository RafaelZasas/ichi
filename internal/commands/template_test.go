package commands

import (
	"strings"
	"testing"

	"github.com/atterpac/dado/components"

	"github.com/atterpac/ichi/internal/git"
	"github.com/atterpac/ichi/internal/selection"
)

func selWithCommit() *selection.Context {
	return &selection.Context{
		ViewName: "Commit Graph",
		Commit: &components.GitCommit{
			Hash:      "abcdef1234567890abcdef1234567890abcdef12",
			ShortHash: "abcdef1",
			Message:   "fix: a bug\n\nbody line",
			Author:    "Ada",
		},
	}
}

func TestExpandCommitTokens(t *testing.T) {
	sel := selWithCommit()
	got, err := Expand("git log {{commitHash}} {{shortHash}} {{commitSubject}}", sel, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "git log 'abcdef1234567890abcdef1234567890abcdef12' 'abcdef1' 'fix: a bug'"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandMissingRequired(t *testing.T) {
	_, err := Expand("git log {{commitHash}}", &selection.Context{}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "selected commit") {
		t.Fatalf("expected missing-commit error, got %v", err)
	}
}

func TestExpandShellEscaping(t *testing.T) {
	sel := &selection.Context{File: &git.StatusEntry{Path: "weird'; rm -rf /.txt"}}
	got, err := Expand("cat {{file}}", sel, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `cat 'weird'\''; rm -rf /.txt'`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandPositionalArgs(t *testing.T) {
	got, err := Expand("echo {{1}} {{args}}", nil, nil, []string{"a", "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "echo 'a' 'a b'" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandOptionalArgs(t *testing.T) {
	cases := []struct {
		name string
		tmpl string
		args []string
		want string
	}{
		{"missing empty default", "echo {{1?}}", nil, "echo ''"},
		{"missing with default", "echo {{1?HEAD}}", nil, "echo 'HEAD'"},
		{"present overrides default", "echo {{1?HEAD}}", []string{"v1"}, "echo 'v1'"},
		{"optional selection token", "git show {{commitHash?HEAD}}", nil, "git show 'HEAD'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Expand(c.tmpl, &selection.Context{}, nil, c.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestExpandUnknownTokenLeftIntact(t *testing.T) {
	got, err := Expand("echo {{nope}}", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "echo {{nope}}" {
		t.Fatalf("got %q", got)
	}
}
