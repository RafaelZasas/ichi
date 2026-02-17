package views

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"

	"github.com/atterpac/gxt/internal/git"
	"github.com/atterpac/gxt/internal/remote"
	_ "github.com/atterpac/gxt/internal/remote/github" // Register GitHub provider
)

// PRListView displays a list of pull requests
type PRListView struct {
	*tview.Box
	split      *components.Split
	prTree     *components.Tree
	detailText *tview.TextView
	app        *layout.App
	repo       *git.Repository

	provider   remote.Provider
	repoPath   string // owner/repo
	prs        []remote.PullRequest
	stateFilter remote.PRState
}

// NewPRListView creates a new PR list view
func NewPRListView(app *layout.App, repo *git.Repository) *PRListView {
	v := &PRListView{
		Box:         tview.NewBox(),
		prTree:      components.NewTree(),
		detailText:  tview.NewTextView(),
		app:         app,
		repo:        repo,
		stateFilter: remote.PROpen,
	}
	v.setup()
	return v
}

func (v *PRListView) setup() {
	v.Box.SetBackgroundColor(theme.Bg())
	theme.Register(v.Box)

	// Configure detail text view
	v.detailText.SetDynamicColors(true)
	v.detailText.SetWordWrap(true)
	v.detailText.SetBackgroundColor(theme.Bg())
	theme.Register(v.detailText)

	// Configure tree
	v.prTree.SetShowLines(false).
		SetShowIcons(true).
		SetIndentSize(2).
		SetOnHighlight(v.onPRHighlight).
		SetOnSelect(v.onPRSelect)

	// Create split layout (50/50)
	v.split = components.NewSplit().
		SetDirection(components.SplitHorizontal).
		SetRatio(0.5).
		SetShowDivider(true).
		SetLeft(components.NewPanel().SetTitle("Pull Requests").SetContent(v.prTree)).
		SetRight(components.NewPanel().SetTitle("Details").SetContent(v.detailText))
}

// nav.Component interface

func (v *PRListView) Name() string {
	return "Pull Requests"
}

func (v *PRListView) Start() {
	v.initProvider()
}

func (v *PRListView) Stop() {}

func (v *PRListView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Navigate"},
		{Key: "Enter", Description: "View PR"},
		{Key: "c", Description: "Checkout"},
		{Key: "o", Description: "Open in browser"},
		{Key: "f", Description: "Filter state"},
		{Key: "r", Description: "Refresh"},
		{Key: "Esc", Description: "Back"},
	}
}

func (v *PRListView) initProvider() {
	// Get remote URL
	remoteURL := v.repo.RemoteURL("origin")
	if remoteURL == "" {
		ShowErrorModal(v.app, "Error", "No remote 'origin' found")
		return
	}

	// Detect provider
	provider, repoPath, err := remote.DetectFromURL(remoteURL)
	if err != nil {
		ShowErrorModal(v.app, "Error", err.Error())
		return
	}

	// Authenticate
	if err := provider.Authenticate(); err != nil {
		ShowErrorModal(v.app, "Authentication Error", err.Error())
		return
	}

	v.provider = provider
	v.repoPath = repoPath
	v.loadPRs()
}

func (v *PRListView) loadPRs() {
	if v.provider == nil {
		return
	}

	prs, err := v.provider.ListPRs(v.repoPath, remote.ListPRsOpts{
		State: v.stateFilter,
		Limit: 50,
	})
	if err != nil {
		ShowErrorModal(v.app, "Error", "Failed to load PRs: "+err.Error())
		return
	}

	v.prs = prs
	v.buildTree()
}

func (v *PRListView) buildTree() {
	root := &components.TreeNode{
		ID:       "root",
		Label:    fmt.Sprintf("Pull Requests (%d)", len(v.prs)),
		Expanded: true,
	}

	for i := range v.prs {
		pr := &v.prs[i]
		node := v.buildPRNode(pr)
		root.AddChild(node)
	}

	v.prTree.SetRoot(root)
	v.prTree.ExpandAll()

	// Trigger initial detail
	if node := v.prTree.GetSelected(); node != nil {
		v.onPRHighlight(node)
	}
}

