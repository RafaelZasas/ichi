package views

import (
	"fmt"
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

// CommitView displays detailed commit information.
type CommitView struct {
	flex         *tview.Flex
	infoPanel    *tview.TextView
	filesTable   *components.Table
	diffPreview  *tview.TextView
	filesPanel   *components.Panel
	diffPanel    *components.Panel
	repo         *git.Repository
	app          *layout.App
	hash         string
	commit       *git.CommitDetail
	actions      *input.ActionRegistry
	selectedFile int
	focusFiles   bool // true = files table focused, false = diff preview focused
}

// NewCommitView creates a new commit detail view.
func NewCommitView(app *layout.App, repo *git.Repository, hash string) *CommitView {
	v := &CommitView{
		flex:        tview.NewFlex(),
		infoPanel:   tview.NewTextView(),
		filesTable:  components.NewTable(),
		diffPreview: tview.NewTextView(),
		repo:        repo,
		app:         app,
		hash:        hash,
		focusFiles:  true, // Start with files table focused
	}
	v.setup()
	return v
}

func (v *CommitView) setup() {
	// Configure info panel
	v.infoPanel.SetDynamicColors(true).SetWordWrap(true)
	v.infoPanel.SetBackgroundColor(theme.Bg())
	theme.Register(v.infoPanel)

	// Configure files table
	v.filesTable.SetHeaders("Status", "File")
	v.filesTable.SetSelectable(true, false)
	v.filesTable.SetOnSelect(func(row int) {
		v.showFileDiff(row)
	})

	// Configure diff preview panel
	v.diffPreview.SetDynamicColors(true).SetWordWrap(false)
	v.diffPreview.SetBackgroundColor(theme.Bg())
	theme.Register(v.diffPreview)

	// Wrap in panels
	infoWrapper := components.NewPanel().
		SetTitle("Commit Info").
		SetContent(v.infoPanel)

	v.filesPanel = components.NewPanel().
		SetTitle("Changed Files").
		SetContent(v.filesTable)

	v.diffPanel = components.NewPanel().
		SetTitle("Diff Preview").
		SetContent(v.diffPreview)

	// Right side: files table (top) + diff preview (bottom)
	rightFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	rightFlex.AddItem(v.filesPanel, 0, 2, true)
	rightFlex.AddItem(v.diffPanel, 0, 3, false)
	rightFlex.SetBackgroundColor(theme.Bg())
	theme.Register(rightFlex)

	// Layout: info (left 40%) | files+preview (right 60%)
	v.flex.SetDirection(tview.FlexColumn)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(infoWrapper, 0, 2, false)
	v.flex.AddItem(rightFlex, 0, 3, true)
	theme.Register(v.flex)

	// Register actions
	v.actions = input.NewActionRegistry().
		AddKey("select", tcell.KeyEnter, "View diff", func() {
			row, _ := v.filesTable.GetSelection()
			v.showFileDiff(row)
		}).
		AddKey("switch_panel", tcell.KeyTab, "Switch panel", v.switchPanel).
		AddSimple("diff", 'd', "Full diff", v.showFullDiff).
		AddSimple("checkout", 'c', "Checkout", v.checkout).
		AddSimple("cherry_pick", 'C', "Cherry-pick", v.cherryPick).
		AddSimple("revert", 'R', "Revert", v.revert).
		AddSimple("copy_hash", 'y', "Copy hash", v.copyHash)

	// Set initial focus styling
	v.updatePanelFocus()
}

// nav.Component interface implementation

func (v *CommitView) Name() string {
	return "Commit Details"
}

func (v *CommitView) Start() {
	v.loadCommit()
}

func (v *CommitView) Stop() {}

func (v *CommitView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Tab", Description: "Switch panel"},
		{Key: "Enter", Description: "File diff"},
		{Key: "d", Description: "Full diff"},
		{Key: "c", Description: "Checkout"},
		{Key: "y", Description: "Copy hash"},
		{Key: "Esc", Description: "Back"},
	}
}

// Business logic

func (v *CommitView) loadCommit() {
	commit, err := v.repo.LoadCommit(v.hash)
	if err != nil {
		v.showError(err)
		return
	}
	v.commit = commit
	v.updateInfo()
	v.updateFiles()
}

