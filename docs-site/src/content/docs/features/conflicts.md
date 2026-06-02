---
title: Conflict Resolution
description: Resolve merge and rebase conflicts in ichi's dedicated three-way view.
---

When a merge, rebase, or pull can't apply cleanly, ichi opens the **conflict
resolution** view instead of leaving you to untangle conflict markers in an
editor.

![The three-way conflict resolution view: ours, base, and theirs with a resolved panel](/images/views/conflicts.png)

## How you get here

Any operation that can conflict drops you into this view automatically:

- `:merge` a branch with divergent changes
- `:rebase` onto a branch that touched the same lines
- `:pull` when local and remote both changed a file

## Resolving

The view walks you through the conflicting files and hunks, showing the
competing sides so you can choose what to keep. Navigate with the usual keys:

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between conflicts |
| <kbd>Tab</kbd> | Switch focus / side |
| <kbd>Enter</kbd> | Choose / confirm |
| <kbd>Esc</kbd> | Back |

Once a conflict is resolved, the file is staged so you can continue the merge or
rebase. Press <kbd>?</kbd> in the view for the exact keys available.

## After resolving

When every conflict is settled, finish the operation and review the result:

- Return to the [graph](/features/graph/) (<kbd>g</kbd>) to confirm history looks
  right.
- [Commit](/features/staging/) the merge if one is pending.

## Next

- [Branches](/features/branches/) for merge and rebase.
- [Push, Pull & Fetch](/features/syncing/) to sync once you're clean.
