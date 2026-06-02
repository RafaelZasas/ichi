---
title: Diffs
description: Inspect changes with ichi's diff view.
---

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

## Next

- [Blame](/features/blame/) to trace line-by-line authorship.
- [File Log](/features/file-log/) for a single file's history.
