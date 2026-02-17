package commands

import (
	"fmt"

	"github.com/atterpac/gxt/internal/views"
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
		Description: "Quit gxt",
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
	if err := ctx.Repo.FetchAll(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	return nil
}
