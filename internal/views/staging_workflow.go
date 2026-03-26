package views

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/app"
	"github.com/atterpac/gxt/internal/git"
)

// nodeData holds data associated with a tree node
type nodeData struct {
	file      *git.StatusEntry
	hunk      *git.DiffHunk
	hunkIndex int
	isFile    bool
	isStaged  bool
}

// StagingWorkflowView provides a tinder-style hunk review workflow.
type StagingWorkflowView struct {
	*tview.Box
	mainSplit      *components.Split
	leftSplit      *components.Split
	unstagedTree   *components.Tree
	stagedTree     *components.Tree
	previewText    *tview.TextView
	unstagedPanel  *components.Panel
	stagedPanel    *components.Panel
	previewPanel   *components.Panel
	repo           *git.Repository
	app            *layout.App

	unstagedFiles []git.StatusEntry          // Unstaged/untracked files
	stagedFiles   []git.StatusEntry          // Staged files
	fileHunks     map[string][]*git.DiffHunk // path -> hunks
	focusPanel    int                        // 0=unstaged, 1=staged, 2=preview
	lastTreePanel int                        // Remember which tree panel (0 or 1) was last focused
}

// NewStagingWorkflowView creates a new staging workflow view.
func NewStagingWorkflowView(app *layout.App, repo *git.Repository) *StagingWorkflowView {
	v := &StagingWorkflowView{
		Box:          tview.NewBox(),
		previewText:  tview.NewTextView(),
		unstagedTree: components.NewTree(),
		stagedTree:   components.NewTree(),
		repo:         repo,
		app:          app,
		fileHunks:    make(map[string][]*git.DiffHunk),
		focusPanel:   0, // Start with unstaged
	}
	v.setup()
	return v
}

func (v *StagingWorkflowView) setup() {
	v.Box.SetBackgroundColor(theme.Bg())
	theme.Register(v.Box)

	// Configure preview text view
	v.previewText.SetDynamicColors(true)
	v.previewText.SetWordWrap(false)
	v.previewText.SetBackgroundColor(theme.Bg())
	v.previewText.SetScrollable(true)
	theme.Register(v.previewText)

	// Configure unstaged tree
	v.unstagedTree.SetShowLines(true).
		SetShowIcons(true).
		SetIndentSize(2).
		SetOnHighlight(v.onNodeHighlight).
		SetOnSelect(v.onNodeSelect)

	// Configure staged tree
	v.stagedTree.SetShowLines(true).
		SetShowIcons(true).
		SetIndentSize(2).
		SetOnHighlight(v.onNodeHighlight).
		SetOnSelect(v.onNodeSelect)

	// Create panels and store references
	v.unstagedPanel = components.NewPanel().SetTitle("Unstaged").SetContent(v.unstagedTree)
	v.stagedPanel = components.NewPanel().SetTitle("Staged").SetContent(v.stagedTree)
	v.previewPanel = components.NewPanel().SetTitle("Preview").SetContent(v.previewText)

	// Create left vertical split (unstaged top, staged bottom)
	v.leftSplit = components.NewSplit().
		SetDirection(components.SplitVertical).
		SetRatio(0.5).
		SetShowDivider(true).
		SetTop(v.unstagedPanel).
		SetBottom(v.stagedPanel)

	// Create main horizontal split (left panels / right preview)
	v.mainSplit = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.4).
		SetShowDivider(true).
		SetLeft(v.leftSplit).
		SetRight(v.previewPanel)

	// Set initial focus state
	v.updateFocusState()
}

// nav.Component interface implementation

func (v *StagingWorkflowView) Name() string {
	return "Stage Changes"
}

func (v *StagingWorkflowView) Start() {
	v.loadFiles()
}

func (v *StagingWorkflowView) Stop() {}