func (v *CommitView) updateInfo() {
	if v.commit == nil {
		return
	}

	c := v.commit
	var text strings.Builder

	// ═══════════════════════════════════════════════════════════
	// Subject (title)
	// ═══════════════════════════════════════════════════════════
	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagAccent(), c.Subject))

	// Body (if different from subject)
	if c.Body != "" && c.Body != c.Subject {
		body := strings.TrimSpace(c.Body)
		// Remove subject from body if it starts with it
		body = strings.TrimPrefix(body, c.Subject)
		body = strings.TrimSpace(body)
		if body != "" {
			text.WriteString(fmt.Sprintf("\n%s\n", body))
		}
	}

	// ═══════════════════════════════════════════════════════════
	// Commit Details Section
	// ═══════════════════════════════════════════════════════════
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Commit ───[-:-:-]\n", theme.TagFgDim()))

	text.WriteString(fmt.Sprintf("[%s]Hash:[-]        %s\n", theme.TagFgDim(), c.Hash))

	// GPG Signature
	if c.GPGStatus.Signed {
		if c.GPGStatus.Valid {
			text.WriteString(fmt.Sprintf("[%s]Signature:[-]   [%s]✓ Verified[-]", theme.TagFgDim(), theme.TagSuccess()))
			if c.GPGStatus.Signer != "" {
				text.WriteString(fmt.Sprintf(" by %s", c.GPGStatus.Signer))
			}
			if c.GPGStatus.TrustLevel != "good" {
				text.WriteString(fmt.Sprintf(" (%s)", c.GPGStatus.TrustLevel))
			}
		} else {
			text.WriteString(fmt.Sprintf("[%s]Signature:[-]   [%s]✗ Invalid[-] (%s)", theme.TagFgDim(), theme.TagError(), c.GPGStatus.TrustLevel))
		}
		text.WriteString("\n")
	} else {
		text.WriteString(fmt.Sprintf("[%s]Signature:[-]   [%s]Unsigned[-]\n", theme.TagFgDim(), theme.TagFgDim()))
	}

	// ═══════════════════════════════════════════════════════════
	// Author Section
	// ═══════════════════════════════════════════════════════════
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Author ───[-:-:-]\n", theme.TagFgDim()))

	text.WriteString(fmt.Sprintf("[%s]Name:[-]        %s <%s>\n", theme.TagFgDim(), c.Author, c.AuthorEmail))
	text.WriteString(fmt.Sprintf("[%s]Date:[-]        %s [%s](%s)[-]\n",
		theme.TagFgDim(),
		c.AuthorDate.Format("2006-01-02 15:04:05 -0700"),
		theme.TagFgDim(),
		relativeTime(c.AuthorDate)))

	// Committer (if different)
	if c.Committer != c.Author || !c.CommitterDate.Equal(c.AuthorDate) {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Committer ───[-:-:-]\n", theme.TagFgDim()))
		text.WriteString(fmt.Sprintf("[%s]Name:[-]        %s <%s>\n", theme.TagFgDim(), c.Committer, c.CommitterEmail))
		text.WriteString(fmt.Sprintf("[%s]Date:[-]        %s [%s](%s)[-]\n",
			theme.TagFgDim(),
			c.CommitterDate.Format("2006-01-02 15:04:05 -0700"),
			theme.TagFgDim(),
			relativeTime(c.CommitterDate)))
	}

	// ═══════════════════════════════════════════════════════════
	// Parents Section
	// ═══════════════════════════════════════════════════════════
	if len(c.Parents) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Parents ───[-:-:-]\n", theme.TagFgDim()))
		for i, parent := range c.Parents {
			shortParent := parent
			if len(parent) > 7 {
				shortParent = parent[:7]
			}
			subject := ""
			if i < len(c.ParentSubjects) && c.ParentSubjects[i] != "" {
				subject = c.ParentSubjects[i]
				if len(subject) > 60 {
					subject = subject[:57] + "..."
				}
			}
			if subject != "" {
				text.WriteString(fmt.Sprintf("[%s]%s[-] %s\n", theme.TagInfo(), shortParent, subject))
			} else {
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), shortParent))
			}
		}
	}

	// ═══════════════════════════════════════════════════════════
	// Refs Section
	// ═══════════════════════════════════════════════════════════
	if len(c.Refs) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Refs ───[-:-:-]\n", theme.TagFgDim()))
		for _, ref := range c.Refs {
			if strings.HasPrefix(ref, "tag:") || strings.HasPrefix(ref, "v") {
				text.WriteString(fmt.Sprintf("[%s]⚑[-] %s\n", theme.TagWarning(), ref))
			} else {
				text.WriteString(fmt.Sprintf("[%s]→[-] %s\n", theme.TagAccent(), ref))
			}
		}
	}

	// ═══════════════════════════════════════════════════════════
	// Branches Containing Section
	// ═══════════════════════════════════════════════════════════
	if len(c.Branches) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Branches Containing ───[-:-:-]\n", theme.TagFgDim()))
		// Show first 10 branches, then count
		maxShow := 10
		for i, branch := range c.Branches {
			if i >= maxShow {
				text.WriteString(fmt.Sprintf("[%s]... and %d more[-]\n", theme.TagFgDim(), len(c.Branches)-maxShow))
				break
			}
			if strings.HasPrefix(branch, "origin/") {
				text.WriteString(fmt.Sprintf("[%s]○[-] %s\n", theme.TagFgDim(), branch))
			} else {
				text.WriteString(fmt.Sprintf("[%s]●[-] %s\n", theme.TagSuccess(), branch))
			}
		}
	}

	// ═══════════════════════════════════════════════════════════
	// Stats Section
	// ═══════════════════════════════════════════════════════════
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Changes ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Files:[-]       %d changed\n", theme.TagFgDim(), c.Stats.FilesChanged))
	text.WriteString(fmt.Sprintf("[%s]Lines:[-]       [%s]+%d[-] / [%s]-%d[-]\n",
		theme.TagFgDim(),
		theme.TagSuccess(), c.Stats.Insertions,
		theme.TagError(), c.Stats.Deletions))

	v.infoPanel.SetText(text.String())
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func (v *CommitView) updateFiles() {
	if v.commit == nil {
		return
	}

	v.filesTable.Clear()
	v.filesTable.SetHeaders("Status", "Changes", "File")

	for _, file := range v.commit.Files {
		statusColor := theme.TagFg()
		switch file.Status {
		case git.FileAdded:
			statusColor = theme.TagSuccess()
		case git.FileDeleted:
			statusColor = theme.TagError()
		case git.FileModified:
			statusColor = theme.TagWarning()
		case git.FileRenamed:
			statusColor = theme.TagInfo()
		}

		path := file.Path
		if file.OldPath != "" && file.OldPath != file.Path {
			path = file.OldPath + " → " + file.Path
		}

		// Format changes column
		var changes string
		if file.Binary {
			changes = fmt.Sprintf("[%s]binary[-]", theme.TagFgDim())
		} else if file.Insertions > 0 || file.Deletions > 0 {
			changes = fmt.Sprintf("[%s]+%d[-] [%s]-%d[-]",
				theme.TagSuccess(), file.Insertions,
				theme.TagError(), file.Deletions)
		} else {
			changes = fmt.Sprintf("[%s]—[-]", theme.TagFgDim())
		}

		v.filesTable.AddRow(
			fmt.Sprintf("[%s]%s[-]", statusColor, file.Status.String()),
			changes,
			path,
		)
	}

	// Select first file and show diff preview
	if v.filesTable.GetRowCount() > 1 {
		v.filesTable.Select(1, 0)
		v.updateDiffPreview()
	}
}

