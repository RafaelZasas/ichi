package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConflictType represents the type of conflict operation in progress.
type ConflictType int

const (
	ConflictNone ConflictType = iota
	ConflictMerge
	ConflictRebase
	ConflictCherryPick
)

// ConflictState represents the current state of a merge/rebase conflict.
type ConflictState struct {
	Type         ConflictType
	InProgress   bool
	TargetBranch string // The branch being merged/rebased onto
	SourceBranch string // The branch being merged/rebased from
	Files        []string
}

// ConflictRegion represents a single conflict region in a file.
type ConflictRegion struct {
	StartLine   int      // Line number where conflict starts (0-indexed)
	EndLine     int      // Line number where conflict ends (0-indexed)
	OursLines   []string // Lines between <<<<<<< and |||||||
	BaseLines   []string // Lines between ||||||| and =======
	TheirsLines []string // Lines between ======= and >>>>>>>
	OursLabel   string   // Usually "HEAD" or branch name
	TheirsLabel string   // Branch/commit name
	HasBase     bool     // Whether this conflict has a base section (diff3)
}

// GetConflictState detects if a merge or rebase is in progress.
func (r *Repository) GetConflictState() (*ConflictState, error) {
	state := &ConflictState{
		Type: ConflictNone,
	}

	gitDir := filepath.Join(r.path, ".git")

	// Check for merge in progress
	mergeHeadPath := filepath.Join(gitDir, "MERGE_HEAD")
	if _, err := os.Stat(mergeHeadPath); err == nil {
		state.Type = ConflictMerge
		state.InProgress = true

		// Read target branch from MERGE_MSG or current branch
		state.TargetBranch = r.CurrentBranch()

		// Read source branch from MERGE_HEAD
		mergeHead, err := os.ReadFile(mergeHeadPath)
		if err == nil {
			state.SourceBranch = strings.TrimSpace(string(mergeHead))[:8] // Short hash
		}

		// Try to get branch name from MERGE_MSG
		mergeMsgPath := filepath.Join(gitDir, "MERGE_MSG")
		if msgData, err := os.ReadFile(mergeMsgPath); err == nil {
			msg := string(msgData)
			// Parse "Merge branch 'branchname'" format
			if strings.HasPrefix(msg, "Merge branch '") {
				parts := strings.SplitN(msg, "'", 3)
				if len(parts) >= 2 {
					state.SourceBranch = parts[1]
				}
			}
		}
	}

	// Check for rebase in progress
	rebaseMergePath := filepath.Join(gitDir, "rebase-merge")
	rebaseApplyPath := filepath.Join(gitDir, "rebase-apply")

	if _, err := os.Stat(rebaseMergePath); err == nil {
		state.Type = ConflictRebase
		state.InProgress = true

		// Read onto branch
		ontoFile := filepath.Join(rebaseMergePath, "onto")
		if data, err := os.ReadFile(ontoFile); err == nil {
			state.TargetBranch = strings.TrimSpace(string(data))[:8]
		}

		// Read head name (current branch)
		headNameFile := filepath.Join(rebaseMergePath, "head-name")
		if data, err := os.ReadFile(headNameFile); err == nil {
			headName := strings.TrimSpace(string(data))
			state.SourceBranch = strings.TrimPrefix(headName, "refs/heads/")
		}
	} else if _, err := os.Stat(rebaseApplyPath); err == nil {
		state.Type = ConflictRebase
		state.InProgress = true

		// For rebase-apply, try to get branch from head-name
		headNameFile := filepath.Join(rebaseApplyPath, "head-name")
		if data, err := os.ReadFile(headNameFile); err == nil {
			headName := strings.TrimSpace(string(data))
			state.SourceBranch = strings.TrimPrefix(headName, "refs/heads/")
		}
	}

	// Get conflicted files
	if state.InProgress {
		files, err := r.ConflictFiles()
		if err == nil {
			state.Files = make([]string, len(files))
			for i, f := range files {
				state.Files[i] = f.Path
			}
		}
	}

	return state, nil
}

