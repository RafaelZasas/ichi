---
title: Stashes
description: Shelve work in progress and restore it later with the stash view.
---

Stashes let you set aside uncommitted work without making a commit. Press
<kbd>S</kbd> from the graph to open the **Stash** view, which lists your stash
entries newest-first.

![The stash view listing shelved work-in-progress entries](/images/views/stash.png)

## Navigating

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between stash entries |
| <kbd>Enter</kbd> | View the stash contents |
| <kbd>/</kbd> | Search stashes |

## Restoring and removing

| Command | Action |
|---|---|
| `:apply` | Apply the selected stash, keeping it in the list |
| `:pop` | Apply the selected stash and remove it |
| `:drop` | Delete the selected stash without applying it |

Use `:apply` when you want to reuse the same shelved change more than once, and
`:pop` for the common "put it back and move on" case.

## When to stash vs. commit

Stash when you need to switch context quickly — pull a fix, check out another
branch — but aren't ready to record a commit. When the work is ready to keep,
[stage and commit](/features/staging/) it instead.

## Next

- [Branches](/features/branches/) to switch context after stashing.
- [Staging & Committing](/features/staging/) when the work is ready to land.
