package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Repository represents a git repository.
type Repository struct {
	path   string
	branch string
}

// OpenRepository opens a git repository at the given path.
func OpenRepository(path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Verify this is a git repo
	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", absPath)
	}

	return &Repository{path: absPath}, nil
}

// Path returns the repository path.
func (r *Repository) Path() string {
	return r.path
}

// SetPath re-points this repository at a different working tree, validating
// that the new path is a git repository. Mutating in place lets every holder
// of this *Repository observe the switch without re-wiring closures.
func (r *Repository) SetPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository: %s", absPath)
	}
	r.path = absPath
	r.branch = ""
	return nil
}

// CurrentBranch returns the current branch name.
func (r *Repository) CurrentBranch() string {
	out, err := r.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out)
}

// AheadBehind returns how many commits ahead and behind the current branch is
// relative to its upstream.
func (r *Repository) AheadBehind() (ahead, behind int) {
	out, err := r.run("rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind
}

// HasUncommitted returns true if there are uncommitted changes.
func (r *Repository) HasUncommitted() bool {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// HEAD returns the current HEAD commit hash.
func (r *Repository) HEAD() string {
	out, err := r.run("rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ShortHEAD returns the short form of the current HEAD commit hash.
func (r *Repository) ShortHEAD() string {
	out, err := r.run("rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// IsDetachedHEAD returns true if HEAD is detached (not on a branch).
func (r *Repository) IsDetachedHEAD() bool {
	out, err := r.run("symbolic-ref", "-q", "HEAD")
	return err != nil || out == ""
}

// Checkout checks out a commit or branch.
func (r *Repository) Checkout(ref string) error {
	_, err := r.run("checkout", ref)
	return err
}

// CheckoutBranch checks out a branch, creating it if it doesn't exist.
func (r *Repository) CheckoutBranch(branch string, create bool) error {
	if create {
		_, err := r.run("checkout", "-b", branch)
		return err
	}
	return r.Checkout(branch)
}

// Fetch fetches from a remote.
func (r *Repository) Fetch(remote string) error {
	_, err := r.run("fetch", remote)
	return err
}

// FetchAll fetches from all remotes.
func (r *Repository) FetchAll() error {
	_, err := r.run("fetch", "--all")
	return err
}

// Pull pulls from the upstream.
func (r *Repository) Pull() error {
	_, err := r.run("pull")
	return err
}

// Push pushes to the upstream.
func (r *Repository) Push() error {
	_, err := r.run("push")
	return err
}

// HasUpstream returns true if the current branch has an upstream tracking branch.
func (r *Repository) HasUpstream() bool {
	_, err := r.run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	return err == nil
}

// PushSetUpstream pushes and sets the upstream.
func (r *Repository) PushSetUpstream(remote, branch string) error {
	_, err := r.run("push", "-u", remote, branch)
	return err
}

// StageFile stages a file.
func (r *Repository) StageFile(file string) error {
	_, err := r.run("add", file)
	return err
}

// StageAll stages all changes.
func (r *Repository) StageAll() error {
	_, err := r.run("add", "-A")
	return err
}

// UnstageFile unstages a file.
func (r *Repository) UnstageFile(file string) error {
	_, err := r.run("reset", "HEAD", "--", file)
	return err
}

// UnstageAll unstages all files.
func (r *Repository) UnstageAll() error {
	_, err := r.run("reset", "HEAD")
	return err
}

// Commit creates a commit with the given message.
func (r *Repository) Commit(message string) error {
	_, err := r.run("commit", "-m", message)
	return err
}

// CommitAmend amends the most recent commit with a new message and any staged changes.
func (r *Repository) CommitAmend(message string) error {
	_, err := r.run("commit", "--amend", "-m", message)
	return err
}

// CherryPick cherry-picks a commit.
func (r *Repository) CherryPick(hash string) error {
	_, err := r.run("cherry-pick", hash)
	return err
}

// Revert reverts a commit.
func (r *Repository) Revert(hash string) error {
	_, err := r.run("revert", "--no-edit", hash)
	return err
}

// ResetHard performs a hard reset to a commit.
func (r *Repository) ResetHard(ref string) error {
	_, err := r.run("reset", "--hard", ref)
	return err
}

// ResetSoft performs a soft reset to a commit.
func (r *Repository) ResetSoft(ref string) error {
	_, err := r.run("reset", "--soft", ref)
	return err
}

// DiscardFileChanges discards all unstaged changes to a file.
func (r *Repository) DiscardFileChanges(file string) error {
	_, err := r.run("checkout", "--", file)
	return err
}

// ChangeStats holds file count and line change stats.
type ChangeStats struct {
	Files      int
	Insertions int
	Deletions  int
	Untracked  int // Untracked files (no +/- available)
}

// StatusCounts returns counts for staged changes, unstaged changes, and stashes.
func (r *Repository) StatusCounts() (staged, unstaged ChangeStats, stashCount int) {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		if out, err := r.run("diff", "--cached", "--numstat"); err == nil {
			staged = parseNumstat(out)
		}
	}()

	go func() {
		defer wg.Done()
		if out, err := r.run("diff", "--numstat"); err == nil {
			unstaged = parseNumstat(out)
		}
	}()

	go func() {
		defer wg.Done()
		if out, err := r.run("ls-files", "--others", "--exclude-standard"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				if strings.TrimSpace(line) != "" {
					unstaged.Untracked++
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		if out, err := r.run("stash", "list"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				if strings.TrimSpace(line) != "" {
					stashCount++
				}
			}
		}
	}()

	wg.Wait()
	return staged, unstaged, stashCount
}

// parseNumstat parses git diff --numstat output
func parseNumstat(output string) ChangeStats {
	var stats ChangeStats
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			stats.Files++
			// Handle binary files (shown as "-")
			if parts[0] != "-" {
				var ins int
				fmt.Sscanf(parts[0], "%d", &ins)
				stats.Insertions += ins
			}
			if parts[1] != "-" {
				var del int
				fmt.Sscanf(parts[1], "%d", &del)
				stats.Deletions += del
			}
		}
	}
	return stats
}

// RemoteURL returns the URL of a remote.
func (r *Repository) RemoteURL(remote string) string {
	out, err := r.run("remote", "get-url", remote)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ListRemotes returns a list of remote names.
func (r *Repository) ListRemotes() []string {
	out, err := r.run("remote")
	if err != nil {
		return nil
	}

	var remotes []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes
}

// run executes a git command and returns the output.
func (r *Repository) run(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.path}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}

// RunWithStdin executes a git command with stdin input.
func (r *Repository) RunWithStdin(input string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", r.path}, args...)...)

	// Use pipes to avoid deadlock with CombinedOutput + stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// Write input and close stdin
	_, writeErr := stdin.Write([]byte(input))
	stdin.Close()

	// Wait for command to complete
	waitErr := cmd.Wait()

	if writeErr != nil {
		return writeErr
	}

	if waitErr != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return waitErr
	}
	return nil
}

// ListFiles returns tracked files matching the given prefix.
// Uses git ls-files for fast file listing.
func (r *Repository) ListFiles(prefix string) []string {
	out, err := r.run("ls-files")
	if err != nil {
		return nil
	}

	var files []string
	prefix = strings.ToLower(prefix)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		// Match on the full path (e.g. "cmd/ichi") or the base name (e.g.
		// "main.go") so a bare file name suggests files in any directory.
		if prefix == "" ||
			strings.HasPrefix(lower, prefix) ||
			strings.HasPrefix(strings.ToLower(filepath.Base(line)), prefix) {
			files = append(files, line)
		}
	}
	return files
}

// ListBranchNames returns branch names matching the given prefix.
func (r *Repository) ListBranchNames(prefix string) []string {
	out, err := r.run("branch", "--format=%(refname:short)")
	if err != nil {
		return nil
	}

	var branches []string
	prefix = strings.ToLower(prefix)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(line), prefix) {
			branches = append(branches, line)
		}
	}
	return branches
}

