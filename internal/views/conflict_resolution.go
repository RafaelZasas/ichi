package views

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/app"
	"github.com/atterpac/gxt/internal/git"
)

// LineSource represents where a resolved line came from.
type LineSource int

const (
	SourceOurs LineSource = iota
	SourceBase
	SourceTheirs
	SourceManual
)

// ResolvedLine represents a line in the resolved pane with its source.
type ResolvedLine struct {
	Source  LineSource
	Content string
}

// DisplayMode controls what content is shown in the diff panes.
type DisplayMode int

const (
	DisplayConflicts DisplayMode = iota // Show only conflict regions
	DisplayFullFile                     // Show full file with conflicts highlighted
)

// ConflictResolutionView provides a UI for resolving merge/rebase conflicts.
type ConflictResolutionView struct {
	*tview.Box
	mainSplit  *components.Split
	topSplit   *components.Split
	leftSplit  *components.Split

	oursText   *tview.TextView
	baseText   *tview.TextView
	theirsText *tview.TextView
	resolvedText *tview.TextView

	oursPanel     *components.Panel
	basePanel     *components.Panel
	theirsPanel   *components.Panel
	resolvedPanel *components.Panel

	repo *git.Repository
	app  *layout.App

	state             *git.ConflictState
	currentFileIndex  int
	conflictFiles     []git.StatusEntry
	currentRegions    []*git.ConflictRegion
	fullFileLines     []string
	resolvedLines     []ResolvedLine
	currentRegionIdx  int

	focusPanel  int         // 0=ours, 1=base, 2=theirs, 3=resolved
	displayMode DisplayMode
}

// NewConflictResolutionView creates a new conflict resolution view.
func NewConflictResolutionView(app *layout.App, repo *git.Repository) *ConflictResolutionView {
	v := &ConflictResolutionView{
		Box:          tview.NewBox(),
		oursText:     tview.NewTextView(),
		baseText:     tview.NewTextView(),
		theirsText:   tview.NewTextView(),
		resolvedText: tview.NewTextView(),
		repo:         repo,
		app:          app,
		displayMode:  DisplayConflicts,
		focusPanel:   0,
	}
	v.setup()
	return v
}

func (v *ConflictResolutionView) setup() {
	v.Box.SetBackgroundColor(theme.Bg())
	theme.Register(v.Box)

	// Configure text views
	for _, tv := range []*tview.TextView{v.oursText, v.baseText, v.theirsText, v.resolvedText} {
		tv.SetDynamicColors(true)
		tv.SetWordWrap(false)
		tv.SetScrollable(true)
		tv.SetBackgroundColor(theme.Bg())
		theme.Register(tv)
	}

	// Create panels
	v.oursPanel = components.NewPanel().SetTitle("Ours (current)").SetContent(v.oursText)
	v.basePanel = components.NewPanel().SetTitle("Base (ancestor)").SetContent(v.baseText)
	v.theirsPanel = components.NewPanel().SetTitle("Theirs (incoming)").SetContent(v.theirsText)
	v.resolvedPanel = components.NewPanel().SetTitle("Resolved").SetContent(v.resolvedText)

	// Create left split: Ours | Base
	v.leftSplit = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.5).
		SetShowDivider(true).
		SetLeft(v.oursPanel).
		SetRight(v.basePanel)

	// Create top split: (Ours | Base) | Theirs
	v.topSplit = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.67).
		SetShowDivider(true).
		SetLeft(v.leftSplit).
		SetRight(v.theirsPanel)

	// Create main split: Top diffs | Bottom resolved
	v.mainSplit = components.NewSplit().
		SetDirection(components.SplitVertical).
		SetRatio(0.6).
		SetShowDivider(true).
		SetTop(v.topSplit).
		SetBottom(v.resolvedPanel)

	v.updateFocusState()
}

// nav.Component interface

func (v *ConflictResolutionView) Name() string {
	if v.state != nil && len(v.conflictFiles) > 0 {
		fileName := v.conflictFiles[v.currentFileIndex].Path
		return fmt.Sprintf("Resolve Conflicts (%d/%d) - %s",
			v.currentFileIndex+1, len(v.conflictFiles), fileName)
	}
	return "Resolve Conflicts"
}

