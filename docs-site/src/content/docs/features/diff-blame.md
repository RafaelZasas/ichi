---
title: Diffs & Blame
description: Inspect changes and trace authorship with ichi's diff, blame, and file-log views.
---

ichi gives you several lenses on a file's history: the diff view for changes,
blame for line-by-line attribution, and the file log for a file's commit history.

## Diffs

A diff is never more than a keypress away:

- In [status](/features/staging/), press <kbd>d</kbd> to diff the selected file.
- In the [commit graph](/features/graph/), press <kbd>d</kbd> (or run `:diff`) to
  see a commit's full diff.

![A commit diff rendered in ichi](/images/views/diff.png)

Inside the diff view:

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Move by line |
| <kbd>J</kbd> / <kbd>K</kbd> | Move by hunk |
| <kbd>Space</kbd> | Select a line |
| <kbd>s</kbd> | Stage the current hunk |
| <kbd>Enter</kbd> | Stage selected lines |
| <kbd>u</kbd> | Unstage |
| <kbd>Esc</kbd> | Back |

The same view that shows changes also lets you stage them — see
[Staging & Committing](/features/staging/).

## Blame

Run `:blame` to open a line-by-line attribution view for a file: who last
touched each line and in which commit.

![The blame view attributing each line to a commit and author](/images/views/blame.png)

| Key | Action |
|---|---|
| <kbd>j</kbd> / <kbd>k</kbd> | Scroll |
| <kbd>Esc</kbd> | Close |

## File log

Run `:log` to see the commits that touched a specific file — a focused history
when you don't want the whole graph.

![The file log listing commits that touched a single file](/images/views/file-log.png)

## Next

- [Commit Graph](/features/graph/) for the full history.
- [Staging & Committing](/features/staging/) to act on what a diff shows you.