func (v *StagingWorkflowView) Hints() []components.KeyHint {
	if v.focusPanel == 2 {
		// Preview panel hints
		return []components.KeyHint{
			{Key: "j/k", Description: "Scroll"},
			{Key: "Tab", Description: "Switch panel"},
			{Key: "Space", Description: "Stage/Unstage"},
			{Key: "e", Description: "Edit"},
			{Key: "d", Description: "Discard"},
		}
	}
	// Tree panel hints
	hints := []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "l/h", Description: "Expand/Collapse"},
		{Key: "Tab", Description: "Switch panel"},
		{Key: "Space", Description: "Stage/Unstage"},
		{Key: "d", Description: "Discard"},
		{Key: "e", Description: "Edit"},
	}
	// Show commit/stash only if there are staged files
	if len(v.stagedFiles) > 0 {
		hints = append(hints, components.KeyHint{Key: "c", Description: "Commit"})
		hints = append(hints, components.KeyHint{Key: "a", Description: "Amend"})
		hints = append(hints, components.KeyHint{Key: "s", Description: "Stash"})
	}
	return hints
}

func (v *StagingWorkflowView) loadFiles() {
	status, err := v.repo.Status()
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to load status: "+err.Error())
		return
	}

	// Separate staged and unstaged files
	v.unstagedFiles = nil
	v.stagedFiles = nil
	v.fileHunks = make(map[string][]*git.DiffHunk)

	for _, entry := range status {
		if entry.IsUntracked || entry.WorkStatus != 0 {
			v.unstagedFiles = append(v.unstagedFiles, entry)
		}
		if entry.IndexStatus != 0 {
			v.stagedFiles = append(v.stagedFiles, entry)
		}
	}

	// If no files at all, go back
	if len(v.unstagedFiles) == 0 && len(v.stagedFiles) == 0 {
		v.app.Pages().Pop()
		return
	}

	v.buildTrees()
}

func (v *StagingWorkflowView) buildTrees() {
	// Build unstaged tree
	unstagedRoot := &components.TreeNode{
		ID:       "unstaged-root",
		Label:    "Unstaged",
		Expanded: true,
	}

	for i := range v.unstagedFiles {
		entry := &v.unstagedFiles[i]
		fileNode := v.buildFileNode(entry, false)
		unstagedRoot.AddChild(fileNode)
	}

	v.unstagedTree.SetRoot(unstagedRoot)
	v.unstagedTree.ExpandAll()

	// Build staged tree
	stagedRoot := &components.TreeNode{
		ID:       "staged-root",
		Label:    "Staged",
		Expanded: true,
	}

	for i := range v.stagedFiles {
		entry := &v.stagedFiles[i]
		fileNode := v.buildFileNode(entry, true)
		stagedRoot.AddChild(fileNode)
	}

	v.stagedTree.SetRoot(stagedRoot)
	v.stagedTree.ExpandAll()

	// Trigger initial preview from the appropriate tree
	if v.focusPanel == 0 {
		if node := v.unstagedTree.GetSelected(); node != nil {
			v.onNodeHighlight(node)
		}
	} else if v.focusPanel == 1 {
		if node := v.stagedTree.GetSelected(); node != nil {
			v.onNodeHighlight(node)
		}
	}
}

func (v *StagingWorkflowView) buildFileNode(entry *git.StatusEntry, isStaged bool) *components.TreeNode {
	// Determine icon based on status
	icon := "M"
	if isStaged {
		if entry.IndexStatus != 0 {
			icon = entry.IndexStatus.String()
		}
	} else {
		if entry.IsUntracked {
			icon = "?"
		} else {
			icon = entry.WorkStatus.String()
		}
	}

	// Get hunks for this file
	var hunks []*git.DiffHunk
	if isStaged {
		diff, err := v.repo.GetStagedFileDiff(entry.Path)
		if err == nil && diff != "" {
			files, err := git.ParseDiff(diff)
			if err == nil && len(files) > 0 {
				hunks = files[0].Hunks
			}
		}
	} else {
		if entry.IsUntracked {
			hunks, _ = v.createSyntheticHunksForUntracked(entry.Path)
		} else {
			diff, err := v.repo.GetWorkingFileDiff(entry.Path)
			if err == nil && diff != "" {
				files, err := git.ParseDiff(diff)
				if err == nil && len(files) > 0 {
					hunks = files[0].Hunks
				}
			}
		}
	}

	// Use a unique key for each tree
	key := entry.Path
	if isStaged {
		key = "staged:" + entry.Path
	}
	v.fileHunks[key] = hunks

	// Create file node
	label := fmt.Sprintf("%s %s", icon, entry.Path)
	if len(hunks) > 1 {
		label = fmt.Sprintf("%s %s (%d hunks)", icon, entry.Path, len(hunks))
	}

	fileNode := &components.TreeNode{
		ID:       key,
		Label:    label,
		Icon:     v.statusIcon(entry),
		Expanded: true,
		Data: &nodeData{
			file:     entry,
			isFile:   true,
			isStaged: isStaged,
		},
	}

	// Only add hunk children if there are multiple hunks
	// For single hunks, the file node itself represents the hunk
	if len(hunks) > 1 {
		for i, hunk := range hunks {
			hunkLabel := hunk.Header
			// Truncate long headers
			if len(hunkLabel) > 50 {
				hunkLabel = hunkLabel[:47] + "..."
			}

			hunkNode := &components.TreeNode{
				ID:    fmt.Sprintf("%s:hunk:%d", key, i),
				Label: hunkLabel,
				Icon:  "~",
				Data: &nodeData{
					file:      entry,
					hunk:      hunk,
					hunkIndex: i,
					isFile:    false,
					isStaged:  isStaged,
				},
			}
			fileNode.AddChild(hunkNode)
		}
	}

	return fileNode
}

