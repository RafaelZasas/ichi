---
title: Quick Start
description: Stage, commit, and push your first change with ichi in under a minute.
---

This walks through the everyday loop — see a change, stage it, commit it, push —
so you learn the core keys by using them.

## 1. Open ichi

From inside a repository:

```sh
ichi
```

ichi opens on the **commit graph**: your history, newest at the top. Move with
<kbd>j</kbd>/<kbd>k</kbd>, jump to the top/bottom with <kbd>g</kbd>/<kbd>G</kbd>.

## 2. Jump to status

Press <kbd>s</kbd> to open the **status** view. This lists your changed files,
split between unstaged and staged.

- <kbd>j</kbd>/<kbd>k</kbd> — move between files
- <kbd>Space</kbd> — stage / unstage the selected file
- <kbd>a</kbd> — stage all, <kbd>u</kbd> — unstage all
- <kbd>d</kbd> — view the diff for the selected file

For finer control, the diff view lets you stage individual **hunks**
(<kbd>s</kbd>) or even selected **lines** (<kbd>Enter</kbd>). See
[Staging & Committing](/features/staging/).

## 3. Commit

With your changes staged, press <kbd>c</kbd> to open the commit dialog. Type your
message, then <kbd>Enter</kbd> to confirm (<kbd>Esc</kbd> cancels).

## 4. Push

Press <kbd>:</kbd> to open the command line and run:

```text
:push
```

ichi pushes the current branch to its remote. (<kbd>:pull</kbd> and
<kbd>:fetch</kbd> work the same way.) See [Push, Pull & Fetch](/features/syncing/).

## 5. Get around

A few keys you'll use constantly:

| Key | Action |
|---|---|
| <kbd>g</kbd> | Back to the commit graph (home) |
| <kbd>s</kbd> | Status |
| <kbd>b</kbd> | Branches |
| <kbd>S</kbd> | Stashes |
| <kbd>p</kbd> | Pull requests |
| <kbd>:</kbd> | Command line |
| <kbd>Ctrl</kbd>+<kbd>P</kbd> | Command palette |
| <kbd>T</kbd> | Theme selector |
| <kbd>?</kbd> | Help |
| <kbd>Esc</kbd> | Go back |
| <kbd>q</kbd> | Quit |

That's the whole loop. Next, see the full
[Keybindings & Commands](/guides/keybindings/) reference, or dig into a feature.
