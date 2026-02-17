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

	"github.com/atterpac/gxt/internal/app"
	"github.com/atterpac/gxt/internal/git"
)

// GraphView displays the commit graph.
type GraphView struct {
	flex        *tview.Flex
	split       *components.Split
	graphPanel  *components.Panel
	gitGraph    *components.GitGraph
	detailView  *tview.TextView
	repo        *git.Repository
	app         *layout.App
	actions     *input.ActionRegistry
	showStashes bool

	// Search
	searchActive  bool
	searchQuery   string
	searchMatches []int // Indices of matching commits
	searchIndex   int   // Current match index
}

// NewGraphView creates a new graph view.
func NewGraphView(app *layout.App, repo *git.Repository) *GraphView {
	v := &GraphView{
		flex:       tview.NewFlex(),
		gitGraph:   components.NewGitGraph(),
		detailView: tview.NewTextView(),
		repo:       repo,
		app:        app,
	}
	v.setup()
	return v
}

func (v *GraphView) setup() {
	// Configure git graph component
	v.gitGraph.
		SetShowRefs(true).
		SetShowHash(true).
		SetShowAuthor(true).
		SetOnChange(v.updateDetail).
		SetOnSelect(v.showCommitView)

	// Configure detail view
	v.detailView.SetDynamicColors(true).SetWordWrap(true)
	v.detailView.SetBackgroundColor(theme.Bg())
	theme.Register(v.detailView)

	// Wrap in panels (Required: rounded borders)
	v.graphPanel = components.NewPanel().
		SetTitle("Commit Graph").
		SetContent(v.gitGraph)

	detailPanel := components.NewPanel().
		SetTitle("Commit Details").
		SetContent(v.detailView)

	// Two-panel layout (60/40 split)
	v.split = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.6).
		SetShowDivider(false).
		SetLeft(v.graphPanel).
		SetRight(detailPanel)

	v.flex.SetDirection(tview.FlexRow)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(v.split, 0, 1, true)
	theme.Register(v.flex)

	// Register actions (search handled separately in InputHandler)
	v.actions = input.NewActionRegistry().
		AddKey("select", tcell.KeyEnter, "Details", func() { v.showCommitView(v.gitGraph.GetSelected()) }).
		AddSimple("diff", 'd', "Diff", v.showDiff).
		AddSimple("checkout", 'c', "Checkout", v.checkout).
		AddSimple("refresh", 'r', "Refresh", v.refresh).
		AddSimple("cherry_pick", 'C', "Cherry-pick", v.cherryPick).
		AddSimple("revert", 'R', "Revert", v.revert).
		AddSimple("edit_message", 'e', "Edit message", v.editCommitMessage).
		AddSimple("copy_hash", 'y', "Copy hash", v.copyHash).
		AddSimple("toggle_stashes", 'z', "Toggle stashes", v.toggleStashes).
		AddSimple("next_match", 'n', "Next match", v.nextMatch).
		AddSimple("prev_match", 'N', "Prev match", v.prevMatch)
}

// nav.Component interface implementation

func (v *GraphView) Name() string {
	return "Commit Graph"
}

func (v *GraphView) Start() {
	v.refresh()
}

func (v *GraphView) Stop() {
	// Cleanup if needed
}

func (v *GraphView) Hints() []components.KeyHint {
	if v.searchActive {
		return []components.KeyHint{
			{Key: "Enter", Description: "Confirm"},
			{Key: "Esc", Description: "Cancel"},
		}
	}
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "g/G", Description: "Top/Bottom"},
		{Key: "/", Description: "Search"},
		{Key: "n/N", Description: "Next/Prev"},
		{Key: "Enter", Description: "Details"},
		{Key: "d", Description: "Diff"},
		{Key: "c", Description: "Checkout"},
	}
}

// Business logic

