package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const (
	configFile     = "config.json"
	defaultTheme   = "tokyonight-night"
)

// Config holds all user preferences.
type Config struct {
	Theme string `json:"theme"`
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
	configPath = filepath.Join(configDir, "gxt", configFile)
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

func load() {
	current = &Config{
		Theme: defaultTheme,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}

	// Only apply valid fields
	if loaded.Theme != "" {
		current.Theme = loaded.Theme
	}
}

func save() {
	mu.RLock()
	data, err := json.MarshalIndent(current, "", "  ")
	mu.RUnlock()
	if err != nil {
		return
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	_ = os.WriteFile(configPath, data, 0644)
}
