package views

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/atterpac/jig/components"
	"github.com/atterpac/jig/layout"
	"github.com/atterpac/jig/theme"
	"github.com/atterpac/jig/validators"
)

// ShowConfirmModal displays a confirmation dialog.
func ShowConfirmModal(app *layout.App, title, message string, onConfirm func()) {
	modal := components.NewConfirmModal(title, message).
		SetOnSubmit(func() {
			app.Pages().Pop()
			onConfirm()
		}).
		SetOnClose(func() {
			app.Pages().Pop()
		})

	app.ShowModal(modal)
}

// ShowErrorModal displays an error message.
func ShowErrorModal(app *layout.App, title, message string) {
	modal := components.NewAlertModal(title, "["+theme.TagError()+"]"+message+"[-]").
		SetOnClose(func() {
			app.Pages().Pop()
		}).
		SetOnSubmit(func() {
			app.Pages().Pop()
		})

	app.ShowModal(modal)
}

// ShowInputModal displays an input dialog.
func ShowInputModal(app *layout.App, title, prompt string, onSubmit func(string)) {
	ShowInputModalWithValidator(app, title, prompt, nil, onSubmit)
}

// ShowInputModalWithValidator displays an input dialog with validation.
func ShowInputModalWithValidator(app *layout.App, title, prompt string, validator validators.Validator, onSubmit func(string)) {
	textField := components.NewTextField("input").
		SetLabel(prompt).
		SetPlaceholder("")

	if validator != nil {
		textField.SetValidator(func(value string) error {
			return validator(value)
		})
	}

	modal := components.NewFormModal(title, 60, 12).
		SetContent(textField).
		SetHints([]components.KeyHint{
			{Key: "Enter", Description: "Submit"},
			{Key: "Esc", Description: "Cancel"},
		}).
		SetDismissOnEsc(true)

	textField.SetOnSubmit(func(event *components.SubmitEvent) {
		// Check validation before submitting
		if validator != nil {
			if value, ok := event.Value.(string); ok {
				if err := validator(value); err != nil {
					return // Don't close if validation fails
				}
			}
		}
		app.Pages().Pop()
		if value, ok := event.Value.(string); ok {
			onSubmit(value)
		}
	})

	modal.SetOnClose(func() {
		app.Pages().Pop()
	})

	app.ShowModal(modal)
}

// ShowTextAreaModal displays a multi-line text input dialog.
// Uses Ctrl+S to submit (Enter adds newlines in the text area).
func ShowTextAreaModal(app *layout.App, title, label, value string, onSubmit func(string)) {
	textArea := components.NewTextArea("input").
		SetLabel(label).
		SetValue(value).
		SetPlaceholder("")

	// Use Form component which handles Ctrl+S for submit
	form := components.NewForm().
		AddField(textArea).
		SetOnSubmit(func(values map[string]any) {
			app.Pages().Pop()
			if v, ok := values["input"].(string); ok {
				onSubmit(v)
			}
		}).
		SetOnCancel(func() {
			app.Pages().Pop()
		})

	modal := components.NewFormModal(title, 70, 18).
		SetContent(form).
		SetHints([]components.KeyHint{
			{Key: "Ctrl+S", Description: "Submit"},
			{Key: "Esc", Description: "Cancel"},
		})

	app.ShowModal(modal)
}

// ShowInfoModal displays an informational message.
func ShowInfoModal(app *layout.App, title, message string) {
	// Use NewModal directly for custom sizing (larger than standard alert)
	modal := components.NewModal(components.ModalConfig{
		Title:    title,
		Width:    70,
		Height:   20,
		Backdrop: true,
	})

	modal.SetBehavior(components.ModalBehavior{
		CapturesAllInput:      true,
		DismissOnEsc:          true,
		RestoreFocusOnDismiss: true,
		Backdrop:              true,
		BlockUntilDismissed:   false,
	})

	messageView := tview.NewTextView().
		SetText(message).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWordWrap(true)
	messageView.SetBackgroundColor(theme.Bg())
	messageView.SetTextColor(theme.Fg())

	modal.SetContent(messageView)
	modal.SetHints([]components.KeyHint{
		{Key: "Enter/Esc", Description: "Close"},
	})
	modal.SetOnClose(func() {
		app.Pages().Pop()
	})
	modal.SetOnSubmit(func() {
		app.Pages().Pop()
	})

	app.ShowModal(modal)
}
