package app

import (
	"github.com/atterpac/jig/components"
	"github.com/rivo/tview"
)

// Global toast manager instance
var toastManager *components.ToastManager

// InitToasts initializes the toast manager with the application.
func InitToasts(app *tview.Application) {
	toastManager = components.NewToastManager(app).
		SetPosition(components.ToastBottomRight).
		SetMaxVisible(3).
		SetMaxWidth(50)
}

// GetToastManager returns the global toast manager.
func GetToastManager() *components.ToastManager {
	return toastManager
}

// ToastSuccess shows a success notification.
func ToastSuccess(message string) {
	if toastManager != nil {
		toastManager.Success(message)
	}
}

// ToastError shows an error notification.
func ToastError(message string) {
	if toastManager != nil {
		toastManager.Error(message)
	}
}

// ToastWarning shows a warning notification.
func ToastWarning(message string) {
	if toastManager != nil {
		toastManager.Warning(message)
	}
}

// ToastInfo shows an info notification.
func ToastInfo(message string) {
	if toastManager != nil {
		toastManager.Info(message)
	}
}
