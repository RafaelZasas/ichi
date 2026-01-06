package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/nav"
	"github.com/atterpac/jig/theme"
	"github.com/atterpac/jig/theme/themes"

	"github.com/atterpac/etch/internal/app"
	"github.com/atterpac/etch/internal/commands"
	"github.com/atterpac/etch/internal/git"
	"github.com/atterpac/etch/internal/views"
)

const sponsorURL = "github.com/sponsors/atterpac"

// ASCII art logo for etch
const etchLogo = `
__/\\\\\\\\\\\\\\\______________________________/\\\_________        
 _\/\\\///////////______________________________\/\\\_________       
  _\/\\\_________________/\\\____________________\/\\\_________      
   _\/\\\\\\\\\\\______/\\\\\\\\\\\_____/\\\\\\\\_\/\\\_________     
    _\/\\\///////______\////\\\////____/\\\//////__\/\\\\\\\\\\__    
     _\/\\\________________\/\\\_______/\\\_________\/\\\/////\\\_   
      _\/\\\________________\/\\\_/\\__\//\\\________\/\\\___\/\\\_  
       _\/\\\\\\\\\\\\\\\____\//\\\\\____\///\\\\\\\\_\/\\\___\/\\\_ 
        _\///////////////______\/////_______\////////__\///____\///__
`

var (
	repoPath  = flag.String("path", ".", "Path to git repository")
	noSplash  = flag.Bool("no-splash", false, "Skip splash screen")
)

func main() {
	flag.Parse()

	// 1. Initialize theme FIRST (Required by jig)
	theme.SetProvider(themes.TokyoNightNight)

	// 2. Initialize git repository
	repo, err := git.OpenRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Show splash screen (unless skipped)
	if !*noSplash {
		if err := showSplash(); err != nil {
			// Splash was cancelled or failed, just continue
		}
	}

	// 4. Create layout components
	statusBar := layout.NewStatusBar()

	// Update status bar with repo info (includes branch in title)
	app.UpdateStatusBar(statusBar, repo)

	menu := layout.NewMenu().
		SetRightText("[" + theme.TagFgDim() + "]♥ " + sponsorURL + "[-]")

	// Track focused primitive before entering command mode
	var previousFocus tview.Primitive

	// 5. Create app with 4-tier layout (Required by jig)
	application := layout.NewApp(layout.AppConfig{
		TopBar:     statusBar,
		BottomBar:  menu,
		ShowCrumbs: true,
		OnComponentChange: func(c nav.Component) {
			if c != nil {
				menu.SetHints(c.Hints())
			}
		},
	})

	// 6. Set up command mode callbacks
	cmdCtx := &commands.Context{
		App:       application,
		Repo:      repo,
		StatusBar: statusBar,
	}

	statusBar.SetOnCommandSubmit(func(text string) {
		statusBar.ExitCommandMode()
		depthBefore := application.Pages().StackDepth()
		if err := commands.Execute(cmdCtx, text); err != nil {
			views.ShowErrorModal(application, "Command Error", err.Error())
		}
		app.UpdateStatusBar(statusBar, repo)
		// If a new view was pushed, focus it directly
		// Otherwise restore previous focus for commands that don't push views
		if application.Pages().StackDepth() > depthBefore {
			if current := application.Pages().Current(); current != nil {
				application.SetFocus(current)
			}
		} else if previousFocus != nil {
			application.SetFocus(previousFocus)
		}
	})

	statusBar.SetOnCommandCancel(func() {
		statusBar.ExitCommandMode()
		// Restore focus to previous primitive
		if previousFocus != nil {
			application.SetFocus(previousFocus)
		}
	})

	statusBar.SetOnComplete(func(input string) []string {
		return commands.GetCompletions(repo, input)
	})

	statusBar.SetOnHistoryPrev(commands.HistoryPrev)
	statusBar.SetOnHistoryNext(commands.HistoryNext)

	// 7. Set up global keys
	application.SetInputCapture(globalInputHandler(application, repo, statusBar, &previousFocus))

	// 8. Push initial view (Graph is the home view)
	graphView := views.NewGraphView(application, repo)
	application.Pages().Push(graphView)
	application.Crumbs().SetPath([]string{"Graph"})

	// 9. Run
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showSplash() error {
	splash := components.NewSplash().
		SetLogo(etchLogo).
		SetStatus("Press any key to continue\n\n[" + theme.TagFgDim() + "]♥ " + sponsorURL + "[-]").
		SetGradient(theme.GradientDiagonal).
		SetAutoDismiss(3 * time.Second).
		SetDismissKeys([]components.DismissKey{components.DismissAnyKey})

	splash.Build()

	splashApp := tview.NewApplication()
	theme.SetApp(splashApp)

	splash.SetOnClose(func() {
		splashApp.Stop()
	})

	splashApp.SetRoot(splash, true)
	return splashApp.Run()
}

func globalInputHandler(app *layout.App, repo *git.Repository, statusBar *layout.StatusBar, previousFocus *tview.Primitive) func(*tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		// Don't handle keys when in command mode
		if statusBar.IsCommandMode() {
			return event
		}

		switch {
		// Enter command mode with ':'
		case event.Rune() == ':':
			*previousFocus = app.GetApplication().GetFocus()
			commands.ResetHistoryIndex()
			statusBar.EnterCommandMode()
			app.SetFocus(statusBar.GetCommandInput())
			return nil

		// Quit on root view only (Required by jig)
		case event.Rune() == 'q' && app.Pages().StackDepth() <= 1:
			app.Stop()
			return nil

		// Go back with Esc (Required by jig)
		case event.Key() == tcell.KeyEscape:
			if app.Pages().CanPop() {
				app.Pages().Pop()
				return nil
			}

		// Help modal (Required by jig)
		case event.Rune() == '?':
			showHelp(app)
			return nil

		// Theme selector (Required by jig)
		case event.Rune() == 'T':
			showThemeSelector(app)
			return nil

		// Git-specific global keys
		case event.Rune() == 'b':
			branchView := views.NewBranchesView(app, repo)
			app.Pages().Push(branchView)
			app.Crumbs().SetPath([]string{"Branches"})
			return nil

		case event.Rune() == 's':
			statusView := views.NewStatusView(app, repo)
			app.Pages().Push(statusView)
			app.Crumbs().SetPath([]string{"Status"})
			return nil

		case event.Rune() == 'S':
			stashView := views.NewStashView(app, repo)
			app.Pages().Push(stashView)
			app.Crumbs().SetPath([]string{"Stash"})
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
	selectorView := views.NewThemeSelectorView(app)
	app.Pages().Push(selectorView)
}