// ListTagNames returns tag names matching the given prefix.
func (r *Repository) ListTagNames(prefix string) []string {
	out, err := r.run("tag", "--list")
	if err != nil {
		return nil
	}

	var tags []string
	prefix = strings.ToLower(prefix)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(line), prefix) {
			tags = append(tags, line)
		}
	}
	return tags
}

// ListStashEntries returns stash entries matching the given prefix.
func (r *Repository) ListStashEntries(prefix string) []string {
	out, err := r.run("stash", "list", "--format=%gd: %s")
	if err != nil {
		return nil
	}

	var entries []string
	prefix = strings.ToLower(prefix)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(line), prefix) {
			entries = append(entries, line)
		}
	}
	return entries
}

// ListRecentCommitHashes returns recent commit hashes matching the given prefix.
func (r *Repository) ListRecentCommitHashes(prefix string, limit int) []string {
	out, err := r.run("log", "--format=%h", fmt.Sprintf("-n%d", limit))
	if err != nil {
		return nil
	}

	var hashes []string
	prefix = strings.ToLower(prefix)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(line), prefix) {
			hashes = append(hashes, line)
		}
	}
	return hashes
}

// IsCommitPushed checks if a commit exists on any remote branch.
func (r *Repository) IsCommitPushed(hash string) bool {
	out, err := r.run("branch", "-r", "--contains", hash)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// GetCommitMessage returns the full commit message for a given hash.
func (r *Repository) GetCommitMessage(hash string) (string, error) {
	out, err := r.run("log", "-1", "--format=%B", hash)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsHeadCommit checks if the given hash is the current HEAD.
func (r *Repository) IsHeadCommit(hash string) bool {
	head := r.HEAD()
	if head == "" {
		return false
	}
	// Compare full hashes or short hash prefixes
	return head == hash || strings.HasPrefix(head, hash) || strings.HasPrefix(hash, head)
}

// RenameCommit renames (rewords) a commit's message.
// For HEAD, uses --amend. For older commits, uses interactive rebase.
func (r *Repository) RenameCommit(hash, newMessage string) error {
	if r.IsHeadCommit(hash) {
		_, err := r.run("commit", "--amend", "-m", newMessage)
		return err
	}
	return r.rebaseReword(hash, newMessage)
}

// rebaseReword uses non-interactive rebase to reword a non-HEAD commit.
func (r *Repository) rebaseReword(hash, newMessage string) error {
	// Find parent of target commit for rebase base
	parentOut, err := r.run("rev-parse", hash+"^")
	if err != nil {
		return fmt.Errorf("cannot find parent commit: %w", err)
	}
	parent := strings.TrimSpace(parentOut)

	// Get short hash for matching in todo list
	shortHashOut, err := r.run("rev-parse", "--short", hash)
	if err != nil {
		return fmt.Errorf("cannot get short hash: %w", err)
	}
	shortHash := strings.TrimSpace(shortHashOut)

	// Create temp file with new commit message
	tmpFile, err := os.CreateTemp("", "ichi-commit-msg-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(newMessage); err != nil {
		tmpFile.Close()
		return fmt.Errorf("cannot write message: %w", err)
	}
	tmpFile.Close()

	// Create sequence editor script to change 'pick' to 'reword'
	// Using sed with backup extension for cross-platform compatibility
	seqEditor := fmt.Sprintf(`sed -i.bak 's/^pick %s/reword %s/' "$1" && rm -f "$1.bak"`, shortHash, shortHash)

	// Create editor script that outputs the new message
	msgEditor := fmt.Sprintf(`cat "%s" >`, tmpFile.Name())

	// Run rebase with both editors set
	cmd := exec.Command("git", "-C", r.path, "rebase", "-i", parent)
	cmd.Env = append(os.Environ(),
		"GIT_SEQUENCE_EDITOR="+seqEditor,
		"GIT_EDITOR="+msgEditor,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rebase failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// DropCommit removes a commit from history.
// For HEAD, uses soft reset. For older commits, uses interactive rebase.
func (r *Repository) DropCommit(hash string) error {
	if r.IsHeadCommit(hash) {
		return r.ResetSoft("HEAD~1")
	}
	return r.rebaseDrop(hash)
}

// rebaseDrop uses non-interactive rebase to drop a non-HEAD commit.
func (r *Repository) rebaseDrop(hash string) error {
	// Find parent of target commit for rebase base
	parentOut, err := r.run("rev-parse", hash+"^")
	if err != nil {
		return fmt.Errorf("cannot find parent commit: %w", err)
	}
	parent := strings.TrimSpace(parentOut)

	// Get short hash for matching in todo list
	shortHashOut, err := r.run("rev-parse", "--short", hash)
	if err != nil {
		return fmt.Errorf("cannot get short hash: %w", err)
	}
	shortHash := strings.TrimSpace(shortHashOut)

	// Create sequence editor script to change 'pick' to 'drop'
	seqEditor := fmt.Sprintf(`sed -i.bak 's/^pick %s/drop %s/' "$1" && rm -f "$1.bak"`, shortHash, shortHash)

	cmd := exec.Command("git", "-C", r.path, "rebase", "-i", parent)
	cmd.Env = append(os.Environ(),
		"GIT_SEQUENCE_EDITOR="+seqEditor,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rebase failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// FileLogEntry represents a commit that affected a file.
type FileLogEntry struct {
	Hash       string
	ShortHash  string
	Subject    string
	Author     string
	Date       string
	Insertions int
	Deletions  int
}

// FileLog returns commits that affected a file, following renames.
func (r *Repository) FileLog(file string, limit int) ([]FileLogEntry, error) {
	format := "%H%x00%h%x00%s%x00%an%x00%ar"
	out, err := r.run("log", "--follow", "--format="+format, "--numstat", fmt.Sprintf("-n%d", limit), "--", file)
	if err != nil {
		return nil, fmt.Errorf("file log failed: %w", err)
	}

	return parseFileLogOutput(out), nil
}

// parseFileLogOutput parses git log --numstat output.
func parseFileLogOutput(output string) []FileLogEntry {
	var entries []FileLogEntry
	lines := strings.Split(output, "\n")

	var current *FileLogEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a commit line (contains null separators)
		if strings.Contains(line, "\x00") {
			// Save previous entry
			if current != nil {
				entries = append(entries, *current)
			}

			parts := strings.Split(line, "\x00")
			if len(parts) >= 5 {
				current = &FileLogEntry{
					Hash:      parts[0],
					ShortHash: parts[1],
					Subject:   parts[2],
					Author:    parts[3],
					Date:      parts[4],
				}
			}
			continue
		}

		// Parse numstat line (insertions deletions filename)
		if current != nil {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if parts[0] != "-" {
					current.Insertions, _ = strconv.Atoi(parts[0])
				}
				if parts[1] != "-" {
					current.Deletions, _ = strconv.Atoi(parts[1])
				}
			}
		}
	}

	// Don't forget the last entry
	if current != nil {
		entries = append(entries, *current)
	}

	return entries
}
