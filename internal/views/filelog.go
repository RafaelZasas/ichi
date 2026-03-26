package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/input"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/git"
	"github.com/atterpac/gxt/internal/selection"
)

// FileLogView displays commits that affected a specific file.
type FileLogView struct {
	flex       *tview.Flex
	split      *components.Split
	logTable   *tview.Table
	detailView *tview.TextView
	repo       *git.Repository
	app        *layout.App
	file       string
	entries    []git.FileLogEntry
	actions    *input.ActionRegistry
}

// NewFileLogView creates a new file log view.
func NewFileLogView(app *layout.App, repo *git.Repository, file string) *FileLogView {
	v := &FileLogView{
		flex:       tview.NewFlex(),
		logTable:   tview.NewTable(),
		detailView: tview.NewTextView(),
		repo:       repo,
		app:        app,
		file:       file,
	}
	v.setup()
	return v
}

func (v *FileLogView) setup() {
	// Configure log table
	v.logTable.SetBorders(false)
	v.logTable.SetSelectable(true, false)
	v.logTable.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.Accent()).
		Foreground(theme.Bg()))
	v.logTable.SetBackgroundColor(theme.Bg())
	// Note: SetSelectionChangedFunc is set after data loads to avoid deadlock
	theme.Register(v.logTable)

	// Configure detail view
	v.detailView.SetDynamicColors(true).SetWordWrap(true)
	v.detailView.SetBackgroundColor(theme.Bg())
	theme.Register(v.detailView)

	// Wrap in panels
	logPanel := components.NewPanel().
		SetTitle(fmt.Sprintf("History: %s", v.file)).
		SetContent(v.logTable)

	detailPanel := components.NewPanel().
		SetTitle("Commit Details").
		SetContent(v.detailView)

	// Two-panel layout (60/40 split)
	v.split = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.6).
		SetShowDivider(false).
		SetLeft(logPanel).
		SetRight(detailPanel)

	v.flex.SetDirection(tview.FlexRow)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(v.split, 0, 1, true)
	theme.Register(v.flex)

	// Register actions
	v.actions = input.NewActionRegistry().
		AddKey("select", tcell.KeyEnter, "View commit", v.showCommit).
		AddSimple("diff", 'd', "Full diff", v.showDiff).
		AddSimple("blame", 'b', "Blame at commit", v.showBlameAtCommit).
		AddSimple("checkout", 'c', "Checkout", v.checkout).
		AddSimple("copy_hash", 'y', "Copy hash", v.copyHash)
}

// nav.Component interface implementation

// Selection implements selection.Provider.
func (v *FileLogView) Selection() *selection.Context {
	sel := &selection.Context{ViewName: v.Name()}
	row, _ := v.logTable.GetSelection()
	if row >= 0 && row < len(v.entries) {
		entry := v.entries[row]
		sel.Commit = &components.GitCommit{
			Hash:      entry.Hash,
			ShortHash: entry.ShortHash,
			Message:   entry.Subject,
			Author:    entry.Author,
		}
		sel.CommitHash = entry.Hash
	}
	return sel
}

func (v *FileLogView) Name() string {
	return "File History"
}

func (v *FileLogView) Start() {
	v.loadLog()
}

func (v *FileLogView) Stop() {}

func (v *FileLogView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "View commit"},
		{Key: "d", Description: "Diff"},
		{Key: "b", Description: "Blame"},
		{Key: "c", Description: "Checkout"},
		{Key: "Esc", Description: "Back"},
	}
}

// Business logic

func (v *FileLogView) loadLog() {
	entries, err := v.repo.FileLog(v.file, 100)
	if err != nil {
		v.showError(err)
		return
	}

	v.entries = entries
	v.renderLog()

	// Select first entry and update detail
	if len(v.entries) > 0 {
		v.logTable.Select(0, 0)
		v.updateDetail(v.entries[0])
	}
}

