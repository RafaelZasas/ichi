package commands

import (
	"fmt"

	"github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/views"
)

func init() {
	// File commands
	Register(&Command{
		Name:        "blame",
		Aliases:     []string{"b"},
		Description: "Show line-by-line file attribution",
		Args:        []ArgSpec{{Name: "file", Type: ArgTypeFile, Required: true}},
		Handler:     handleBlame,
	})

	Register(&Command{
		Name:        "log",
		Aliases:     []string{"l"},
		Description: "Show commits for file",
		Args:        []ArgSpec{{Name: "file", Type: ArgTypeFile, Required: true}},
		Handler:     handleFileLog,
	})

	// Vim-style commands
	Register(&Command{
		Name:        "w",
		Description: "Stage all changes",
		Handler:     handleStageAll,
	})

	Register(&Command{
		Name:        "q",
		Description: "Quit ichi",
		Handler:     handleQuit,
	})

	Register(&Command{
		Name:        "wq",
		Description: "Open commit dialog and quit",
		Handler:     handleCommitAndQuit,
	})

	// Git operations
	Register(&Command{
		Name:        "checkout",
		Aliases:     []string{"co"},
		Description: "Checkout branch or commit",
		Args:        []ArgSpec{{Name: "ref", Type: ArgTypeRef, Required: true}},
		Handler:     handleCheckout,
	})

	Register(&Command{
		Name:        "fetch",
		Aliases:     []string{"f"},
		Description: "Fetch from all remotes",
		Handler:     handleFetch,
	})

	Register(&Command{
		Name:        "push",
		Aliases:     []string{"P"},
		Description: "Push to remote",
		Handler:     handlePush,
	})

	Register(&Command{
		Name:        "pull",
		Aliases:     []string{"p"},
		Description: "Pull from remote",
		Handler:     handlePull,
	})

	// Context-aware commands (operate on current selection)
	Register(&Command{
		Name:        "pick",
		Description: "Cherry-pick selected commit",
		Handler:     handlePick,
	})

	Register(&Command{
		Name:        "revert",
		Description: "Revert selected commit",
		Handler:     handleRevert,
	})

	Register(&Command{
		Name:        "drop",
		Description: "Drop selected commit or stash",
		Handler:     handleDrop,
	})

	Register(&Command{
		Name:        "apply",
		Description: "Apply selected stash",
		Handler:     handleApply,
	})

	Register(&Command{
		Name:        "pop",
		Description: "Pop selected stash",
		Handler:     handlePop,
	})

	Register(&Command{
		Name:        "merge",
		Description: "Merge selected branch",
		Handler:     handleMerge,
	})

	Register(&Command{
		Name:        "rebase",
		Description: "Rebase onto selected branch",
		Handler:     handleRebase,
	})

	Register(&Command{
		Name:        "diff",
		Aliases:     []string{"d"},
		Description: "Show diff for selected commit",
		Handler:     handleDiff,
	})

	Register(&Command{
		Name:        "reset",
		Description: "Reset to selected commit (soft)",
		Handler:     handleReset,
	})

	Register(&Command{
		Name:        "reset!",
		Description: "Reset to selected commit (hard)",
		Handler:     handleResetHard,
	})
}

func handleBlame(ctx *Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blame requires a file argument")
	}
	blameView := views.NewBlameView(ctx.App, ctx.Repo, args[0])
	ctx.App.Pages().Push(blameView)
	ctx.App.Crumbs().SetPath([]string{"Blame", args[0]})
	return nil
}

func handleFileLog(ctx *Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("log requires a file argument")
	}
	fileLogView := views.NewFileLogView(ctx.App, ctx.Repo, args[0])
	ctx.App.Pages().Push(fileLogView)
	ctx.App.Crumbs().SetPath([]string{"History", args[0]})
	return nil
}

func handleStageAll(ctx *Context, args []string) error {
	if err := ctx.Repo.StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	return nil
}

