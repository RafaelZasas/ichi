package app

import (
	"fmt"
	"sync"

	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"

	"github.com/atterpac/gxt/internal/git"
)

// Nerd Font icons for git
const (
	IconGitBranch   = "" // nf-dev-git_branch
	IconGitCommit   = "" // nf-dev-git_commit
	IconGitMerge    = "" // nf-dev-git_merge
	IconGitPull     = "" // nf-fa-arrow_down
	IconGitPush     = "" // nf-fa-arrow_up
	IconGitStash    = "" // nf-fa-inbox (stash)
	IconGitStaged   = "" // nf-fa-check (staged)
	IconGitUnstaged = "" // nf-fa-pencil (unstaged/modified)
	IconGitAdded    = "" // nf-fa-plus
	IconGitRemoved  = "" // nf-fa-minus
)

// UpdateStatusBar updates the status bar with repository information.
func UpdateStatusBar(statusBar *layout.StatusBar, repo *git.Repository) {
	var (
		branch     string
		shortHead  string
		ahead      int
		behind     int
		isDetached bool
		staged     git.ChangeStats
		unstaged   git.ChangeStats
		stashCount int
	)

	var wg sync.WaitGroup
	wg.Add(5)
	go func() { defer wg.Done(); branch = repo.CurrentBranch() }()
	go func() { defer wg.Done(); shortHead = repo.ShortHEAD() }()
	go func() { defer wg.Done(); ahead, behind = repo.AheadBehind() }()
	go func() { defer wg.Done(); isDetached = repo.IsDetachedHEAD() }()
	go func() { defer wg.Done(); staged, unstaged, stashCount = repo.StatusCounts() }()
	wg.Wait()

	statusBar.ClearSections()

	// Set title
	statusBar.SetTitle("gxt")

	// Connection status (repo path)
	statusBar.SetConnectionStatus(true, repo.Path())

	// Branch/HEAD section with icon
	if isDetached {
		statusBar.AddSection(layout.StatusSection{
			Icon:  theme.IconWarning,
			Text:  "DETACHED",
			Color: theme.Warning(),
		})
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitCommit,
			Text:  shortHead,
			Color: theme.Accent(),
		})
	} else {
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitBranch,
			Text:  branch,
			Color: theme.Accent(),
		})
		// Also show the commit hash
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitCommit,
			Text:  shortHead,
			Color: theme.FgDim(),
		})
	}

	// Ahead/behind info
	if ahead > 0 || behind > 0 {
		text := ""
		if ahead > 0 {
			text += fmt.Sprintf("↑%d", ahead)
		}
		if behind > 0 {
			if text != "" {
				text += " "
			}
			text += fmt.Sprintf("↓%d", behind)
		}
		statusBar.AddSection(layout.StatusSection{
			Icon:  theme.IconRefresh,
			Text:  text,
			Color: theme.Warning(),
		})
	}

	// Staged changes (files ready to commit)
	if staged.Files > 0 {
		text := fmt.Sprintf("[%s]%d[-] [%s]+%d[-] [%s]-%d[-]",
			theme.TagSuccess(), staged.Files,
			theme.TagSuccess(), staged.Insertions,
			theme.TagError(), staged.Deletions)
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitStaged,
			Text:  text,
			Color: theme.Success(),
		})
	}

	// Unstaged changes (modified tracked files)
	if unstaged.Files > 0 {
		text := fmt.Sprintf("[%s]%d[-] [%s]+%d[-] [%s]-%d[-]",
			theme.TagWarning(), unstaged.Files,
			theme.TagSuccess(), unstaged.Insertions,
			theme.TagError(), unstaged.Deletions)
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitUnstaged,
			Text:  text,
			Color: theme.Warning(),
		})
	}

	// Untracked files (new files not yet added)
	if unstaged.Untracked > 0 {
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitAdded,
			Text:  fmt.Sprintf("[%s]%d[-]", theme.TagInfo(), unstaged.Untracked),
			Color: theme.Info(),
		})
	}

	// Stash count
	if stashCount > 0 {
		statusBar.AddSection(layout.StatusSection{
			Icon:  IconGitStash,
			Text:  fmt.Sprintf("[%s]%d[-]", theme.TagFgDim(), stashCount),
			Color: theme.FgDim(),
		})
	}
}
