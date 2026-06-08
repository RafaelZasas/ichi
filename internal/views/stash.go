package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/binding"
	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/input"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/git"
	"github.com/atterpac/ichi/internal/selection"
)

// StashView displays the stash list.
type StashView struct {
	flex    *core.Flex
	table   *components.Table
	repo    *git.Repository
	app     *layout.App
	binding *binding.TableBinding[git.Stash]
	actions *input.ActionRegistry
}

// NewStashView creates a new stash view.
func NewStashView(app *layout.App, repo *git.Repository) *StashView {
	v := &StashView{
		flex:  core.NewFlex(),
		table: components.NewTable(),
		repo:  repo,
		app:   app,
	}
	v.setup()
	return v
}

func (v *StashView) setup() {
	v.table.SetHeaders("Index", "Branch", "Message")
	v.table.SetSelectable(true, false)

	panel := components.NewPanel().
		SetTitle("Stash List").
		SetContent(v.table)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)

	// Setup binding
	v.binding = binding.NewTableBinding[git.Stash](v.table).
		SetMapper(func(s git.Stash) []string {
			return []string{
				fmt.Sprintf("stash@{%d}", s.Index),
				s.Branch,
				s.Message,
			}
		}).
		SetKeyMapper(func(s git.Stash) string {
			return fmt.Sprintf("%d", s.Index)
		}).
		SetFetcher(v.repo.ListStashes).
		SetOnSelect(func(s git.Stash) {
			// When Enter is pressed, view the diff
			v.viewDiff()
		}).
		SetOnRefresh(func(_ []git.Stash, err error) {
			if err != nil {
				ShowErrorModal(v.app, "Error", err.Error())
			}
		})

	v.actions = input.NewActionRegistry().
		AddSimple("apply", 'a', "Apply", v.applyStash).
		AddSimple("pop", 'p', "Pop", v.popStash).
		AddSimple("drop", 'd', "Drop", v.dropStash).
		AddSimple("new", 'n', "New stash", v.newStash).
		AddSimple("diff", 'D', "View diff", v.viewDiff).
		AddSimple("branch", 'b', "Create branch", v.createBranch).
		AddSimple("refresh", 'r', "Refresh", func() { v.binding.RefreshAsync() })
}

// nav.Component interface

// Selection implements selection.Provider.
func (v *StashView) Selection() *selection.Context {
	sel := &selection.Context{ViewName: v.Name()}
	if stash := v.binding.GetSelected(); stash != nil {
		sel.Stash = stash
	}
	return sel
}

func (v *StashView) Name() string {
	return "Stash"
}

func (v *StashView) Start() {
	v.binding.Start()
}

func (v *StashView) Stop() {
	v.binding.Stop()
}

func (v *StashView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "View diff"},
		{Key: "a", Description: "Apply"},
		{Key: "p", Description: "Pop"},
		{Key: "d", Description: "Drop"},
		{Key: "n", Description: "New"},
	}
}

func (v *StashView) applyStash() {
	stash := v.binding.GetSelected()
	if stash == nil {
		return
	}

	ShowConfirmModal(v.app, "Apply Stash",
		fmt.Sprintf("Apply stash@{%d}?\n\n%s", stash.Index, stash.Message),
		func() {
			if err := v.repo.StashApplyIndex(stash.Index); err != nil {
				ShowErrorModal(v.app, "Apply Failed", err.Error())
				return
			}
			app.ToastSuccess("Stash applied")
		})
}

func (v *StashView) popStash() {
	stash := v.binding.GetSelected()
	if stash == nil {
		return
	}

	ShowConfirmModal(v.app, "Pop Stash",
		fmt.Sprintf("Pop stash@{%d}?\n\nThis will apply and remove the stash.\n\n%s", stash.Index, stash.Message),
		func() {
			if err := v.repo.StashPopIndex(stash.Index); err != nil {
				ShowErrorModal(v.app, "Pop Failed", err.Error())
				return
			}
			app.ToastSuccess("Stash popped")
			v.binding.RefreshAsync()
		})
}

func (v *StashView) dropStash() {
	stash := v.binding.GetSelected()
	if stash == nil {
		return
	}

	ShowConfirmModal(v.app, "Drop Stash",
		fmt.Sprintf("Drop stash@{%d}?\n\nThis cannot be undone.\n\n%s", stash.Index, stash.Message),
		func() {
			if err := v.repo.StashDropIndex(stash.Index); err != nil {
				ShowErrorModal(v.app, "Drop Failed", err.Error())
				return
			}
			app.ToastSuccess("Stash dropped")
			v.binding.RefreshAsync()
		})
}

