package app

import (
	"context"

	"github.com/atterpac/dado/anim"
	"github.com/atterpac/dado/async"
	"github.com/atterpac/dado/layout"
	"github.com/atterpac/ichi/internal/git"
)

var circleFrames = []string{"◐", "◓", "◑", "◒"}

func BusyIndicator(sb *layout.StatusBar, msg string) async.LoadingIndicator {
	var cancel func()
	frame := 0

	return async.Callback(
		func() { // Show
			cancel = anim.Subscribe(0, func() {
				frame = (frame + 1) % len(circleFrames)
				sb.ClearSections()
				sb.AddSection(layout.StatusSection{Icon: circleFrames[frame], Text: msg})
			})
		},
		func() { // Hide
			if cancel != nil {
				cancel()
				cancel = nil
			}
		},
	)
}

// RunBusy is the common case: run `work`, show msg, then toast on outcome.
func RunBusy(sb *layout.StatusBar, repo *git.Repository, busyMsg, okMsg string, work func() error) {
	async.NewLoader[any]().
		WithIndicator(BusyIndicator(sb, busyMsg)).
		OnSuccess(func(any) {
			UpdateStatusBar(sb, repo)
			ToastSuccess(okMsg)
		}).
		OnError(func(err error) {
			UpdateStatusBar(sb, repo)
			ToastError(err.Error())
		}).
		Run(func(context.Context) (any, error) { return nil, work() })
}
