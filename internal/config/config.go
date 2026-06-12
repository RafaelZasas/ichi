package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
#
# Custom context-aware commands run from the command bar (":<name>"). The
# command string is run via "sh -c" with {{tokens}} filled from the current
# selection. Available tokens include: commitHash, shortHash, commitMessage,
# commitSubject, author, branch, branchUpstream, currentBranch, file, fileName,
# fileDir, stash, stashIndex, stashMessage, ref, repoRoot, view, editor,
# {{env:VAR}}, and positional {{1}}/{{args}}.
#
# view:     none (default, toast result) | pager | diff | graph | commit |
#           branch | editor (suspends ichi for $EDITOR, returns on exit)
# requires: optional guard - commit | branch | file | stash
# confirm:  prompt before running when true
#
# commands:
#   - name: showlog
#     aliases: [sl]
#     description: Show full log for the selected commit
#     command: git log -p {{commitHash}}
#     view: pager
#     requires: commit
#   - name: edit
#     command: $EDITOR {{filePath}}
#     view: editor
#     requires: file
#   - name: vim
#     command: $EDITOR
#     view: editor
#
# Saved repos power the repo switcher (Ctrl+R) and ":repo <alias/name>".
# Each entry needs a name and path; alias is an optional short handle.
#
# repos:
#   - name: ichi
#     path: /home/me/projects/ichi
#     alias: i

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

	// Commands defines user-supplied context-aware commands invokable from the
	// command bar via ":<name>". See CustomCommand for the available fields.
	Commands []CustomCommand `yaml:"commands"`

	// Repos defines saved git repositories that can be switched between via the
	// repo switcher (Ctrl+R) or ":repo <alias/name>".
	Repos []Repo `yaml:"repos"`
}

// Repo is a saved git repository the user can quickly switch between.
type Repo struct {
	// Name is the human-readable label shown in the repo switcher.
	Name string `yaml:"name"`
	// Path is the absolute path to the repository working tree.
	Path string `yaml:"path"`
	// Alias is an optional short handle for ":repo <alias>".
	Alias string `yaml:"alias"`
}

// CustomCommand is a user-defined command run from the command bar. The
// Command string is expanded with {{template}} tokens drawn from the current
// selection (commit hash, file, branch, ...) and executed via "sh -c".
type CustomCommand struct {
	// Name is the command keyword typed after ':' (e.g. "showlog").
	Name string `yaml:"name"`
	// Aliases are alternative keywords for the same command.
	Aliases []string `yaml:"aliases"`
	// Description appears in the command palette and help.
	Description string `yaml:"description"`
	// Command is the shell command template, e.g. "git log -p {{commitHash}}".
	Command string `yaml:"command"`
	// View selects how output is rendered: "none" (default, toast result),
	// "pager", "diff", "graph", "commit", "branch", or "editor".
	View string `yaml:"view"`
	// Requires guards execution on a selection kind: "", "commit", "branch",
	// "file", or "stash".
	Requires string `yaml:"requires"`
	// Confirm shows a confirmation modal before running when true.
	Confirm bool `yaml:"confirm"`
	// Key optionally binds the command to a global key chord, e.g. "ctrl+g",
	// "alt+x", "f5", or a single rune "x". Empty means no key binding.
	Key string `yaml:"key"`
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
	current.Commands = loaded.Commands
	current.Repos = loaded.Repos
}

// GetRepos returns the saved repositories.
func GetRepos() []Repo {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Repo, len(current.Repos))
	copy(out, current.Repos)
	return out
}

// FindRepo looks up a saved repo by name or alias (case-insensitive).
func FindRepo(query string) (Repo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(query))
	for _, r := range current.Repos {
		if strings.ToLower(r.Name) == q || (r.Alias != "" && strings.ToLower(r.Alias) == q) {
			return r, true
		}
	}
	return Repo{}, false
}

// SaveRepo adds or updates a saved repo and persists the config. When oldName
// is non-empty the existing entry with that name is replaced (supporting
// renames); otherwise the repo is upserted by name.
func SaveRepo(oldName string, r Repo) {
	mu.Lock()
	key := oldName
	if key == "" {
		key = r.Name
	}
	updated := false
	for i := range current.Repos {
		if current.Repos[i].Name == key {
			current.Repos[i] = r
			updated = true
			break
		}
	}
	if !updated {
		current.Repos = append(current.Repos, r)
	}
	mu.Unlock()
	save()
}

// DeleteRepo removes a saved repo by name and persists the config.
func DeleteRepo(name string) {
	mu.Lock()
	filtered := current.Repos[:0]
	for _, r := range current.Repos {
		if r.Name != name {
			filtered = append(filtered, r)
		}
	}
	current.Repos = filtered
	mu.Unlock()
	save()
}

// GetCommands returns the user-defined custom commands.
func GetCommands() []CustomCommand {
	mu.RLock()
	defer mu.RUnlock()
	return current.Commands
}

// validViews and validRequires bound the accepted enum-like fields.
var validViews = map[string]bool{
	"": true, "none": true, "pager": true, "diff": true,
	"graph": true, "commit": true, "branch": true, "editor": true,
}

var validRequires = map[string]bool{
	"": true, "commit": true, "branch": true, "file": true, "stash": true,
}

// ValidateCommand reports the first problem with a custom command definition,
// or nil if it is well-formed.
func ValidateCommand(c CustomCommand) error {
	if c.Name == "" {
		return fmt.Errorf("custom command missing 'name'")
	}
	if c.Command == "" {
		return fmt.Errorf("custom command %q missing 'command'", c.Name)
	}
	if !validViews[c.View] {
		return fmt.Errorf("custom command %q has unknown view %q", c.Name, c.View)
	}
	if !validRequires[c.Requires] {
		return fmt.Errorf("custom command %q has unknown requires %q", c.Name, c.Requires)
	}
	return nil
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
