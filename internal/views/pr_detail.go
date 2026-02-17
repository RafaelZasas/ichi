package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/remote"
)

// PRDetailView shows detailed information about a single PR
type PRDetailView struct {
	*tview.Box
	split      *components.Split
	fileTree   *components.Tree
	contentView *tview.TextView
	app        *layout.App

	provider   remote.Provider
	repoPath   string
	pr         *remote.PullRequest
	files      []remote.ChangedFile
	reviews    []remote.Review
	comments   []remote.Comment
	checks     []remote.Check

	mode       prViewMode // files, conversation, checks
}

type prViewMode int

const (
	prModeFiles prViewMode = iota
	prModeConversation
	prModeChecks
)

// NewPRDetailView creates a new PR detail view
func NewPRDetailView(app *layout.App, provider remote.Provider, repoPath string, pr *remote.PullRequest) *PRDetailView {
	v := &PRDetailView{
		Box:         tview.NewBox(),
		fileTree:    components.NewTree(),
		contentView: tview.NewTextView(),
		app:         app,
		provider:    provider,
		repoPath:    repoPath,
		pr:          pr,
		mode:        prModeFiles,
	}
	v.setup()
	return v
}

func (v *PRDetailView) setup() {
	v.Box.SetBackgroundColor(theme.Bg())
	theme.Register(v.Box)

	// Configure content view
	v.contentView.SetDynamicColors(true)
	v.contentView.SetWordWrap(false)
	v.contentView.SetBackgroundColor(theme.Bg())
	theme.Register(v.contentView)

	// Configure tree
	v.fileTree.SetShowLines(true).
		SetShowIcons(true).
		SetIndentSize(2).
		SetOnHighlight(v.onNodeHighlight).
		SetOnSelect(v.onNodeSelect)

	// Create split layout
	v.split = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.35).
		SetShowDivider(true).
		SetLeft(components.NewPanel().SetTitle("Files").SetContent(v.fileTree)).
		SetRight(components.NewPanel().SetTitle("Diff").SetContent(v.contentView))
}

// nav.Component interface

func (v *PRDetailView) Name() string {
	return "PR Details"
}

func (v *PRDetailView) Start() {
	v.loadData()
}

func (v *PRDetailView) Stop() {}

func (v *PRDetailView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "1", Description: "Files"},
		{Key: "2", Description: "Conversation"},
		{Key: "3", Description: "Checks"},
		{Key: "a", Description: "Approve"},
		{Key: "x", Description: "Request changes"},
		{Key: "c", Description: "Comment"},
		{Key: "m", Description: "Merge"},
		{Key: "o", Description: "Open in browser"},
		{Key: "Esc", Description: "Back"},
	}
}

func (v *PRDetailView) loadData() {
	// Load files
	files, err := v.provider.GetPRFiles(v.repoPath, v.pr.Number)
	if err == nil {
		v.files = files
	}

	// Load reviews
	reviews, err := v.provider.ListReviews(v.repoPath, v.pr.Number)
	if err == nil {
		v.reviews = reviews
	}

	// Load comments
	comments, err := v.provider.ListComments(v.repoPath, v.pr.Number)
	if err == nil {
		v.comments = comments
	}

	// Load checks
	checks, err := v.provider.GetChecks(v.repoPath, v.pr.Number)
	if err == nil {
		v.checks = checks
	}

	v.buildTree()
}

func (v *PRDetailView) buildTree() {
	switch v.mode {
	case prModeFiles:
		v.buildFilesTree()
	case prModeConversation:
		v.buildConversationTree()
	case prModeChecks:
		v.buildChecksTree()
	}
}

func (v *PRDetailView) buildFilesTree() {
	root := &components.TreeNode{
		ID:       "root",
		Label:    fmt.Sprintf("Changed Files (%d)", len(v.files)),
		Expanded: true,
	}

	for i := range v.files {
		file := &v.files[i]
		icon := v.fileStatusIcon(file.Status)
		stats := fmt.Sprintf("+%d -%d", file.Additions, file.Deletions)

		node := &components.TreeNode{
			ID:    file.Path,
			Label: fmt.Sprintf("%s %s", file.Path, stats),
			Icon:  icon,
			Data:  file,
		}
		root.AddChild(node)
	}

	v.fileTree.SetRoot(root)
	v.fileTree.ExpandAll()

	// Show first file
	if node := v.fileTree.GetSelected(); node != nil {
		v.onNodeHighlight(node)
	}
}