func (v *FileLogView) renderLog() {
	v.logTable.Clear()

	for i, entry := range v.entries {
		// Graph marker
		marker := "●"
		if i == len(v.entries)-1 {
			marker = "○" // Initial commit marker
		}

		// Hash cell
		hashCell := tview.NewTableCell(fmt.Sprintf("[%s]%s[-]", theme.TagAccent(), entry.ShortHash)).
			SetAlign(tview.AlignLeft)
		v.logTable.SetCell(i, 0, hashCell)

		// Marker cell (graph)
		markerCell := tview.NewTableCell(fmt.Sprintf("[%s]%s[-]", theme.TagInfo(), marker)).
			SetAlign(tview.AlignCenter)
		v.logTable.SetCell(i, 1, markerCell)

		// Stats cell
		statsCell := tview.NewTableCell(fmt.Sprintf("[%s]+%d[-] [%s]-%d[-]",
			theme.TagSuccess(), entry.Insertions,
			theme.TagError(), entry.Deletions)).
			SetAlign(tview.AlignRight)
		v.logTable.SetCell(i, 2, statsCell)

		// Subject cell (truncate if needed)
		subject := entry.Subject
		if len(subject) > 50 {
			subject = subject[:47] + "..."
		}
		subjectCell := tview.NewTableCell(subject).
			SetExpansion(1).
			SetAlign(tview.AlignLeft)
		v.logTable.SetCell(i, 3, subjectCell)

		// Author cell
		author := entry.Author
		if len(author) > 15 {
			author = author[:14] + "…"
		}
		authorCell := tview.NewTableCell(fmt.Sprintf("[%s]%s[-]", theme.TagFgDim(), author)).
			SetAlign(tview.AlignLeft)
		v.logTable.SetCell(i, 4, authorCell)

		// Date cell
		dateCell := tview.NewTableCell(fmt.Sprintf("[%s]%s[-]", theme.TagFgDim(), entry.Date)).
			SetAlign(tview.AlignRight)
		v.logTable.SetCell(i, 5, dateCell)
	}
}

func (v *FileLogView) updateDetail(entry git.FileLogEntry) {
	var text strings.Builder

	// Subject
	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n\n", theme.TagAccent(), entry.Subject))

	// Commit info
	text.WriteString(fmt.Sprintf("[%s::b]─── Commit ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Hash:[-]   %s\n", theme.TagFgDim(), entry.Hash))
	text.WriteString(fmt.Sprintf("[%s]Short:[-]  %s\n", theme.TagFgDim(), entry.ShortHash))

	// Author
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Author ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Name:[-]   %s\n", theme.TagFgDim(), entry.Author))
	text.WriteString(fmt.Sprintf("[%s]Date:[-]   %s\n", theme.TagFgDim(), entry.Date))

	// Changes
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Changes ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]+%d[-] [%s]-%d[-] in %s\n",
		theme.TagSuccess(), entry.Insertions,
		theme.TagError(), entry.Deletions,
		v.file))

	// Actions hint
	text.WriteString(fmt.Sprintf("\n[%s]Press Enter to view full commit[-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Press 'd' for full diff[-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Press 'b' to blame at this commit[-]\n", theme.TagFgDim()))

	v.detailView.SetText(text.String())
}

func (v *FileLogView) showCommit() {
	row, _ := v.logTable.GetSelection()
	if row < 0 || row >= len(v.entries) {
		return
	}

	entry := v.entries[row]
	commitView := NewCommitView(v.app, v.repo, entry.Hash)
	v.app.Pages().Push(commitView)
	v.app.Crumbs().SetPath([]string{"History", v.file, entry.ShortHash})
}

func (v *FileLogView) showDiff() {
	row, _ := v.logTable.GetSelection()
	if row < 0 || row >= len(v.entries) {
		return
	}

	entry := v.entries[row]
	diffView := NewFileDiffView(v.app, v.repo, entry.Hash, v.file)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"History", "Diff"})
}

