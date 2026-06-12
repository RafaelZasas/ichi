package views

import (
	"path/filepath"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/validators"

	"github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/config"
	"github.com/atterpac/ichi/internal/git"
)

// RepoSwitcherModal lists saved repositories and exposes CRUD keybinds plus
// switching to the selected repo. Mirrors tempo's profile switcher.
type RepoSwitcherModal struct {
	*components.Modal
	table     *components.Table
	app       *layout.App
	repo      *git.Repository
	statusBar *layout.StatusBar
	repos     []config.Repo
}

// NewRepoSwitcherModal builds the repo switcher modal.
func NewRepoSwitcherModal(a *layout.App, repo *git.Repository, statusBar *layout.StatusBar) *RepoSwitcherModal {
	m := &RepoSwitcherModal{
		Modal: components.NewModal(components.ModalConfig{
			Title:    "Repositories",
			Width:    70,
			Height:   20,
			Backdrop: true,
		}),
		table:     components.NewTable(),
		app:       a,
		repo:      repo,
		statusBar: statusBar,
	}
	m.setup()
	return m
}

func (m *RepoSwitcherModal) setup() {
	m.table.SetHeaders("", "NAME", "ALIAS", "PATH")
	m.table.ConfigureEmpty("", "No saved repositories", "Press 'n' to add the current repository")

	m.table.SetOnSelect(func(row int) {
		if row >= 0 && row < len(m.repos) {
			path := m.repos[row].Path
			m.app.Pages().Pop()
			if err := SwitchRepo(m.app, m.repo, m.statusBar, path); err != nil {
				ShowErrorModal(m.app, "Switch Failed", err.Error())
			} else {
				app.ToastSuccess("Switched to " + m.repos[row].Name)
			}
		}
	})

	m.Modal.SetContent(m.table)
	m.Modal.SetHints([]components.KeyHint{
		{Key: "Enter", Description: "Switch"},
		{Key: "n", Description: "New"},
		{Key: "e", Description: "Edit"},
		{Key: "d", Description: "Delete"},
		{Key: "Esc", Description: "Close"},
	})
	m.Modal.SetOnCancel(func() {
		m.app.Pages().Pop()
	})
}

func (m *RepoSwitcherModal) loadRepos() {
	m.repos = config.GetRepos()
	m.table.ClearRows()

	current := m.repo.Path()
	selectedIdx := 0
	for i, r := range m.repos {
		marker := " "
		if sameRepoPath(r.Path, current) {
			marker = "●"
			selectedIdx = i
		}
		m.table.AddRow(marker, r.Name, r.Alias, r.Path)
	}
	if len(m.repos) > 0 {
		m.table.SelectRow(selectedIdx)
	}
}

// Start loads the repo list when shown.
func (m *RepoSwitcherModal) Start() { m.loadRepos() }

// Stop is called when the modal is hidden.
func (m *RepoSwitcherModal) Stop() {}

// Hints returns key hints for the menu bar.
func (m *RepoSwitcherModal) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Switch"},
		{Key: "n", Description: "New"},
		{Key: "e", Description: "Edit"},
		{Key: "d", Description: "Delete"},
		{Key: "Esc", Description: "Close"},
	}
}

// HandleKey intercepts CRUD keys, then delegates to the table.
func (m *RepoSwitcherModal) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Rune() {
	case 'n':
		m.app.Pages().Pop()
		ShowRepoForm(m.app, m.repo, m.statusBar, config.Repo{}, newRepoPrefill(m.repo))
		return true
	case 'e':
		row := m.table.SelectedRow()
		if row >= 0 && row < len(m.repos) {
			existing := m.repos[row]
			m.app.Pages().Pop()
			ShowRepoForm(m.app, m.repo, m.statusBar, existing, existing)
		}
		return true
	case 'd':
		row := m.table.SelectedRow()
		if row >= 0 && row < len(m.repos) {
			name := m.repos[row].Name
			ShowConfirmModal(m.app, "Delete Repository", "Remove '"+name+"' from saved repos?", func() {
				config.DeleteRepo(name)
				app.ToastSuccess("Removed " + name)
				m.loadRepos()
			})
		}
		return true
	}
	return m.table.HandleKey(ev)
}

