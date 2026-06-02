package views

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/layout"

	"github.com/atterpac/gxt/internal/app"
	"github.com/atterpac/gxt/internal/git"
)

// FinderModal provides a command palette / fuzzy finder using dado's modal.
type FinderModal struct {
	*components.Modal
	finder    *components.Finder
	app       *layout.App
	repo      *git.Repository
	statusBar *layout.StatusBar
}

// NewFinderModal creates a new finder modal.
func NewFinderModal(app *layout.App, repo *git.Repository, statusBar *layout.StatusBar) *FinderModal {
	m := &FinderModal{
		Modal: components.NewModal(components.ModalConfig{
			Title:    "Command Palette",
			Width:    70,
			Height:   22,
			Backdrop: true,
		}),
		finder:    components.NewFinder(),
		app:       app,
		repo:      repo,
		statusBar: statusBar,
	}
	m.setup()
	return m
}

func (m *FinderModal) setup() {
	m.finder.SetPlaceholder("Search branches, commands...")
	m.finder.SetPrompt("> ")
	m.finder.SetMaxVisible(12)
	m.finder.SetMinScore(1) // Only show items that actually match
	m.finder.SetShowCategories(true)
	m.finder.SetShowDescription(true)
	m.finder.SetBorder(false)

	m.finder.SetCategories([]components.FinderCategory{
		{Name: "Recent", Priority: 0},
		{Name: "Branches", Priority: 1},
		{Name: "Remote Branches", Priority: 2},
		{Name: "Commands", Priority: 3},
		{Name: "Navigation", Priority: 4},
	})

	m.finder.SetOnSelect(func(item components.FinderItem) {
		m.app.Pages().Pop()
		if item.Data != nil {
			if action, ok := item.Data.(func()); ok {
				action()
			}
		}
	})

	m.finder.SetOnCancel(func() {
		m.app.Pages().Pop()
	})

	m.Modal.SetContent(m.finder)
	m.Modal.SetHints([]components.KeyHint{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "Esc", Description: "Cancel"},
	})
	m.Modal.SetOnCancel(func() {
		m.app.Pages().Pop()
	})
}