func (v *FileLogView) showBlameAtCommit() {
	row, _ := v.logTable.GetSelection()
	if row < 0 || row >= len(v.entries) {
		return
	}

	entry := v.entries[row]
	blameView := NewBlameViewAtCommit(v.app, v.repo, v.file, entry.Hash)
	v.app.Pages().Push(blameView)
	v.app.Crumbs().SetPath([]string{"History", "Blame", entry.ShortHash})
}

func (v *FileLogView) checkout() {
	row, _ := v.logTable.GetSelection()
	if row < 0 || row >= len(v.entries) {
		return
	}

	entry := v.entries[row]
	ShowConfirmModal(v.app, "Checkout",
		fmt.Sprintf("Checkout commit %s?\n\n%s", entry.ShortHash, entry.Subject),
		func() {
			if err := v.repo.Checkout(entry.Hash); err != nil {
				ShowErrorModal(v.app, "Checkout Failed", err.Error())
				return
			}
		})
}

func (v *FileLogView) copyHash() {
	row, _ := v.logTable.GetSelection()
	if row < 0 || row >= len(v.entries) {
		return
	}
	// TODO: Implement clipboard copy
}

func (v *FileLogView) showError(err error) {
	v.detailView.SetText(fmt.Sprintf("[%s]Error:[-] %v", theme.TagError(), err))
}

// tview.Primitive delegation

func (v *FileLogView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *FileLogView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *FileLogView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *FileLogView) Focus(d func(tview.Primitive)) { v.flex.Focus(d) }
func (v *FileLogView) Blur()                         { v.flex.Blur() }
func (v *FileLogView) HasFocus() bool                { return true }

func (v *FileLogView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.flex.MouseHandler()
}

func (v *FileLogView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *FileLogView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		if v.actions.Handle(event) {
			return
		}

		if len(v.entries) == 0 {
			return
		}

		row, col := v.logTable.GetSelection()
		oldRow := row

		updateRow := func(newRow int) {
			if newRow != oldRow {
				v.logTable.Select(newRow, col)
				v.updateDetail(v.entries[newRow])
			}
		}

		bindings := input.NewKeyBindings().
			On(tcell.KeyDown, func(e *tcell.EventKey) bool {
				if row < len(v.entries)-1 {
					updateRow(row + 1)
				}
				return true
			}).
			On(tcell.KeyUp, func(e *tcell.EventKey) bool {
				if row > 0 {
					updateRow(row - 1)
				}
				return true
			}).
			On(tcell.KeyPgDn, func(e *tcell.EventKey) bool {
				_, _, _, height := v.logTable.GetInnerRect()
				newRow := row + height/2
				if newRow >= len(v.entries) {
					newRow = len(v.entries) - 1
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyCtrlD, func(e *tcell.EventKey) bool {
				_, _, _, height := v.logTable.GetInnerRect()
				newRow := row + height/2
				if newRow >= len(v.entries) {
					newRow = len(v.entries) - 1
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyPgUp, func(e *tcell.EventKey) bool {
				_, _, _, height := v.logTable.GetInnerRect()
				newRow := row - height/2
				if newRow < 0 {
					newRow = 0
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyCtrlU, func(e *tcell.EventKey) bool {
				_, _, _, height := v.logTable.GetInnerRect()
				newRow := row - height/2
				if newRow < 0 {
					newRow = 0
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyHome, func(e *tcell.EventKey) bool {
				updateRow(0)
				return true
			}).
			On(tcell.KeyEnd, func(e *tcell.EventKey) bool {
				updateRow(len(v.entries) - 1)
				return true
			}).
			OnRune('j', func(e *tcell.EventKey) bool {
				if row < len(v.entries)-1 {
					updateRow(row + 1)
				}
				return true
			}).
			OnRune('k', func(e *tcell.EventKey) bool {
				if row > 0 {
					updateRow(row - 1)
				}
				return true
			}).
			OnRune('g', func(e *tcell.EventKey) bool {
				updateRow(0)
				return true
			}).
			OnRune('G', func(e *tcell.EventKey) bool {
				updateRow(len(v.entries) - 1)
				return true
			})

		bindings.Handle(event)
	}
}
