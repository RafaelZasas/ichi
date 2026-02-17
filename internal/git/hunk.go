package git

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// FileStatus represents the status of a file in the working tree.
type FileStatus int

const (
	FileUnchanged FileStatus = iota // 0 = no status / unchanged
	FileModified                    // 1
	FileAdded                       // 2
	FileDeleted                     // 3
	FileRenamed                     // 4
	FileCopied                      // 5
	FileUntracked                   // 6
	FileIgnored                     // 7
	FileConflict                    // 8
)

func (s FileStatus) String() string {
	switch s {
	case FileUnchanged:
		return " "
	case FileModified:
		return "M"
	case FileAdded:
		return "A"
	case FileDeleted:
		return "D"
	case FileRenamed:
		return "R"
	case FileCopied:
		return "C"
	case FileUntracked:
		return "?"
	case FileIgnored:
		return "!"
	case FileConflict:
		return "U"
	default:
		return " "
	}
}

// LineType represents the type of a diff line.
type LineType int

const (
	LineContext LineType = iota // Unchanged line (starts with space)
	LineAdded                   // Added line (starts with +)
	LineRemoved                 // Removed line (starts with -)
	LineHeader                  // Hunk header (@@ ... @@)
)

// DiffLine represents a single line in a diff.
type DiffLine struct {
	Type      LineType
	Content   string
	OldLineNo int  // Line number in old file (0 if added)
	NewLineNo int  // Line number in new file (0 if removed)
	Selected  bool // Whether this line is selected for staging
}

// DiffHunk represents a contiguous block of changes.
type DiffHunk struct {
	Header   string      // @@ -start,count +start,count @@ context
	OldStart int         // Starting line in old file
	OldCount int         // Number of lines from old file
	NewStart int         // Starting line in new file
	NewCount int         // Number of lines in new file
	Lines    []*DiffLine // Lines in this hunk
	Selected bool        // Entire hunk selected
	Expanded bool        // Whether hunk is expanded in view
}

// FileDiff represents all changes to a single file.
type FileDiff struct {
	Path     string      // File path
	OldPath  string      // Old path (for renames)
	Status   FileStatus  // Added, Modified, Deleted, Renamed
	Hunks    []*DiffHunk // All hunks in this file
	Binary   bool        // True if binary file
	NewFile  bool        // True if this is a new file
	Deleted  bool        // True if this file was deleted
}

// ParseDiff parses git diff output into structured hunks.
func ParseDiff(diffOutput string) ([]*FileDiff, error) {
	var files []*FileDiff
	var currentFile *FileDiff
	var currentHunk *DiffHunk

	lines := strings.Split(diffOutput, "\n")

	hunkHeaderRe := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file header
		if strings.HasPrefix(line, "diff --git") {
			if currentFile != nil {
				files = append(files, currentFile)
			}
			currentFile = &FileDiff{
				Hunks: make([]*DiffHunk, 0),
			}
			currentHunk = nil

			// Parse file paths from "diff --git a/path b/path"
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 4 {
				currentFile.OldPath = strings.TrimPrefix(parts[2], "a/")
				currentFile.Path = strings.TrimPrefix(parts[3], "b/")
			}
			continue
		}

		if currentFile == nil {
			continue
		}

		// File metadata
		if strings.HasPrefix(line, "new file mode") {
			currentFile.NewFile = true
			currentFile.Status = FileAdded
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			currentFile.Deleted = true
			currentFile.Status = FileDeleted
			continue
		}
		if strings.HasPrefix(line, "rename from ") {
			currentFile.OldPath = strings.TrimPrefix(line, "rename from ")
			currentFile.Status = FileRenamed
			continue
		}
		if strings.HasPrefix(line, "rename to ") {
			currentFile.Path = strings.TrimPrefix(line, "rename to ")
			continue
		}
		if strings.HasPrefix(line, "Binary files") {
			currentFile.Binary = true
			continue
		}
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			continue
		}

		// Hunk header
		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			currentHunk = &DiffHunk{
				Header:   line,
				Lines:    make([]*DiffLine, 0),
				Expanded: true,
			}

			currentHunk.OldStart, _ = strconv.Atoi(matches[1])
			if matches[2] != "" {
				currentHunk.OldCount, _ = strconv.Atoi(matches[2])
			} else {
				currentHunk.OldCount = 1
			}
			currentHunk.NewStart, _ = strconv.Atoi(matches[3])
			if matches[4] != "" {
				currentHunk.NewCount, _ = strconv.Atoi(matches[4])
			} else {
				currentHunk.NewCount = 1
			}

			currentFile.Hunks = append(currentFile.Hunks, currentHunk)

			// Set status if not already set
			if currentFile.Status == 0 && !currentFile.NewFile && !currentFile.Deleted {
				currentFile.Status = FileModified
			}
			continue
		}

		// Diff line content
		if currentHunk != nil && len(line) > 0 {
			diffLine := &DiffLine{
				Content: line[1:], // Remove prefix character
			}

			switch line[0] {
			case '+':
				diffLine.Type = LineAdded
			case '-':
				diffLine.Type = LineRemoved
			case ' ':
				diffLine.Type = LineContext
			default:
				// Skip non-diff lines (like "\ No newline at end of file")
				continue
			}

			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}

	// Don't forget the last file
	if currentFile != nil {
		files = append(files, currentFile)
	}

	// Compute line numbers for all hunks
	for _, file := range files {
		for _, hunk := range file.Hunks {
			computeLineNumbers(hunk)
		}
	}

	return files, nil
}