func (v *StagingWorkflowView) updateFocusState() {
	v.unstagedPanel.SetFocused(v.focusPanel == 0)
	v.stagedPanel.SetFocused(v.focusPanel == 1)
	v.previewPanel.SetFocused(v.focusPanel == 2)
}

func (v *StagingWorkflowView) statusIcon(entry *git.StatusEntry) string {
	if entry.IsUntracked {
		return "+"
	}
	switch entry.WorkStatus {
	case git.FileModified:
		return "~"
	case git.FileAdded:
		return "+"
	case git.FileDeleted:
		return "-"
	case git.FileRenamed:
		return "→"
	default:
		return "•"
	}
}

func (v *StagingWorkflowView) onNodeHighlight(node *components.TreeNode) {
	if node == nil || node.Data == nil {
		return
	}

	data, ok := node.Data.(*nodeData)
	if !ok {
		return
	}

	if data.isFile {
		// Show file overview
		v.renderFilePreview(data.file, data.isStaged)
	} else {
		// Show specific hunk
		v.renderHunkPreview(data.file, data.hunk, data.hunkIndex, data.isStaged)
	}
}

func (v *StagingWorkflowView) onNodeSelect(node *components.TreeNode) {
	// Enter triggers staging
	v.stageSelected()
}

func (v *StagingWorkflowView) renderFilePreview(entry *git.StatusEntry, isStaged bool) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n\n", theme.TagAccent(), entry.Path))

	if entry.IsUntracked {
		text.WriteString(fmt.Sprintf("[%s]Status:[-] ? (untracked)\n\n", theme.TagFgDim()))

		// Show file contents
		fullPath := filepath.Join(v.repo.Path(), entry.Path)
		content, err := os.ReadFile(fullPath)
		if err == nil && !isBinaryContent(content) {
			lines := strings.Split(string(content), "\n")
			maxLines := 100
			for i, line := range lines {
				if i >= maxLines {
					text.WriteString(fmt.Sprintf("\n[%s]... +%d more lines[-]", theme.TagFgDim(), len(lines)-maxLines))
					break
				}
				text.WriteString(fmt.Sprintf("[%s]+%s[-]\n", theme.TagSuccess(), tview.Escape(line)))
			}
		}
	} else {
		status := entry.WorkStatus
		if isStaged {
			status = entry.IndexStatus
		}
		text.WriteString(fmt.Sprintf("[%s]Status:[-] %s\n\n", theme.TagFgDim(), status.String()))

		// Show all hunks for this file (staged files use "staged:" prefix key)
		hunkKey := entry.Path
		if isStaged {
			hunkKey = "staged:" + entry.Path
		}
		hunks := v.fileHunks[hunkKey]
		for i, hunk := range hunks {
			if i > 0 {
				text.WriteString("\n")
			}
			text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), hunk.Header))
			for _, line := range hunk.Lines {
				switch line.Type {
				case git.LineAdded:
					text.WriteString(fmt.Sprintf("[%s]+%s[-]\n", theme.TagSuccess(), tview.Escape(line.Content)))
				case git.LineRemoved:
					text.WriteString(fmt.Sprintf("[%s]-%s[-]\n", theme.TagError(), tview.Escape(line.Content)))
				default:
					text.WriteString(fmt.Sprintf(" %s\n", tview.Escape(line.Content)))
				}
			}
		}
	}

	v.previewText.SetText(text.String())
	v.previewText.ScrollToBeginning()
}