func (v *ConflictResolutionView) Start() {
	v.loadConflictFiles()
	if len(v.conflictFiles) > 0 {
		v.loadFile(0)
	} else {
		// No conflicts found, return
		ShowErrorModal(v.app, "No Conflicts", "No conflicted files found")
		v.app.Pages().Pop()
	}
}

func (v *ConflictResolutionView) Stop() {}

func (v *ConflictResolutionView) Hints() []components.KeyHint {
	if v.focusPanel == 3 {
		// Resolved pane hints
		return []components.KeyHint{
			{Key: "d", Description: "Delete line"},
			{Key: "e", Description: "Edit in $EDITOR"},
			{Key: "c", Description: "Clear all"},
			{Key: "s", Description: "Save & next"},
		}
	}
	// Diff panes hints
	hints := []components.KeyHint{
		{Key: "Tab", Description: "Next pane"},
		{Key: "o/b/t", Description: "Accept Ours/Base/Theirs"},
		{Key: "s", Description: "Save & next"},
	}
	if v.currentFileIndex+1 < len(v.conflictFiles) || v.areAllFilesResolved() {
		hints = append(hints, components.KeyHint{Key: "C", Description: "Continue operation"})
	}
	hints = append(hints, components.KeyHint{Key: "x", Description: "Abort"})
	return hints
}

func (v *ConflictResolutionView) loadConflictFiles() {
	state, err := v.repo.GetConflictState()
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to get conflict state: "+err.Error())
		return
	}
	v.state = state

	files, err := v.repo.ConflictFiles()
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to load conflict files: "+err.Error())
		return
	}
	v.conflictFiles = files
}

func (v *ConflictResolutionView) loadFile(index int) {
	if index < 0 || index >= len(v.conflictFiles) {
		return
	}

	file := v.conflictFiles[index]
	v.currentFileIndex = index

	// Parse conflict regions
	regions, fullLines, err := v.repo.ParseConflictFile(file.Path)
	if err != nil {
		ShowErrorModal(v.app, "Parse Error", fmt.Sprintf("Failed to parse %s: %s", file.Path, err.Error()))
		return
	}

	v.currentRegions = regions
	v.fullFileLines = fullLines
	v.currentRegionIdx = 0

	// Clear resolved lines
	v.resolvedLines = nil

	// Display the first conflict region
	v.displayCurrentRegion()
	v.updateResolvedPane()
}

func (v *ConflictResolutionView) displayCurrentRegion() {
	if len(v.currentRegions) == 0 {
		v.oursText.SetText("[" + theme.TagFgDim() + "]No conflicts found[-]")
		v.baseText.SetText("")
		v.theirsText.SetText("")
		return
	}

	if v.currentRegionIdx >= len(v.currentRegions) {
		v.currentRegionIdx = len(v.currentRegions) - 1
	}

	region := v.currentRegions[v.currentRegionIdx]

	// Build display text for each pane
	var oursText, baseText, theirsText strings.Builder

	// Ours section
	oursText.WriteString(fmt.Sprintf("[%s]<<<<<<< %s[-]\n", theme.TagError(), region.OursLabel))
	for _, line := range region.OursLines {
		oursText.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagFg(), tview.Escape(line)))
	}

	// Base section
	if region.HasBase && len(region.BaseLines) > 0 {
		baseText.WriteString(fmt.Sprintf("[%s]||||||| merged common ancestors[-]\n", theme.TagFgDim()))
		for _, line := range region.BaseLines {
			baseText.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagFg(), tview.Escape(line)))
		}
	} else {
		baseText.WriteString(fmt.Sprintf("[%s]No base available[-]", theme.TagFgDim()))
	}

	// Theirs section
	theirsText.WriteString(fmt.Sprintf("[%s]>>>>>>> %s[-]\n", theme.TagSuccess(), region.TheirsLabel))
	for _, line := range region.TheirsLines {
		theirsText.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagFg(), tview.Escape(line)))
	}

	v.oursText.SetText(oursText.String())
	v.baseText.SetText(baseText.String())
	v.theirsText.SetText(theirsText.String())

	// Update panel titles with region info
	if len(v.currentRegions) > 1 {
		v.oursPanel.SetTitle(fmt.Sprintf("Ours (%s) - Region %d/%d",
			region.OursLabel, v.currentRegionIdx+1, len(v.currentRegions)))
		v.theirsPanel.SetTitle(fmt.Sprintf("Theirs (%s) - Region %d/%d",
			region.TheirsLabel, v.currentRegionIdx+1, len(v.currentRegions)))
	} else {
		v.oursPanel.SetTitle(fmt.Sprintf("Ours (%s)", region.OursLabel))
		v.theirsPanel.SetTitle(fmt.Sprintf("Theirs (%s)", region.TheirsLabel))
	}
}