// computeLineNumbers assigns line numbers to each line in a hunk.
func computeLineNumbers(hunk *DiffHunk) {
	oldLine := hunk.OldStart
	newLine := hunk.NewStart

	for _, line := range hunk.Lines {
		switch line.Type {
		case LineContext:
			line.OldLineNo = oldLine
			line.NewLineNo = newLine
			oldLine++
			newLine++
		case LineAdded:
			line.OldLineNo = 0
			line.NewLineNo = newLine
			newLine++
		case LineRemoved:
			line.OldLineNo = oldLine
			line.NewLineNo = 0
			oldLine++
		}
	}
}

// GenerateHunkPatch generates a patch for a single hunk.
func GenerateHunkPatch(file string, hunk *DiffHunk) string {
	var sb strings.Builder

	// File header
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
	sb.WriteString(fmt.Sprintf("--- a/%s\n", file))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", file))
	sb.WriteString(hunk.Header + "\n")

	// Hunk content
	for _, line := range hunk.Lines {
		prefix := " "
		switch line.Type {
		case LineAdded:
			prefix = "+"
		case LineRemoved:
			prefix = "-"
		case LineContext:
			prefix = " "
		}
		sb.WriteString(prefix + line.Content + "\n")
	}

	// Debug: write patch to temp file
	_ = os.WriteFile("/tmp/gxt_debug_patch.txt", []byte(sb.String()), 0644)

	return sb.String()
}

// GenerateLinesPatch generates a patch for selected lines within a hunk.
func GenerateLinesPatch(file string, hunk *DiffHunk, selectedLines []*DiffLine) string {
	if len(selectedLines) == 0 {
		return ""
	}

	// Create a set of selected lines for quick lookup
	selected := make(map[*DiffLine]bool)
	for _, l := range selectedLines {
		selected[l] = true
	}

	// Count additions and deletions in selected lines
	var additions, deletions int
	for _, line := range selectedLines {
		if line.Type == LineAdded {
			additions++
		} else if line.Type == LineRemoved {
			deletions++
		}
	}

	var sb strings.Builder

	// File header
	sb.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file, file))
	sb.WriteString(fmt.Sprintf("--- a/%s\n", file))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", file))

	// Modified hunk header
	newOldCount := hunk.OldCount - deletions
	newNewCount := hunk.NewCount - additions
	sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
		hunk.OldStart, newOldCount, hunk.NewStart, newNewCount))

	// Output lines, converting non-selected +/- lines to context
	for _, line := range hunk.Lines {
		if selected[line] {
			// Keep the line as-is
			prefix := " "
			switch line.Type {
			case LineAdded:
				prefix = "+"
			case LineRemoved:
				prefix = "-"
			}
			sb.WriteString(prefix + line.Content + "\n")
		} else {
			// Convert to context if it's a change line
			switch line.Type {
			case LineContext:
				sb.WriteString(" " + line.Content + "\n")
			case LineAdded:
				// Skip unselected additions
			case LineRemoved:
				// Convert unselected removals to context
				sb.WriteString(" " + line.Content + "\n")
			}
		}
	}

	return sb.String()
}

// StageHunk stages a single hunk using git apply.
func (r *Repository) StageHunk(file string, hunk *DiffHunk) error {
	patch := GenerateHunkPatch(file, hunk)
	return r.RunWithStdin(patch, "apply", "--cached", "-")
}

// StageLines stages specific lines within a hunk.
func (r *Repository) StageLines(file string, hunk *DiffHunk, lines []*DiffLine) error {
	patch := GenerateLinesPatch(file, hunk, lines)
	if patch == "" {
		return nil
	}
	return r.RunWithStdin(patch, "apply", "--cached", "-")
}

// UnstageHunk unstages a single hunk.
func (r *Repository) UnstageHunk(file string, hunk *DiffHunk) error {
	patch := GenerateHunkPatch(file, hunk)
	return r.RunWithStdin(patch, "apply", "--cached", "--reverse", "-")
}

// UnstageLines unstages specific lines within a hunk.
func (r *Repository) UnstageLines(file string, hunk *DiffHunk, lines []*DiffLine) error {
	patch := GenerateLinesPatch(file, hunk, lines)
	if patch == "" {
		return nil
	}
	return r.RunWithStdin(patch, "apply", "--cached", "--reverse", "-")
}

// DiscardHunk discards changes in a single hunk.
func (r *Repository) DiscardHunk(file string, hunk *DiffHunk) error {
	patch := GenerateHunkPatch(file, hunk)
	return r.RunWithStdin(patch, "apply", "--reverse", "-")
}