func (v *PRListView) buildPRNode(pr *remote.PullRequest) *components.TreeNode {
	// Status icon
	icon := v.prStateIcon(pr)

	// Title with number
	label := fmt.Sprintf("#%d %s", pr.Number, pr.Title)
	if len(label) > 60 {
		label = label[:57] + "..."
	}

	return &components.TreeNode{
		ID:    fmt.Sprintf("pr:%d", pr.Number),
		Label: label,
		Icon:  icon,
		Data:  pr,
	}
}

func (v *PRListView) prStateIcon(pr *remote.PullRequest) string {
	if pr.Draft {
		return "◌" // Draft
	}
	switch pr.State {
	case remote.PROpen:
		return "○" // Open
	case remote.PRMerged:
		return "●" // Merged
	case remote.PRClosed:
		return "✕" // Closed
	default:
		return "•"
	}
}

func (v *PRListView) onPRHighlight(node *components.TreeNode) {
	if node == nil || node.Data == nil {
		return
	}

	pr, ok := node.Data.(*remote.PullRequest)
	if !ok {
		return
	}

	v.renderPRDetail(pr)
}

func (v *PRListView) onPRSelect(node *components.TreeNode) {
	if node == nil || node.Data == nil {
		return
	}

	pr, ok := node.Data.(*remote.PullRequest)
	if !ok {
		return
	}

	// Open PR detail view
	v.openPRView(pr)
}

func (v *PRListView) renderPRDetail(pr *remote.PullRequest) {
	var text strings.Builder

	// Title
	text.WriteString(fmt.Sprintf("[%s::b]#%d %s[-:-:-]\n\n", theme.TagAccent(), pr.Number, tview.Escape(pr.Title)))

	// State and author
	stateColor := theme.TagSuccess()
	if pr.State == remote.PRClosed {
		stateColor = theme.TagError()
	} else if pr.State == remote.PRMerged {
		stateColor = theme.TagInfo()
	}

	text.WriteString(fmt.Sprintf("[%s]State:[-] [%s]%s[-]", theme.TagFgDim(), stateColor, pr.State))
	if pr.Draft {
		text.WriteString(fmt.Sprintf(" [%s](draft)[-]", theme.TagFgDim()))
	}
	text.WriteString("\n")

	text.WriteString(fmt.Sprintf("[%s]Author:[-] %s\n", theme.TagFgDim(), pr.Author.Login))
	text.WriteString(fmt.Sprintf("[%s]Branch:[-] %s → %s\n", theme.TagFgDim(), pr.HeadBranch, pr.BaseBranch))

	// Stats
	text.WriteString(fmt.Sprintf("[%s]Changed:[-] [%s]+%d[-] [%s]-%d[-] in %d files\n",
		theme.TagFgDim(),
		theme.TagSuccess(), pr.Additions,
		theme.TagError(), pr.Deletions,
		pr.ChangedFiles))

	// Time
	text.WriteString(fmt.Sprintf("[%s]Created:[-] %s\n", theme.TagFgDim(), formatTimeAgo(pr.CreatedAt)))
	text.WriteString(fmt.Sprintf("[%s]Updated:[-] %s\n", theme.TagFgDim(), formatTimeAgo(pr.UpdatedAt)))

	// Labels
	if len(pr.Labels) > 0 {
		text.WriteString(fmt.Sprintf("\n[%s]Labels:[-] %s\n", theme.TagFgDim(), strings.Join(pr.Labels, ", ")))
	}

	// Reviewers
	if len(pr.Reviewers) > 0 {
		reviewers := make([]string, len(pr.Reviewers))
		for i, r := range pr.Reviewers {
			reviewers[i] = r.Login
		}
		text.WriteString(fmt.Sprintf("[%s]Reviewers:[-] %s\n", theme.TagFgDim(), strings.Join(reviewers, ", ")))
	}

	// Mergeable
	if pr.Mergeable != nil {
		mergeStatus := "yes"
		mergeColor := theme.TagSuccess()
		if !*pr.Mergeable {
			mergeStatus = "conflicts"
			mergeColor = theme.TagError()
		}
		text.WriteString(fmt.Sprintf("[%s]Mergeable:[-] [%s]%s[-]\n", theme.TagFgDim(), mergeColor, mergeStatus))
	}

	// Body preview
	if pr.Body != "" {
		text.WriteString(fmt.Sprintf("\n[%s::b]Description[-:-:-]\n", theme.TagFg()))
		body := pr.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		text.WriteString(tview.Escape(body))
	}

	v.detailText.SetText(text.String())
	v.detailText.ScrollToBeginning()
}

