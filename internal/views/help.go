package views

import (
	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/input"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"
)

// HelpView displays keyboard shortcuts and help information.
type HelpView struct {
	flex    *core.Flex
	content *core.TextView
	app     *layout.App
}

// NewHelpView creates a new help view.
func NewHelpView(app *layout.App) *HelpView {
	v := &HelpView{
		flex:    core.NewFlex(),
		content: core.NewTextView(),
		app:     app,
	}
	v.setup()
	return v
}

func (v *HelpView) setup() {
	v.content.SetDynamicColors(true)
	v.content.SetWordWrap(true)
	v.content.SetBackgroundColor(theme.Bg())

	helpText := `[` + theme.TagAccent() + `::b]ichi - Terminal Git Client[-:-:-]

[` + theme.TagAccent() + `]Global Keys[-]
  [` + theme.TagWarning() + `]q[-]       Quit (from root view)
  [` + theme.TagWarning() + `]Esc[-]     Go back
  [` + theme.TagWarning() + `]?[-]       This help
  [` + theme.TagWarning() + `]T[-]       Theme selector
  [` + theme.TagWarning() + `]g[-]       Go to graph (home)
  [` + theme.TagWarning() + `]b[-]       Branches
  [` + theme.TagWarning() + `]s[-]       Status
  [` + theme.TagWarning() + `]S[-]       Stash

[` + theme.TagAccent() + `]Graph View[-]
  [` + theme.TagWarning() + `]j/k[-]     Navigate up/down
  [` + theme.TagWarning() + `]g/G[-]     Jump to top/bottom
  [` + theme.TagWarning() + `]Enter[-]   View commit details
  [` + theme.TagWarning() + `]d[-]       Show diff
  [` + theme.TagWarning() + `]c[-]       Checkout commit
  [` + theme.TagWarning() + `]p[-]       Toggle preview panel
  [` + theme.TagWarning() + `]r[-]       Refresh
  [` + theme.TagWarning() + `]C[-]       Cherry-pick
  [` + theme.TagWarning() + `]R[-]       Revert

[` + theme.TagAccent() + `]Status View[-]
  [` + theme.TagWarning() + `]j/k[-]     Navigate
  [` + theme.TagWarning() + `]Space[-]   Stage/unstage file
  [` + theme.TagWarning() + `]a[-]       Stage all
  [` + theme.TagWarning() + `]u[-]       Unstage all
  [` + theme.TagWarning() + `]c[-]       Commit staged
  [` + theme.TagWarning() + `]d[-]       Show file diff
  [` + theme.TagWarning() + `]Tab[-]     Switch panels
  [` + theme.TagWarning() + `]Enter[-]   Interactive staging

[` + theme.TagAccent() + `]Interactive Staging[-]
  [` + theme.TagWarning() + `]j/k[-]     Navigate lines
  [` + theme.TagWarning() + `]J/K[-]     Navigate hunks
  [` + theme.TagWarning() + `]Space[-]   Select line
  [` + theme.TagWarning() + `]s[-]       Stage hunk
  [` + theme.TagWarning() + `]l[-]       Stage selected lines
  [` + theme.TagWarning() + `]S[-]       Stage all
  [` + theme.TagWarning() + `]u[-]       Unstage hunk
  [` + theme.TagWarning() + `]D[-]       Discard hunk

[` + theme.TagAccent() + `]Branch View[-]
  [` + theme.TagWarning() + `]Enter[-]   Checkout branch
  [` + theme.TagWarning() + `]n[-]       New branch
  [` + theme.TagWarning() + `]d[-]       Delete branch
  [` + theme.TagWarning() + `]r[-]       Rename branch
  [` + theme.TagWarning() + `]m[-]       Merge into current
  [` + theme.TagWarning() + `]R[-]       Toggle remote branches

[` + theme.TagAccent() + `]Stash View[-]
  [` + theme.TagWarning() + `]a[-]       Apply stash
  [` + theme.TagWarning() + `]p[-]       Pop stash
  [` + theme.TagWarning() + `]d[-]       Drop stash
  [` + theme.TagWarning() + `]n[-]       New stash
  [` + theme.TagWarning() + `]D[-]       View stash diff
  [` + theme.TagWarning() + `]b[-]       Create branch from stash

[` + theme.TagFgDim() + `]github.com/atterpac/ichi[-]
`

	v.content.SetText(helpText)

	panel := components.NewPanel().
		SetTitle("Help").
		SetContent(v.content)

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)
}

// nav.Component interface

func (v *HelpView) Name() string {
	return "Help"
}

func (v *HelpView) Start() {}
func (v *HelpView) Stop()  {}

func (v *HelpView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Scroll"},
		{Key: "Esc", Description: "Close"},
	}
}

// core.Widget interface

func (v *HelpView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *HelpView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *HelpView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }
func (v *HelpView) Blur()                         { v.flex.Blur() }
func (v *HelpView) HasFocus() bool                { return v.flex.HasFocus() }

func (v *HelpView) HandleKey(event *tcell.EventKey) bool {
	row, col := v.content.GetScrollOffset()

	bindings := input.NewKeyBindings().
		On(tcell.KeyDown, func(e *tcell.EventKey) bool {
			v.content.ScrollTo(row+1, col)
			return true
		}).
		On(tcell.KeyUp, func(e *tcell.EventKey) bool {
			if row > 0 {
				v.content.ScrollTo(row-1, col)
			}
			return true
		}).
		OnRune('j', func(e *tcell.EventKey) bool {
			v.content.ScrollTo(row+1, col)
			return true
		}).
		OnRune('k', func(e *tcell.EventKey) bool {
			if row > 0 {
				v.content.ScrollTo(row-1, col)
			}
			return true
		}).
		OnRune('g', func(e *tcell.EventKey) bool {
			v.content.ScrollTo(0, col)
			return true
		}).
		OnRune('G', func(e *tcell.EventKey) bool {
			v.content.ScrollTo(999999, col)
			return true
		})

	return bindings.Handle(event)
}
