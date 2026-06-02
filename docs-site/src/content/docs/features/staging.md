---
title: Staging & Committing
description: Stage changes by file, hunk, or line, then commit — all from the keyboard.
---

ichi's staging workflow lets you build a commit at whatever granularity you
need: whole files, individual hunks, or hand-picked lines.

## The status view

Press <kbd>s</kbd> from the graph to open **Status**. It lists your changed
files, separated into unstaged and staged.

![The status view listing staged and unstaged files with a details panel](/images/views/status.png)

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between files |
| <kbd>Space</kbd> | Stage / unstage the selected file |
| <kbd>a</kbd> | Stage all changes |
| <kbd>u</kbd> | Unstage all |
| <kbd>d</kbd> | Open the diff for the selected file |
| <kbd>Tab</kbd> | Switch between staged and unstaged |
| <kbd>c</kbd> | Commit |

The `:w` command also stages all changes in one step.

## The staging workflow

For a focused review, the staging workflow puts your unstaged and staged file
trees side by side with a live diff preview: move through the files on the left
and see each one's changes on the right.

![The split staging workflow: unstaged and staged file trees on the left, a diff preview on the right](/images/views/staging.png)

## Staging hunks and lines

For surgical commits, press <kbd>d</kbd> to open the diff, then stage at a finer
grain — by hunk or by hand-picked lines:

![The single-file diff with a selected line highlighted for staging](/images/views/staging-hunks.png)

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move by line |
| <kbd>J</kbd> / <kbd>K</kbd> | Move by hunk |
| <kbd>s</kbd> | Stage the current hunk |
| <kbd>Space</kbd> | Select a line |
| <kbd>Enter</kbd> | Stage the selected lines |
| <kbd>u</kbd> | Unstage |

This is how you split a messy working tree into clean, focused commits without
dropping to `git add -p`.

## Committing

Once you've staged what you want, press <kbd>c</kbd> to open the commit dialog.
Type your message and press <kbd>Enter</kbd> to confirm, or <kbd>Esc</kbd> to
cancel.

In a hurry to commit and leave? `:wq` stages everything, opens the commit
dialog, and quits when you're done.

## Next

- [Push, Pull & Fetch](/features/syncing/) to share your commits.
- [Stashes](/features/stash/) to shelve work in progress instead of committing.
