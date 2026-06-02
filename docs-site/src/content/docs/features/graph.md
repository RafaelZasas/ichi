---
title: Commit Graph
description: Browse history, inspect commits, and act on them from ichi's home view.
---

The **commit graph** is ichi's home view — it's what you see when the app opens
and what <kbd>g</kbd> always returns you to. It renders your history as a graph,
newest commit at the top, with branch and merge lines drawn alongside.

![The ichi commit graph, with a working-changes panel on the right](/images/views/graph.png)

## Navigating

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between commits |
| <kbd>g</kbd> / <kbd>G</kbd> | Jump to top / bottom |
| <kbd>/</kbd> | Search the log |
| <kbd>Enter</kbd> | Open commit details |

## Acting on a commit

With a commit selected, you can act on it without leaving the graph:

| Key / Command | Action |
|---|---|
| <kbd>d</kbd> / `:diff` | View the commit's diff |
| <kbd>c</kbd> / `:checkout` | Check out the commit |
| <kbd>y</kbd> | Copy the commit hash to the clipboard |
| `:pick` | Cherry-pick the commit onto the current branch |
| `:revert` | Create a commit that undoes it |
| `:reset` | Reset the current branch to it (soft) |

Press <kbd>:</kbd> to run any of the command-line actions against the highlighted
commit. See the full list in [Keybindings & Commands](/guides/keybindings/).

## From here

- Press <kbd>s</kbd> for [Status & staging](/features/staging/).
- Press <kbd>b</kbd> for [Branches](/features/branches/).
- Press <kbd>Enter</kbd> on a commit, then explore its [diff](/features/diff/).