func (v *PRListView) openPRView(pr *remote.PullRequest) {
	prView := NewPRDetailView(v.app, v.provider, v.repoPath, pr)
	v.app.Pages().Push(prView)
	v.app.Crumbs().SetPath([]string{"PRs", fmt.Sprintf("#%d", pr.Number)})
}

func (v *PRListView) checkoutPR() {
	node := v.prTree.GetSelected()
	if node == nil || node.Data == nil {
		return
	}

	pr, ok := node.Data.(*remote.PullRequest)
	if !ok {
		return
	}

	// Checkout the PR branch
	if err := v.repo.Checkout(pr.HeadBranch); err != nil {
		ShowErrorModal(v.app, "Checkout Failed", err.Error())
		return
	}

	ShowInfoModal(v.app, "Checked Out", fmt.Sprintf("Switched to branch: %s", pr.HeadBranch))
}

func (v *PRListView) openInBrowser() {
	node := v.prTree.GetSelected()
	if node == nil || node.Data == nil {
		return
	}

	pr, ok := node.Data.(*remote.PullRequest)
	if !ok {
		return
	}

	// Use xdg-open/open to open URL
	openURL(pr.URL)
}

func (v *PRListView) cycleStateFilter() {
	switch v.stateFilter {
	case remote.PROpen:
		v.stateFilter = remote.PRClosed
	case remote.PRClosed:
		v.stateFilter = remote.PRMerged
	case remote.PRMerged:
		v.stateFilter = remote.PRAll
	default:
		v.stateFilter = remote.PROpen
	}
	v.loadPRs()
}

// Draw renders the view
func (v *PRListView) Draw(screen tcell.Screen) {
	v.Box.DrawForSubclass(screen, v)
	x, y, width, height := v.GetInnerRect()

	if width <= 0 || height <= 0 {
		return
	}

	v.split.SetRect(x, y, width, height)
	v.split.Draw(screen)
}

// tview.Primitive delegation

func (v *PRListView) GetRect() (int, int, int, int) { return v.Box.GetRect() }
func (v *PRListView) SetRect(x, y, w, h int)        { v.Box.SetRect(x, y, w, h) }
func (v *PRListView) Focus(d func(tview.Primitive)) { v.Box.Focus(d) }
func (v *PRListView) Blur()                         { v.Box.Blur() }
func (v *PRListView) HasFocus() bool                { return v.Box.HasFocus() }

func (v *PRListView) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return v.Box.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(tview.Primitive)) (bool, tview.Primitive) {
		if handler := v.split.MouseHandler(); handler != nil {
			return handler(action, event, setFocus)
		}
		return false, nil
	})
}

func (v *PRListView) PasteHandler() func(string, func(tview.Primitive)) { return nil }

func (v *PRListView) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.Box.WrapInputHandler(func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		if event.Key() == tcell.KeyEscape {
			v.app.Pages().Pop()
			return
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'c':
				v.checkoutPR()
				return
			case 'o':
				v.openInBrowser()
				return
			case 'f':
				v.cycleStateFilter()
				return
			case 'r':
				v.loadPRs()
				return
			case 'q':
				v.app.Pages().Pop()
				return
			}
		}

		// Pass to tree for navigation
		if handler := v.prTree.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

// Helper functions

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

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
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
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

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
