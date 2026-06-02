# ichi docs site

Marketing + how-to documentation for [ichi](https://github.com/atterpac/ichi),
built with [Astro Starlight](https://starlight.astro.build). Shares the visual
style of the [dado docs site](https://dado.atterpac.dev).

## Develop

```sh
npm install
npm run dev      # http://localhost:4321
```

## Build

```sh
npm run build    # output in ./dist
npm run preview  # preview the production build
```

## Structure

- `src/content/docs/guides/` — getting-started guides
- `src/content/docs/features/` — feature / how-to pages
- `src/content/docs/index.mdx` — splash landing page
- `src/styles/custom.css` — Starlight token overrides (IBM Plex, plum accent)
- `astro.config.mjs` — site config and sidebar

Production URL is `https://ichi.atterpac.dev` (set in `astro.config.mjs`).
