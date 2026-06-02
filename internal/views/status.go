package views

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/binding"
	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/input"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/gxt/internal/git"
	"github.com/atterpac/gxt/internal/selection"
)

// StatusView displays the working tree status.
type StatusView struct {
	flex            *core.Flex
	stagedTbl       *components.Table
	unstagedTbl     *components.Table
	repo            *git.Repository
	app             *layout.App
	entries         *binding.Value[[]git.StatusEntry]
	stagedBinding   *binding.TableBinding[git.StatusEntry]
	unstagedBinding *binding.TableBinding[git.StatusEntry]
	actions         *input.ActionRegistry
	focusStaged     *binding.Value[bool]
}

// NewStatusView creates a new status view.
func NewStatusView(app *layout.App, repo *git.Repository) *StatusView {
	v := &StatusView{
		flex:        core.NewFlex(),
		stagedTbl:   components.NewTable(),
		unstagedTbl: components.NewTable(),
		repo:        repo,
		app:         app,
		entries:     binding.NewValue[[]git.StatusEntry](nil),
		focusStaged: binding.NewValue(false),
	}
	v.setup()
	return v
}

func (v *StatusView) setup() {
	// Configure tables
	v.stagedTbl.SetHeaders("Status", "File")
	v.stagedTbl.SetSelectable(true, false)

	v.unstagedTbl.SetHeaders("Status", "File")
	v.unstagedTbl.SetSelectable(true, false)

	// Wrap in panels
	stagedPanel := components.NewPanel().
		SetTitle("Staged Changes").
		SetContent(v.stagedTbl)

	unstagedPanel := components.NewPanel().
		SetTitle("Unstaged Changes").
		SetContent(v.unstagedTbl)

	// Horizontal split
	split := components.NewSplit().
		SetDirection(components.SplitVertical).
		SetRatio(0.5).
		SetLeft(unstagedPanel).
		SetRight(stagedPanel)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(split, 0, 1, true)

	// Staged binding - filters for staged entries
	v.stagedBinding = binding.NewTableBinding[git.StatusEntry](v.stagedTbl).
		SetMapper(func(e git.StatusEntry) []string { return v.statusMapper(e, true) }).
		SetKeyMapper(func(e git.StatusEntry) string { return "staged:" + e.Path }).
		SetFilter(func(e git.StatusEntry, _ string) bool { return e.IsStaged })

	// Unstaged binding - filters for working tree changes
	v.unstagedBinding = binding.NewTableBinding[git.StatusEntry](v.unstagedTbl).
		SetMapper(func(e git.StatusEntry) []string { return v.statusMapper(e, false) }).
		SetKeyMapper(func(e git.StatusEntry) string { return "unstaged:" + e.Path }).
		SetFilter(func(e git.StatusEntry, _ string) bool {
			return e.WorkStatus != git.FileUnchanged || e.IsUntracked
		})

	// When entries change, push to both bindings
	v.entries.Subscribe(func(_, newEntries []git.StatusEntry) {
		v.stagedBinding.SetData(newEntries)
		v.unstagedBinding.SetData(newEntries)
	})

	v.actions = input.NewActionRegistry().
		AddSimple("stage", ' ', "Stage/Unstage", v.toggleStage).
		AddSimple("stage_all", 'a', "Stage all", v.stageAll).
		AddSimple("unstage_all", 'u', "Unstage all", v.unstageAll).
		AddSimple("commit", 'c', "Commit", v.commit).
		AddSimple("diff", 'd', "View diff", v.viewDiff).
		AddSimple("refresh", 'r', "Refresh", v.refresh).
		AddSimple("discard", 'D', "Discard changes", v.discardChanges).
		AddKey("switch_panel", tcell.KeyTab, "Switch panel", v.switchPanel).
		AddKey("staging_view", tcell.KeyEnter, "Interactive staging", v.openStagingView)
}

// nav.Component interface

// Selection implements selection.Provider.
func (v *StatusView) Selection() *selection.Context {
	sel := &selection.Context{ViewName: v.Name()}
	if entry := v.currentBinding().GetSelected(); entry != nil {
		sel.File = entry
	}
	return sel
}

func (v *StatusView) Name() string {
	return "Status"
}

func (v *StatusView) Start() {
	v.refresh()
}

func (v *StatusView) Stop() {}

func (v *StatusView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Space", Description: "Stage/Unstage"},
		{Key: "a/u", Description: "All"},
		{Key: "c", Description: "Commit"},
		{Key: "d", Description: "Diff"},
		{Key: "Tab", Description: "Switch"},
	}
}

func (v *StatusView) refresh() {
	entries, err := v.repo.Status()
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	v.entries.SetAndDraw(entries)
}

func (v *StatusView) getStatusColor(e git.StatusEntry) string {
	switch {
	case e.IndexStatus == git.FileAdded || e.WorkStatus == git.FileAdded:
		return theme.TagSuccess()
	case e.IndexStatus == git.FileDeleted || e.WorkStatus == git.FileDeleted:
		return theme.TagError()
	case e.IndexStatus == git.FileModified || e.WorkStatus == git.FileModified:
		return theme.TagWarning()
	case e.IsUntracked:
		return theme.TagFgDim()
	case e.IsConflict:
		return theme.TagError()
	}
	return theme.TagFg()
}

