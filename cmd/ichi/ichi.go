package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/nav"
	"github.com/atterpac/dado/theme"
	"github.com/atterpac/dado/theme/themes"

	"github.com/atterpac/ichi/internal/app"
	appkg "github.com/atterpac/ichi/internal/app"
	"github.com/atterpac/ichi/internal/commands"
	"github.com/atterpac/ichi/internal/config"
	"github.com/atterpac/ichi/internal/git"
	"github.com/atterpac/ichi/internal/views"
)

const footerURL = "atterpac.dev"

// ASCII art logo for gxt
const ichiLogo = `
░▒▓█▓▒░░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
░▒▓█▓▒░▒▓█▓▒░      ░▒▓████████▓▒░▒▓█▓▒░ 
░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░ 
`

var (
	repoPath = flag.String("path", ".", "Path to git repository")
	noSplash = flag.Bool("no-splash", false, "Skip splash screen")
	showVer  = flag.Bool("version", false, "Print version and exit")
)

// Build info, injected via -ldflags by goreleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	flag.Parse()

	if *showVer {
		fmt.Printf("ichi %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}

	if extraArgs := flag.Args(); len(extraArgs) > 0 {
		path := extraArgs[0]
		if info, err := os.Stat(path); err != nil && !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: Invalid path '%s'\n", path)
			os.Exit(1)
		}
		*repoPath = path
	}

	// 1. Initialize theme FIRST (Required by dado)
	// Load saved theme from config, fallback to TokyoNightNight
	savedTheme := config.GetTheme()
	if t := themes.Get(savedTheme); t != nil {
		theme.SetProvider(t)
	} else {
		theme.SetProvider(themes.TokyoNightNight)
	}

	// 2. Initialize git repository
	repo, err := git.OpenRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Start loading data in background while splash shows
	type graphResult struct {
		graph *views.PreloadedGraph
		err   error
	}
	graphCh := make(chan graphResult, 1)
	dataReady := make(chan struct{})
	go func() {
		g, err := views.PreloadGraph(repo)
		graphCh <- graphResult{g, err}
		close(dataReady)
	}()

	// Show splash screen (unless skipped) — dismisses when data is ready
	if !*noSplash {
		if err := showSplash(dataReady); err != nil {
			// Splash was cancelled or failed, just continue
		}
	}

	// 4. Create layout components
	statusBar := layout.NewStatusBar()

	// Update status bar with repo info (includes branch in title)
	app.UpdateStatusBar(statusBar, repo)

	menu := layout.NewMenu().
		SetRightText("♥ " + footerURL)

	// Track focused widget before entering command mode
	var previousFocus core.Widget

	// 5. Create app with 4-tier layout (Required by dado)
	application := layout.NewApp(layout.AppConfig{
		TopBar:     statusBar,
		BottomBar:  menu,
		ShowCrumbs: true,
		Debug:      true,
		OnComponentChange: func(c nav.Component) {
			if c != nil {
				menu.SetHints(c.Hints())
			}
		},
	})

	// Wire dado's built-in theme selector (live preview + cancel restore).
	// Persisting the choice happens through OnChange.
	//
	// themes.All() excludes hidden themes (e.g. "atterpac"), so EnableThemes
	// would reject a hidden saved theme as its Default and fall back to the
	// built-in default. Inject the saved theme into the map so it survives as
	// the active theme; keep Names as the visible (non-hidden) list so hidden
	// themes stay out of the selector.
	themeSet := themes.All()
	if themeSet[savedTheme] == nil {
		if t := themes.Get(savedTheme); t != nil {
			themeSet[savedTheme] = t
		}
	}
	application.EnableThemes(layout.ThemeOptions{
		Themes:   themeSet,
		Names:    themes.Names(),
		Default:  savedTheme,
		OnChange: config.SetTheme,
	})

	// Initialize toast notifications and draw them on top of every frame via the
	// after-draw hook. Without this the toast manager exists but is never
	// rendered.
	app.InitToasts()
	if toasts := app.GetToastManager(); toasts != nil {
		if coreApp := application.GetApp(); coreApp != nil {
			coreApp.SetAfterDrawFunc(func(screen tcell.Screen) {
				w, h := screen.Size()
				toasts.Draw(screen, w, h)
			})
		}
		// tcell only redraws on input events; tick while toasts are active so
		// they animate and auto-dismiss on time.
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				if toasts.HasActive() {
					application.QueueUpdateDraw(func() {})
				}
			}
		}()
	}

	// 6. Set up command mode callbacks
	cmdCtx := &commands.Context{
		App:       application,
		Repo:      repo,
		StatusBar: statusBar,
	}

	// Register user-defined context-aware commands from config.
	for _, w := range commands.RegisterCustom(config.GetCommands()) {
		app.ToastError("config: " + w)
	}

	statusBar.SetOnCommandSubmit(func(text string) {
		statusBar.ExitCommandMode()
		depthBefore := application.Pages().StackDepth()
		if err := commands.Execute(cmdCtx, text); err != nil {
			app.ToastError(err.Error())
		}
		app.UpdateStatusBar(statusBar, repo)
		// A command-opened modal focuses itself; don't steal focus back.
		if application.Pages().CurrentIsModal() {
			return
		}
		if application.Pages().StackDepth() > depthBefore {
			if current := application.Pages().Current(); current != nil {
				if w, ok := current.(core.Widget); ok {
					application.SetFocus(w)
				} else if previousFocus != nil {
					// View isn't a core.Widget (most views route input through
					// the Pages container). Restore focus there so keys reach
					// the new view's HandleKey instead of staying on the
					// command bar.
					application.SetFocus(previousFocus)
				}
			}
		} else {
			if current := application.Pages().Current(); current != nil {
				current.Start()
			}
			if previousFocus != nil {
				application.SetFocus(previousFocus)
			}
		}
	})

	statusBar.SetOnCommandCancel(func() {
		statusBar.ExitCommandMode()
		// Restore focus to previous widget
		if previousFocus != nil {
			application.SetFocus(previousFocus)
		}
	})

	statusBar.SetOnComplete(func(input string) []string {
		return commands.GetCompletions(repo, input)
	})

	// Live auto-complete: as the user types, show the inline ghost-text
	// suggestion (e.g. ":log cmd/g" suggests ":log cmd/ichi/gxt.go").
	statusBar.SetOnCommandChange(func(input string) {
		statusBar.SetSuggestion(commands.GetSuggestion(repo, input))
	})

	statusBar.SetOnHistoryPrev(commands.HistoryPrev)
	statusBar.SetOnHistoryNext(commands.HistoryNext)

	// 7. Set up global keys
	application.SetInputCapture(globalInputHandler(application, repo, statusBar, cmdCtx, &previousFocus))

	// 8. Push initial view (Graph is the home view)
	// Wait for preloaded graph data
	preloaded := <-graphCh
	graphView := views.NewGraphView(application, repo)
	if preloaded.err == nil && preloaded.graph != nil {
		graphView.SetPreloadedGraph(preloaded.graph)
	}
	application.Pages().Push(graphView)
	application.Crumbs().SetPath([]string{"Graph"})

	// 9. Run
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showSplash(ready <-chan struct{}) error {
	splash := components.NewSplash().
		SetLogo(ichiLogo).
		SetLogoWidth(40).
		SetLogoHeight(9).
		SetStatusHeight(1).
		SetStatus("[" + theme.TagFgDim() + "]Made with ♥ by " + footerURL + "[-]").
		SetGradient(theme.GradientDiagonal).
		SetAutoDismiss(10 * time.Second). // Fallback max; normally dismissed earlier
		SetDismissKeys([]components.DismissKey{components.DismissAnyKey})

	splash.Build()

	splashApp := core.NewApp()

	splash.SetOnClose(func() {
		splashApp.Stop()
	})

	// Dismiss splash once data is ready (with a brief minimum display)
	go func() {
		minDisplay := time.After(500 * time.Millisecond)
		<-ready      // Wait for graph data
		<-minDisplay // Ensure splash shows for at least 500ms
		splashApp.QueueUpdateDraw(func() {
			splashApp.Stop()
		})
	}()

	splashApp.SetRoot(splash)
	return splashApp.Run()
}