func (v *CommitView) showFileDiff(row int) {
	if v.commit == nil || row <= 0 || row > len(v.commit.Files) {
		return
	}

	file := v.commit.Files[row-1] // -1 for header
	diffView := NewFileDiffView(v.app, v.repo, v.hash, file.Path)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Graph", v.commit.ShortHash, file.Path})
}

func (v *CommitView) showFullDiff() {
	diffView := NewDiffView(v.app, v.repo, v.hash)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Graph", v.commit.ShortHash, "Diff"})
}

func (v *CommitView) checkout() {
	if v.commit == nil {
		return
	}

	ShowConfirmModal(v.app, "Checkout",
		fmt.Sprintf("Checkout commit %s?\n\n%s", v.commit.ShortHash, v.commit.Subject),
		func() {
			if err := v.repo.Checkout(v.hash); err != nil {
				ShowErrorModal(v.app, "Checkout Failed", err.Error())
				return
			}
		})
}

func (v *CommitView) cherryPick() {
	if v.commit == nil {
		return
	}

	ShowConfirmModal(v.app, "Cherry Pick",
		fmt.Sprintf("Cherry-pick commit %s?\n\n%s", v.commit.ShortHash, v.commit.Subject),
		func() {
			if err := v.repo.CherryPick(v.hash); err != nil {
				ShowErrorModal(v.app, "Cherry-pick Failed", err.Error())
			}
		})
}

