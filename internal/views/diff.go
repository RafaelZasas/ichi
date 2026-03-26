package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/git"
)

// DiffView displays a commit diff using jig's DiffViewer.
type DiffView struct {
	flex       *tview.Flex
	diffViewer *components.DiffViewer
	repo       *git.Repository
	app        *layout.App
	hash       string
}

// NewDiffView creates a new diff view for a commit.
func NewDiffView(app *layout.App, repo *git.Repository, hash string) *DiffView {
	v := &DiffView{
		flex:       tview.NewFlex(),
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

	v.flex.SetDirection(tview.FlexRow)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)
	theme.Register(v.flex)
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
	// Create an error diff result
	v.diffViewer.SetUnifiedDiff(fmt.Sprintf("Error: %v", err))
}

// tview.Primitive delegation

func (v *DiffView) Draw(screen tcell.Screen)       { v.flex.Draw(screen) }
func (v *DiffView) GetRect() (int, int, int, int)  { return v.flex.GetRect() }
func (v *DiffView) SetRect(x, y, w, h int)         { v.flex.SetRect(x, y, w, h) }
func (v *DiffView) Focus(d func(tview.Primitive)) { d(v.diffViewer) }
func (v *DiffView) Blur()                          { v.diffViewer.Blur() }
func (v *DiffView) HasFocus() bool                 { return v.diffViewer.HasFocus() }

func (v *DiffView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.diffViewer.MouseHandler()
}

func (v *DiffView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *DiffView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.flex.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		// Delegate to diffViewer's input handler
		if handler := v.diffViewer.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
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