func globalInputHandler(app *layout.App, repo *git.Repository, statusBar *layout.StatusBar, cmdCtx *commands.Context, previousFocus *core.Widget) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		// Don't handle keys when in command mode
		if statusBar.IsCommandMode() {
			return event
		}

		// Don't handle most keys when a modal is active (let modal handle input)
		// Exception: Escape key should still work to dismiss modals
		if app.Pages().CurrentIsModal() && event.Key() != tcell.KeyEscape {
			return event
		}

		// User-defined key-bound custom commands take precedence over builtins.
		if cmd := commands.MatchKey(event); cmd != nil {
			if err := cmd.Handler(cmdCtx, nil); err != nil {
				appkg.ToastError(err.Error())
			}
			return nil
		}

		switch {
		// Enter command mode with ':'
		case event.Rune() == ':':
			*previousFocus = app.GetApp().GetFocus()
			commands.ResetHistoryIndex()
			statusBar.EnterCommandMode()
			app.SetFocus(statusBar)
			return nil

		// Quit on root view only (Required by dado)
		case event.Rune() == 'q' && app.Pages().StackDepth() <= 1:
			app.Stop()
			return nil

		// Go back with Esc (Required by dado)
		// Optionally go back with vim motion Ctrl-O
		case event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyCtrlO:
			if app.Pages().CanPop() {
				app.Pages().Pop()
				return nil
			}

		// Help modal (Required by dado)
		case event.Rune() == '?':
			showHelp(app)
			return nil

		// Theme selector (Required by dado)
		case event.Rune() == 'T':
			showThemeSelector(app)
			return nil

		// Command palette (Ctrl+P or Ctrl+K)
		case event.Key() == tcell.KeyCtrlP || event.Key() == tcell.KeyCtrlK:
			views.ShowFinder(app, repo, statusBar)
			return nil

		// Repo switcher (Ctrl+R)
		case event.Key() == tcell.KeyCtrlR:
			views.ShowRepoSwitcher(app, repo, statusBar)
			return nil

		// Git-specific global keys (only from root/graph view to avoid conflicts)
		case event.Rune() == 'b' && app.Pages().StackDepth() <= 1:
			branchView := views.NewBranchesView(app, repo)
			app.Pages().Push(branchView)
			app.Crumbs().SetPath([]string{"Branches"})
			return nil

		case event.Rune() == 's' && app.Pages().StackDepth() <= 1:
			statusView := views.NewStatusView(app, repo)
			app.Pages().Push(statusView)
			app.Crumbs().SetPath([]string{"Status"})
			return nil

		case event.Rune() == 'S' && app.Pages().StackDepth() <= 1:
			stashView := views.NewStashView(app, repo)
			app.Pages().Push(stashView)
			app.Crumbs().SetPath([]string{"Stash"})
			return nil

		case event.Rune() == 'p' && app.Pages().StackDepth() <= 1:
			prView := views.NewPRListView(app, repo)
			app.Pages().Push(prView)
			app.Crumbs().SetPath([]string{"PRs"})
			return nil

		case event.Rune() == 'g':
			// Return to graph view (home)
			if app.Pages().StackDepth() > 1 {
				// Pop until we're at root
				for app.Pages().CanPop() {
					app.Pages().Pop()
				}
				app.Crumbs().SetPath([]string{"Graph"})
			}
			return nil
		}
		return event
	}
}

func showHelp(app *layout.App) {
	helpView := views.NewHelpView(app)
	app.Pages().Push(helpView)
}

func showThemeSelector(app *layout.App) {
	app.OpenThemeSelector()
}