func (v *GraphView) refresh() {
	graph, err := v.repo.LoadGraph(500)
	if err != nil {
		v.showError(err)
		return
	}

	// Add stashes to graph if enabled - insert chronologically
	if v.showStashes {
		stashes, _ := v.repo.LoadStashes()
		for _, stash := range stashes {
			graph.CommitMap[stash.Hash] = stash
			// Find correct chronological position (commits are newest first)
			inserted := false
			for i, commit := range graph.Commits {
				if stash.Date.After(commit.Date) {
					// Insert before this commit
					graph.Commits = append(graph.Commits[:i], append([]*components.GitCommit{stash}, graph.Commits[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted {
				// Stash is older than all commits, add at end
				graph.Commits = append(graph.Commits, stash)
			}
		}
		// Re-layout with stashes included
		graph.LayoutGraph()
	}

	// Check for unstaged changes and prepend pseudo-node
	if v.hasUnstagedChanges() {
		pseudoNode := &components.GitCommit{
			Hash:         "unstaged",
			ShortHash:    "○",
			Message:      "Unstaged Changes",
			IsPseudoNode: true,
			PseudoType:   "unstaged",
			Column:       0,
		}
		graph.Commits = append([]*components.GitCommit{pseudoNode}, graph.Commits...)
		graph.LayoutGraph()
	}

	v.gitGraph.SetGraph(graph)

	// Update detail for currently selected
	if commit := v.gitGraph.GetSelected(); commit != nil {
		v.updateDetail(commit)
	}
}

// hasUnstagedChanges returns true if there are unstaged changes in the repo.
func (v *GraphView) hasUnstagedChanges() bool {
	status, err := v.repo.Status()
	if err != nil {
		return false
	}
	for _, entry := range status {
		// Check for unstaged modifications or untracked files
		if entry.WorkStatus != 0 || entry.IsUntracked {
			return true
		}
	}
	return false
}

func (v *GraphView) toggleStashes() {
	v.showStashes = !v.showStashes
	v.refresh()
}

func (v *GraphView) updateDetail(commit *components.GitCommit) {
	if commit == nil {
		v.detailView.SetText("")
		return
	}

	// Handle pseudo-node - show unstaged changes summary
	if commit.IsPseudoNode && commit.PseudoType == "unstaged" {
		v.updateDetailUnstaged()
		return
	}

	// Load full commit details for enhanced preview
	detail, err := v.repo.LoadCommit(commit.Hash)
	if err != nil {
		// Fallback to basic info from graph commit
		v.updateDetailBasic(commit)
		return
	}

	var text strings.Builder

	// Subject
	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagAccent(), detail.Subject))

	// Commit type indicator
	if commit.IsStash {
		text.WriteString(fmt.Sprintf("[%s]◇ Stash Entry[-]\n", theme.TagInfo()))
	} else if commit.IsMerge {
		text.WriteString(fmt.Sprintf("[%s]◆ Merge Commit[-]\n", theme.TagAccent()))
	}

	// Show branch info prominently right after subject
	if len(detail.Branches) > 0 {
		// Find the primary branch (prefer local over remote, current branch first)
		var localBranches, remoteBranches []string
		for _, branch := range detail.Branches {
			if strings.HasPrefix(branch, "origin/") {
				remoteBranches = append(remoteBranches, branch)
			} else {
				localBranches = append(localBranches, branch)
			}
		}

		// Display primary branch
		if len(localBranches) > 0 {
			text.WriteString(fmt.Sprintf("[%s]Branch:[-] [%s]%s[-]", theme.TagFgDim(), theme.TagSuccess(), localBranches[0]))
			if len(localBranches) > 1 {
				text.WriteString(fmt.Sprintf(" [%s](+%d more)[-]", theme.TagFgDim(), len(localBranches)-1))
			}
			text.WriteString("\n")
		} else if len(remoteBranches) > 0 {
			text.WriteString(fmt.Sprintf("[%s]Branch:[-] [%s]%s[-]", theme.TagFgDim(), theme.TagFgDim(), remoteBranches[0]))
			if len(remoteBranches) > 1 {
				text.WriteString(fmt.Sprintf(" [%s](+%d more)[-]", theme.TagFgDim(), len(remoteBranches)-1))
			}
			text.WriteString("\n")
		}
	}

	// ─── Commit ───
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Commit ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Hash:[-]   %s\n", theme.TagFgDim(), detail.ShortHash))

	// GPG Signature (compact)
	if detail.GPGStatus.Signed {
		if detail.GPGStatus.Valid {
			text.WriteString(fmt.Sprintf("[%s]Signed:[-] [%s]✓ Verified[-]\n", theme.TagFgDim(), theme.TagSuccess()))
		} else {
			text.WriteString(fmt.Sprintf("[%s]Signed:[-] [%s]✗ Invalid[-]\n", theme.TagFgDim(), theme.TagError()))
		}
	}

	// ─── Author ───
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Author ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Name:[-]   %s\n", theme.TagFgDim(), detail.Author))
	text.WriteString(fmt.Sprintf("[%s]Date:[-]   %s [%s](%s)[-]\n",
		theme.TagFgDim(),
		detail.AuthorDate.Format("2006-01-02 15:04"),
		theme.TagFgDim(),
		relativeTimeCompact(detail.AuthorDate)))

	// Committer (only if different)
	if detail.Committer != detail.Author {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Committer ───[-:-:-]\n", theme.TagFgDim()))
		text.WriteString(fmt.Sprintf("[%s]Name:[-]   %s\n", theme.TagFgDim(), detail.Committer))
	}

	// ─── Parents ─── (with subjects)
	if len(detail.Parents) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Parents ───[-:-:-]\n", theme.TagFgDim()))
		for i, parent := range detail.Parents {
			shortParent := parent
			if len(parent) > 7 {
				shortParent = parent[:7]
			}
			subject := ""
			if i < len(detail.ParentSubjects) && detail.ParentSubjects[i] != "" {
				subject = detail.ParentSubjects[i]
				if len(subject) > 40 {
					subject = subject[:37] + "..."
				}
			}
			if subject != "" {
				text.WriteString(fmt.Sprintf("[%s]%s[-] %s\n", theme.TagInfo(), shortParent, subject))
			} else {
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), shortParent))
			}
		}
	}

	// ─── Refs ───
	if len(detail.Refs) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Refs ───[-:-:-]\n", theme.TagFgDim()))
		for _, ref := range detail.Refs {
			if strings.HasPrefix(ref, "tag:") || (len(ref) > 0 && ref[0] == 'v' && len(ref) > 1 && ref[1] >= '0' && ref[1] <= '9') {
				text.WriteString(fmt.Sprintf("[%s]⚑[-] %s\n", theme.TagWarning(), ref))
			} else {
				text.WriteString(fmt.Sprintf("[%s]→[-] %s\n", theme.TagAccent(), ref))
			}
		}
	}

	// ─── Branches ─── (compact, max 5)
	if len(detail.Branches) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s::b]─── Branches ───[-:-:-]\n", theme.TagFgDim()))
		maxShow := 5
		for i, branch := range detail.Branches {
			if i >= maxShow {
				text.WriteString(fmt.Sprintf("[%s]... +%d more[-]\n", theme.TagFgDim(), len(detail.Branches)-maxShow))
				break
			}
			if strings.HasPrefix(branch, "origin/") {
				text.WriteString(fmt.Sprintf("[%s]○[-] %s\n", theme.TagFgDim(), branch))
			} else {
				text.WriteString(fmt.Sprintf("[%s]●[-] %s\n", theme.TagSuccess(), branch))
			}
		}
	}

	// ─── Changes ───
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Changes ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Total:[-]  %d files  [%s]+%d[-] [%s]-%d[-]\n",
		theme.TagFgDim(),
		detail.Stats.FilesChanged,
		theme.TagSuccess(), detail.Stats.Insertions,
		theme.TagError(), detail.Stats.Deletions))

	// Per-file stats
	if len(detail.Files) > 0 {
		text.WriteString("\n")
		maxFiles := 15 // Limit files shown in preview
		for i, file := range detail.Files {
			if i >= maxFiles {
				text.WriteString(fmt.Sprintf("[%s]... +%d more files[-]\n", theme.TagFgDim(), len(detail.Files)-maxFiles))
				break
			}

			// Status color
			statusColor := theme.TagFg()
			statusChar := "M"
			switch file.Status {
			case git.FileAdded:
				statusColor = theme.TagSuccess()
				statusChar = "A"
			case git.FileDeleted:
				statusColor = theme.TagError()
				statusChar = "D"
			case git.FileModified:
				statusColor = theme.TagWarning()
				statusChar = "M"
			case git.FileRenamed:
				statusColor = theme.TagInfo()
				statusChar = "R"
			case git.FileCopied:
				statusColor = theme.TagInfo()
				statusChar = "C"
			}

			// Format stats
			var stats string
			if file.Binary {
				stats = fmt.Sprintf("[%s]bin[-]", theme.TagFgDim())
			} else {
				stats = fmt.Sprintf("[%s]+%-3d[-] [%s]-%-3d[-]",
					theme.TagSuccess(), file.Insertions,
					theme.TagError(), file.Deletions)
			}

			// Truncate path if needed
			path := file.Path
			if len(path) > 35 {
				path = "..." + path[len(path)-32:]
			}

			text.WriteString(fmt.Sprintf("[%s]%s[-] %s %s\n", statusColor, statusChar, stats, path))
		}
	}

	v.detailView.SetText(text.String())
}

