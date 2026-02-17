package app

import (
	"github.com/atterpac/jig/clipboard"
)

// CopyToClipboard copies text to the system clipboard and shows a toast.
func CopyToClipboard(text string, description string) error {
	err := clipboard.Copy(text)
	if err != nil {
		ToastError("Failed to copy to clipboard")
		return err
	}

	if description != "" {
		ToastSuccess("Copied " + description + " to clipboard")
	} else {
		ToastSuccess("Copied to clipboard")
	}
	return nil
}

// CopyCommitHash copies a commit hash to clipboard.
func CopyCommitHash(hash string) error {
	return CopyToClipboard(hash, "commit hash")
}

// CopyBranchName copies a branch name to clipboard.
func CopyBranchName(name string) error {
	return CopyToClipboard(name, "branch name")
}

// CopyFilePath copies a file path to clipboard.
func CopyFilePath(path string) error {
	return CopyToClipboard(path, "file path")
}

// ClipboardAvailable returns true if clipboard is available.
func ClipboardAvailable() bool {
	return clipboard.Available()
}
