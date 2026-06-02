package views

import (
	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/input"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/git"
)

// StagingView provides interactive hunk/line staging using dado's DiffViewer.
type StagingView struct {
	*core.Flex
	diffViewer *components.DiffViewer
	repo       *git.Repository
	app        *layout.App
	file       string
	isStaged   bool
	hunks      []*git.DiffHunk // Keep reference to parsed hunks for staging operations
	actions    *input.ActionRegistry
}

// NewStagingView creates a new interactive staging view.
func NewStagingView(app *layout.App, repo *git.Repository, file string, staged bool) *StagingView {
	v := &StagingView{
		Flex:       core.NewFlex(),
		diffViewer: components.NewDiffViewer(),
		repo:       repo,
		app:        app,
		file:       file,
		isStaged:   staged,
	}
	v.setup()
	return v
}

func (v *StagingView) setup() {
	v.SetDirection(core.Column)
	v.SetBackgroundColor(theme.Bg())

	// Configure diff viewer for staging mode
	v.diffViewer.SetShowLineNumbers(true)
	v.diffViewer.SetSelectionEnabled(true)
	v.diffViewer.SetTitle(v.file)

	// Wrap in panel
	panel := components.NewPanel().
		SetTitle("Staging: " + v.file).
		SetContent(v.diffViewer)

	v.AddItem(panel, 0, 1, true)

	v.actions = input.NewActionRegistry().
		AddSimple("stage_hunk", 's', "Stage hunk", v.stageHunk).
		AddKey("stage_lines", tcell.KeyEnter, "Stage lines", v.stageLines).
		AddSimple("stage_all", 'S', "Stage all", v.stageAll).
		AddSimple("unstage_hunk", 'u', "Unstage hunk", v.unstageHunk).
		AddSimple("discard_hunk", 'D', "Discard hunk", v.discardHunk).
		AddSimple("toggle_select", ' ', "Select line", v.toggleSelect).
		AddSimple("select_all", 'a', "Select all in hunk", v.selectAllInHunk).
		AddSimple("clear_select", 'c', "Clear selection", v.clearSelection).
		AddSimple("refresh", 'r', "Refresh", v.loadDiff)
}

// nav.Component interface

func (v *StagingView) Name() string {
	return "Stage: " + v.file
}

func (v *StagingView) Start() {
	v.loadDiff()
}

func (v *StagingView) Stop() {}

func (v *StagingView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Line"},
		{Key: "J/K", Description: "Hunk"},
		{Key: "Space", Description: "Select"},
		{Key: "s", Description: "Stage hunk"},
		{Key: "Enter", Description: "Stage lines"},
		{Key: "u", Description: "Unstage"},
	}
}

func (v *StagingView) loadDiff() {
	var diff string
	var err error

	if v.isStaged {
		diff, err = v.repo.GetStagedFileDiff(v.file)
	} else {
		diff, err = v.repo.GetWorkingFileDiff(v.file)
	}

	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}

	// Parse for staging operations
	files, err := git.ParseDiff(diff)
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to parse diff")
		return
	}

	// No more changes - go back to previous view
	if len(files) == 0 || len(files[0].Hunks) == 0 {
		app.ToastSuccess("All changes staged")
		v.app.Pages().Pop()
		return
	}

	v.hunks = files[0].Hunks

	// Set the diff in the viewer
	v.diffViewer.SetUnifiedDiff(diff)
}

func (v *StagingView) stageHunk() {
	hunkIdx := v.diffViewer.GetCurrentHunkIndex()
	if hunkIdx < 0 || hunkIdx >= len(v.hunks) {
		return
	}

	hunk := v.hunks[hunkIdx]
	if err := v.repo.StageHunk(v.file, hunk); err != nil {
		ShowErrorModal(v.app, "Stage Failed", err.Error())
		return
	}
	app.ToastSuccess("Hunk staged")
	v.loadDiff()
}