func (v *GraphView) updateDetailUnstaged() {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]○ Unstaged Changes[-:-:-]\n", theme.TagAccent()))
	text.WriteString(fmt.Sprintf("\n[%s]Press Enter to open the staging workflow[-]\n", theme.TagFgDim()))

	// Get status
	status, err := v.repo.Status()
	if err != nil {
		text.WriteString(fmt.Sprintf("\n[%s]Error loading status[-]", theme.TagError()))
		v.detailView.SetText(text.String())
		return
	}

	// Categorize files
	var modified, added, deleted, untracked []git.StatusEntry
	for _, entry := range status {
		if entry.IsUntracked {
			untracked = append(untracked, entry)
		} else if entry.WorkStatus == git.FileModified {
			modified = append(modified, entry)
		} else if entry.WorkStatus == git.FileAdded {
			added = append(added, entry)
		} else if entry.WorkStatus == git.FileDeleted {
			deleted = append(deleted, entry)
		} else if entry.WorkStatus != 0 {
			modified = append(modified, entry)
		}
	}

	// Summary
	total := len(modified) + len(added) + len(deleted) + len(untracked)
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Summary ───[-:-:-]\n", theme.TagFgDim()))
	text.WriteString(fmt.Sprintf("[%s]Total:[-] %d files\n", theme.TagFgDim(), total))

	if len(modified) > 0 {
		text.WriteString(fmt.Sprintf("[%s]Modified:[-] [%s]%d[-]\n", theme.TagFgDim(), theme.TagWarning(), len(modified)))
	}
	if len(added) > 0 {
		text.WriteString(fmt.Sprintf("[%s]Added:[-] [%s]%d[-]\n", theme.TagFgDim(), theme.TagSuccess(), len(added)))
	}
	if len(deleted) > 0 {
		text.WriteString(fmt.Sprintf("[%s]Deleted:[-] [%s]%d[-]\n", theme.TagFgDim(), theme.TagError(), len(deleted)))
	}
	if len(untracked) > 0 {
		text.WriteString(fmt.Sprintf("[%s]Untracked:[-] [%s]%d[-]\n", theme.TagFgDim(), theme.TagInfo(), len(untracked)))
	}

	// File list
	text.WriteString(fmt.Sprintf("\n[%s::b]─── Files ───[-:-:-]\n", theme.TagFgDim()))
	maxFiles := 20
	shown := 0

	// Show modified files
	for _, entry := range modified {
		if shown >= maxFiles {
			break
		}
		text.WriteString(fmt.Sprintf("[%s]M[-] %s\n", theme.TagWarning(), entry.Path))
		shown++
	}

	// Show added files
	for _, entry := range added {
		if shown >= maxFiles {
			break
		}
		text.WriteString(fmt.Sprintf("[%s]A[-] %s\n", theme.TagSuccess(), entry.Path))
		shown++
	}

	// Show deleted files
	for _, entry := range deleted {
		if shown >= maxFiles {
			break
		}
		text.WriteString(fmt.Sprintf("[%s]D[-] %s\n", theme.TagError(), entry.Path))
		shown++
	}

	// Show untracked files
	for _, entry := range untracked {
		if shown >= maxFiles {
			break
		}
		text.WriteString(fmt.Sprintf("[%s]?[-] %s\n", theme.TagInfo(), entry.Path))
		shown++
	}

	if total > maxFiles {
		text.WriteString(fmt.Sprintf("\n[%s]... and %d more files[-]\n", theme.TagFgDim(), total-maxFiles))
	}

	v.detailView.SetText(text.String())
}

