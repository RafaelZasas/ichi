---
title: GitHub Pull Requests
description: List, read, review, and merge GitHub pull requests without leaving ichi.
---

ichi brings your GitHub pull requests into the terminal. Press <kbd>p</kbd> from
the graph to open the **Pull Requests** view.

:::note
PR features use the [`gh` CLI](https://cli.github.com/) under the hood. Run
`gh auth login` once so ichi can read your repository's pull requests.
:::

![The pull request list](/images/views/pr-list.png)

## The PR list

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move between PRs |
| <kbd>Enter</kbd> | Open the PR |
| <kbd>c</kbd> | Check out the PR's branch locally |
| <kbd>o</kbd> | Open the PR in your browser |
| <kbd>f</kbd> | Filter by state (open / closed / all) |
| <kbd>r</kbd> | Refresh the list |
| <kbd>Esc</kbd> | Back |

## Reviewing a PR

Press <kbd>Enter</kbd> on a PR to open its detail view, which is split into tabs:

![A pull request detail view showing changed files](/images/views/pr-detail.png)

| Key | Tab |
|---|---|
| <kbd>1</kbd> | Files — the diff |
| <kbd>2</kbd> | Conversation — comments and reviews |
| <kbd>3</kbd> | Checks — CI status |

From the detail view you can review and act on the PR directly:

| Key | Action |
|---|---|
| <kbd>a</kbd> | Approve |
| <kbd>x</kbd> | Request changes |
| <kbd>c</kbd> | Comment |
| <kbd>m</kbd> | Merge |
| <kbd>o</kbd> | Open in browser |
| <kbd>Esc</kbd> | Back |

This means you can read the diff, leave a review, and merge — without opening a
browser tab.

## Next

- [Push, Pull & Fetch](/features/syncing/) to update before reviewing.
- [Branches](/features/branches/) after checking out a PR branch with <kbd>c</kbd>.
