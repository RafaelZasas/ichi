// Package exec runs user-defined shell commands for context-aware commands.
package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// RunShell runs command via "sh -c" with cwd set to repoRoot and returns its
// captured stdout and stderr. The process inherits the current environment so
// $EDITOR and friends resolve as expected.
func RunShell(ctx context.Context, repoRoot, command string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// RunShellInteractive runs command via "sh -c" wired directly to the real
// terminal so interactive programs (e.g. $EDITOR) take over the screen. It
// MUST be called from inside app.Suspend so the TUI releases the terminal.
func RunShellInteractive(repoRoot, command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
