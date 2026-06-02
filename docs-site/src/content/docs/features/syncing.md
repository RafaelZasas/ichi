---
title: Push, Pull & Fetch
description: Sync with your remotes using ichi's command line.
---

ichi syncs with remotes through its command line. Press <kbd>:</kbd> from most
views and run one of:

| Command | Action |
|---|---|
| `:push` | Push the current branch to its remote |
| `:pull` | Pull the current branch from its remote |
| `:fetch` | Fetch from all configured remotes |

These work from the [commit graph](/features/graph/), the
[status](/features/staging/) view, and the [branches](/features/branches/) view —
wherever you happen to be when you want to sync.

## A typical loop

```text
:w        # stage everything
c         # commit
:push     # send it up
```

## After fetching

`:fetch` updates your remote-tracking refs without touching your working tree.
Once fetched, you can:

- Return to the [graph](/features/graph/) (<kbd>g</kbd>) to see what landed.
- Open [branches](/features/branches/) (<kbd>b</kbd>) and `:merge` or `:rebase`
  onto an updated branch.

## When things conflict

If a pull, merge, or rebase can't apply cleanly, ichi opens the
[conflict resolution](/features/conflicts/) view so you can sort it out in place.

## Next

- [Branches](/features/branches/) for merge and rebase.
- [GitHub Pull Requests](/features/pull-requests/) to review what you've pushed.
