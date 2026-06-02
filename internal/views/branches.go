package views

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/binding"
	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/input"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/gxt/internal/app"
	"github.com/atterpac/gxt/internal/git"
	"github.com/atterpac/gxt/internal/selection"
)

// BranchesView displays the branch list.
type BranchesView struct {
	flex       *core.Flex
	table      *components.Table
	repo       *git.Repository
	app        *layout.App
	binding    *binding.TableBinding[git.Branch]
	showRemote *binding.Value[bool]
	actions    *input.ActionRegistry
}

// NewBranchesView creates a new branches view.
func NewBranchesView(app *layout.App, repo *git.Repository) *BranchesView {
	v := &BranchesView{
		flex:       core.NewFlex(),
		table:      components.NewTable(),
		repo:       repo,
		app:        app,
		showRemote: binding.NewValue(true),
	}
	v.setup()
	return v
}

func (v *BranchesView) setup() {
	v.table.SetHeaders("", "Branch", "Tracking", "Last Commit")
	v.table.SetSelectable(true, false)

	panel := components.NewPanel().
		SetTitle("Branches").
		SetContent(v.table)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)

	// Setup binding
	v.binding = binding.NewTableBinding[git.Branch](v.table).
		SetMapper(v.branchMapper).
		SetKeyMapper(func(b git.Branch) string { return b.Name }).
		SetFetcher(v.fetchBranches).
		SetOnSelect(func(b git.Branch) { v.checkoutBranch(b) }).
		SetOnRefresh(func(_ []git.Branch, err error) {
			if err != nil {
				ShowErrorModal(v.app, "Error", err.Error())
			}
		})

	// React to remote toggle
	v.showRemote.Subscribe(func(_, _ bool) {
		v.binding.RefreshAsync()
	})

	v.actions = input.NewActionRegistry().
		AddKey("checkout", tcell.KeyEnter, "Checkout", func() {
			if branch := v.binding.GetSelected(); branch != nil {
				v.checkoutBranch(*branch)
			}
		}).
		AddSimple("new", 'n', "New branch", v.newBranch).
		AddSimple("delete", 'd', "Delete", v.deleteBranch).
		AddSimple("rename", 'r', "Rename", v.renameBranch).
		AddSimple("merge", 'm', "Merge into current", v.mergeBranch).
		AddSimple("rebase", 'b', "Rebase onto selected", v.rebaseBranch).
		AddSimple("toggle_remote", 'R', "Toggle remotes", v.toggleRemote).
		AddSimple("refresh", 'g', "Refresh", func() { v.binding.RefreshAsync() }).
		AddSimple("copy", 'y', "Copy name", v.copyBranchName)
}

// nav.Component interface

// Selection implements selection.Provider.
func (v *BranchesView) Selection() *selection.Context {
	sel := &selection.Context{ViewName: v.Name()}
	if branch := v.binding.GetSelected(); branch != nil {
		sel.Branch = branch
	}
	return sel
}

func (v *BranchesView) Name() string {
	return "Branches"
}

func (v *BranchesView) Start() {
	v.binding.Start()
}

func (v *BranchesView) Stop() {
	v.binding.Stop()
}

func (v *BranchesView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "Checkout"},
		{Key: "n", Description: "New"},
		{Key: "d", Description: "Delete"},
		{Key: "m", Description: "Merge"},
		{Key: "b", Description: "Rebase"},
		{Key: "R", Description: "Remotes"},
		{Key: "y", Description: "Copy"},
	}
}

func (v *BranchesView) fetchBranches() ([]git.Branch, error) {
	if v.showRemote.Get() {
		return v.repo.ListBranches()
	}
	return v.repo.ListLocalBranches()
}

func (v *BranchesView) branchMapper(branch git.Branch) []string {
	indicator := " "
	if branch.IsCurrent {
		indicator = "*"
	}

	nameColor := theme.TagFg()
	if branch.IsRemote {
		nameColor = theme.TagFgDim()
	} else if branch.IsCurrent {
		nameColor = theme.TagAccent()
	}

	tracking := ""
	if branch.IsTracking {
		tracking = branch.Upstream
		if branch.Ahead > 0 || branch.Behind > 0 {
			tracking += fmt.Sprintf(" [%s]+%d[%s]-%d[-]",
				theme.TagSuccess(), branch.Ahead,
				theme.TagError(), branch.Behind)
		}
	}

	return []string{
		fmt.Sprintf("[%s]%s[-]", nameColor, indicator),
		fmt.Sprintf("[%s]%s[-]", nameColor, branch.Name),
		tracking,
		fmt.Sprintf("[%s]%s[-] %s", theme.TagFgDim(), branch.LastCommit, truncate(branch.LastMsg, 40)),
	}
}