// ShowRepoSwitcher displays the repo switcher modal.
func ShowRepoSwitcher(a *layout.App, repo *git.Repository, statusBar *layout.StatusBar) {
	modal := NewRepoSwitcherModal(a, repo, statusBar)
	a.Pages().Push(modal)
	a.SetFocus(modal)
}

// ShowNewRepoForm opens the create-repo form prefilled with the current repo
// when it isn't already saved.
func ShowNewRepoForm(a *layout.App, repo *git.Repository, statusBar *layout.StatusBar) {
	ShowRepoForm(a, repo, statusBar, config.Repo{}, newRepoPrefill(repo))
}

// RepoFormModal is the create/edit form for a saved repository.
type RepoFormModal struct {
	*components.Modal
	form *components.Form
	app  *layout.App
}

// ShowRepoForm displays the new/edit repo form. When original.Name is empty the
// form creates a new repo; otherwise it edits (and may rename) that repo. The
// prefill values populate the fields.
func ShowRepoForm(a *layout.App, repo *git.Repository, statusBar *layout.StatusBar, original config.Repo, prefill config.Repo) {
	isEdit := original.Name != ""
	title := "New Repository"
	if isEdit {
		title = "Edit Repository: " + original.Name
	}

	m := &RepoFormModal{
		Modal: components.NewModal(components.ModalConfig{
			Title:    title,
			Width:    70,
			Height:   14,
			Backdrop: true,
		}),
		app: a,
	}

	builder := components.NewFormBuilder()
	builder.Text("name", "Name").
		Placeholder("my-project").
		Value(prefill.Name).
		Validate(validators.Required(), validators.MinLength(1)).
		Done()
	builder.Text("alias", "Alias (optional)").
		Placeholder("mp").
		Value(prefill.Alias).
		Done()
	builder.Text("path", "Path").
		Placeholder("/path/to/repo").
		Value(prefill.Path).
		Validate(validators.Required()).
		Done()

	builder.OnSubmit(func(values map[string]any) {
		name, _ := values["name"].(string)
		alias, _ := values["alias"].(string)
		path, _ := values["path"].(string)
		if name == "" || path == "" {
			return
		}
		r := config.Repo{Name: name, Path: path, Alias: alias}
		config.SaveRepo(original.Name, r)
		a.Pages().Pop()
		app.ToastSuccess("Saved " + name)
		if err := SwitchRepo(a, repo, statusBar, path); err != nil {
			ShowErrorModal(a, "Switch Failed", err.Error())
		}
	})
	builder.OnCancel(func() {
		a.Pages().Pop()
	})

	m.form = builder.Build()
	m.Modal.SetContent(m.form)
	m.Modal.SetHints([]components.KeyHint{
		{Key: "Tab", Description: "Next field"},
		{Key: "Ctrl+S", Description: "Save"},
		{Key: "Esc", Description: "Cancel"},
	})
	m.Modal.SetOnCancel(func() {
		a.Pages().Pop()
	})

	a.Pages().Push(m)
	a.SetFocus(m)
}

// HandleKey delegates to the form's key handler.
func (m *RepoFormModal) HandleKey(ev *tcell.EventKey) bool {
	return m.form.HandleKey(ev)
}

// Start/Stop satisfy the component lifecycle.
func (m *RepoFormModal) Start() {}
func (m *RepoFormModal) Stop()  {}

// SwitchRepo re-points the repository at path and rebuilds the graph view.
func SwitchRepo(a *layout.App, repo *git.Repository, statusBar *layout.StatusBar, path string) error {
	if err := repo.SetPath(path); err != nil {
		return err
	}
	for a.Pages().CanPop() {
		a.Pages().Pop()
	}
	graphView := NewGraphView(a, repo)
	a.Pages().Replace(graphView)
	a.Crumbs().SetPath([]string{"Graph"})
	if statusBar != nil {
		app.UpdateStatusBar(statusBar, repo)
	}
	return nil
}

// newRepoPrefill returns prefill values for a new-repo form: the current repo's
// details when it isn't already saved, otherwise an empty form.
func newRepoPrefill(repo *git.Repository) config.Repo {
	current := repo.Path()
	for _, r := range config.GetRepos() {
		if sameRepoPath(r.Path, current) {
			return config.Repo{}
		}
	}
	return config.Repo{Name: filepath.Base(current), Path: current}
}

// sameRepoPath reports whether two repo paths point at the same directory.
func sameRepoPath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return absA == absB
}
