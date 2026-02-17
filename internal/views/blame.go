package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/input"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/git"
)

// BlameView displays line-by-line file attribution.
type BlameView struct {
	flex        *tview.Flex
	split       *components.Split
	blameTable  *tview.Table
	detailView  *tview.TextView
	repo        *git.Repository
	app         *layout.App
	file        string
	hash        string // Optional: blame at specific commit
	lines       []git.BlameLine
	actions     *input.ActionRegistry
	authorColor map[string]tcell.Color
	colorIndex  int
}

// blameColors returns theme-aware colors for author distinction.
// Called dynamically to support theme switching.
func blameColors() []tcell.Color {
	return []tcell.Color{
		theme.Accent(),
		theme.Success(),
		theme.Warning(),
		theme.Error(),
		theme.Info(),
		theme.Highlight(),
		theme.AccentDim(),
		theme.FgDim(),
	}
}

// NewBlameView creates a new blame view for a file.
func NewBlameView(app *layout.App, repo *git.Repository, file string) *BlameView {
	v := &BlameView{
		flex:        tview.NewFlex(),
		blameTable:  tview.NewTable(),
		detailView:  tview.NewTextView(),
		repo:        repo,
		app:         app,
		file:        file,
		authorColor: make(map[string]tcell.Color),
	}
	v.setup()
	return v
}

// NewBlameViewAtCommit creates a blame view for a file at a specific commit.
func NewBlameViewAtCommit(app *layout.App, repo *git.Repository, file, hash string) *BlameView {
	v := NewBlameView(app, repo, file)
	v.hash = hash
	return v
}

func (v *BlameView) setup() {
	// Configure blame table
	v.blameTable.SetBorders(false)
	v.blameTable.SetSelectable(true, false)
	v.blameTable.SetSelectedStyle(tcell.StyleDefault.
		Background(theme.Accent()).
		Foreground(theme.Bg()))
	v.blameTable.SetBackgroundColor(theme.Bg())
	// Note: SetSelectionChangedFunc is set after data loads to avoid deadlock
	theme.Register(v.blameTable)

	// Configure detail view
	v.detailView.SetDynamicColors(true).SetWordWrap(true)
	v.detailView.SetBackgroundColor(theme.Bg())
	theme.Register(v.detailView)

	// Wrap in panels
	blamePanel := components.NewPanel().
		SetTitle(fmt.Sprintf("Blame: %s", v.file)).
		SetContent(v.blameTable)

	detailPanel := components.NewPanel().
		SetTitle("Commit Info").
		SetContent(v.detailView)

	// Two-panel layout (70/30 split)
	v.split = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.7).
		SetShowDivider(false).
		SetLeft(blamePanel).
		SetRight(detailPanel)

	v.flex.SetDirection(tview.FlexRow)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(v.split, 0, 1, true)
	theme.Register(v.flex)

	// Register actions
	v.actions = input.NewActionRegistry().
		AddKey("select", tcell.KeyEnter, "Show commit", v.showCommit).
		AddSimple("diff", 'd', "Show diff", v.showDiff).
		AddSimple("file_log", 'L', "File history", v.showFileLog).
		AddSimple("copy_hash", 'y', "Copy hash", v.copyHash)
}

// nav.Component interface implementation

func (v *BlameView) Name() string {
	return "Blame"
}

func (v *BlameView) Start() {
	v.loadBlame()
}

func (v *BlameView) Stop() {}

func (v *BlameView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "Show commit"},
		{Key: "d", Description: "Diff"},
		{Key: "L", Description: "File history"},
		{Key: "Esc", Description: "Back"},
	}
}

// Business logic

func (v *BlameView) loadBlame() {
	var err error
	if v.hash != "" {
		v.lines, err = v.repo.BlameAtCommit(v.file, v.hash)
	} else {
		v.lines, err = v.repo.Blame(v.file)
	}

	if err != nil {
		v.showError(err)
		return
	}

	v.renderBlame()

	// Select first line and update detail
	if len(v.lines) > 0 {
		v.blameTable.Select(0, 0)
		v.updateDetail(v.lines[0])
	}
}