func (v *ConflictResolutionView) updateResolvedPane() {
	var text strings.Builder

	if len(v.resolvedLines) == 0 {
		text.WriteString(fmt.Sprintf("[%s]No lines selected yet.[-]\n\n", theme.TagFgDim()))
		text.WriteString(fmt.Sprintf("[%s]Press 'o' to accept Ours, 'b' for Base, 't' for Theirs[-]\n", theme.TagFgDim()))
		text.WriteString(fmt.Sprintf("[%s]Or press 'e' to edit manually in $EDITOR[-]", theme.TagFgDim()))
	} else {
		for _, line := range v.resolvedLines {
			prefix := v.getSourcePrefix(line.Source)
			color := v.getSourceColor(line.Source)
			text.WriteString(fmt.Sprintf("[%s][%s][-] %s\n", color, prefix, tview.Escape(line.Content)))
		}
	}

	v.resolvedText.SetText(text.String())
	v.resolvedText.ScrollToBeginning()
}

func (v *ConflictResolutionView) getSourcePrefix(source LineSource) string {
	switch source {
	case SourceOurs:
		return "O"
	case SourceBase:
		return "B"
	case SourceTheirs:
		return "T"
	case SourceManual:
		return "M"
	default:
		return "?"
	}
}

func (v *ConflictResolutionView) getSourceColor(source LineSource) string {
	switch source {
	case SourceOurs:
		return theme.TagError()
	case SourceBase:
		return theme.TagFgDim()
	case SourceTheirs:
		return theme.TagSuccess()
	case SourceManual:
		return theme.TagInfo()
	default:
		return theme.TagFg()
	}
}

func (v *ConflictResolutionView) acceptOurs() {
	if v.currentRegionIdx >= len(v.currentRegions) {
		return
	}
	region := v.currentRegions[v.currentRegionIdx]
	for _, line := range region.OursLines {
		v.resolvedLines = append(v.resolvedLines, ResolvedLine{
			Source:  SourceOurs,
			Content: line,
		})
	}
	v.updateResolvedPane()
	v.moveToNextRegion()
}

func (v *ConflictResolutionView) acceptBase() {
	if v.currentRegionIdx >= len(v.currentRegions) {
		return
	}
	region := v.currentRegions[v.currentRegionIdx]
	if !region.HasBase || len(region.BaseLines) == 0 {
		app.ToastWarning("No base available for this conflict")
		return
	}
	for _, line := range region.BaseLines {
		v.resolvedLines = append(v.resolvedLines, ResolvedLine{
			Source:  SourceBase,
			Content: line,
		})
	}
	v.updateResolvedPane()
	v.moveToNextRegion()
}

func (v *ConflictResolutionView) acceptTheirs() {
	if v.currentRegionIdx >= len(v.currentRegions) {
		return
	}
	region := v.currentRegions[v.currentRegionIdx]
	for _, line := range region.TheirsLines {
		v.resolvedLines = append(v.resolvedLines, ResolvedLine{
			Source:  SourceTheirs,
			Content: line,
		})
	}
	v.updateResolvedPane()
	v.moveToNextRegion()
}

func (v *ConflictResolutionView) acceptBoth() {
	if v.currentRegionIdx >= len(v.currentRegions) {
		return
	}
	region := v.currentRegions[v.currentRegionIdx]

	// Add ours first, then theirs
	for _, line := range region.OursLines {
		v.resolvedLines = append(v.resolvedLines, ResolvedLine{
			Source:  SourceOurs,
			Content: line,
		})
	}
	for _, line := range region.TheirsLines {
		v.resolvedLines = append(v.resolvedLines, ResolvedLine{
			Source:  SourceTheirs,
			Content: line,
		})
	}
	v.updateResolvedPane()
	v.moveToNextRegion()
}

