// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import remarkGfm from 'remark-gfm';

export default defineConfig({
	// Canonical production URL. Starlight renders nav links / canonical tags
	// absolute against this; if unset it leaks the dev host
	// (http://localhost:8080), which trips browser "local network access"
	// prompts on the deployed site. Keep it hardcoded to the prod domain.
	site: 'https://ichi.atterpac.dev',
	markdown: {
		remarkPlugins: [remarkGfm],
	},
	integrations: [
		starlight({
			customCss: ['./src/styles/custom.css'],
			favicon: '/favicon.svg',
			title: 'ichi',
			description: 'A keyboard-driven Git client for the terminal',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/atterpac/ichi' }],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'guides/introduction' },
						{ label: 'Installation', slug: 'guides/installation' },
						{ label: 'Quick Start', slug: 'guides/quick-start' },
						{ label: 'Keybindings & Commands', slug: 'guides/keybindings' },
					],
				},
				{
					label: 'Features',
					items: [
						{ label: 'Commit Graph', slug: 'features/graph' },
						{ label: 'Staging & Committing', slug: 'features/staging' },
						{ label: 'Branches', slug: 'features/branches' },
						{ label: 'Push, Pull & Fetch', slug: 'features/syncing' },
						{ label: 'Stashes', slug: 'features/stash' },
						{ label: 'Diffs', slug: 'features/diff' },
						{ label: 'Blame', slug: 'features/blame' },
						{ label: 'File Log', slug: 'features/file-log' },
						{ label: 'Conflict Resolution', slug: 'features/conflicts' },
						{ label: 'GitHub Pull Requests', slug: 'features/pull-requests' },
						{ label: 'Command Palette', slug: 'features/command-palette' },
						{ label: 'Custom Commands', slug: 'features/custom-commands' },
						{ label: 'Themes', slug: 'features/themes' },
					],
				},
			],
		}),
	],
});