func (v *BranchesView) checkoutBranch(branch git.Branch) {
	if branch.IsCurrent {
		return // Already on this branch
	}

	branchName := branch.Name
	if branch.IsRemote {
		// For remote branches, create a local tracking branch
		// Extract branch name from remote/branch format
		parts := splitRemoteBranch(branch.Name)
		if len(parts) == 2 {
			branchName = parts[1]
		}
	}

	ShowConfirmModal(v.app, "Checkout Branch",
		fmt.Sprintf("Switch to branch %s?", branchName),
		func() {
			if err := v.repo.Checkout(branchName); err != nil {
				ShowErrorModal(v.app, "Checkout Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Switched to %s", branchName))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) newBranch() {
	ShowInputModalWithValidator(v.app, "New Branch", "Branch name:",
		app.BranchNameValidator(),
		func(name string) {
			if name == "" {
				return
			}
			if err := v.repo.CreateBranch(name); err != nil {
				ShowErrorModal(v.app, "Create Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Branch '%s' created", name))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) deleteBranch() {
	branch := v.binding.GetSelected()
	if branch == nil {
		return
	}

	if branch.IsCurrent {
		ShowErrorModal(v.app, "Cannot Delete", "Cannot delete the current branch")
		return
	}

	ShowConfirmModal(v.app, "Delete Branch",
		fmt.Sprintf("Delete branch %s?", branch.Name),
		func() {
			var err error
			if branch.IsRemote {
				parts := splitRemoteBranch(branch.Name)
				if len(parts) == 2 {
					err = v.repo.DeleteRemoteBranch(parts[0], parts[1])
				}
			} else {
				err = v.repo.DeleteBranch(branch.Name, false)
			}
			if err != nil {
				ShowErrorModal(v.app, "Delete Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Branch '%s' deleted", branch.Name))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) renameBranch() {
	branch := v.binding.GetSelected()
	if branch == nil {
		return
	}

	if branch.IsRemote {
		ShowErrorModal(v.app, "Cannot Rename", "Cannot rename remote branches")
		return
	}

	ShowInputModalWithValidator(v.app, "Rename Branch", fmt.Sprintf("New name for %s:", branch.Name),
		app.BranchNameValidator(),
		func(newName string) {
			if newName == "" || newName == branch.Name {
				return
			}
			if err := v.repo.RenameBranch(branch.Name, newName); err != nil {
				ShowErrorModal(v.app, "Rename Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Renamed to '%s'", newName))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) mergeBranch() {
	branch := v.binding.GetSelected()
	if branch == nil {
		return
	}

	if branch.IsCurrent {
		ShowErrorModal(v.app, "Cannot Merge", "Cannot merge current branch into itself")
		return
	}

	currentBranch := v.repo.CurrentBranch()
	ShowConfirmModal(v.app, "Merge Branch",
		fmt.Sprintf("Merge %s into %s?", branch.Name, currentBranch),
		func() {
			if err := v.repo.MergeBranch(branch.Name); err != nil {
				if git.IsConflictError(err) {
					// Launch conflict resolution view
					conflictView := NewConflictResolutionView(v.app, v.repo)
					v.app.Pages().Push(conflictView)
					return
				}
				ShowErrorModal(v.app, "Merge Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Merged %s", branch.Name))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) rebaseBranch() {
	branch := v.binding.GetSelected()
	if branch == nil {
		return
	}

	if branch.IsCurrent {
		ShowErrorModal(v.app, "Cannot Rebase", "Cannot rebase onto current branch")
		return
	}

	currentBranch := v.repo.CurrentBranch()
	ShowConfirmModal(v.app, "Rebase Branch",
		fmt.Sprintf("Rebase %s onto %s?", currentBranch, branch.Name),
		func() {
			if err := v.repo.RebaseBranch(branch.Name); err != nil {
				if git.IsConflictError(err) {
					// Launch conflict resolution view
					conflictView := NewConflictResolutionView(v.app, v.repo)
					v.app.Pages().Push(conflictView)
					return
				}
				ShowErrorModal(v.app, "Rebase Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Rebased onto %s", branch.Name))
			v.binding.RefreshAsync()
		})
}

func (v *BranchesView) toggleRemote() {
	v.showRemote.Update(func(b bool) bool { return !b })
}

func (v *BranchesView) copyBranchName() {
	branch := v.binding.GetSelected()
	if branch == nil {
		return
	}
	app.CopyBranchName(branch.Name)
}

func splitRemoteBranch(name string) []string {
	// Split "origin/main" into ["origin", "main"]
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			return []string{name[:i], name[i+1:]}
		}
	}
	return []string{name}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "..."
}

// core.Widget interface

func (v *BranchesView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *BranchesView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *BranchesView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *BranchesView) Blur()                         { v.flex.Blur() }
func (v *BranchesView) HasFocus() bool                { return v.flex.HasFocus() }

func (v *BranchesView) HandleKey(event *tcell.EventKey) bool {
	if v.actions.Handle(event) {
		return true
	}

	row, col := v.table.GetSelection()

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			if row < v.table.GetRowCount()-1 {
				v.table.Select(row+1, col)
			}
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			minRow := 1 // Skip header
			if row > minRow {
				v.table.Select(row-1, col)
			}
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			if row < v.table.GetRowCount()-1 {
				v.table.Select(row+1, col)
			}
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			minRow := 1 // Skip header
			if row > minRow {
				v.table.Select(row-1, col)
			}
			return true
		})

	return bindings.Handle(event)
}