func (v *StagingWorkflowView) renderHunkPreview(entry *git.StatusEntry, hunk *git.DiffHunk, hunkIndex int, isStaged bool) {
	var text strings.Builder

	hunkKey := entry.Path
	if isStaged {
		hunkKey = "staged:" + entry.Path
	}
	hunks := v.fileHunks[hunkKey]
	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagAccent(), entry.Path))
	text.WriteString(fmt.Sprintf("[%s]Hunk %d/%d[-]\n\n", theme.TagFgDim(), hunkIndex+1, len(hunks)))

	text.WriteString(fmt.Sprintf("[%s]%s[-]\n\n", theme.TagInfo(), hunk.Header))

	for _, line := range hunk.Lines {
		switch line.Type {
		case git.LineAdded:
			text.WriteString(fmt.Sprintf("[%s]+%s[-]\n", theme.TagSuccess(), tview.Escape(line.Content)))
		case git.LineRemoved:
			text.WriteString(fmt.Sprintf("[%s]-%s[-]\n", theme.TagError(), tview.Escape(line.Content)))
		default:
			text.WriteString(fmt.Sprintf(" %s\n", tview.Escape(line.Content)))
		}
	}

	text.WriteString(fmt.Sprintf("\n[%s]Space: Stage  d: Discard  e: Edit[-]", theme.TagFgDim()))

	v.previewText.SetText(text.String())
	v.previewText.ScrollToBeginning()
}

func (v *StagingWorkflowView) stageSelected() {
	var tree *components.Tree
	var node *components.TreeNode
	var isUnstaging bool

	if v.focusPanel == 0 {
		tree = v.unstagedTree
		node = tree.GetSelected()
		isUnstaging = false
	} else if v.focusPanel == 1 {
		tree = v.stagedTree
		node = tree.GetSelected()
		isUnstaging = true
	} else {
		return
	}

	if node == nil || node.Data == nil {
		return
	}

	data, ok := node.Data.(*nodeData)
	if !ok {
		return
	}

	// Remember cursor position to restore after reload
	savedIndex := tree.GetSelectedIndex()

	if isUnstaging {
		// Unstage operation
		if data.isFile {
			// Unstage entire file
			if err := v.repo.UnstageFile(data.file.Path); err != nil {
				ShowErrorModal(v.app, "Unstage Failed", err.Error())
				return
			}
		} else {
			// Unstage specific hunk
			if err := v.repo.UnstageHunk(data.file.Path, data.hunk); err != nil {
				ShowErrorModal(v.app, "Unstage Failed", err.Error())
				return
			}
		}
	} else {
		// Stage operation
		if data.isFile {
			// Stage entire file
			if err := v.repo.StageFile(data.file.Path); err != nil {
				ShowErrorModal(v.app, "Stage Failed", err.Error())
				return
			}
		} else {
			// Stage specific hunk
			if data.file.IsUntracked {
				// For untracked files, must stage whole file
				if err := v.repo.StageFile(data.file.Path); err != nil {
					ShowErrorModal(v.app, "Stage Failed", err.Error())
					return
				}
			} else {
				if err := v.repo.StageHunk(data.file.Path, data.hunk); err != nil {
					ShowErrorModal(v.app, "Stage Failed", err.Error())
					return
				}
			}
		}
	}

	v.loadFiles()

	// Restore cursor position in the same tree (clamped to new bounds)
	tree.SetSelectedIndex(savedIndex)
}

func (v *StagingWorkflowView) discardSelected() {
	var tree *components.Tree
	var node *components.TreeNode

	if v.focusPanel == 0 {
		tree = v.unstagedTree
		node = tree.GetSelected()
	} else if v.focusPanel == 1 {
		tree = v.stagedTree
		node = tree.GetSelected()
	} else {
		return
	}

	if node == nil || node.Data == nil {
		return
	}

	data, ok := node.Data.(*nodeData)
	if !ok {
		return
	}

	savedIndex := tree.GetSelectedIndex()

	ShowConfirmModal(v.app, "Discard Changes",
		"Discard selected changes? This cannot be undone.",
		func() {
			if data.isFile {
				if data.file.IsUntracked {
					// Delete untracked file
					fullPath := filepath.Join(v.repo.Path(), data.file.Path)
					os.Remove(fullPath)
				} else {
					// Discard all changes to file
					v.repo.DiscardFileChanges(data.file.Path)
				}
			} else {
				if data.file.IsUntracked {
					// Delete untracked file
					fullPath := filepath.Join(v.repo.Path(), data.file.Path)
					os.Remove(fullPath)
				} else {
					// Discard specific hunk
					v.repo.DiscardHunk(data.file.Path, data.hunk)
				}
			}
			v.loadFiles()
			tree.SetSelectedIndex(savedIndex)
		})
}