func (v *GraphView) updateDetailBasic(commit *components.GitCommit) {
	// Fallback basic view when LoadCommit fails
	var text strings.Builder
	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n\n", theme.TagAccent(), commit.Message))
	text.WriteString(fmt.Sprintf("[%s]Hash:[-]   %s\n", theme.TagFgDim(), commit.Hash))
	text.WriteString(fmt.Sprintf("[%s]Author:[-] %s\n", theme.TagFgDim(), commit.Author))
	text.WriteString(fmt.Sprintf("[%s]Date:[-]   %s\n", theme.TagFgDim(), commit.Date.Format("2006-01-02 15:04")))

	if len(commit.Parents) > 0 {
		text.WriteString(fmt.Sprintf("[%s]Parents:[-] %v\n", theme.TagFgDim(), commit.Parents))
	}

	if len(commit.Refs) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s]Refs:[-] [%s]%v[-]\n",
			theme.TagFgDim(), theme.TagWarning(), commit.Refs))
	}

	if commit.IsStash {
		text.WriteString(fmt.Sprintf("\n[%s]◇ Stash entry[-]", theme.TagInfo()))
	} else if commit.IsMerge {
		text.WriteString(fmt.Sprintf("\n[%s]◆ Merge commit[-]", theme.TagAccent()))
	}

	v.detailView.SetText(text.String())
}