func (v *CommitView) revert() {
	if v.commit == nil {
		return
	}

	ShowConfirmModal(v.app, "Revert Commit",
		fmt.Sprintf("Revert commit %s?\n\n%s", v.commit.ShortHash, v.commit.Subject),
		func() {
			if err := v.repo.Revert(v.hash); err != nil {
				ShowErrorModal(v.app, "Revert Failed", err.Error())
			}
		})
}

func (v *CommitView) copyHash() {
	// TODO: Implement clipboard copy
}

func (v *CommitView) switchPanel() {
	v.focusFiles = !v.focusFiles
	v.updatePanelFocus()
}

func (v *CommitView) updatePanelFocus() {
	if v.focusFiles {
		v.filesPanel.SetFocused(true)
		v.diffPanel.SetFocused(false)
	} else {
		v.filesPanel.SetFocused(false)
		v.diffPanel.SetFocused(true)
	}
}

func (v *CommitView) updateDiffPreview() {
	row, _ := v.filesTable.GetSelection()
	if v.commit == nil || row <= 0 || row > len(v.commit.Files) {
		v.diffPreview.SetText(fmt.Sprintf("[%s]Select a file to preview changes[-]", theme.TagFgDim()))
		return
	}

	file := v.commit.Files[row-1] // -1 for header

	// Get diff for this file (limited to first ~50 lines for preview)
	diff, err := v.repo.GetFileDiff(v.hash, file.Path)
	if err != nil {
		v.diffPreview.SetText(fmt.Sprintf("[%s]Unable to load diff[-]", theme.TagFgDim()))
		return
	}

	// Format and limit diff for preview
	v.diffPreview.SetText(v.formatDiffPreview(diff))
}

func (v *CommitView) formatDiffPreview(diff string) string {
	lines := strings.Split(diff, "\n")
	var result strings.Builder

	maxLines := 100 // Limit preview lines
	lineCount := 0

	for _, line := range lines {
		if lineCount >= maxLines {
			result.WriteString(fmt.Sprintf("\n[%s]... (truncated, press Enter for full diff)[-]", theme.TagFgDim()))
			break
		}

		if len(line) == 0 {
			result.WriteString("\n")
			lineCount++
			continue
		}

		// Color based on diff line type
		switch line[0] {
		case '+':
			if strings.HasPrefix(line, "+++") {
				result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagFgDim(), tview.Escape(line)))
			} else {
				result.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagSuccess(), tview.Escape(line)))
			}
		case '-':
			if strings.HasPrefix(line, "---") {
				result.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagFgDim(), tview.Escape(line)))
			} else {
				result.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagError(), tview.Escape(line)))
			}
		case '@':
			result.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), tview.Escape(line)))
		case 'd', 'i', 'n', 'o', 's', 'B': // diff headers
			if strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") ||
				strings.HasPrefix(line, "new ") || strings.HasPrefix(line, "old ") ||
				strings.HasPrefix(line, "similarity") || strings.HasPrefix(line, "Binary") {
				result.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagFgDim(), tview.Escape(line)))
			} else {
				result.WriteString(tview.Escape(line) + "\n")
			}
		default:
			result.WriteString(tview.Escape(line) + "\n")
		}
		lineCount++
	}

	return result.String()
}

func (v *CommitView) showError(err error) {
	v.infoPanel.SetText(fmt.Sprintf("[%s]Error:[-] %v", theme.TagError(), err))
}

// tview.Primitive delegation

func (v *CommitView) Draw(screen tcell.Screen)       { v.flex.Draw(screen) }
func (v *CommitView) GetRect() (int, int, int, int)  { return v.flex.GetRect() }
func (v *CommitView) SetRect(x, y, w, h int)         { v.flex.SetRect(x, y, w, h) }
func (v *CommitView) Focus(d func(tview.Primitive)) {
	// Delegate focus to the appropriate panel based on focus state
	if v.focusFiles {
		d(v.filesTable)
	} else {
		d(v.diffPreview)
	}
}
func (v *CommitView) Blur()                          { v.flex.Blur() }
func (v *CommitView) HasFocus() bool                 { return true }

func (v *CommitView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.flex.MouseHandler()
}

