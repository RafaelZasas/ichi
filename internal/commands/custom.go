package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/config"
	ichiexec "github.com/atterpac/ichi/internal/exec"
	"github.com/atterpac/ichi/internal/selection"
	"github.com/atterpac/ichi/internal/views"
)

// RegisterCustom registers user-defined commands from config. Invalid command
// definitions are skipped and returned as warnings. Custom commands override
// builtins of the same name (replacing them in the registry).
func RegisterCustom(cmds []config.CustomCommand) []string {
	var warnings []string
	for _, c := range cmds {
		if err := config.ValidateCommand(c); err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		cmd := buildCommand(c)
		if c.Key != "" {
			chord, err := parseKeyChord(c.Key)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("custom command %q: %v", c.Name, err))
				continue
			}
			bindKey(chord, cmd)
		}
		registerOrReplace(cmd)
	}
	return warnings
}

// registerOrReplace adds cmd, replacing any existing command with the same name
// (custom commands override builtins).
func registerOrReplace(cmd *Command) {
	if existing, ok := registry[cmd.Name]; ok {
		for i, c := range allCommands {
			if c == existing {
				allCommands[i] = cmd
				break
			}
		}
	} else {
		allCommands = append(allCommands, cmd)
	}
	registry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		registry[alias] = cmd
	}
}

func buildCommand(c config.CustomCommand) *Command {
	desc := c.Description
	if desc == "" {
		desc = "Custom: " + c.Command
	}
	return &Command{
		Name:        c.Name,
		Aliases:     c.Aliases,
		Description: desc,
		Handler:     customHandler(c),
	}
}

func customHandler(c config.CustomCommand) Handler {
	return func(ctx *Context, args []string) error {
		sel := ctx.Selection()
		if err := checkRequires(c.Requires, sel); err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}

		expanded, err := Expand(c.Command, sel, ctx.Repo, args)
		if err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}

		run := func() { runCustom(ctx, c, expanded) }
		if c.Confirm {
			// Defer the modal until after the command-submit handler returns and
			// restores focus to the underlying view. A modal captures the
			// currently-focused widget on push and restores it on dismiss; if we
			// showed it inline here, it would capture the (now hidden) command bar
			// and cancelling would leave the app unfocused.
			go ctx.App.QueueUpdateDraw(func() {
				views.ShowConfirmModal(ctx.App, c.Name,
					fmt.Sprintf("Run:\n%s", expanded), run)
			})
			return nil
		}
		run()
		return nil
	}
}

func checkRequires(requires string, sel *selection.Context) error {
	switch requires {
	case "commit":
		if !sel.HasCommit() {
			return fmt.Errorf("no commit selected")
		}
	case "branch":
		if !sel.HasBranch() {
			return fmt.Errorf("no branch selected")
		}
	case "file":
		if !sel.HasFile() {
			return fmt.Errorf("no file selected")
		}
	case "stash":
		if !sel.HasStash() {
			return fmt.Errorf("no stash selected")
		}
	}
	return nil
}

func runCustom(ctx *Context, c config.CustomCommand, command string) {
	repoRoot := ctx.Repo.Path()

	switch c.View {
	case "editor":
		ctx.App.Suspend(func() {
			_ = ichiexec.RunShellInteractive(repoRoot, command)
		})
		return

	case "pager", "diff":
		var content string
		app.RunAsyncSimple(
			fmt.Sprintf("Running %s...", c.Name),
			func(cctx context.Context) error {
				out, errOut, err := ichiexec.RunShell(cctx, repoRoot, command)
				if err != nil {
					return fmt.Errorf("%s\n%s", strings.TrimSpace(errOut), err)
				}
				content = out
				return nil
			},
			func() {
				view := views.NewOutputView(ctx.App, c.Name, content, c.View == "diff")
				ctx.App.Pages().Push(view)
				ctx.App.Crumbs().SetPath([]string{c.Name})
			},
			func(err error) { views.ShowErrorModal(ctx.App, c.Name+" failed", err.Error()) },
		)
		return

	case "commit", "graph":
		out, _, err := ichiexec.RunShell(context.Background(), repoRoot, command)
		if err != nil {
			views.ShowErrorModal(ctx.App, c.Name+" failed", err.Error())
			return
		}
		ref := firstToken(out)
		if ref == "" {
			app.ToastError(fmt.Sprintf("%s: command produced no commit ref", c.Name))
			return
		}
		view := views.NewCommitView(ctx.App, ctx.Repo, ref)
		ctx.App.Pages().Push(view)
		ctx.App.Crumbs().SetPath([]string{c.Name, ref})
		return

	case "branch":
		// Run for side effects (e.g. create/checkout), then show the branch list.
		if _, errOut, err := ichiexec.RunShell(context.Background(), repoRoot, command); err != nil {
			views.ShowErrorModal(ctx.App, c.Name+" failed", strings.TrimSpace(errOut)+"\n"+err.Error())
			return
		}
		view := views.NewBranchesView(ctx.App, ctx.Repo)
		ctx.App.Pages().Push(view)
		ctx.App.Crumbs().SetPath([]string{"Branches"})
		return

	default: // "none" / ""
		out, errOut, err := ichiexec.RunShell(context.Background(), repoRoot, command)
		if err != nil {
			msg := strings.TrimSpace(errOut)
			if msg == "" {
				msg = err.Error()
			}
			views.ShowErrorModal(ctx.App, c.Name+" failed", msg)
			return
		}
		// Toast the command's output so the result is visible; fall back to a
		// generic confirmation when the command is silent.
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = c.Name + " done"
		}
		app.ToastSuccess(msg)
	}
}

// firstToken returns the first whitespace-delimited token of s.
func firstToken(s string) string {
	return strings.TrimSpace(strings.SplitN(strings.TrimSpace(s), "\n", 2)[0])
}
