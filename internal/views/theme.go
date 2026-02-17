package views

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"
	"github.com/atterpac/jig/theme/themes"

	"github.com/atterpac/gxt/internal/config"
)

// ThemeSelectorModal allows selecting themes in a modal dialog.
type ThemeSelectorModal struct {
	*components.Modal
	table       *components.Table
	themes      []string
	currentIdx  int
	originalIdx int
	onSelect    func(string)
	onCancel    func()
}

// NewThemeSelectorModal creates a new theme selector modal.
func NewThemeSelectorModal() *ThemeSelectorModal {
	m := &ThemeSelectorModal{
		Modal: components.NewModal(components.ModalConfig{
			Title:    "Select Theme",
			Width:    45,
			Height:   20,
			Backdrop: true,
		}),
	}
	m.setup()
	return m
}

func (m *ThemeSelectorModal) setup() {
	m.table = components.NewTable()
	m.table.SetHeaders("", "Theme")
	m.table.SetBorder(false)
	m.table.SetSelectable(true, false)

	// Get themes from jig
	m.themes = themes.Names()
	currentTheme := config.GetTheme()

	// Populate table and find current theme index
	for i, name := range m.themes {
		m.table.AddRow(" ", name)
		if name == currentTheme {
			m.originalIdx = i
			m.currentIdx = i
		}
	}

	// Select current theme row (add 1 for header)
	if len(m.themes) > 0 {
		m.table.Select(m.originalIdx+1, 0)
	}

	// Live preview on selection change
	m.table.SetSelectionChangedFunc(func(row, col int) {
		idx := row - 1 // Account for header
		if idx >= 0 && idx < len(m.themes) {
			m.currentIdx = idx
			m.applyTheme(m.themes[idx])
		}
	})

	// Confirm selection on Enter
	m.table.SetOnSelect(func(row int) {
		idx := row - 1
		if idx >= 0 && idx < len(m.themes) {
			if m.onSelect != nil {
				m.onSelect(m.themes[idx])
			}
		}
	})

	m.Modal.SetContent(m.table)
	m.Modal.SetHints([]components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "Esc", Description: "Cancel"},
	})

	// Restore original theme on cancel
	m.Modal.SetOnCancel(func() {
		if m.originalIdx >= 0 && m.originalIdx < len(m.themes) {
			m.applyTheme(m.themes[m.originalIdx])
		}
		if m.onCancel != nil {
			m.onCancel()
		}
	})
}

func (m *ThemeSelectorModal) applyTheme(name string) {
	if t := themes.Get(name); t != nil {
		theme.SetProvider(t)
	}
}

// SetOnSelect sets the callback for when a theme is selected.
func (m *ThemeSelectorModal) SetOnSelect(fn func(string)) {
	m.onSelect = fn
}

// SetOnCancel sets the callback for when selection is cancelled.
func (m *ThemeSelectorModal) SetOnCancel(fn func()) {
	m.onCancel = fn
}

// Focus delegates focus to the table.
func (m *ThemeSelectorModal) Focus(delegate func(tview.Primitive)) {
	delegate(m.table)
}

// InputHandler handles keyboard input.
func (m *ThemeSelectorModal) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return m.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		row, col := m.table.GetSelection()

		switch event.Rune() {
		case 'j':
			if row < m.table.GetRowCount()-1 {
				m.table.Select(row+1, col)
			}
			return
		case 'k':
			if row > 1 { // Skip header
				m.table.Select(row-1, col)
			}
			return
		}

		switch event.Key() {
		case tcell.KeyDown:
			if row < m.table.GetRowCount()-1 {
				m.table.Select(row+1, col)
			}
			return
		case tcell.KeyUp:
			if row > 1 {
				m.table.Select(row-1, col)
			}
			return
		case tcell.KeyEnter:
			idx := row - 1
			if idx >= 0 && idx < len(m.themes) && m.onSelect != nil {
				m.onSelect(m.themes[idx])
			}
			return
		}

		// Let modal handle Escape
		if handler := m.Modal.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

// ShowThemeSelector displays the theme selector modal.
func ShowThemeSelector(app *layout.App) {
	modal := NewThemeSelectorModal()
	modal.SetOnSelect(func(themeName string) {
		config.SetTheme(themeName)
		app.Pages().Pop()
	})
	modal.SetOnCancel(func() {
		app.Pages().Pop()
	})
	app.Pages().Push(modal)
	app.SetFocus(modal)
}
