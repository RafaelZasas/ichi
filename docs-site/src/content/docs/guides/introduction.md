---
title: Introduction
description: What ichi is, who it's for, and how it fits into your Git workflow.
---

**ichi** is a keyboard-driven Git client that runs in your terminal. It puts the
parts of Git you reach for every day — the commit graph, staging, committing,
branching, syncing, stashing, conflict resolution, and pull requests — behind a
fast, consistent, vim-flavored interface.

Instead of memorizing porcelain flags or tabbing out to a desktop app, you stay
in one screen: move with <kbd>j</kbd>/<kbd>k</kbd>, act with a single key, and
drop into a command line with <kbd>:</kbd> when you want the full Git verb.

## What you can do

- **Read history** in a live commit graph and inspect any commit's diff.
- **Stage and commit** interactively — by file, by hunk, or line by line.
- **Manage branches**: create, check out, merge, and rebase.
- **Sync** with remotes: push, pull, and fetch.
- **Stash** work in progress and apply, pop, or drop entries.
- **Resolve conflicts** in a dedicated three-way view.
- **Work with GitHub pull requests** — list, read, and review them in place.
- **Run any Git action** from a command palette or `:` command line.

## Who it's for

ichi is for people who live in the terminal and want their Git workflow to feel
the same way: fast, modal, and keyboard-first. If you like the ergonomics of vim
or a TUI file manager, ichi will feel familiar immediately.

## Built on dado

ichi is built in Go on top of [dado](https://dado.atterpac.dev), a terminal UI
component library. That's where ichi gets its themeable look, smooth navigation,
and 26 built-in color schemes — see [Themes](/features/themes/).

## Next steps

- [Install ichi](/guides/installation/) and open a repository.
- Follow the [Quick Start](/guides/quick-start/) to make your first commit.
- Skim the [Keybindings & Commands](/guides/keybindings/) reference.