func (v *StagingWorkflowView) editSelected() {
	var node *components.TreeNode

	if v.focusPanel == 0 {
		node = v.unstagedTree.GetSelected()
	} else if v.focusPanel == 1 {
		node = v.stagedTree.GetSelected()
	} else {
		return
	}

	if node == nil || node.Data == nil {
		return
	}

	data, ok := node.Data.(*nodeData)
	if !ok {
		return
	}

	if data.isFile {
		// Open file directly in editor
		fullPath := filepath.Join(v.repo.Path(), data.file.Path)
		v.openInEditor(fullPath)
		v.loadFiles()
		return
	}

	// For hunks, write to a .patch file so editor shows diff colors
	if data.hunk != nil {
		v.editHunkAsPatch(data.file, data.hunk)
	}
}

func (v *StagingWorkflowView) editHunkAsPatch(entry *git.StatusEntry, hunk *git.DiffHunk) {
	// Build patch content with header for syntax highlighting
	var content strings.Builder
	content.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", entry.Path, entry.Path))
	content.WriteString(fmt.Sprintf("--- a/%s\n", entry.Path))
	content.WriteString(fmt.Sprintf("+++ b/%s\n", entry.Path))
	content.WriteString(hunk.Header + "\n")

	for _, line := range hunk.Lines {
		switch line.Type {
		case git.LineAdded:
			content.WriteString("+" + line.Content + "\n")
		case git.LineRemoved:
			content.WriteString("-" + line.Content + "\n")
		default:
			content.WriteString(" " + line.Content + "\n")
		}
	}

	// Write to temp .patch file
	tmpFile, err := os.CreateTemp("", "gxt-*.patch")
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	tmpPath := tmpFile.Name()
	tmpFile.WriteString(content.String())
	tmpFile.Close()

	// Open in editor
	v.openInEditor(tmpPath)

	// Read back edited content
	newContent, err := os.ReadFile(tmpPath)
	os.Remove(tmpPath)
	if err != nil {
		v.loadFiles()
		return
	}

	// Apply the edited patch
	v.applyEditedPatch(entry.Path, hunk, string(newContent))
}

func (v *StagingWorkflowView) applyEditedPatch(filePath string, originalHunk *git.DiffHunk, patchContent string) {
	// Parse the edited patch to extract hunk lines
	lines := strings.Split(patchContent, "\n")
	var newLines []*git.DiffLine
	inHunk := false

	for _, line := range lines {
		// Skip diff headers
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") {
			continue
		}
		// Start of hunk
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			continue
		}
		if !inHunk || len(line) == 0 {
			continue
		}

		diffLine := &git.DiffLine{}
		if len(line) > 0 {
			diffLine.Content = line[1:]
			switch line[0] {
			case '+':
				diffLine.Type = git.LineAdded
			case '-':
				diffLine.Type = git.LineRemoved
			default:
				diffLine.Type = git.LineContext
			}
			newLines = append(newLines, diffLine)
		}
	}

	if len(newLines) == 0 {
		v.loadFiles()
		return
	}

	modifiedHunk := &git.DiffHunk{
		Header:   originalHunk.Header,
		OldStart: originalHunk.OldStart,
		OldCount: originalHunk.OldCount,
		NewStart: originalHunk.NewStart,
		NewCount: originalHunk.NewCount,
		Lines:    newLines,
	}

	if err := v.repo.StageHunk(filePath, modifiedHunk); err != nil {
		ShowErrorModal(v.app, "Stage Failed", err.Error())
		v.loadFiles()
		return
	}

	v.loadFiles()
}

func (v *StagingWorkflowView) openInEditor(path string) {
	v.openInEditorAtLine(path, 0)
}

