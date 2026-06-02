package app

import (
	"sync"
	"time"

	"github.com/atterpac/dado/components"
	"github.com/atterpac/dado/layout"
	"github.com/gdamore/tcell/v2"
)

var (
	busyMu        sync.Mutex
	busySpinner   *components.Spinner
	busyLabel     string
	busyActive    bool
	busyStatusBar *layout.StatusBar
)

func IsBusy() bool {
	busyMu.Lock()
	defer busyMu.Unlock()
	return busyActive
}

func StopBusy() {
	busyMu.Lock()
	if busySpinner != nil {
		busySpinner.Stop()
		busySpinner = nil
	}
	busyLabel = ""
	busyActive = false
	busyMu.Unlock()
}

func StartBusy(label string) {
	busyMu.Lock()

	busyActive = true
	busyLabel = label

	busySpinner = components.NewSpinner().
		SetStyle(components.SpinnerCircle).
		SetInterval(100 * time.Millisecond).
		SetLabel(label)

	sp := busySpinner

	busyMu.Unlock()
	busyStatusBar.ClearSections().ClearRightSections()
	sp.Start()
}

func InitBusyOverlay(application *layout.App, statusBar *layout.StatusBar) {
	busyStatusBar = statusBar

	application.GetApp().SetAfterDrawFunc(func(screen tcell.Screen) {
		if !busyActive || busySpinner == nil {
			return
		}

		x, y, _, _ := statusBar.GetInnerRect()
		spinnerW := 3 + len(busyLabel) // 2 for glyph cols + space + label
		busySpinner.SetRect(x+1, y+1, spinnerW, 1)

		busySpinner.Draw(screen)
	})
}
