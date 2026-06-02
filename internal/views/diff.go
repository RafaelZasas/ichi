package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/gxt/internal/git"
)

// DiffView displays a commit diff using dado's DiffViewer.
type DiffView struct {
	flex       *core.Flex
	diffViewer *components.DiffViewer
	repo       *git.Repository
	app        *layout.App
	hash       string
}

// NewDiffView creates a new diff view for a commit.
func NewDiffView(app *layout.App, repo *git.Repository, hash string) *DiffView {
	v := &DiffView{
		flex:       core.NewFlex(),
		diffViewer: components.NewDiffViewer(),
		repo:       repo,
		app:        app,
		hash:       hash,
	}
	v.setup()
	return v
}

func (v *DiffView) setup() {
	v.diffViewer.SetShowLineNumbers(true)

	title := "Diff"
	if v.hash != "" {
		if len(v.hash) == 40 {
			title = fmt.Sprintf("Diff: %s", v.hash[:8])
		} else {
			title = fmt.Sprintf("Diff: %s", v.hash)
		}
	}

	panel := components.NewPanel().
		SetTitle(title).
		SetContent(v.diffViewer)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)
}

// nav.Component interface

func (v *DiffView) Name() string {
	return "Diff"
}

func (v *DiffView) Start() {
	v.loadDiff()
}

func (v *DiffView) Stop() {}

func (v *DiffView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Scroll"},
		{Key: "n/N", Description: "Next/Prev change"},
		{Key: "g/G", Description: "Top/Bottom"},
		{Key: "l", Description: "Toggle line numbers"},
		{Key: "Esc", Description: "Back"},
	}
}

func (v *DiffView) loadDiff() {
	diff, err := v.repo.GetCommitDiff(v.hash)
	if err != nil {
		v.showError(err)
		return
	}
	v.diffViewer.SetUnifiedDiff(diff)
}

func (v *DiffView) showError(err error) {
	v.diffViewer.SetUnifiedDiff(fmt.Sprintf("Error: %v", err))
}

// core.Widget interface

func (v *DiffView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *DiffView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *DiffView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *DiffView) Blur()                         { v.diffViewer.Blur() }
func (v *DiffView) HasFocus() bool                { return v.diffViewer.HasFocus() }

func (v *DiffView) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Rune() {
	case 'j':
		v.diffViewer.NextChange()
		return true
	case 'k':
		v.diffViewer.PrevChange()
		return true
	case 'n':
		v.diffViewer.NextChange()
		return true
	case 'N':
		v.diffViewer.PrevChange()
		return true
	case 'J':
		v.diffViewer.NextHunk()
		return true
	case 'K':
		v.diffViewer.PrevHunk()
		return true
	}
	switch ev.Key() {
	case tcell.KeyDown:
		v.diffViewer.NextChange()
		return true
	case tcell.KeyUp:
		v.diffViewer.PrevChange()
		return true
	}
	return false
}

// FileDiffView displays a diff for a single file.
type FileDiffView struct {
	*DiffView
	file string
}

func (v *FileDiffView) Name() string {
	return "File Diff"
}

// NewFileDiffView creates a diff view for a specific file in a commit.
func NewFileDiffView(app *layout.App, repo *git.Repository, hash, file string) *FileDiffView {
	v := &FileDiffView{
		DiffView: NewDiffView(app, repo, hash),
		file:     file,
	}
	return v
}

func (v *FileDiffView) Start() {
	diff, err := v.repo.GetFileDiff(v.hash, v.file)
	if err != nil {
		v.showError(err)
		return
	}
	v.diffViewer.SetUnifiedDiff(diff)
}
