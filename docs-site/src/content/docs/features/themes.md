---
title: Themes
description: Switch ichi's color scheme at runtime and have it remembered.
---

ichi ships with 26 built-in color schemes from [dado](https://dado.atterpac.dev),
the TUI library it's built on — Nord, Dracula, Catppuccin, Tokyo Night, Gruvbox,
Rosé Pine, and many more.

## Switching themes

Press <kbd>T</kbd> from anywhere to open the **theme selector**. Move through the
list to preview each scheme, and select one to apply it instantly across every
view.

## It's remembered

Your choice is saved to ichi's config, so the next time you launch it the app
opens in the theme you picked. If no theme is saved yet, ichi starts on
**Tokyo Night (Night)**.

## The full set

All 26 built-in schemes, rendered on the commit graph.

### Tokyo Night

**Tokyo Night (Night)** — `tokyonight-night`
![Tokyo Night (Night)](/images/views/tokyonight-night.png)

**Tokyo Night (Storm)** — `tokyonight-storm`
![Tokyo Night (Storm)](/images/views/tokyonight-storm.png)

**Tokyo Night (Moon)** — `tokyonight-moon`
![Tokyo Night (Moon)](/images/views/tokyonight-moon.png)

**Tokyo Night (Day)** — `tokyonight-day`
![Tokyo Night (Day)](/images/views/tokyonight-day.png)

### Catppuccin

**Catppuccin Mocha** — `catppuccin-mocha`
![Catppuccin Mocha](/images/views/catppuccin-mocha.png)

**Catppuccin Macchiato** — `catppuccin-macchiato`
![Catppuccin Macchiato](/images/views/catppuccin-macchiato.png)

**Catppuccin Frappé** — `catppuccin-frappe`
![Catppuccin Frappé](/images/views/catppuccin-frappe.png)

**Catppuccin Latte** — `catppuccin-latte`
![Catppuccin Latte](/images/views/catppuccin-latte.png)

### Rosé Pine

**Rosé Pine** — `rosepine`
![Rosé Pine](/images/views/rosepine.png)

**Rosé Pine Moon** — `rosepine-moon`
![Rosé Pine Moon](/images/views/rosepine-moon.png)

**Rosé Pine Dawn** — `rosepine-dawn`
![Rosé Pine Dawn](/images/views/rosepine-dawn.png)

### Gruvbox

**Gruvbox Dark** — `gruvbox-dark`
![Gruvbox Dark](/images/views/gruvbox-dark.png)

**Gruvbox Light** — `gruvbox-light`
![Gruvbox Light](/images/views/gruvbox-light.png)

### Everforest

**Everforest Dark** — `everforest-dark`
![Everforest Dark](/images/views/everforest-dark.png)

**Everforest Light** — `everforest-light`
![Everforest Light](/images/views/everforest-light.png)

### Dracula

**Dracula** — `dracula`
![Dracula](/images/views/dracula.png)

**Dracula Light** — `dracula-light`
![Dracula Light](/images/views/dracula-light.png)

### GitHub

**GitHub Dark** — `github-dark`
![GitHub Dark](/images/views/github-dark.png)

**GitHub Light** — `github-light`
![GitHub Light](/images/views/github-light.png)

### Solarized

**Solarized Dark** — `solarized-dark`
![Solarized Dark](/images/views/solarized-dark.png)

**Solarized Light** — `solarized-light`
![Solarized Light](/images/views/solarized-light.png)

### One

**One Dark** — `onedark`
![One Dark](/images/views/onedark.png)

**One Light** — `onelight`
![One Light](/images/views/onelight.png)

### And a few standalones

**Nord** — `nord`
![Nord](/images/views/nord.png)

**Kanagawa** — `kanagawa`
![Kanagawa](/images/views/kanagawa.png)

**Monokai** — `monokai`
![Monokai](/images/views/monokai.png)

## Why it looks consistent

Because ichi renders through [dado](https://dado.atterpac.dev)'s theming system, every component — the commit
graph, diffs, modals, the command palette — honors the active theme together.
Switching is a single keypress, not a per-view setting.

## Next

- Back to the [Commit Graph](/features/graph/) to see a theme applied.
- [Command Palette](/features/command-palette/) for the other keyboard-driven
  niceties.