// relativeTimeCompact returns a short relative time string.
func relativeTimeCompact(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(diff.Hours()/24))
	case diff < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(diff.Hours()/24/7))
	case diff < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(diff.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy", int(diff.Hours()/24/365))
	}
}

func (v *GraphView) showCommitView(commit *components.GitCommit) {
	if commit == nil {
		return
	}

	// Handle pseudo-node selection - open staging workflow
	if commit.IsPseudoNode && commit.PseudoType == "unstaged" {
		stagingWorkflow := NewStagingWorkflowView(v.app, v.repo)
		v.app.Pages().Push(stagingWorkflow)
		v.app.Crumbs().SetPath([]string{"Graph", "Staging"})
		return
	}

	commitView := NewCommitView(v.app, v.repo, commit.Hash)
	v.app.Pages().Push(commitView)
	v.app.Crumbs().SetPath([]string{"Graph", commit.ShortHash})
}

func (v *GraphView) showDiff() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}
	diffView := NewDiffView(v.app, v.repo, commit.Hash)
	v.app.Pages().Push(diffView)
	v.app.Crumbs().SetPath([]string{"Graph", "Diff"})
}

func (v *GraphView) checkout() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}

	// Show confirmation modal
	ShowConfirmModal(v.app, "Checkout",
		fmt.Sprintf("Checkout commit %s?\n\n%s", commit.ShortHash, commit.Message),
		func() {
			if err := v.repo.Checkout(commit.Hash); err != nil {
				ShowErrorModal(v.app, "Checkout Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Checked out %s", commit.ShortHash))
			v.refresh()
		})
}