func handleQuit(ctx *Context, args []string) error {
	ctx.App.Stop()
	return nil
}

func handleCommitAndQuit(ctx *Context, args []string) error {
	// Stage all first
	if err := ctx.Repo.StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Show commit input modal
	views.ShowInputModal(ctx.App, "Commit", "Message: ", func(message string) {
		if message == "" {
			return
		}
		if err := ctx.Repo.Commit(message); err != nil {
			views.ShowErrorModal(ctx.App, "Error", fmt.Sprintf("Failed to commit: %v", err))
			return
		}
		ctx.App.Stop()
	})
	return nil
}

func handleCheckout(ctx *Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("checkout requires a ref argument")
	}
	if err := ctx.Repo.Checkout(args[0]); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", args[0], err)
	}
	return nil
}

func handleFetch(ctx *Context, args []string) error {
	app.RunBusy(
		ctx.StatusBar,
		ctx.Repo,
		"Fetching...",
		"Fetched from all remotes",
		ctx.Repo.FetchAll,
	)
	return nil
}

func handlePush(ctx *Context, args []string) error {
	if !ctx.Repo.HasUpstream() {
		branch := ctx.Repo.CurrentBranch()
		views.ShowInputModalWithDefault(ctx.App, "Set Upstream", "Remote branch name:", branch, func(remoteBranch string) {
			if remoteBranch == "" {
				return
			}
			if err := ctx.Repo.PushSetUpstream("origin", remoteBranch); err != nil {
				views.ShowErrorModal(ctx.App, "Push Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Pushed and set upstream to origin/%s", remoteBranch))
		})
		return nil
	}
	if err := ctx.Repo.Push(); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}
	app.ToastSuccess("Pushed to remote")
	return nil
}

func handlePull(ctx *Context, args []string) error {
	app.RunBusy(
		ctx.StatusBar,
		ctx.Repo,
		"Pulling...",
		"Pulled from remote",
		ctx.Repo.Pull,
	)
	return nil
}

// Context-aware command handlers

func handlePick(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasCommit() {
		return fmt.Errorf("pick: no commit selected")
	}
	commit := sel.Commit
	views.ShowConfirmModal(ctx.App, "Cherry-pick",
		fmt.Sprintf("Cherry-pick %s?\n%s", commit.ShortHash, commit.Message),
		func() {
			if err := ctx.Repo.CherryPick(commit.Hash); err != nil {
				views.ShowErrorModal(ctx.App, "Cherry-pick Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Cherry-picked %s", commit.ShortHash))
		})
	return nil
}

func handleRevert(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasCommit() {
		return fmt.Errorf("revert: no commit selected")
	}
	commit := sel.Commit
	views.ShowConfirmModal(ctx.App, "Revert",
		fmt.Sprintf("Revert %s?\n%s", commit.ShortHash, commit.Message),
		func() {
			if err := ctx.Repo.Revert(commit.Hash); err != nil {
				views.ShowErrorModal(ctx.App, "Revert Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Reverted %s", commit.ShortHash))
		})
	return nil
}

func handleDrop(ctx *Context, args []string) error {
	sel := ctx.Selection()
	// Stash drop takes priority if in stash view
	if sel.HasStash() {
		stash := sel.Stash
		views.ShowConfirmModal(ctx.App, "Drop Stash",
			fmt.Sprintf("Drop stash@{%d}?\n%s", stash.Index, stash.Message),
			func() {
				if err := ctx.Repo.StashDropIndex(stash.Index); err != nil {
					views.ShowErrorModal(ctx.App, "Drop Failed", err.Error())
					return
				}
				app.ToastSuccess(fmt.Sprintf("Dropped stash@{%d}", stash.Index))
			})
		return nil
	}
	if sel.HasCommit() {
		commit := sel.Commit
		views.ShowConfirmModal(ctx.App, "Drop Commit",
			fmt.Sprintf("Drop %s from history?\n%s\nThis rewrites history!", commit.ShortHash, commit.Message),
			func() {
				if err := ctx.Repo.DropCommit(commit.Hash); err != nil {
					views.ShowErrorModal(ctx.App, "Drop Failed", err.Error())
					return
				}
				app.ToastSuccess(fmt.Sprintf("Dropped %s", commit.ShortHash))
			})
		return nil
	}
	return fmt.Errorf("drop: no commit or stash selected")
}