func (v *ConflictResolutionView) moveToNextRegion() {
	if v.currentRegionIdx+1 < len(v.currentRegions) {
		v.currentRegionIdx++
		v.displayCurrentRegion()
	}
}

func (v *ConflictResolutionView) moveToPrevRegion() {
	if v.currentRegionIdx > 0 {
		v.currentRegionIdx--
		v.displayCurrentRegion()
	}
}

func (v *ConflictResolutionView) openInEditor() {
	// Write current resolved content to temp file
	tmpFile, err := os.CreateTemp("", "gxt-conflict-*.txt")
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write resolved lines or current file if no resolution yet
	if len(v.resolvedLines) > 0 {
		for _, line := range v.resolvedLines {
			tmpFile.WriteString(line.Content + "\n")
		}
	} else {
		// Write full file with conflicts
		tmpFile.WriteString(strings.Join(v.fullFileLines, "\n"))
	}
	tmpFile.Close()

	// Open in $EDITOR
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	v.app.Suspend(func() {
		cmd := exec.Command(editor, tmpPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})

	// Read back edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		v.updateResolvedPane()
		return
	}

	lines := strings.Split(string(content), "\n")
	v.resolvedLines = nil
	for _, line := range lines {
		if line != "" || len(v.resolvedLines) > 0 {
			v.resolvedLines = append(v.resolvedLines, ResolvedLine{
				Source:  SourceManual,
				Content: line,
			})
		}
	}
	v.updateResolvedPane()
}

func (v *ConflictResolutionView) saveResolution() {
	if len(v.resolvedLines) == 0 {
		ShowErrorModal(v.app, "Cannot Save", "No resolution provided. Use 'o', 'b', 't', or 'e' to resolve.")
		return
	}

	file := v.conflictFiles[v.currentFileIndex]

	// Extract content lines (remove source prefixes)
	lines := make([]string, len(v.resolvedLines))
	for i, rl := range v.resolvedLines {
		lines[i] = rl.Content
	}

	// Write to file
	if err := v.repo.WriteResolvedFile(file.Path, lines); err != nil {
		ShowErrorModal(v.app, "Save Failed", err.Error())
		return
	}

	// Stage file
	if err := v.repo.StageResolvedFile(file.Path); err != nil {
		ShowErrorModal(v.app, "Stage Failed", err.Error())
		return
	}

	app.ToastSuccess(fmt.Sprintf("Resolved %s", file.Path))

	// Move to next file or finish
	if v.currentFileIndex+1 < len(v.conflictFiles) {
		v.loadFile(v.currentFileIndex + 1)
	} else {
		// All files resolved, show continue prompt
		v.promptContinue()
	}
}

func (v *ConflictResolutionView) areAllFilesResolved() bool {
	// Check if all conflict files have been staged
	files, err := v.repo.ConflictFiles()
	if err != nil {
		return false
	}
	return len(files) == 0
}

func (v *ConflictResolutionView) promptContinue() {
	if !v.areAllFilesResolved() {
		ShowErrorModal(v.app, "Not Ready", "Some files still have conflicts. Resolve all files first.")
		return
	}

	opType := "merge"
	if v.state.Type == git.ConflictRebase {
		opType = "rebase"
	}

	ShowConfirmModal(v.app, "Continue "+strings.Title(opType),
		fmt.Sprintf("All conflicts resolved. Continue %s?", opType),
		func() {
			v.continueOperation()
		})
}

func (v *ConflictResolutionView) continueOperation() {
	var err error
	if v.state.Type == git.ConflictMerge {
		err = v.repo.MergeContinue()
	} else if v.state.Type == git.ConflictRebase {
		err = v.repo.RebaseContinue()
	}

	if err != nil {
		ShowErrorModal(v.app, "Continue Failed", err.Error())
		return
	}

	v.app.Pages().Pop()
	app.ToastSuccess("Operation completed successfully")
}

