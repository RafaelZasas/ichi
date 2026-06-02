---
title: Installation
description: Install ichi and open your first repository.
---

ichi is a single Go binary. You'll need [Go 1.25+](https://go.dev/dl/) and `git`
available on your `PATH`.

## Install with `go install`

```sh
go install github.com/atterpac/ichi/cmd/ichi@latest
```

This drops the `ichi` binary into `$(go env GOPATH)/bin`. Make sure that
directory is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Build from source

```sh
git clone https://github.com/atterpac/ichi
cd ichi
go build -o ichi ./cmd/ichi
```

## Run it

From inside any Git repository:

```sh
ichi
```

Or point it at a repository elsewhere:

```sh
ichi -path /path/to/repo
```

To skip the splash screen on launch:

```sh
ichi -no-splash
```

| Flag | Default | Description |
|---|---|---|
| `-path` | `.` | Path to the Git repository to open |
| `-no-splash` | `false` | Skip the splash screen |

## GitHub features

The [pull request](/features/pull-requests/) views talk to GitHub. Authenticate
with the [`gh` CLI](https://cli.github.com/) (`gh auth login`) so ichi can read
your repository's PRs.

## Next steps

Head to the [Quick Start](/guides/quick-start/) to make your first commit.
