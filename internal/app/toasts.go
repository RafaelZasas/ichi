package app

import (
	"github.com/atterpac/dado/components"
)

// Global toast manager instance
var toastManager *components.ToastManager

// InitToasts initializes the toast manager.
func InitToasts() {
	toastManager = components.NewToastManager().
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