func (m *FinderModal) loadItems() {
	items := make([]components.FinderItem, 0)

	// Add all branches (local and remote)
	branches, err := m.repo.ListBranches()
	if err == nil {
		for _, branch := range branches {
			branchCopy := branch
			category := "Branches"
			if branch.IsRemote {
				category = "Remote Branches"
			}
			items = append(items, components.FinderItem{
				ID:          "branch:" + branch.Name,
				Label:       branch.Name,
				Description: branch.LastMsg,
				Category:    category,
				Keywords:    []string{branch.Name, "branch", "checkout"},
				Data: func() {
					// For remote branches like "origin/feature", strip the remote prefix
					// so git creates/switches to a local tracking branch
					checkoutName := branchCopy.Name
					if branchCopy.IsRemote {
						if idx := strings.Index(checkoutName, "/"); idx != -1 {
							checkoutName = checkoutName[idx+1:]
						}
					}
					if err := m.repo.Checkout(checkoutName); err != nil {
						ShowErrorModal(m.app, "Checkout Failed", err.Error())
					} else {
						app.ToastSuccess("Switched to " + checkoutName)
						// Update status bar to reflect new branch
						if m.statusBar != nil {
							app.UpdateStatusBar(m.statusBar, m.repo)
						}
					}
				},
			})
		}
	}

	// Add navigation commands
	navCommands := []struct {
		id, label, desc string
		action          func()
	}{
		{"nav:graph", "Graph View", "Show commit graph", func() {
			graphView := NewGraphView(m.app, m.repo)
			m.app.Pages().Replace(graphView)
			m.app.Crumbs().SetPath([]string{"Graph"})
		}},
		{"nav:branches", "Branches View", "Manage branches", func() {
			branchView := NewBranchesView(m.app, m.repo)
			m.app.Pages().Push(branchView)
			m.app.Crumbs().SetPath([]string{"Branches"})
		}},
		{"nav:status", "Status View", "View working tree status", func() {
			statusView := NewStatusView(m.app, m.repo)
			m.app.Pages().Push(statusView)
			m.app.Crumbs().SetPath([]string{"Status"})
		}},
		{"nav:stash", "Stash View", "Manage stashes", func() {
			stashView := NewStashView(m.app, m.repo)
			m.app.Pages().Push(stashView)
			m.app.Crumbs().SetPath([]string{"Stash"})
		}},
		{"nav:prs", "Pull Requests", "View pull requests", func() {
			prView := NewPRListView(m.app, m.repo)
			m.app.Pages().Push(prView)
			m.app.Crumbs().SetPath([]string{"PRs"})
		}},
	}

	for _, cmd := range navCommands {
		cmdCopy := cmd
		items = append(items, components.FinderItem{
			ID:          cmdCopy.id,
			Label:       cmdCopy.label,
			Description: cmdCopy.desc,
			Category:    "Navigation",
			Data:        cmdCopy.action,
		})
	}

	// Add git commands
	gitCommands := []struct {
		id, label, desc string
		action          func()
	}{
		{"cmd:commit", "Commit", "Commit staged changes", func() {
			ShowCommitModal(m.app, "Commit", "", func(message string) {
				if err := m.repo.Commit(message); err != nil {
					ShowErrorModal(m.app, "Commit Failed", err.Error())
				} else {
					app.ToastSuccess("Changes committed")
				}
			})
		}},
		{"cmd:stash-push", "Stash Push", "Stash current changes", func() {
			ShowInputModal(m.app, "Stash", "Stash message (optional):", func(message string) {
				if err := m.repo.StashPush(message, true); err != nil {
					ShowErrorModal(m.app, "Stash Failed", err.Error())
				} else {
					app.ToastSuccess("Changes stashed")
				}
			})
		}},
		{"cmd:stash-pop", "Stash Pop", "Pop the latest stash", func() {
			if err := m.repo.StashPop(); err != nil {
				ShowErrorModal(m.app, "Stash Pop Failed", err.Error())
			} else {
				app.ToastSuccess("Stash popped")
			}
		}},
		{"cmd:fetch", "Fetch", "Fetch from remote", func() {
			if err := m.repo.Fetch("origin"); err != nil {
				ShowErrorModal(m.app, "Fetch Failed", err.Error())
			} else {
				app.ToastSuccess("Fetched from origin")
			}
		}},
		{"cmd:pull", "Pull", "Pull from remote", func() {
			if err := m.repo.Pull(); err != nil {
				ShowErrorModal(m.app, "Pull Failed", err.Error())
			} else {
				app.ToastSuccess("Pulled from remote")
			}
		}},
		{"cmd:push", "Push", "Push to remote", func() {
			if err := m.repo.Push(); err != nil {
				ShowErrorModal(m.app, "Push Failed", err.Error())
			} else {
				app.ToastSuccess("Pushed to remote")
			}
		}},
		{"cmd:new-branch", "New Branch", "Create a new branch", func() {
			ShowInputModalWithValidator(m.app, "New Branch", "Branch name:",
				app.BranchNameValidator(),
				func(name string) {
					if name == "" {
						return
					}
					if err := m.repo.CreateBranch(name); err != nil {
						ShowErrorModal(m.app, "Create Failed", err.Error())
					} else {
						app.ToastSuccess("Branch '" + name + "' created")
					}
				})
		}},
		{"cmd:merge", "Merge Branch", "Merge a branch into current", func() {
			branchView := NewBranchesView(m.app, m.repo)
			m.app.Pages().Push(branchView)
			app.ToastInfo("Select a branch and press 'm' to merge")
		}},
		{"cmd:rebase", "Rebase Branch", "Rebase current onto another", func() {
			branchView := NewBranchesView(m.app, m.repo)
			m.app.Pages().Push(branchView)
			app.ToastInfo("Select a branch and press 'b' to rebase")
		}},
	}

	for _, cmd := range gitCommands {
		cmdCopy := cmd
		items = append(items, components.FinderItem{
			ID:          cmdCopy.id,
			Label:       cmdCopy.label,
			Description: cmdCopy.desc,
			Category:    "Commands",
			Keywords:    []string{"git", cmdCopy.label},
			Data:        cmdCopy.action,
		})
	}

	m.finder.SetItems(items)
}

// Start loads items when the modal is shown.
func (m *FinderModal) Start() {
	m.loadItems()
}

// Stop is called when the modal is hidden.
func (m *FinderModal) Stop() {}

// Hints returns key hints for the menu bar.
func (m *FinderModal) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "Esc", Description: "Cancel"},
	}
}

// HandleKey delegates to finder's key handler.
func (m *FinderModal) HandleKey(ev *tcell.EventKey) bool {
	return m.finder.HandleKey(ev)
}

// ShowFinder displays the command palette modal.
func ShowFinder(app *layout.App, repo *git.Repository, statusBar *layout.StatusBar) {
	modal := NewFinderModal(app, repo, statusBar)
	app.Pages().Push(modal)
	app.SetFocus(modal)
}