func (v *PRDetailView) buildConversationTree() {
	root := &components.TreeNode{
		ID:       "root",
		Label:    "Conversation",
		Expanded: true,
	}

	// Add reviews
	for i := range v.reviews {
		review := &v.reviews[i]
		icon := v.reviewStateIcon(review.State)
		label := fmt.Sprintf("%s %s", review.Author.Login, review.State)

		node := &components.TreeNode{
			ID:    fmt.Sprintf("review:%d", review.ID),
			Label: label,
			Icon:  icon,
			Data:  review,
		}
		root.AddChild(node)
	}

	// Add inline comments grouped by file
	fileComments := make(map[string][]*remote.Comment)
	for i := range v.comments {
		c := &v.comments[i]
		if c.Path != "" {
			fileComments[c.Path] = append(fileComments[c.Path], c)
		}
	}

	for path, comments := range fileComments {
		fileNode := &components.TreeNode{
			ID:       "file:" + path,
			Label:    fmt.Sprintf("%s (%d comments)", path, len(comments)),
			Icon:     "💬",
			Expanded: true,
		}

		for _, c := range comments {
			commentNode := &components.TreeNode{
				ID:    fmt.Sprintf("comment:%d", c.ID),
				Label: fmt.Sprintf("L%d: %s", c.Line, truncateString(c.Body, 40)),
				Icon:  "•",
				Data:  c,
			}
			fileNode.AddChild(commentNode)
		}

		root.AddChild(fileNode)
	}

	v.fileTree.SetRoot(root)
	v.fileTree.ExpandAll()

	if node := v.fileTree.GetSelected(); node != nil {
		v.onNodeHighlight(node)
	}
}

func (v *PRDetailView) buildChecksTree() {
	root := &components.TreeNode{
		ID:       "root",
		Label:    fmt.Sprintf("Checks (%d)", len(v.checks)),
		Expanded: true,
	}

	for i := range v.checks {
		check := &v.checks[i]
		icon := v.checkStatusIcon(check)

		node := &components.TreeNode{
			ID:    fmt.Sprintf("check:%d", i),
			Label: check.Name,
			Icon:  icon,
			Data:  check,
		}
		root.AddChild(node)
	}

	v.fileTree.SetRoot(root)
	v.fileTree.ExpandAll()

	if node := v.fileTree.GetSelected(); node != nil {
		v.onNodeHighlight(node)
	}
}

func (v *PRDetailView) fileStatusIcon(status remote.FileStatus) string {
	switch status {
	case remote.FileAdded:
		return "+"
	case remote.FileModified:
		return "~"
	case remote.FileDeleted:
		return "-"
	case remote.FileRenamed:
		return "→"
	default:
		return "•"
	}
}

func (v *PRDetailView) reviewStateIcon(state remote.ReviewState) string {
	switch state {
	case remote.ReviewApproved:
		return "✓"
	case remote.ReviewChangesRequested:
		return "✗"
	case remote.ReviewCommented:
		return "💬"
	case remote.ReviewPending:
		return "○"
	default:
		return "•"
	}
}

func (v *PRDetailView) checkStatusIcon(check *remote.Check) string {
	if check.Status != remote.CheckCompleted {
		return "○" // In progress
	}
	switch check.Conclusion {
	case remote.CheckSuccess:
		return "✓"
	case remote.CheckFailure:
		return "✗"
	case remote.CheckSkipped:
		return "⊘"
	default:
		return "•"
	}
}

func (v *PRDetailView) onNodeHighlight(node *components.TreeNode) {
	if node == nil || node.Data == nil {
		return
	}

	switch data := node.Data.(type) {
	case *remote.ChangedFile:
		v.renderFileDiff(data)
	case *remote.Review:
		v.renderReview(data)
	case *remote.Comment:
		v.renderComment(data)
	case *remote.Check:
		v.renderCheck(data)
	}
}