func (v *StatusView) statusMapper(e git.StatusEntry, useIndex bool) []string {
	statusColor := v.getStatusColor(e)
	status := e.WorkStatus
	if useIndex {
		status = e.IndexStatus
	}
	if e.IsUntracked {
		status = git.FileUntracked
	}

	path := e.Path
	if e.OldPath != "" && e.OldPath != e.Path {
		path = e.OldPath + " -> " + e.Path
	}
	return []string{
		fmt.Sprintf("[%s]%s[-]", statusColor, status.String()),
		path,
	}
}

func (v *StatusView) currentBinding() *binding.TableBinding[git.StatusEntry] {
	if v.focusStaged.Get() {
		return v.stagedBinding
	}
	return v.unstagedBinding
}

func (v *StatusView) currentTable() *components.Table {
	if v.focusStaged.Get() {
		return v.stagedTbl
	}
	return v.unstagedTbl
}

func (v *StatusView) toggleStage() {
	entry := v.currentBinding().GetSelected()
	if entry == nil {
		return
	}

	var err error
	if v.focusStaged.Get() {
		err = v.repo.UnstageFile(entry.Path)
	} else {
		err = v.repo.StageFile(entry.Path)
	}
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	v.refresh()
}

func (v *StatusView) stageAll() {
	if err := v.repo.StageAll(); err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	v.refresh()
}

func (v *StatusView) unstageAll() {
	if err := v.repo.UnstageAll(); err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	v.refresh()
}

func (v *StatusView) commit() {
	// Check if there are staged changes
	if v.stagedBinding.Count() == 0 {
		ShowErrorModal(v.app, "No Changes", "No staged changes to commit")
		return
	}

	ShowInputModal(v.app, "Commit", "Commit message:", func(message string) {
		if message == "" {
			return
		}
		if err := v.repo.Commit(message); err != nil {
			ShowErrorModal(v.app, "Commit Failed", err.Error())
			return
		}
		v.refresh()
	})
}

func (v *StatusView) viewDiff() {
	entry := v.currentBinding().GetSelected()
	if entry == nil {
		return
	}

	isStaged := v.focusStaged.Get()
	diffView := NewWorkingDiffView(v.app, v.repo, entry.Path, isStaged)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Status", entry.Path})
}

func (v *StatusView) discardChanges() {
	if v.focusStaged.Get() {
		return // Can't discard staged changes
	}

	entry := v.unstagedBinding.GetSelected()
	if entry == nil {
		return
	}

	ShowConfirmModal(v.app, "Discard Changes",
		fmt.Sprintf("Discard all changes to %s?", entry.Path),
		func() {
			if err := v.repo.DiscardFileChanges(entry.Path); err != nil {
				ShowErrorModal(v.app, "Error", err.Error())
				return
			}
			v.refresh()
		})
}

func (v *StatusView) switchPanel() {
	v.focusStaged.Update(func(b bool) bool { return !b })
}

func (v *StatusView) openStagingView() {
	entry := v.currentBinding().GetSelected()

	os.WriteFile("/tmp/gxt_status_debug.txt", []byte(fmt.Sprintf("openStagingView: focusStaged=%v, entry=%v\n", v.focusStaged.Get(), entry)), 0644)

	if entry == nil {
		return
	}

	os.WriteFile("/tmp/gxt_status_debug2.txt", []byte(fmt.Sprintf("found file: '%s'\n", entry.Path)), 0644)

	stagingView := NewStagingView(v.app, v.repo, entry.Path, v.focusStaged.Get())
	v.app.Pages().Push(stagingView)
	v.app.Crumbs().SetPath([]string{"Status", "Staging", entry.Path})
}

// core.Widget interface

func (v *StatusView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *StatusView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *StatusView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *StatusView) Blur()                         { v.flex.Blur() }
func (v *StatusView) HasFocus() bool                { return v.flex.HasFocus() }

func (v *StatusView) HandleKey(event *tcell.EventKey) bool {
	if v.actions.Handle(event) {
		return true
	}

	table := v.currentTable()
	row, col := table.GetSelection()

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			if row < table.GetRowCount()-1 {
				table.Select(row+1, col)
			}
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			if row > 1 {
				table.Select(row-1, col)
			}
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			if row < table.GetRowCount()-1 {
				table.Select(row+1, col)
			}
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			if row > 1 {
				table.Select(row-1, col)
			}
			return true
		})

	return bindings.Handle(event)
}

// WorkingDiffView shows diff for working tree changes.
type WorkingDiffView struct {
	*DiffView
	file     string
	isStaged bool
}

func (v *WorkingDiffView) Name() string {
	return "Working Diff"
}

// NewWorkingDiffView creates a diff view for working tree changes.
func NewWorkingDiffView(app *layout.App, repo *git.Repository, file string, staged bool) *WorkingDiffView {
	v := &WorkingDiffView{
		DiffView: NewDiffView(app, repo, file),
		file:     file,
		isStaged: staged,
	}
	return v
}

func (v *WorkingDiffView) Start() {
	var diff string
	var err error

	if v.isStaged {
		diff, err = v.repo.GetStagedFileDiff(v.file)
	} else {
		diff, err = v.repo.GetWorkingFileDiff(v.file)
	}

	if err != nil {
		v.showError(err)
		return
	}
	v.diffViewer.SetUnifiedDiff(diff)
}
