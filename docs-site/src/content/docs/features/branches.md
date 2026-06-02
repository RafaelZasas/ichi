---
title: Branches
description: Create, check out, merge, and rebase branches from the branch view.
---

Press <kbd>b</kbd> from the graph to open the **Branches** view. It lists your
local (and tracking) branches with the current branch marked.

![The branches view listing local branches](/images/views/branches.png)

## Navigating

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between branches |
| <kbd>g</kbd> / <kbd>G</kbd> | Top / bottom |
| <kbd>/</kbd> | Search branches |
| <kbd>Enter</kbd> | Branch details |

## Working with branches

| Key / Command | Action |
|---|---|
| <kbd>b</kbd> | Create a new branch |
| <kbd>c</kbd> / `:checkout` | Check out the selected branch |
| <kbd>d</kbd> | Diff against the selected branch |
| `:merge` | Merge the selected branch into the current one |
| `:rebase` | Rebase the current branch onto the selected one |

You can also sync straight from the branch view:

| Command | Action |
|---|---|
| `:push` | Push the current branch |
| `:pull` | Pull the current branch |
| `:fetch` | Fetch from all remotes |

If a merge or rebase produces conflicts, ichi drops you into the
[conflict resolution](/features/conflicts/) view.

## Next

- [Push, Pull & Fetch](/features/syncing/) for the full syncing story.
- [Conflict Resolution](/features/conflicts/) when histories diverge.