func (v *PRDetailView) onNodeSelect(node *components.TreeNode) {
	// Could open inline comment editor, etc.
}

func (v *PRDetailView) renderFileDiff(file *remote.ChangedFile) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n", theme.TagAccent(), file.Path))
	text.WriteString(fmt.Sprintf("[%s]%s | +%d -%d[-]\n\n", theme.TagFgDim(), file.Status, file.Additions, file.Deletions))

	if file.Patch == "" {
		text.WriteString("[" + theme.TagFgDim() + "]No patch available (binary file?)[-]")
	} else {
		// Parse and render the patch with colors
		lines := strings.Split(file.Patch, "\n")
		for _, line := range lines {
			if len(line) == 0 {
				text.WriteString("\n")
				continue
			}

			switch line[0] {
			case '+':
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagSuccess(), tview.Escape(line)))
			case '-':
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagError(), tview.Escape(line)))
			case '@':
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagInfo(), tview.Escape(line)))
			default:
				text.WriteString(tview.Escape(line) + "\n")
			}
		}
	}

	v.contentView.SetText(text.String())
	v.contentView.ScrollToBeginning()
}

func (v *PRDetailView) renderReview(review *remote.Review) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]Review by %s[-:-:-]\n", theme.TagAccent(), review.Author.Login))

	stateColor := theme.TagFgDim()
	switch review.State {
	case remote.ReviewApproved:
		stateColor = theme.TagSuccess()
	case remote.ReviewChangesRequested:
		stateColor = theme.TagError()
	}
	text.WriteString(fmt.Sprintf("[%s]%s[-]\n", stateColor, review.State))
	text.WriteString(fmt.Sprintf("[%s]%s[-]\n\n", theme.TagFgDim(), formatTimeAgo(review.CreatedAt)))

	if review.Body != "" {
		text.WriteString(tview.Escape(review.Body))
	}

	v.contentView.SetText(text.String())
	v.contentView.ScrollToBeginning()
}

func (v *PRDetailView) renderComment(comment *remote.Comment) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]Comment by %s[-:-:-]\n", theme.TagAccent(), comment.Author.Login))
	text.WriteString(fmt.Sprintf("[%s]%s[-]\n\n", theme.TagFgDim(), formatTimeAgo(comment.CreatedAt)))

	if comment.Path != "" {
		text.WriteString(fmt.Sprintf("[%s]File:[-] %s:%d\n\n", theme.TagFgDim(), comment.Path, comment.Line))
	}

	if comment.DiffHunk != "" {
		text.WriteString(fmt.Sprintf("[%s]Context:[-]\n", theme.TagFgDim()))
		lines := strings.Split(comment.DiffHunk, "\n")
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			switch line[0] {
			case '+':
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagSuccess(), tview.Escape(line)))
			case '-':
				text.WriteString(fmt.Sprintf("[%s]%s[-]\n", theme.TagError(), tview.Escape(line)))
			default:
				text.WriteString(tview.Escape(line) + "\n")
			}
		}
		text.WriteString("\n")
	}

	text.WriteString(tview.Escape(comment.Body))

	v.contentView.SetText(text.String())
	v.contentView.ScrollToBeginning()
}

func (v *PRDetailView) renderCheck(check *remote.Check) {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("[%s::b]%s[-:-:-]\n\n", theme.TagAccent(), check.Name))

	statusColor := theme.TagFgDim()
	statusText := string(check.Status)
	if check.Status == remote.CheckCompleted {
		statusText = string(check.Conclusion)
		switch check.Conclusion {
		case remote.CheckSuccess:
			statusColor = theme.TagSuccess()
		case remote.CheckFailure:
			statusColor = theme.TagError()
		}
	}
	text.WriteString(fmt.Sprintf("[%s]Status:[-] [%s]%s[-]\n", theme.TagFgDim(), statusColor, statusText))

	if !check.StartedAt.IsZero() {
		text.WriteString(fmt.Sprintf("[%s]Started:[-] %s\n", theme.TagFgDim(), formatTimeAgo(check.StartedAt)))
	}
	if !check.CompletedAt.IsZero() {
		text.WriteString(fmt.Sprintf("[%s]Completed:[-] %s\n", theme.TagFgDim(), formatTimeAgo(check.CompletedAt)))
	}

	if check.URL != "" {
		text.WriteString(fmt.Sprintf("\n[%s]URL:[-] %s\n", theme.TagFgDim(), check.URL))
	}

	v.contentView.SetText(text.String())
	v.contentView.ScrollToBeginning()
}

