---
title: Command Palette
description: Fuzzy-find branches, commands, and views — or type a Git command directly.
---

ichi gives you two ways to drive it by typing: a fuzzy-finding **command
palette** and a Vim-style **command line**.

## The command palette

Press <kbd>Ctrl</kbd>+<kbd>P</kbd> (or <kbd>Ctrl</kbd>+<kbd>K</kbd>) from
anywhere to open the palette. Start typing and it fuzzy-matches across:

![The ichi command palette](/images/views/command-palette.png)

- **Recent** — what you used last
- **Branches** and **Remote Branches** — jump to or check out a branch
- **Commands** — any Git action ichi knows
- **Navigation** — switch to a view

Move through results with <kbd>j</kbd>/<kbd>k</kbd> (or the arrows) and press
<kbd>Enter</kbd> to run the highlighted item. <kbd>Esc</kbd> closes the palette.

The palette is the fastest way to act when you know *what* you want but not which
key or view it lives under — just type a few letters.

## The command line

Prefer typing the verb? Press <kbd>:</kbd> to open a command line and run a Git
command directly — like <code>:rebase main</code>.

The command line acts on whatever is selected in the current view — a commit, a
branch, a stash. See the full command list in
[Keybindings & Commands](/guides/keybindings/).

## Palette vs. command line

- Use the **palette** to *search* — branches, views, and actions, fuzzily.
- Use the **command line** when you already know the *verb* you want to type.

Both reach the same set of Git actions; pick whichever is faster in the moment.

## Going further

You can extend the command line with your own context-aware commands — shell
templates that expand with the selected commit, branch, or file, render into any
view, and optionally bind to a global key. See
[Custom Commands](/features/custom-commands/).