func (v *StagingWorkflowView) openInEditorAtLine(path string, line int) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	var args []string
	if line > 0 {
		// Most editors support +N for line number
		args = []string{fmt.Sprintf("+%d", line), path}
	} else {
		args = []string{path}
	}

	v.app.Suspend(func() {
		cmd := exec.Command(editor, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})
}

func (v *StagingWorkflowView) createSyntheticHunksForUntracked(filePath string) ([]*git.DiffHunk, error) {
	fullPath := filepath.Join(v.repo.Path(), filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	if isBinaryContent(content) {
		return nil, nil
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	var diffLines []*git.DiffLine
	for i, line := range lines {
		diffLines = append(diffLines, &git.DiffLine{
			Type:      git.LineAdded,
			Content:   line,
			OldLineNo: 0,
			NewLineNo: i + 1,
		})
	}

	hunk := &git.DiffHunk{
		Header:   fmt.Sprintf("@@ -0,0 +1,%d @@ (new file)", len(lines)),
		OldStart: 0,
		OldCount: 0,
		NewStart: 1,
		NewCount: len(lines),
		Lines:    diffLines,
	}

	return []*git.DiffHunk{hunk}, nil
}

func isBinaryContent(content []byte) bool {
	checkLen := min(len(content), 8000)
	for i := range checkLen {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

func (v *StagingWorkflowView) commit() {
	if len(v.stagedFiles) == 0 {
		ShowErrorModal(v.app, "No Staged Changes", "Stage some changes before committing")
		return
	}

	ShowCommitModal(v.app, "Commit", "", func(message string) {
		if err := v.repo.Commit(message); err != nil {
			ShowErrorModal(v.app, "Commit Failed", err.Error())
			return
		}

		app.ToastSuccess("Changes committed successfully")
		v.loadFiles()

		// If no more changes, pop back to previous view
		if len(v.unstagedFiles) == 0 && len(v.stagedFiles) == 0 {
			v.app.Pages().Pop()
		}
	})
}

func (v *StagingWorkflowView) amend() {
	// Get the previous commit message to pre-populate
	prevMsg, err := v.repo.GetCommitMessage("HEAD")
	if err != nil {
		ShowErrorModal(v.app, "Amend Failed", "Could not read previous commit: "+err.Error())
		return
	}

	ShowCommitModal(v.app, "Amend Commit", prevMsg, func(message string) {
		if err := v.repo.CommitAmend(message); err != nil {
			ShowErrorModal(v.app, "Amend Failed", err.Error())
			return
		}

		app.ToastSuccess("Commit amended successfully")
		v.loadFiles()

		// If no more changes, pop back to previous view
		if len(v.unstagedFiles) == 0 && len(v.stagedFiles) == 0 {
			v.app.Pages().Pop()
		}
	})
}

func (v *StagingWorkflowView) stash() {
	if len(v.stagedFiles) == 0 {
		ShowErrorModal(v.app, "No Staged Changes", "Stage some changes before stashing")
		return
	}

	ShowInputModal(v.app, "Stash Staged Changes", "Stash message (optional):", func(message string) {
		if err := v.repo.StashStaged(message); err != nil {
			ShowErrorModal(v.app, "Stash Failed", err.Error())
			return
		}

		app.ToastSuccess("Staged changes stashed successfully")
		v.loadFiles()

		// If no more changes, pop back to previous view
		if len(v.unstagedFiles) == 0 && len(v.stagedFiles) == 0 {
			v.app.Pages().Pop()
		}
	})
}

// Draw renders the staging workflow view.
func (v *StagingWorkflowView) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	v.mainSplit.SetRect(x, y, width, height)
	v.mainSplit.Draw(screen)
}

// tview.Primitive delegation

func (v *StagingWorkflowView) GetRect() (int, int, int, int) { return v.Box.GetRect() }
func (v *StagingWorkflowView) SetRect(x, y, w, h int)        { v.Box.SetRect(x, y, w, h) }
func (v *StagingWorkflowView) Focus(d func(tview.Primitive)) { v.Box.Focus(d) }
func (v *StagingWorkflowView) Blur()                         { v.Box.Blur() }
func (v *StagingWorkflowView) HasFocus() bool                { return v.Box.HasFocus() }

func (v *StagingWorkflowView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		if handler := v.mainSplit.MouseHandler(); handler != nil {
			return handler(action, event, setFocus)
		}
		return false, nil
	})
}

func (v *StagingWorkflowView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *StagingWorkflowView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.Box.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		// Handle escape
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().Pop()
			return
		}

		// Handle Tab to cycle between unstaged, staged, and preview
		if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
			if event.Key() == tcell.KeyBacktab {
				// Reverse cycle
				if v.focusPanel == 0 {
					v.focusPanel = 2
				} else {
					v.focusPanel--
				}
			} else {
				v.focusPanel = (v.focusPanel + 1) % 3
			}
			v.updateFocusState()
			// Trigger preview update when switching to a tree
			if v.focusPanel == 0 {
				if node := v.unstagedTree.GetSelected(); node != nil {
					v.onNodeHighlight(node)
				}
			} else if v.focusPanel == 1 {
				if node := v.stagedTree.GetSelected(); node != nil {
					v.onNodeHighlight(node)
				}
			}
			return
		}

		// Get current tree based on focus
		var currentTree *components.Tree
		if v.focusPanel == 0 {
			currentTree = v.unstagedTree
		} else if v.focusPanel == 1 {
			currentTree = v.stagedTree
		}

		// Handle our custom keys (only when tree has focus)
		if v.focusPanel == 0 || v.focusPanel == 1 {
			switch event.Key() {
			case tcell.KeyRune:
				switch event.Rune() {
				case ' ':
					v.stageSelected()
					return
				case 'd':
					v.discardSelected()
					return
				case 'e', 'E':
					v.editSelected()
					return
				case 'c', 'C':
					v.commit()
					return
				case 'a', 'A':
					v.amend()
					return
				case 's', 'S':
					v.stash()
					return
				case 'q':
					v.app.Pages().Pop()
					return
				}
			}

			// Pass to appropriate tree for navigation
			if currentTree != nil {
				if handler := currentTree.InputHandler(); handler != nil {
					handler(event, setFocus)
				}
			}
		} else if v.focusPanel == 2 {
			// Preview panel has focus - handle scrolling
			switch event.Key() {
			case tcell.KeyDown:
				row, col := v.previewText.GetScrollOffset()
				v.previewText.ScrollTo(row+1, col)
				return
			case tcell.KeyUp:
				row, col := v.previewText.GetScrollOffset()
				if row > 0 {
					v.previewText.ScrollTo(row-1, col)
				}
				return
			case tcell.KeyPgDn:
				row, col := v.previewText.GetScrollOffset()
				v.previewText.ScrollTo(row+10, col)
				return
			case tcell.KeyPgUp:
				row, col := v.previewText.GetScrollOffset()
				v.previewText.ScrollTo(row-10, col)
				return
			case tcell.KeyRune:
				switch event.Rune() {
				case 'h':
					// Move back to the tree panel we came from
					v.focusPanel = v.lastTreePanel
					v.updateFocusState()
					// Trigger preview update
					if v.focusPanel == 0 {
						if node := v.unstagedTree.GetSelected(); node != nil {
							v.onNodeHighlight(node)
						}
					} else if v.focusPanel == 1 {
						if node := v.stagedTree.GetSelected(); node != nil {
							v.onNodeHighlight(node)
						}
					}
					return
				case 'e', 'E':
					v.editSelected()
					return
				case ' ':
					v.stageSelected()
					return
				case 'd':
					v.discardSelected()
					return
				case 'j':
					row, col := v.previewText.GetScrollOffset()
					v.previewText.ScrollTo(row+1, col)
					return
				case 'k':
					row, col := v.previewText.GetScrollOffset()
					if row > 0 {
						v.previewText.ScrollTo(row-1, col)
					}
					return
				case 'g':
					v.previewText.ScrollToBeginning()
					return
				case 'G':
					v.previewText.ScrollToEnd()
					return
				case 'q':
					v.app.Pages().Pop()
					return
				}
			case tcell.KeyCtrlD:
				row, col := v.previewText.GetScrollOffset()
				v.previewText.ScrollTo(row+5, col)
				return
			case tcell.KeyCtrlU:
				row, col := v.previewText.GetScrollOffset()
				v.previewText.ScrollTo(row-5, col)
				return
			}
		}
	})
}
