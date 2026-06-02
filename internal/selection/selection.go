package selection

import (
	"github.com/atterpac/dado/components"

	"github.com/atterpac/ichi/internal/git"
)

// Context carries the current view's selection state to command handlers.
type Context struct {
	// ViewName is the name of the active view (e.g. "Commit Graph", "Branches").
	ViewName string

	// Commit is set when a commit is selected (graph, file log, commit detail).
	Commit *components.GitCommit

	// Branch is set when a branch is selected (branches view).
	Branch *git.Branch

	// Stash is set when a stash entry is selected (stash view).
	Stash *git.Stash

	// File is set when a file is selected (status view, commit view).
	File *git.StatusEntry

	// CommitHash is set when viewing a specific commit (commit detail view).
	CommitHash string
}

// HasCommit returns true if a non-pseudo commit is selected.
func (s *Context) HasCommit() bool {
	return s != nil && s.Commit != nil && !s.Commit.IsPseudoNode
}

// HasBranch returns true if a branch is selected.
func (s *Context) HasBranch() bool {
	return s != nil && s.Branch != nil
}

// HasStash returns true if a stash is selected.
func (s *Context) HasStash() bool {
	return s != nil && s.Stash != nil
}

// HasFile returns true if a file is selected.
func (s *Context) HasFile() bool {
	return s != nil && s.File != nil
}

// Provider is implemented by views that expose their current selection.
type Provider interface {
	// Selection returns the current selection context for the view.
	Selection() *Context
}