// ParseConflictFile parses a file with conflict markers into structured regions.
// Returns conflict regions, full file lines, and any error.
func (r *Repository) ParseConflictFile(path string) ([]*ConflictRegion, []string, error) {
	fullPath := filepath.Join(r.path, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, err
	}

	// Check for binary content
	if isBinaryContent(content) {
		return nil, nil, fmt.Errorf("cannot parse binary file")
	}

	lines := strings.Split(string(content), "\n")
	var regions []*ConflictRegion
	var currentRegion *ConflictRegion
	var state int // 0=normal, 1=ours, 2=base, 3=theirs

	for i, line := range lines {
		// Detect conflict markers
		if strings.HasPrefix(line, "<<<<<<<") {
			// Start of conflict region
			currentRegion = &ConflictRegion{
				StartLine: i,
				OursLines: []string{},
				BaseLines: []string{},
				TheirsLines: []string{},
			}
			// Extract label (e.g., "<<<<<<< HEAD")
			parts := strings.SplitN(line, " ", 2)
			if len(parts) > 1 {
				currentRegion.OursLabel = parts[1]
			} else {
				currentRegion.OursLabel = "HEAD"
			}
			state = 1 // Now reading "ours" section
			continue
		}

		if strings.HasPrefix(line, "|||||||") {
			// Start of base section (diff3 style)
			if currentRegion != nil {
				currentRegion.HasBase = true
				state = 2 // Now reading "base" section
			}
			continue
		}

		if strings.HasPrefix(line, "=======") {
			// Start of theirs section
			if currentRegion != nil {
				state = 3 // Now reading "theirs" section
			}
			continue
		}

		if strings.HasPrefix(line, ">>>>>>>") {
			// End of conflict region
			if currentRegion != nil {
				currentRegion.EndLine = i
				// Extract label (e.g., ">>>>>>> feature-branch")
				parts := strings.SplitN(line, " ", 2)
				if len(parts) > 1 {
					currentRegion.TheirsLabel = parts[1]
				} else {
					currentRegion.TheirsLabel = "incoming"
				}
				regions = append(regions, currentRegion)
				currentRegion = nil
				state = 0
			}
			continue
		}

		// Collect lines based on current state
		if currentRegion != nil {
			switch state {
			case 1: // Ours
				currentRegion.OursLines = append(currentRegion.OursLines, line)
			case 2: // Base
				currentRegion.BaseLines = append(currentRegion.BaseLines, line)
			case 3: // Theirs
				currentRegion.TheirsLines = append(currentRegion.TheirsLines, line)
			}
		}
	}

	return regions, lines, nil
}

// WriteResolvedFile writes the resolved content to a file.
func (r *Repository) WriteResolvedFile(path string, lines []string) error {
	fullPath := filepath.Join(r.path, path)
	content := strings.Join(lines, "\n")

	// Write to temp file first for atomic operation
	tmpFile := fullPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, fullPath)
}

// StageResolvedFile stages a resolved conflict file.
func (r *Repository) StageResolvedFile(path string) error {
	_, err := r.run("add", path)
	return err
}

// MergeContinue continues a merge after resolving conflicts.
func (r *Repository) MergeContinue() error {
	_, err := r.run("merge", "--continue")
	return err
}

// MergeAbort aborts a merge in progress.
func (r *Repository) MergeAbort() error {
	_, err := r.run("merge", "--abort")
	return err
}

// RebaseContinue continues a rebase after resolving conflicts.
func (r *Repository) RebaseContinue() error {
	_, err := r.run("rebase", "--continue")
	return err
}

// RebaseAbort aborts a rebase in progress.
func (r *Repository) RebaseAbort() error {
	_, err := r.run("rebase", "--abort")
	return err
}

// IsConflictError checks if an error message indicates a conflict.
func IsConflictError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "conflict") ||
		strings.Contains(msg, "merge conflict") ||
		strings.Contains(msg, "needs merge")
}

// isBinaryContent checks if content appears to be binary.
func isBinaryContent(content []byte) bool {
	checkLen := len(content)
	if checkLen > 8000 {
		checkLen = 8000
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}
