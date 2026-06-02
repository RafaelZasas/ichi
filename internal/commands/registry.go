package commands

import (
	"github.com/atterpac/dado/layout"

	"github.com/atterpac/ichi/internal/git"
	"github.com/atterpac/ichi/internal/selection"
)

// ArgType represents the type of command argument for completion.
type ArgType int

const (
	ArgTypeString ArgType = iota // Plain string, no special completion
	ArgTypeFile                  // Completes with repo files
	ArgTypeBranch                // Completes with branch names
	ArgTypeCommit                // Completes with commit hashes
	ArgTypeRef                   // Completes with any ref (branch/tag/commit)
	ArgTypeTag                   // Completes with tag names
	ArgTypeRemote                // Completes with remote names
	ArgTypeStash                 // Completes with stash entries
)

// ArgSpec defines a command argument.
type ArgSpec struct {
	Name     string  // Argument name for display/help
	Type     ArgType // Type determines completion behavior
	Required bool    // Whether the argument is required
}

// Context provides access to app resources for command handlers.
type Context struct {
	App       *layout.App
	Repo      *git.Repository
	StatusBar *layout.StatusBar
}

// Selection returns the current view's selection context.
// Returns nil if the current view doesn't implement selection.Provider.
func (c *Context) Selection() *selection.Context {
	current := c.App.Pages().Current()
	if provider, ok := current.(selection.Provider); ok {
		return provider.Selection()
	}
	return nil
}

// Handler is the function signature for command handlers.
type Handler func(ctx *Context, args []string) error

// Command defines a registered command.
type Command struct {
	Name        string    // Primary command name
	Aliases     []string  // Alternative names
	Description string    // Brief description for help/completion
	Args        []ArgSpec // Argument specifications
	Handler     Handler   // Function to execute the command
}

// registry maps command names and aliases to commands.
var registry = make(map[string]*Command)

// allCommands stores all registered commands (without duplicates from aliases).
var allCommands []*Command

// Register adds a command to the registry.
func Register(cmd *Command) {
	registry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		registry[alias] = cmd
	}
	allCommands = append(allCommands, cmd)
}

// Get returns a command by name or alias, or nil if not found.
func Get(name string) *Command {
	return registry[name]
}

// All returns all registered commands.
func All() []*Command {
	return allCommands
}

// Names returns all command names (not aliases) for completion.
func Names() []string {
	names := make([]string, 0, len(allCommands))
	for _, cmd := range allCommands {
		names = append(names, cmd.Name)
	}
	return names
}

// NamesWithAliases returns all command names and aliases.
func NamesWithAliases() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