func (v *ConflictResolutionView) abortOperation() {
	opType := "merge"
	if v.state.Type == git.ConflictRebase {
		opType = "rebase"
	}

	ShowConfirmModal(v.app, "Abort "+strings.Title(opType),
		fmt.Sprintf("Discard all resolutions and abort %s?", opType),
		func() {
			var err error
			if v.state.Type == git.ConflictMerge {
				err = v.repo.MergeAbort()
			} else if v.state.Type == git.ConflictRebase {
				err = v.repo.RebaseAbort()
			}

			if err != nil {
				ShowErrorModal(v.app, "Abort Failed", err.Error())
				return
			}

			v.app.Pages().Pop()
			app.ToastInfo(fmt.Sprintf("%s aborted", strings.Title(opType)))
		})
}

func (v *ConflictResolutionView) clearResolved() {
	ShowConfirmModal(v.app, "Clear Resolved", "Clear all resolved lines?", func() {
		v.resolvedLines = nil
		v.updateResolvedPane()
	})
}

func (v *ConflictResolutionView) updateFocusState() {
	v.oursPanel.SetFocused(v.focusPanel == 0)
	v.basePanel.SetFocused(v.focusPanel == 1)
	v.theirsPanel.SetFocused(v.focusPanel == 2)
	v.resolvedPanel.SetFocused(v.focusPanel == 3)
}

// tview.Primitive implementation

func (v *ConflictResolutionView) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	v.mainSplit.SetRect(x, y, width, height)
	v.mainSplit.Draw(screen)
}

func (v *ConflictResolutionView) GetRect() (int, int, int, int) { return v.Box.GetRect() }
func (v *ConflictResolutionView) SetRect(x, y, w, h int)        { v.Box.SetRect(x, y, w, h) }
func (v *ConflictResolutionView) Focus(d func(tview.Primitive)) { v.Box.Focus(d) }
func (v *ConflictResolutionView) Blur()                         { v.Box.Blur() }
func (v *ConflictResolutionView) HasFocus() bool                { return v.Box.HasFocus() }

func (v *ConflictResolutionView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		if handler := v.mainSplit.MouseHandler(); handler != nil {
			return handler(action, event, setFocus)
		}
		return false, nil
	})
}

func (v *ConflictResolutionView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *ConflictResolutionView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.Box.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		// Handle escape
		if event.Key() == tcell.KeyEscape {
			v.abortOperation()
			return
		}

		// Handle Tab to cycle focus
		if event.Key() == tcell.KeyTab {
			v.focusPanel = (v.focusPanel + 1) % 4
			v.updateFocusState()
			return
		}

		// Handle key shortcuts
		switch event.Rune() {
		case 'o', 'O':
			v.acceptOurs()
			return
		case 'b', 'B':
			if event.Rune() == 'B' {
				// Shift+B = accept both
				v.acceptBoth()
			} else {
				v.acceptBase()
			}
			return
		case 't', 'T':
			v.acceptTheirs()
			return
		case 'e', 'E':
			v.openInEditor()
			return
		case 's', 'S':
			v.saveResolution()
			return
		case 'C':
			// Shift+C = continue operation
			v.promptContinue()
			return
		case 'x', 'X':
			v.abortOperation()
			return
		case 'c':
			// lowercase c = clear resolved
			if v.focusPanel == 3 {
				v.clearResolved()
			}
			return
		case 'n':
			// Next region
			v.moveToNextRegion()
			return
		case 'p':
			// Previous region
			v.moveToPrevRegion()
			return
		case 'j':
			// Scroll down in focused pane
			v.scrollFocusedPane(1)
			return
		case 'k':
			// Scroll up in focused pane
			v.scrollFocusedPane(-1)
			return
		}

		// Delegate to split for other navigation
		if handler := v.mainSplit.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

func (v *ConflictResolutionView) scrollFocusedPane(delta int) {
	var targetText *tview.TextView
	switch v.focusPanel {
	case 0:
		targetText = v.oursText
	case 1:
		targetText = v.baseText
	case 2:
		targetText = v.theirsText
	case 3:
		targetText = v.resolvedText
	}

	if targetText != nil {
		row, col := targetText.GetScrollOffset()
		newRow := row + delta
		if newRow < 0 {
			newRow = 0
		}
		targetText.ScrollTo(newRow, col)
	}
}