func (v *BlameView) renderBlame() {
	v.blameTable.Clear()

	for i, line := range v.lines {
		// Get author color
		color := v.getAuthorColor(line.Author)
		colorTag := theme.ColorToHex(color)

		// Format line number (right-aligned, 4 chars)
		lineNum := fmt.Sprintf("%4d", line.LineNumber)

		// Format author (truncate to 12 chars)
		author := line.Author
		if len(author) > 12 {
			author = author[:11] + "…"
		} else {
			author = fmt.Sprintf("%-12s", author)
		}

		// Format date
		date := v.formatDate(line.Date)

		// Create cells
		// Hash cell
		hashCell := tview.NewTableCell(line.ShortHash).
			SetTextColor(color).
			SetAlign(tview.AlignLeft)
		v.blameTable.SetCell(i, 0, hashCell)

		// Author cell
		authorCell := tview.NewTableCell(author).
			SetTextColor(color).
			SetAlign(tview.AlignLeft)
		v.blameTable.SetCell(i, 1, authorCell)

		// Date cell
		dateCell := tview.NewTableCell(date).
			SetTextColor(theme.FgMuted()).
			SetAlign(tview.AlignLeft)
		v.blameTable.SetCell(i, 2, dateCell)

		// Line number cell (dimmed)
		lineNumCell := tview.NewTableCell(lineNum).
			SetTextColor(theme.FgMuted()).
			SetAlign(tview.AlignRight)
		v.blameTable.SetCell(i, 3, lineNumCell)

		// Separator
		sepCell := tview.NewTableCell("│").
			SetTextColor(theme.FgMuted())
		v.blameTable.SetCell(i, 4, sepCell)

		// Content cell - escape and color based on commit
		content := tview.Escape(line.Content)
		// Show with subtle background tint for same-commit lines
		contentCell := tview.NewTableCell(fmt.Sprintf("[%s]%s[-]", colorTag, content)).
			SetExpansion(1).
			SetAlign(tview.AlignLeft)
		v.blameTable.SetCell(i, 5, contentCell)
	}
}

func (v *BlameView) getAuthorColor(author string) tcell.Color {
	if color, ok := v.authorColor[author]; ok {
		return color
	}

	colors := blameColors()
	color := colors[v.colorIndex%len(colors)]
	v.colorIndex++
	v.authorColor[author] = color
	return color
}

func (v *BlameView) formatDate(dateStr string) string {
	// dateStr is unix timestamp as string
	ts, err := strconv.ParseInt(dateStr, 10, 64)
	if err != nil {
		return dateStr
	}

	t := time.Unix(ts, 0)
	diff := time.Since(t)

	switch {
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(diff.Hours()/24/7))
	case diff < 365*24*time.Hour:
		return t.Format("Jan 02")
	default:
		return t.Format("2006-01-02")
	}
}

func (v *BlameView) updateDetail(line git.BlameLine) {
	var text strings.Builder

	// Commit hash
	text.WriteString(fmt.Sprintf("[%s::b]Commit[-:-:-]\n", theme.TagAccent()))
	text.WriteString(fmt.Sprintf("%s\n\n", line.Hash))

	// Author info
	text.WriteString(fmt.Sprintf("[%s::b]Author[-:-:-]\n", theme.TagAccent()))
	text.WriteString(fmt.Sprintf("%s\n", line.Author))
	if line.AuthorMail != "" {
		text.WriteString(fmt.Sprintf("[%s]<%s>[-]\n", theme.TagFgDim(), line.AuthorMail))
	}

	// Date
	text.WriteString(fmt.Sprintf("\n[%s::b]Date[-:-:-]\n", theme.TagAccent()))
	ts, err := strconv.ParseInt(line.Date, 10, 64)
	if err == nil {
		t := time.Unix(ts, 0)
		text.WriteString(fmt.Sprintf("%s\n", t.Format("2006-01-02 15:04:05")))
		text.WriteString(fmt.Sprintf("[%s](%s)[-]\n", theme.TagFgDim(), v.formatDate(line.Date)))
	}

	// Line info
	text.WriteString(fmt.Sprintf("\n[%s::b]Line[-:-:-]\n", theme.TagAccent()))
	text.WriteString(fmt.Sprintf("Current: %d\n", line.LineNumber))
	if line.OrigLine != line.LineNumber {
		text.WriteString(fmt.Sprintf("Original: %d\n", line.OrigLine))
	}

	v.detailView.SetText(text.String())
}

