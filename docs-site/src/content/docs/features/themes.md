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

The same commit graph under a few of the built-in schemes:

![ichi in the Nord theme](/images/views/theme-nord.png)
![ichi in the Dracula theme](/images/views/theme-dracula.png)
![ichi in the Catppuccin Mocha theme](/images/views/theme-catppuccin-mocha.png)
![ichi in the Gruvbox Dark theme](/images/views/theme-gruvbox-dark.png)
![ichi in the Rosé Pine theme](/images/views/theme-rosepine.png)

## It's remembered

Your choice is saved to ichi's config, so the next time you launch it the app
opens in the theme you picked. If no theme is saved yet, ichi starts on
**Tokyo Night (Night)**.

## Why it looks consistent

Because ichi renders through dado's theming system, every component — the commit
graph, diffs, modals, the command palette — honors the active theme together.
Switching is a single keypress, not a per-view setting.

## Next

- Back to the [Commit Graph](/features/graph/) to see a theme applied.
- [Command Palette](/features/command-palette/) for the other keyboard-driven
  niceties.
