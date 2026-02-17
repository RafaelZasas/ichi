package commands

import (
	"sort"
	"strings"

	"github.com/atterpac/gxt/internal/git"
)

// GetCompletions returns completion suggestions for the current input.
func GetCompletions(repo *git.Repository, input string) []string {
	input = strings.TrimSpace(input)
	parts := strings.Fields(input)

	// No input or typing first word - complete command names
	if len(parts) == 0 || (len(parts) == 1 && !IsPartialArg(input)) {
		prefix := ""
		if len(parts) == 1 {
			prefix = parts[0]
		}
		return completeCommandName(prefix)
	}

	// Get the command
	cmd := Get(parts[0])
	if cmd == nil {
		return nil
	}

	// Determine which argument we're completing
	argIndex := len(parts) - 2 // -1 for command, -1 for 0-based
	if IsPartialArg(input) {
		argIndex = len(parts) - 1 // Starting a new argument
	}

	// Check if we have an ArgSpec for this position
	if argIndex < 0 || argIndex >= len(cmd.Args) {
		return nil
	}

	// Get partial text being typed
	partial := ""
	if !IsPartialArg(input) && len(parts) > 1 {
		partial = parts[len(parts)-1]
	}

	return completeArg(repo, cmd.Args[argIndex].Type, partial)
}

// completeCommandName returns command names matching the prefix.
func completeCommandName(prefix string) []string {
	prefix = strings.ToLower(prefix)
	var matches []string

	// Include both names and aliases
	for name := range registry {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			matches = append(matches, name)
		}
	}

	sort.Strings(matches)
	return matches
}

// completeArg returns completions based on argument type.
func completeArg(repo *git.Repository, argType ArgType, prefix string) []string {
	switch argType {
	case ArgTypeFile:
		return repo.ListFiles(prefix)
	case ArgTypeBranch:
		return repo.ListBranchNames(prefix)
	case ArgTypeTag:
		return repo.ListTagNames(prefix)
	case ArgTypeRef:
		// Combine branches and tags
		refs := repo.ListBranchNames(prefix)
		refs = append(refs, repo.ListTagNames(prefix)...)
		sort.Strings(refs)
		return refs
	case ArgTypeRemote:
		return filterByPrefix(repo.ListRemotes(), prefix)
	case ArgTypeStash:
		return repo.ListStashEntries(prefix)
	case ArgTypeCommit:
		return repo.ListRecentCommitHashes(prefix, 20)
	case ArgTypeString:
		return nil // No completion for plain strings
	}
	return nil
}

// filterByPrefix filters a slice of strings by prefix.
func filterByPrefix(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}
	prefix = strings.ToLower(prefix)
	var matches []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), prefix) {
			matches = append(matches, item)
		}
	}
	return matches
}

// GetSuggestion returns an inline suggestion for the current input.
// This is the "ghost text" that appears after the cursor.
func GetSuggestion(repo *git.Repository, input string) string {
	completions := GetCompletions(repo, input)
	if len(completions) == 0 {
		return ""
	}

	// Return the first completion as suggestion
	// Reconstruct full command with this completion
	parts := strings.Fields(input)

	if len(parts) == 0 {
		return completions[0]
	}

	if len(parts) == 1 && !IsPartialArg(input) {
		// Completing command name
		return completions[0]
	}

	// Completing an argument - replace last part with completion
	if IsPartialArg(input) {
		return input + completions[0]
	}

	parts[len(parts)-1] = completions[0]
	return strings.Join(parts, " ")
}