func (v *StagingView) stageLines() {
	hunkIdx := v.diffViewer.GetCurrentHunkIndex()
	if hunkIdx < 0 || hunkIdx >= len(v.hunks) {
		return
	}

	hunk := v.hunks[hunkIdx]

	// Get selected lines from diff viewer
	selectedDiffLines := v.diffViewer.GetSelectedLines()

	// Convert to git.DiffLine for staging
	var selected []*git.DiffLine
	for _, dl := range selectedDiffLines {
		if dl.HunkIndex == hunkIdx {
			// Find matching line in our hunks
			for _, gitLine := range hunk.Lines {
				if gitLine.OldLineNo == dl.OldLineNo && gitLine.NewLineNo == dl.NewLineNo {
					selected = append(selected, gitLine)
					break
				}
			}
		}
	}

	// If none selected, use current line
	if len(selected) == 0 {
		currentLine := v.diffViewer.GetSelectedLine()
		if currentLine != nil && (currentLine.Type == components.DiffLineAdded || currentLine.Type == components.DiffLineRemoved) {
			// Find matching line in hunk
			for _, gitLine := range hunk.Lines {
				if gitLine.OldLineNo == currentLine.OldLineNo && gitLine.NewLineNo == currentLine.NewLineNo {
					selected = []*git.DiffLine{gitLine}
					break
				}
			}
		}
	}

	if len(selected) == 0 {
		return
	}

	if err := v.repo.StageLines(v.file, hunk, selected); err != nil {
		ShowErrorModal(v.app, "Stage Failed", err.Error())
		return
	}
	app.ToastSuccess("Lines staged")
	v.loadDiff()
}

func (v *StagingView) stageAll() {
	for _, hunk := range v.hunks {
		if err := v.repo.StageHunk(v.file, hunk); err != nil {
			ShowErrorModal(v.app, "Stage Failed", err.Error())
			return
		}
	}
	app.ToastSuccess("All hunks staged")
	v.loadDiff()
}

func (v *StagingView) unstageHunk() {
	hunkIdx := v.diffViewer.GetCurrentHunkIndex()
	if hunkIdx < 0 || hunkIdx >= len(v.hunks) {
		return
	}

	hunk := v.hunks[hunkIdx]
	if err := v.repo.UnstageHunk(v.file, hunk); err != nil {
		ShowErrorModal(v.app, "Unstage Failed", err.Error())
		return
	}
	app.ToastSuccess("Hunk unstaged")
	v.loadDiff()
}

func (v *StagingView) discardHunk() {
	hunkIdx := v.diffViewer.GetCurrentHunkIndex()
	if hunkIdx < 0 || hunkIdx >= len(v.hunks) {
		return
	}

	hunk := v.hunks[hunkIdx]
	ShowConfirmModal(v.app, "Discard Changes",
		"Discard this hunk? This cannot be undone.",
		func() {
			if err := v.repo.DiscardHunk(v.file, hunk); err != nil {
				ShowErrorModal(v.app, "Discard Failed", err.Error())
				return
			}
			app.ToastSuccess("Hunk discarded")
			v.loadDiff()
		})
}

func (v *StagingView) toggleSelect() {
	v.diffViewer.ToggleLineSelection()
}

func (v *StagingView) selectAllInHunk() {
	v.diffViewer.SelectAllInHunk()
}

func (v *StagingView) clearSelection() {
	v.diffViewer.ClearSelection()
}

// HandleKey handles keyboard input.
func (v *StagingView) HandleKey(event *tcell.EventKey) bool {
	// Handle staging-specific actions first
	if v.actions.Handle(event) {
		return true
	}

	// Handle J/K for hunk navigation
	switch event.Rune() {
	case 'J':
		v.diffViewer.NextHunk()
		return true
	case 'K':
		v.diffViewer.PrevHunk()
		return true
	}

	// Delegate to diff viewer for navigation
	switch event.Rune() {
	case 'j':
		v.diffViewer.MoveDown()
		return true
	case 'k':
		v.diffViewer.MoveUp()
		return true
	}
	switch event.Key() {
	case tcell.KeyDown:
		v.diffViewer.MoveDown()
		return true
	case tcell.KeyUp:
		v.diffViewer.MoveUp()
		return true
	}
	return false
}