func (v *CommitView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *CommitView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		if v.actions.Handle(event) {
			return
		}

		if v.focusFiles {
			v.handleFilesInput(event)
		} else {
			v.handleDiffInput(event)
		}
	}
}

func (v *CommitView) handleFilesInput(event *tcell.EventKey) {
	row, col := v.filesTable.GetSelection()
	prevRow := row

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			if row < v.filesTable.GetRowCount()-1 {
				v.filesTable.Select(row+1, col)
				v.updateDiffPreview()
			}
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			if row > 1 { // Skip header
				v.filesTable.Select(row-1, col)
				v.updateDiffPreview()
			}
			return true
		}).
		On(tcell.KeyPgDn, func(e *tcell.EventKey) bool {
			_, _, _, height := v.filesTable.GetInnerRect()
			newRow := row + height/2
			if newRow >= v.filesTable.GetRowCount() {
				newRow = v.filesTable.GetRowCount() - 1
			}
			if newRow > 0 {
				v.filesTable.Select(newRow, col)
				v.updateDiffPreview()
			}
			return true
		}).
		On(tcell.KeyCtrlD, func(e *tcell.EventKey) bool {
			_, _, _, height := v.filesTable.GetInnerRect()
			newRow := row + height/2
			if newRow >= v.filesTable.GetRowCount() {
				newRow = v.filesTable.GetRowCount() - 1
			}
			if newRow > 0 {
				v.filesTable.Select(newRow, col)
				v.updateDiffPreview()
			}
			return true
		}).
		On(tcell.KeyPgUp, func(e *tcell.EventKey) bool {
			_, _, _, height := v.filesTable.GetInnerRect()
			newRow := row - height/2
			if newRow < 1 {
				newRow = 1 // Skip header
			}
			v.filesTable.Select(newRow, col)
			v.updateDiffPreview()
			return true
		}).
		On(tcell.KeyCtrlU, func(e *tcell.EventKey) bool {
			_, _, _, height := v.filesTable.GetInnerRect()
			newRow := row - height/2
			if newRow < 1 {
				newRow = 1 // Skip header
			}
			v.filesTable.Select(newRow, col)
			v.updateDiffPreview()
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			if row < v.filesTable.GetRowCount()-1 {
				v.filesTable.Select(row+1, col)
				v.updateDiffPreview()
			}
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			if row > 1 { // Skip header
				v.filesTable.Select(row-1, col)
				v.updateDiffPreview()
			}
			return true
		}).
		OnRune('g', func(e *tcell.EventKey) bool {
			v.filesTable.Select(1, col) // First data row
			if prevRow != 1 {
				v.updateDiffPreview()
			}
			return true
		}).
		OnRune('G', func(e *tcell.EventKey) bool {
			lastRow := v.filesTable.GetRowCount() - 1
			v.filesTable.Select(lastRow, col)
			if prevRow != lastRow {
				v.updateDiffPreview()
			}
			return true
		})

	bindings.Handle(event)
}

func (v *CommitView) handleDiffInput(event *tcell.EventKey) {
	_, _, _, height := v.diffPreview.GetInnerRect()
	row, col := v.diffPreview.GetScrollOffset()

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollTo(row+1, col)
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			if row > 0 {
				v.diffPreview.ScrollTo(row-1, col)
			}
			return true
		}).
		On(tcell.KeyPgDn, func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollTo(row+height/2, col)
			return true
		}).
		On(tcell.KeyCtrlD, func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollTo(row+height/2, col)
			return true
		}).
		On(tcell.KeyPgUp, func(e *tcell.EventKey) bool {
			newRow := row - height/2
			if newRow < 0 {
				newRow = 0
			}
			v.diffPreview.ScrollTo(newRow, col)
			return true
		}).
		On(tcell.KeyCtrlU, func(e *tcell.EventKey) bool {
			newRow := row - height/2
			if newRow < 0 {
				newRow = 0
			}
			v.diffPreview.ScrollTo(newRow, col)
			return true
		}).
		On(tcell.KeyHome, func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollToBeginning()
			return true
		}).
		On(tcell.KeyEnd, func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollToEnd()
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollTo(row+1, col)
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			if row > 0 {
				v.diffPreview.ScrollTo(row-1, col)
			}
			return true
		}).
		OnRune('g', func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollToBeginning()
			return true
		}).
		OnRune('G', func(e *tcell.EventKey) bool {
			v.diffPreview.ScrollToEnd()
			return true
		})

	bindings.Handle(event)
}