func (v *PRDetailView) submitReview(state remote.ReviewState) {
	ShowInputModal(v.app, "Review Comment", "Add a comment (optional):", func(body string) {
		err := v.provider.SubmitReview(v.repoPath, v.pr.Number, &remote.SubmitReviewInput{
			State: state,
			Body:  body,
		})
		if err != nil {
			ShowErrorModal(v.app, "Review Failed", err.Error())
			return
		}
		ShowInfoModal(v.app, "Review Submitted", fmt.Sprintf("Review submitted: %s", state))
		v.loadData()
	})
}

func (v *PRDetailView) addComment() {
	ShowInputModal(v.app, "Add Comment", "Comment:", func(body string) {
		if body == "" {
			return
		}
		_, err := v.provider.AddComment(v.repoPath, v.pr.Number, &remote.CommentInput{
			Body: body,
		})
		if err != nil {
			ShowErrorModal(v.app, "Comment Failed", err.Error())
			return
		}
		v.loadData()
	})
}

func (v *PRDetailView) mergePR() {
	ShowConfirmModal(v.app, "Merge PR",
		fmt.Sprintf("Merge PR #%d?\n\n%s", v.pr.Number, v.pr.Title),
		func() {
			err := v.provider.MergePR(v.repoPath, v.pr.Number, remote.MergeOpts{
				Method:       remote.MergeSquash,
				DeleteBranch: true,
			})
			if err != nil {
				ShowErrorModal(v.app, "Merge Failed", err.Error())
				return
			}
			ShowInfoModal(v.app, "Merged", "PR merged successfully")
			v.app.Pages().Pop()
		})
}

func (v *PRDetailView) setMode(mode prViewMode) {
	v.mode = mode

	// Update panel title
	var title string
	switch mode {
	case prModeFiles:
		title = "Files"
	case prModeConversation:
		title = "Conversation"
	case prModeChecks:
		title = "Checks"
	}

	v.split.SetLeft(components.NewPanel().SetTitle(title).SetContent(v.fileTree))
	v.buildTree()
}

// Draw renders the view
func (v *PRDetailView) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	v.split.SetRect(x, y, width, height)
	v.split.Draw(screen)
}

// tview.Primitive delegation

func (v *PRDetailView) GetRect() (int, int, int, int) { return v.Box.GetRect() }
func (v *PRDetailView) SetRect(x, y, w, h int)        { v.Box.SetRect(x, y, w, h) }
func (v *PRDetailView) Focus(d func(tview.Primitive)) { v.Box.Focus(d) }
func (v *PRDetailView) Blur()                         { v.Box.Blur() }
func (v *PRDetailView) HasFocus() bool                { return v.Box.HasFocus() }

func (v *PRDetailView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		if handler := v.split.MouseHandler(); handler != nil {
			return handler(action, event, setFocus)
		}
		return false, nil
	})
}

func (v *PRDetailView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *PRDetailView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.Box.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().Pop()
			return
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case '1':
				v.setMode(prModeFiles)
				return
			case '2':
				v.setMode(prModeConversation)
				return
			case '3':
				v.setMode(prModeChecks)
				return
			case 'a':
				v.submitReview(remote.ReviewApproved)
				return
			case 'x':
				v.submitReview(remote.ReviewChangesRequested)
				return
			case 'c':
				v.addComment()
				return
			case 'm':
				v.mergePR()
				return
			case 'o':
				openURL(v.pr.URL)
				return
			case 'q':
				v.app.Pages().Pop()
				return
			}
		}

		// Pass to tree for navigation
		if handler := v.fileTree.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