func (v *StashView) newStash() {
	// Check for uncommitted changes
	if !v.repo.HasUncommitted() {
		ShowErrorModal(v.app, "No Changes", "No changes to stash")
		return
	}

	ShowInputModal(v.app, "New Stash", "Stash message (optional):", func(message string) {
		if err := v.repo.StashPush(message, true); err != nil {
			ShowErrorModal(v.app, "Stash Failed", err.Error())
			return
		}
		app.ToastSuccess("Changes stashed")
		v.binding.RefreshAsync()
	})
}

func (v *StashView) viewDiff() {
	stash := v.binding.GetSelected()
	if stash == nil {
		return
	}

	diff, err := v.repo.StashShow(stash.Index)
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}

	diffView := NewStashDiffView(v.app, stash.Index, diff)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Stash", fmt.Sprintf("stash@{%d}", stash.Index)})
}

func (v *StashView) createBranch() {
	stash := v.binding.GetSelected()
	if stash == nil {
		return
	}

	ShowInputModalWithValidator(v.app, "Create Branch from Stash", "Branch name:",
		app.BranchNameValidator(),
		func(name string) {
			if name == "" {
				return
			}
			if err := v.repo.StashBranch(name, stash.Index); err != nil {
				ShowErrorModal(v.app, "Create Branch Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Branch '%s' created", name))
			v.binding.RefreshAsync()
		})
}

// core.Widget interface

func (v *StashView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *StashView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *StashView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *StashView) Blur()                         { v.flex.Blur() }
func (v *StashView) HasFocus() bool                { return v.flex.HasFocus() }

func (v *StashView) HandleKey(event *tcell.EventKey) bool {
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
			if row > 1 {
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
			if row > 1 {
				v.table.Select(row-1, col)
			}
			return true
		})

	return bindings.Handle(event)
}

// StashDiffView displays a stash diff.
type StashDiffView struct {
	flex     *core.Flex
	diffView *core.TextView
	app      *layout.App
	index    int
	content  []string
}

// NewStashDiffView creates a diff view for a stash.
func NewStashDiffView(app *layout.App, index int, diff string) *StashDiffView {
	v := &StashDiffView{
		flex:     core.NewFlex(),
		diffView: core.NewTextView(),
		app:      app,
		index:    index,
	}

	v.diffView.SetDynamicColors(true)
	v.diffView.SetWordWrap(false)
	v.diffView.SetScrollable(true)
	v.diffView.SetBackgroundColor(theme.Bg())

	panel := components.NewPanel().
		SetTitle(fmt.Sprintf("Stash Diff: stash@{%d}", index)).
		SetContent(v.diffView)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)

	// Render diff
	v.renderDiff(diff)

	return v
}

func (v *StashDiffView) renderDiff(diff string) {
	var sb strings.Builder

	for _, line := range strings.Split(diff, "\n") {
		v.content = append(v.content, line)
		escaped := core.EscapeMarkup(line)

		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			sb.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagFgDim(), escaped))
		case strings.HasPrefix(line, "@@"):
			sb.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), escaped))
		case strings.HasPrefix(line, "+"):
			sb.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagSuccess(), escaped))
		case strings.HasPrefix(line, "-"):
			sb.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagError(), escaped))
		case strings.HasPrefix(line, "diff --git"):
			sb.WriteString(fmt.Sprintf("\n[%s::b]%s[-:-:-]\n", theme.TagAccent(), escaped))
		default:
			sb.WriteString(escaped + "\n")
		}
	}

	v.diffView.SetText(sb.String())
}

func (v *StashDiffView) Name() string {
	return "Stash Diff"
}

func (v *StashDiffView) Start() {}
func (v *StashDiffView) Stop()  {}

func (v *StashDiffView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Scroll"},
		{Key: "g/G", Description: "Top/Bottom"},
		{Key: "Esc", Description: "Back"},
	}
}

func (v *StashDiffView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *StashDiffView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *StashDiffView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *StashDiffView) Blur()                         { v.flex.Blur() }
func (v *StashDiffView) HasFocus() bool                { return v.flex.HasFocus() }

func (v *StashDiffView) HandleKey(event *tcell.EventKey) bool {
	row, col := v.diffView.GetScrollOffset()

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			v.diffView.ScrollTo(row+1, col)
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			if row > 0 {
				v.diffView.ScrollTo(row-1, col)
			}
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			v.diffView.ScrollTo(row+1, col)
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			if row > 0 {
				v.diffView.ScrollTo(row-1, col)
			}
			return true
		}).
		OnRune('g', func(e *tcell.EventKey) bool {
			v.diffView.ScrollTo(0, col)
			return true
		}).
		OnRune('G', func(e *tcell.EventKey) bool {
			v.diffView.ScrollTo(len(v.content), col)
			return true
		})

	return bindings.Handle(event)
}
