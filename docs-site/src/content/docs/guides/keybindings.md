---
title: Keybindings & Commands
description: The complete key map and command-line reference for ichi.
---

ichi is modal and keyboard-first. Most keys are contextual — the same letter
does the sensible thing for the view you're in — but a handful are global.

:::tip
Press <kbd>?</kbd> at any time to see the keys available in the current view.
:::

## Global keys

These work from (almost) anywhere:

| Key | Action |
|---|---|
| <kbd>g</kbd> | Return to the commit graph (home) |
| <kbd>s</kbd> | Open Status |
| <kbd>b</kbd> | Open Branches |
| <kbd>S</kbd> | Open Stashes |
| <kbd>p</kbd> | Open Pull Requests |
| <kbd>:</kbd> | Open the command line |
| <kbd>Ctrl</kbd>+<kbd>P</kbd> / <kbd>Ctrl</kbd>+<kbd>K</kbd> | Open the command palette |
| <kbd>T</kbd> | Theme selector |
| <kbd>?</kbd> | Help for the current view |
| <kbd>Esc</kbd> | Go back / close |
| <kbd>q</kbd> | Quit (from the graph) |

## Navigation

Movement keys are consistent across every list and view:

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>↓</kbd> | Down |
| <kbd>k</kbd> / <kbd>↑</kbd> | Up |
| <kbd>g</kbd> / <kbd>G</kbd> | Top / bottom |
| <kbd>/</kbd> | Search within the view |
| <kbd>Enter</kbd> | Select / open details |
| <kbd>Tab</kbd> | Switch panel or focus |

## Commit graph

| Key | Action |
|---|---|
| <kbd>Enter</kbd> | Commit details |
| <kbd>d</kbd> | Diff for the selected commit |
| <kbd>c</kbd> | Check out the selected commit |
| <kbd>y</kbd> | Copy the commit hash |

## Status & staging

| Key | Action |
|---|---|
| <kbd>Space</kbd> | Stage / unstage file |
| <kbd>a</kbd> / <kbd>u</kbd> | Stage all / unstage all |
| <kbd>c</kbd> | Commit |
| <kbd>d</kbd> | Diff selected file |
| <kbd>Tab</kbd> | Switch between staged / unstaged |

In the diff view:

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move by line |
| <kbd>J</kbd> / <kbd>K</kbd> | Move by hunk |
| <kbd>Space</kbd> | Select a line |
| <kbd>s</kbd> | Stage the current hunk |
| <kbd>Enter</kbd> | Stage selected lines |
| <kbd>u</kbd> | Unstage |

## Branches

| Key | Action |
|---|---|
| <kbd>c</kbd> | Check out branch |
| <kbd>b</kbd> | New branch |
| <kbd>d</kbd> | Diff |
| <kbd>/</kbd> | Search branches |

## Command line

Press <kbd>:</kbd> to enter a Git command. The available commands:

| Command | Action |
|---|---|
| `:push` | Push to remote |
| `:pull` | Pull from remote |
| `:fetch` | Fetch from all remotes |
| `:merge <ref>` | Merge the selected branch |
| `:rebase <ref>` | Rebase onto the selected branch |
| `:checkout <ref>` | Check out a branch or commit |
| `:diff` | Show diff for the selected commit |
| `:pick` | Cherry-pick the selected commit |
| `:revert` | Revert the selected commit |
| `:reset` | Reset to the selected commit (soft) |
| `:drop` | Drop the selected commit or stash |
| `:apply` | Apply the selected stash |
| `:pop` | Pop the selected stash |
| `:blame` | Line-by-line attribution for a file |
| `:log` | Show commits for a file |
| `:w` | Stage all changes |
| `:wq` | Open the commit dialog and quit |
| `:q` | Quit |

The [command palette](/features/command-palette/)
(<kbd>Ctrl</kbd>+<kbd>P</kbd>) fuzzy-finds the same actions if you'd rather
search than type.
