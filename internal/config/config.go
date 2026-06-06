package config

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	configFile             = "config.yaml"
	defaultTheme           = "tokyonight-night"
	defaultStagingViewMode = 0 // tree view

)

// configHeader is prepended when ichi writes the config file so a freshly
// generated config.yaml documents itself.
const configHeader = `# ichi configuration
# Location: $XDG_CONFIG_HOME/ichi/config.yaml (usually ~/.config/ichi/config.yaml)
#
# This file is rewritten by ichi when you change settings from the UI, but you
# can also edit it by hand. Unknown keys are ignored.

`

// Config holds all user preferences.
type Config struct {
	// Theme is the name of the active color theme, e.g. "tokyonight-night",
	// "catppuccin-mocha", "nord", "dracula", "gruvbox-dark", "rosepine".
	// Run any theme from the command palette to see the full list.
	Theme string `yaml:"theme"`

	// StagingViewMode controls how staged files are displayed in the staging
	// panel. 0 for tree view, 1 for flat file list.
	StagingViewMode int `yaml:"staging_view_mode"`
}

var (
	current    *Config
	configPath string
	mu         sync.RWMutex
)

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	configPath = filepath.Join(configDir, "ichi", configFile)
	load()
}

// Get returns the current config.
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// GetTheme returns the current theme name.
func GetTheme() string {
	mu.RLock()
	defer mu.RUnlock()
	return current.Theme
}

// SetTheme updates the theme and saves config.
func SetTheme(name string) {
	mu.Lock()
	current.Theme = name
	mu.Unlock()
	save()
}

// GetStagingViewMode returns the current staging view mode (0 for tree view, 1
// for flat file list).
func GetStagingViewMode() int {
	mu.RLock()
	defer mu.RUnlock()
	return current.StagingViewMode
}

// SetStagingViewMode updates the staging view mode and saves config. Valid
// modes are 0 for tree view and 1 for flat file list.
func SetStagingViewMode(mode int) {
	mu.Lock()
	current.StagingViewMode = mode
	mu.Unlock()
	save()
}

func load() {
	current = &Config{
		Theme:           defaultTheme,
		StagingViewMode: defaultStagingViewMode,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return
	}

	// Only apply valid fields
	if loaded.Theme != "" {
		current.Theme = loaded.Theme
	}

	if loaded.StagingViewMode == 0 || loaded.StagingViewMode == 1 {
		current.StagingViewMode = loaded.StagingViewMode
	}
}

func save() {
	mu.RLock()
	data, err := yaml.Marshal(current)
	mu.RUnlock()
	if err != nil {
		return
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	_ = os.WriteFile(configPath, append([]byte(configHeader), data...), 0644)
}
