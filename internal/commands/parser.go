package commands

import (
	"strings"
)

// ParsedCommand holds the parsed command name and arguments.
type ParsedCommand struct {
	Name string   // Command name (first word)
	Args []string // Remaining arguments
	Raw  string   // Original input
}

// Parse splits a command string into command name and arguments.
// It handles quoted strings and preserves empty arguments.
func Parse(input string) *ParsedCommand {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ParsedCommand{Raw: input}
	}

	parts := parseWithQuotes(input)
	if len(parts) == 0 {
		return &ParsedCommand{Raw: input}
	}

	return &ParsedCommand{
		Name: parts[0],
		Args: parts[1:],
		Raw:  input,
	}
}

// parseWithQuotes splits input respecting quoted strings.
// Supports both single and double quotes.
func parseWithQuotes(input string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range input {
		switch {
		case (r == '"' || r == '\'') && !inQuote:
			// Start of quoted string
			inQuote = true
			quoteChar = r
		case r == quoteChar && inQuote:
			// End of quoted string
			inQuote = false
			quoteChar = 0
		case r == ' ' && !inQuote:
			// Space outside quotes - end current part
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add final part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// IsPartialArg returns true if the input ends with a space,
// indicating the user is starting to type a new argument.
func IsPartialArg(input string) bool {
	return strings.HasSuffix(input, " ")
}

// LastPartial returns the last partial word being typed.
// Returns empty string if input ends with space.
func LastPartial(input string) string {
	if IsPartialArg(input) {
		return ""
	}
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// CurrentArgIndex returns which argument position the user is currently typing.
// Returns -1 for command name, 0 for first arg, etc.
func CurrentArgIndex(input string) int {
	parts := strings.Fields(input)
	count := len(parts)

	if count == 0 {
		return -1 // Typing command name
	}

	if IsPartialArg(input) {
		return count - 1 // Starting new argument
	}

	return count - 2 // Typing partial of current argument (-1 for command, -1 for partial)
}