func handleApply(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasStash() {
		return fmt.Errorf("apply: no stash selected")
	}
	if err := ctx.Repo.StashApplyIndex(sel.Stash.Index); err != nil {
		return fmt.Errorf("failed to apply stash@{%d}: %w", sel.Stash.Index, err)
	}
	app.ToastSuccess(fmt.Sprintf("Applied stash@{%d}", sel.Stash.Index))
	return nil
}

func handlePop(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasStash() {
		return fmt.Errorf("pop: no stash selected")
	}
	if err := ctx.Repo.StashPopIndex(sel.Stash.Index); err != nil {
		return fmt.Errorf("failed to pop stash@{%d}: %w", sel.Stash.Index, err)
	}
	app.ToastSuccess(fmt.Sprintf("Popped stash@{%d}", sel.Stash.Index))
	return nil
}

func handleMerge(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasBranch() {
		return fmt.Errorf("merge: no branch selected")
	}
	branch := sel.Branch.Name
	views.ShowConfirmModal(ctx.App, "Merge",
		fmt.Sprintf("Merge %s into %s?", branch, ctx.Repo.CurrentBranch()),
		func() {
			if err := ctx.Repo.MergeBranch(branch); err != nil {
				views.ShowErrorModal(ctx.App, "Merge Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Merged %s", branch))
		})
	return nil
}

func handleRebase(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasBranch() {
		return fmt.Errorf("rebase: no branch selected")
	}
	branch := sel.Branch.Name
	views.ShowConfirmModal(ctx.App, "Rebase",
		fmt.Sprintf("Rebase %s onto %s?", ctx.Repo.CurrentBranch(), branch),
		func() {
			if err := ctx.Repo.RebaseBranch(branch); err != nil {
				views.ShowErrorModal(ctx.App, "Rebase Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Rebased onto %s", branch))
		})
	return nil
}

func handleDiff(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasCommit() {
		return fmt.Errorf("diff: no commit selected")
	}
	diffView := views.NewDiffView(ctx.App, ctx.Repo, sel.Commit.Hash)
	ctx.App.Pages().Push(diffView)
	ctx.App.Crumbs().SetPath([]string{"Diff", sel.Commit.ShortHash})
	return nil
}

func handleReset(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasCommit() {
		return fmt.Errorf("reset: no commit selected")
	}
	commit := sel.Commit
	views.ShowConfirmModal(ctx.App, "Soft Reset",
		fmt.Sprintf("Soft reset to %s?\n%s\nChanges will be kept staged.", commit.ShortHash, commit.Message),
		func() {
			if err := ctx.Repo.ResetSoft(commit.Hash); err != nil {
				views.ShowErrorModal(ctx.App, "Reset Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Reset to %s", commit.ShortHash))
		})
	return nil
}

func handleResetHard(ctx *Context, args []string) error {
	sel := ctx.Selection()
	if !sel.HasCommit() {
		return fmt.Errorf("reset!: no commit selected")
	}
	commit := sel.Commit
	views.ShowConfirmModal(ctx.App, "Hard Reset",
		fmt.Sprintf("Hard reset to %s?\n%s\nAll uncommitted changes will be LOST!", commit.ShortHash, commit.Message),
		func() {
			if err := ctx.Repo.ResetHard(commit.Hash); err != nil {
				views.ShowErrorModal(ctx.App, "Reset Failed", err.Error())
				return
			}
			app.ToastSuccess(fmt.Sprintf("Hard reset to %s", commit.ShortHash))
		})
	return nil
}
