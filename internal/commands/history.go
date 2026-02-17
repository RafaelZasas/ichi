package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	maxHistorySize = 100
	historyFile    = "history"
)

var (
	history      []string
	historyIndex int
	historyMu    sync.Mutex
	historyPath  string
)

func init() {
	// Determine history file path
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	historyPath = filepath.Join(configDir, "gxt", historyFile)

	// Load history from file
	loadHistory()
}

// loadHistory reads command history from disk.
func loadHistory() {
	historyMu.Lock()
	defer historyMu.Unlock()

	file, err := os.Open(historyPath)
	if err != nil {
		return // No history file yet
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			history = append(history, line)
		}
	}

	// Keep only the last maxHistorySize entries
	if len(history) > maxHistorySize {
		history = history[len(history)-maxHistorySize:]
	}

	historyIndex = len(history)
}

// saveHistory writes command history to disk.
func saveHistory() {
	historyMu.Lock()
	defer historyMu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	file, err := os.Create(historyPath)
	if err != nil {
		return
	}
	defer file.Close()

	for _, cmd := range history {
		file.WriteString(cmd + "\n")
	}
}

// AddHistory adds a command to history.
func AddHistory(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	historyMu.Lock()
	defer historyMu.Unlock()

	// Don't add duplicates of the last command
	if len(history) > 0 && history[len(history)-1] == cmd {
		historyIndex = len(history)
		return
	}

	history = append(history, cmd)

	// Trim if too long
	if len(history) > maxHistorySize {
		history = history[1:]
	}

	historyIndex = len(history)

	// Save asynchronously
	go saveHistory()
}

// HistoryPrev returns the previous history entry.
// Called when up arrow is pressed.
func HistoryPrev(current string) string {
	historyMu.Lock()
	defer historyMu.Unlock()

	if len(history) == 0 {
		return current
	}

	if historyIndex > 0 {
		historyIndex--
	}

	return history[historyIndex]
}

// HistoryNext returns the next history entry.
// Called when down arrow is pressed.
func HistoryNext(current string) string {
	historyMu.Lock()
	defer historyMu.Unlock()

	if len(history) == 0 {
		return current
	}

	if historyIndex < len(history)-1 {
		historyIndex++
		return history[historyIndex]
	}

	// At end of history, return to empty
	historyIndex = len(history)
	return ""
}

// ResetHistoryIndex resets the history navigation index.
// Called when entering command mode.
func ResetHistoryIndex() {
	historyMu.Lock()
	defer historyMu.Unlock()
	historyIndex = len(history)
}

// GetHistory returns a copy of the command history.
func GetHistory() []string {
	historyMu.Lock()
	defer historyMu.Unlock()

	result := make([]string, len(history))
	copy(result, history)
	return result
}