func (v *GraphView) cherryPick() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}

	ShowConfirmModal(v.app, "Cherry Pick",
		fmt.Sprintf("Cherry-pick commit %s?\n\n%s", commit.ShortHash, commit.Message),
		func() {
			if err := v.repo.CherryPick(commit.Hash); err != nil {
				ShowErrorModal(v.app, "Cherry-pick Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Cherry-picked %s", commit.ShortHash))
			v.refresh()
		})
}

func (v *GraphView) revert() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}

	ShowConfirmModal(v.app, "Revert Commit",
		fmt.Sprintf("Revert commit %s?\n\n%s", commit.ShortHash, commit.Message),
		func() {
			if err := v.repo.Revert(commit.Hash); err != nil {
				ShowErrorModal(v.app, "Revert Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Reverted %s", commit.ShortHash))
			v.refresh()
		})
}

func (v *GraphView) copyHash() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}
	app.CopyCommitHash(commit.Hash)
}

func (v *GraphView) editCommitMessage() {
	commit := v.gitGraph.GetSelected()
	if commit == nil {
		return
	}

	// Don't allow editing stash entries
	if commit.IsStash {
		ShowErrorModal(v.app, "Cannot Edit", "Stash entries cannot be renamed.")
		return
	}

	// Safety check: is this commit pushed to a remote?
	if v.repo.IsCommitPushed(commit.Hash) {
		ShowErrorModal(v.app, "Cannot Edit",
			"This commit has been pushed to a remote.\n\n"+
				"Editing the message would require a force push,\n"+
				"which could disrupt other collaborators.")
		return
	}

	// Get current commit message
	currentMsg, err := v.repo.GetCommitMessage(commit.Hash)
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to get commit message: "+err.Error())
		return
	}

	// Show text area modal with current message pre-filled
	ShowTextAreaModal(v.app, "Edit Commit Message",
		fmt.Sprintf("Commit: %s", commit.ShortHash),
		currentMsg,
		func(newMessage string) {
			newMessage = strings.TrimSpace(newMessage)
			if newMessage == "" {
				ShowErrorModal(v.app, "Error", "Commit message cannot be empty.")
				return
			}
			if newMessage == currentMsg {
				return // No change
			}

			if err := v.repo.RenameCommit(commit.Hash, newMessage); err != nil {
				ShowErrorModal(v.app, "Edit Failed", err.Error())
				return
			}
			v.refresh()
		})
}

func (v *GraphView) showError(err error) {
	v.detailView.SetText(fmt.Sprintf("[%s]Error:[-] %v", theme.TagError(), err))
}

// Search functionality

func (v *GraphView) startSearch() {
	v.searchActive = true
	v.searchQuery = ""
	v.searchMatches = nil
	v.searchIndex = 0
	v.graphPanel.SetTitle("/█")
}