func (v *BlameView) showCommit() {
	row, _ := v.blameTable.GetSelection()
	if row < 0 || row >= len(v.lines) {
		return
	}

	line := v.lines[row]
	commitView := NewCommitView(v.app, v.repo, line.Hash)
	v.app.Pages().Push(commitView)
	v.app.Crumbs().SetPath([]string{"Blame", v.file, line.ShortHash})
}

func (v *BlameView) showDiff() {
	row, _ := v.blameTable.GetSelection()
	if row < 0 || row >= len(v.lines) {
		return
	}

	line := v.lines[row]
	diffView := NewFileDiffView(v.app, v.repo, line.Hash, v.file)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Blame", "Diff"})
}

func (v *BlameView) showFileLog() {
	fileLogView := NewFileLogView(v.app, v.repo, v.file)
	v.app.Pages().Push(fileLogView)
	v.app.Crumbs().SetPath([]string{"Blame", "History"})
}

func (v *BlameView) copyHash() {
	row, _ := v.blameTable.GetSelection()
	if row < 0 || row >= len(v.lines) {
		return
	}
	// TODO: Implement clipboard copy
}

func (v *BlameView) showError(err error) {
	v.detailView.SetText(fmt.Sprintf("[%s]Error:[-] %v", theme.TagError(), err))
}

// tview.Primitive delegation

func (v *BlameView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *BlameView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *BlameView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *BlameView) Focus(d func(tview.Primitive)) { v.flex.Focus(d) }
func (v *BlameView) Blur()                         { v.flex.Blur() }
func (v *BlameView) HasFocus() bool                { return true }

func (v *BlameView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.flex.MouseHandler()
}

func (v *BlameView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *BlameView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		if v.actions.Handle(event) {
			return
		}

		if len(v.lines) == 0 {
			return
		}

		row, col := v.blameTable.GetSelection()
		oldRow := row

		updateRow := func(newRow int) {
			if newRow != oldRow {
				v.blameTable.Select(newRow, col)
				v.updateDetail(v.lines[newRow])
			}
		}

		bindings := input.NewKeyBindings().
			On(tcell.KeyDown, func(e *tcell.EventKey) bool {
				if row < len(v.lines)-1 {
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
				_, _, _, height := v.blameTable.GetInnerRect()
				newRow := row + height/2
				if newRow >= len(v.lines) {
					newRow = len(v.lines) - 1
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyCtrlD, func(e *tcell.EventKey) bool {
				_, _, _, height := v.blameTable.GetInnerRect()
				newRow := row + height/2
				if newRow >= len(v.lines) {
					newRow = len(v.lines) - 1
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyPgUp, func(e *tcell.EventKey) bool {
				_, _, _, height := v.blameTable.GetInnerRect()
				newRow := row - height/2
				if newRow < 0 {
					newRow = 0
				}
				updateRow(newRow)
				return true
			}).
			On(tcell.KeyCtrlU, func(e *tcell.EventKey) bool {
				_, _, _, height := v.blameTable.GetInnerRect()
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
				updateRow(len(v.lines) - 1)
				return true
			}).
			OnRune('j', func(e *tcell.EventKey) bool {
				if row < len(v.lines)-1 {
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
				updateRow(len(v.lines) - 1)
				return true
			})

		bindings.Handle(event)
	}
}
