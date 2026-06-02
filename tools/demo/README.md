# Demo & screenshot tooling

Automated, deterministic screenshots of every ichi view for the docs site. No
real terminal, no GitHub account, no manual capture.

## One command

```sh
tools/demo/shots.sh
```

This rebuilds the demo repos, renders every view to
`docs-site/public/images/views/*.png`, and adds a theme gallery. Output is
reproducible for a given theme (default: the hidden **atterpac** theme).

Options:

```sh
tools/demo/shots.sh -t nord -o /tmp/out   # different theme / output dir
```

## How it works

ichi shells out to `git` and `gh`, so the demo controls both:

| Piece | Purpose |
|---|---|
| `make-repo.sh` | Builds a deterministic git repo — pinned identity, commit dates, and contents — with a dense branch/merge history, stashes, and a dirty working tree. `--conflict` leaves it mid-merge. |
| `fake-gh` | A stand-in for the `gh` CLI that serves canned JSON from `gh-fixtures/`, so the pull-request views render offline. Put first on PATH as `gh`. |
| `screenshot/` | A Go program that builds each view on a tcell **SimulationScreen**, loads its data via `Start()`, drives selection with synthetic key presses, and rasterizes the frame to PNG with dado's `snapshots` package. Renders at 2× (nearest-neighbor) for crisp, larger images. |
| `shots.sh` | Orchestrates the above: clean repo → all views, conflict repo → conflict view, plus a theme gallery. |

## Run the app on the demo repo

To explore (not screenshot) the demo repo interactively:

```sh
tools/demo/demo.sh            # clean repo
tools/demo/demo.sh --conflict # mid-merge, for the conflict view
```

## Adding a view

Add a `sh.shoot("name", func() any { return views.NewXView(...) })` line in
`screenshot/main.go`. Pass an optional driver to position selection, e.g.
`selectHunk` to highlight a hunk. Then reference
`/images/views/name.png` from a docs page.

## Updating GitHub fixtures

Edit the JSON in `gh-fixtures/`. File names map to `gh` subcommands:
`pr-list.json`, `pr-<n>-view.json`, `pr-<n>-files.json`, `pr-<n>-checks.json`,
`pr-<n>-reviews.json`, `pr-<n>-comments.json`, `pr-<n>-diff.txt`, `user.json`.