func (v *GraphView) handleSearchInput(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEscape:
		v.searchActive = false
		v.searchQuery = ""
		v.searchMatches = nil
		v.graphPanel.SetTitle("Commit Graph")
	case tcell.KeyEnter:
		v.searchActive = false
		if len(v.searchMatches) > 0 {
			v.graphPanel.SetTitle(fmt.Sprintf("Commit Graph [%d matches]", len(v.searchMatches)))
			app.ToastSuccess(fmt.Sprintf("Found %d matches", len(v.searchMatches)))
		} else {
			v.graphPanel.SetTitle("Commit Graph")
			if v.searchQuery != "" {
				app.ToastInfo("No matches found")
			}
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
			v.updateSearchMatches()
			v.graphPanel.SetTitle("/" + v.searchQuery + "█")
		}
	case tcell.KeyRune:
		v.searchQuery += string(event.Rune())
		v.updateSearchMatches()
		v.graphPanel.SetTitle("/" + v.searchQuery + "█")
		// Jump to first match as user types
		if len(v.searchMatches) > 0 {
			v.searchIndex = 0
			v.gitGraph.SetSelectedIndex(v.searchMatches[0])
		}
	}
	// All keys are consumed during search - don't let anything through
}

func (v *GraphView) updateSearchMatches() {
	v.searchMatches = nil
	if v.searchQuery == "" {
		return
	}

	graph := v.gitGraph.GetGraph()
	if graph == nil {
		return
	}

	for i, commit := range graph.Commits {
		// Match against message (case-insensitive)
		if strings.Contains(strings.ToLower(commit.Message), v.searchQuery) {
			v.searchMatches = append(v.searchMatches, i)
			continue
		}
		// Match against hash
		if strings.Contains(strings.ToLower(commit.Hash), v.searchQuery) ||
			strings.Contains(strings.ToLower(commit.ShortHash), v.searchQuery) {
			v.searchMatches = append(v.searchMatches, i)
			continue
		}
		// Match against author
		if strings.Contains(strings.ToLower(commit.Author), v.searchQuery) {
			v.searchMatches = append(v.searchMatches, i)
			continue
		}
		// Match against branch
		if strings.Contains(strings.ToLower(commit.Branch), v.searchQuery) {
			v.searchMatches = append(v.searchMatches, i)
		}
	}
}

func (v *GraphView) nextMatch() {
	if len(v.searchMatches) == 0 {
		if v.searchQuery != "" {
			app.ToastInfo("No matches")
		}
		return
	}

	v.searchIndex = (v.searchIndex + 1) % len(v.searchMatches)
	v.gitGraph.SetSelectedIndex(v.searchMatches[v.searchIndex])
	app.ToastInfo(fmt.Sprintf("Match %d/%d", v.searchIndex+1, len(v.searchMatches)))
}

func (v *GraphView) prevMatch() {
	if len(v.searchMatches) == 0 {
		if v.searchQuery != "" {
			app.ToastInfo("No matches")
		}
		return
	}

	v.searchIndex--
	if v.searchIndex < 0 {
		v.searchIndex = len(v.searchMatches) - 1
	}
	v.gitGraph.SetSelectedIndex(v.searchMatches[v.searchIndex])
	app.ToastInfo(fmt.Sprintf("Match %d/%d", v.searchIndex+1, len(v.searchMatches)))
}

// tview.Primitive delegation (Required pattern)

func (v *GraphView) Draw(screen tcell.Screen)       { v.flex.Draw(screen) }
func (v *GraphView) GetRect() (int, int, int, int)  { return v.flex.GetRect() }
func (v *GraphView) SetRect(x, y, w, h int)         { v.flex.SetRect(x, y, w, h) }
func (v *GraphView) Focus(d func(tview.Primitive)) { v.flex.Focus(d) }
func (v *GraphView) Blur()                          { v.flex.Blur() }
func (v *GraphView) HasFocus() bool                 { return v.flex.HasFocus() }

func (v *GraphView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.flex.MouseHandler()
}

func (v *GraphView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *GraphView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.flex.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		// When search is active, consume all input
		if v.searchActive {
			v.handleSearchInput(event)
			return
		}

		// Start search with /
		if event.Rune() == '/' {
			v.startSearch()
			return
		}

		if v.actions.Handle(event) {
			return
		}
		if handler := v.gitGraph.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}
