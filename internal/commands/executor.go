package commands

import (
	"fmt"
)

// Execute parses and runs a command string.
func Execute(ctx *Context, input string) error {
	parsed := Parse(input)
	if parsed.Name == "" {
		return nil // Empty command, nothing to do
	}

	cmd := Get(parsed.Name)
	if cmd == nil {
		return fmt.Errorf("unknown command: %s", parsed.Name)
	}

	// Validate required arguments
	for i, arg := range cmd.Args {
		if arg.Required && i >= len(parsed.Args) {
			return fmt.Errorf("%s: missing required argument <%s>", cmd.Name, arg.Name)
		}
	}

	// Add to history
	AddHistory(input)

	return cmd.Handler(ctx, parsed.Args)
}

// ExecuteResult holds the result of command execution for display.
type ExecuteResult struct {
	Success bool
	Message string
	Error   error
}

// ExecuteWithResult executes a command and returns a result struct.
func ExecuteWithResult(ctx *Context, input string) ExecuteResult {
	err := Execute(ctx, input)
	if err != nil {
		return ExecuteResult{
			Success: false,
			Error:   err,
		}
	}
	return ExecuteResult{
		Success: true,
	}
}
