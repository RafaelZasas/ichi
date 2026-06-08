---
title: Custom Commands
description: Define your own context-aware commands in config — shell templates, output views, argument passing, and global key bindings.
---

Beyond the built-in command line, ichi lets you define **your own** commands in
config. A custom command is a shell template that ichi expands with values from
the current selection (the commit, branch, file, or stash you're looking at),
runs, and renders however you choose.

Custom commands are invoked the same way as built-ins — type <kbd>:</kbd>
followed by the command name — and can optionally be bound to a global key.

## Where they live

Commands are defined in ichi's config file:

```
$XDG_CONFIG_HOME/ichi/config.yaml      # usually ~/.config/ichi/config.yaml
```

Add a `commands:` list:

```yaml
theme: tokyonight-night

commands:
  - name: showlog
    aliases: [sl]
    description: Full log for the selected commit
    command: git log -p -1 {{commitHash}}
    view: pager
    requires: commit
```

That defines `:showlog` (and `:sl`), which runs `git log -p -1 <hash>` for the
selected commit and shows the result in a scrollable pager. Invalid command
definitions are skipped at startup and reported as a toast — the rest still load.

:::note
A custom command whose `name` matches a built-in **overrides** the built-in.
:::

## Command fields

| Field | Required | Description |
|---|---|---|
| `name` | yes | The keyword typed after `:` (e.g. `showlog`). |
| `command` | yes | The shell template, run via `sh -c`. Supports `{{tokens}}`. |
| `aliases` | no | Alternative keywords for the same command. |
| `description` | no | Shown in the command palette and help. |
| `view` | no | How output is rendered (see [Views](#views)). Default `none`. |
| `requires` | no | Guard the command on a selection kind (see [Requires](#requires)). |
| `confirm` | no | When `true`, prompt with a confirmation modal before running. |
| `key` | no | Bind to a global key chord (see [Key bindings](#key-bindings)). |

## Template tokens

The `command` string is expanded with `{{token}}` placeholders drawn from the
current selection, the repository, and any arguments you pass. Data-derived
values are **shell-escaped automatically**, so they're safe to drop into the
command unquoted.

### Commit tokens

| Token | Expands to |
|---|---|
| `{{commitHash}}` | Full hash of the selected commit |
| `{{shortHash}}` | Abbreviated hash |
| `{{commitMessage}}` | Full commit message |
| `{{commitSubject}}` | First line of the message |
| `{{author}}` | Commit author |

### Branch tokens

| Token | Expands to |
|---|---|
| `{{branch}}` | Selected branch name |
| `{{branchUpstream}}` | Upstream of the selected branch |
| `{{currentBranch}}` | The currently checked-out branch |

### File tokens

| Token | Expands to |
|---|---|
| `{{file}}` / `{{filePath}}` | Path of the selected file |
| `{{fileName}}` | Base name of the selected file |
| `{{fileDir}}` | Directory of the selected file |

### Stash tokens

| Token | Expands to |
|---|---|
| `{{stash}}` | The selected stash ref, e.g. `stash@{0}` |
| `{{stashIndex}}` | The stash index number |
| `{{stashMessage}}` | The stash message |

### Repository & environment

| Token | Expands to |
|---|---|
| `{{ref}}` | Smart ref: commit, else branch, else stash |
| `{{repoRoot}}` | Absolute path to the repository root |
| `{{view}}` | Name of the current view |
| `{{editor}}` | `$EDITOR` (falls back to `vi`); inserted unquoted |
| `{{env:NAME}}` | The value of environment variable `NAME` |

### Arguments

Anything typed after the command name fills positional tokens:

```
:query foo bar
```

| Token | Expands to |
|---|---|
| `{{1}}`, `{{2}}`, … | The Nth argument (1-indexed) |
| `{{args}}` | All arguments joined with spaces |

Quote to group words: `:query "two words"` makes `{{1}}` → `two words`.

A referenced positional token is **required** by default and errors if absent.
Append `?` to make it optional, with an optional default value after it:

| Token | With an argument | Without one |
|---|---|---|
| `{{1}}` | the argument | **errors** |
| `{{1?}}` | the argument | empty string |
| `{{1?HEAD}}` | the argument | `HEAD` |

The `?default` suffix works on any token, not just positional ones — e.g.
`git show {{commitHash?HEAD}}` shows the selected commit, or `HEAD` when nothing
is selected.

## Views

The `view` field controls how the command's output is rendered:

| View | Behavior |
|---|---|
| `none` (default) | Run silently; toast the command's output (or a confirmation). |
| `pager` | Show stdout in a scrollable pager. |
| `diff` | Show stdout in the pager with diff syntax highlighting. |
| `commit` | Treat the first token of stdout as a commit ref and open its commit view. |
| `branch` | Run for side effects, then refresh and show the branch list. |
| `graph` | Treat the first token of stdout as a commit ref and open its commit view. |
| `editor` | Suspend ichi and run the command interactively (e.g. open `$EDITOR`); resume on exit. |

## Requires

Use `requires` to guard a command so it only runs when the right thing is
selected. If nothing matching is selected, ichi reports the error instead of
running a malformed command.

| Value | Runs only when… |
|---|---|
| `commit` | a commit is selected |
| `branch` | a branch is selected |
| `file` | a file is selected |
| `stash` | a stash is selected |

## Confirm

Set `confirm: true` to require a confirmation modal before the command runs —
useful for anything destructive or with side effects. The modal shows the fully
expanded command so you can see exactly what will execute.

```yaml
  - name: fixup
    description: Create a fixup commit for the selected commit
    command: git commit --fixup {{commitHash}}
    requires: commit
    confirm: true
```

## Key bindings

Set `key` to bind a command to a **global** key chord, so it fires from any view
without opening the command line.

```yaml
  - name: countcommits
    description: Count commits on the current branch
    command: git rev-list --count HEAD
    key: ctrl+n
```

Accepted forms:

| Form | Examples |
|---|---|
| Single rune | `x`, `1` |
| Function key | `f5`, `f12` |
| Named key | `enter`, `tab`, `esc`, `space`, `up`, `home`, … |
| Modifier combo | `ctrl+g`, `alt+x`, `shift+a` (join with `+`) |

Key-bound commands fire from any view — **except** while a modal or the command
line is open — and take precedence over built-in keys.

:::caution
Global bindings run before a view's own keys, so binding a bare letter like `c`
would shadow per-view shortcuts (Checkout, Commit, …). Prefer `ctrl+*` chords or
function keys for global commands to avoid clobbering view keys.
:::

## Worked examples

```yaml
commands:
  # Open the selected file in your editor.
  - name: edit
    description: Edit the selected file in $EDITOR
    command: $EDITOR {{filePath}}
    view: editor
    requires: file

  # Diff the selected commit against its parent, with highlighting.
  - name: cdiff
    description: Diff of the selected commit against its parent
    command: git show {{commitHash}}
    view: diff
    requires: commit

  # Search the log for a pattern; show everything when none is given.
  - name: query
    description: Search the log (optional pattern)
    command: git log --oneline {{1?}}
    view: pager

  # Push the current branch, bound to a key, with a confirmation.
  - name: pushup
    description: Push the current branch
    command: git push
    confirm: true
    key: ctrl+u
```
