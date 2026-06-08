package views

import (
	"github.com/gdamore/tcell/v2"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/core"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/dado/theme"
)

// OutputView displays captured command output in a scrollable pager. When
// asDiff is set the content is rendered through dado's DiffViewer; otherwise it
// is shown as plain (line-numbered) text.
type OutputView struct {
	flex    *core.Flex
	app     *layout.App
	title   string
	content string
	asDiff  bool

	code *components.CodeView
	diff *components.DiffViewer
}

// NewOutputView creates a pager view for arbitrary command output.
func NewOutputView(app *layout.App, title, content string, asDiff bool) *OutputView {
	v := &OutputView{
		flex:    core.NewFlex(),
		app:     app,
		title:   title,
		content: content,
		asDiff:  asDiff,
	}
	v.setup()
	return v
}

func (v *OutputView) setup() {
	panel := components.NewPanel().SetTitle(v.title)

	if v.asDiff {
		v.diff = components.NewDiffViewer()
		v.diff.SetShowLineNumbers(true)
		v.diff.SetUnifiedDiff(v.content)
		panel.SetContent(v.diff)
	} else {
		v.code = components.NewCodeView()
		v.code.SetShowLineNumbers(true)
		v.code.SetCode(v.content)
		panel.SetContent(v.code)
	}

	v.flex.SetDirection(core.Column)
	v.flex.SetBackgroundColor(theme.Bg())
	v.flex.AddItem(panel, 0, 1, true)
}

// nav.Component interface

func (v *OutputView) Name() string { return v.title }

func (v *OutputView) Start() {}

func (v *OutputView) Stop() {}

func (v *OutputView) Hints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "j/k", Description: "Scroll"},
		{Key: "g/G", Description: "Top/Bottom"},
		{Key: "Esc", Description: "Back"},
	}
}

// core.Widget interface

func (v *OutputView) Draw(screen tcell.Screen)      { v.flex.Draw(screen) }
func (v *OutputView) GetRect() (int, int, int, int) { return v.flex.GetRect() }
func (v *OutputView) SetRect(x, y, w, h int)        { v.flex.SetRect(x, y, w, h) }

func (v *OutputView) Blur() {
	if v.diff != nil {
		v.diff.Blur()
		return
	}
	v.code.Blur()
}

func (v *OutputView) HasFocus() bool {
	if v.diff != nil {
		return v.diff.HasFocus()
	}
	return v.code.HasFocus()
}

func (v *OutputView) HandleKey(ev *tcell.EventKey) bool {
	if v.diff != nil {
		switch ev.Rune() {
		case 'j', 'n':
			v.diff.NextChange()
			return true
		case 'k', 'N':
			v.diff.PrevChange()
			return true
		}
		switch ev.Key() {
		case tcell.KeyDown:
			v.diff.NextChange()
			return true
		case tcell.KeyUp:
			v.diff.PrevChange()
			return true
		}
		return false
	}
	return v.code.HandleKey(ev)
}
