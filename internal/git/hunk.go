package git

import (
	"strconv"
	"strings"
)

// parseHunkHeader parses "@@ -old[,n] +new[,n] @@"; ok=false if not a hunk
// header. Hand-rolled (no regex) to avoid a per-call []string allocation.
func parseHunkHeader(line string) (oldStart, oldCount, newStart, newCount int, ok bool) {
	// "@@ -"
	if len(line) < 4 || line[0] != '@' || line[1] != '@' || line[2] != ' ' || line[3] != '-' {
		return 0, 0, 0, 0, false
	}
	i := 4

	// oldStart[,oldCount]
	oldStart, i, ok = readInt(line, i)
	if !ok {
		return 0, 0, 0, 0, false
	}
	oldCount = 1
	if i < len(line) && line[i] == ',' {
		oldCount, i, ok = readInt(line, i+1)
		if !ok {
			return 0, 0, 0, 0, false
		}
	}

	// " +"
	if i+1 >= len(line) || line[i] != ' ' || line[i+1] != '+' {
		return 0, 0, 0, 0, false
	}
	i += 2

	// newStart[,newCount]
	newStart, i, ok = readInt(line, i)
	if !ok {
		return 0, 0, 0, 0, false
	}
	newCount = 1
	if i < len(line) && line[i] == ',' {
		newCount, i, ok = readInt(line, i+1)
		if !ok {
			return 0, 0, 0, 0, false
		}
	}

	// " @@"
	if i+2 >= len(line) || line[i] != ' ' || line[i+1] != '@' || line[i+2] != '@' {
		return 0, 0, 0, 0, false
	}
	return oldStart, oldCount, newStart, newCount, true
}

// readInt reads ASCII digits from s at i, returning the value and next index; ok=false if none.
func readInt(s string, i int) (val, next int, ok bool) {
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		val = val*10 + int(s[i]-'0')
		i++
	}
	return val, i, i > start
}

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
	Path    string      // File path
	OldPath string      // Old path (for renames)
	Status  FileStatus  // Added, Modified, Deleted, Renamed
	Hunks   []*DiffHunk // All hunks in this file
	Binary  bool        // True if binary file
	NewFile bool        // True if this is a new file
	Deleted bool        // True if this file was deleted
}

// ParseDiff parses git diff output into structured hunks.
func ParseDiff(diffOutput string) ([]*FileDiff, error) {
	var files []*FileDiff
	var currentFile *FileDiff
	var currentHunk *DiffHunk

	lines := strings.Split(diffOutput, "\n")

	// Arena: hand out *DiffLine from reusable 256-blocks (~1 alloc per 256 lines).
	var arena []DiffLine
	newLine := func() *DiffLine {
		if len(arena) == cap(arena) {
			arena = make([]DiffLine, 0, 256)
		}
		arena = append(arena, DiffLine{})
		return &arena[len(arena)-1]
	}

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
		if oldStart, oldCount, newStart, newCount, ok := parseHunkHeader(line); ok {
			currentHunk = &DiffHunk{
				Header:   line,
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				// Pre-size from header counts so the append loop never regrows.
				Lines:    make([]*DiffLine, 0, oldCount+newCount),
				Expanded: true,
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
			diffLine := newLine()
			diffLine.Content = line[1:] // Remove prefix character

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

// writeFileHeader writes the "diff --git / --- / +++" patch preamble.
func writeFileHeader(sb *strings.Builder, file string) {
	sb.WriteString("diff --git a/")
	sb.WriteString(file)
	sb.WriteString(" b/")
	sb.WriteString(file)
	sb.WriteString("\n--- a/")
	sb.WriteString(file)
	sb.WriteString("\n+++ b/")
	sb.WriteString(file)
	sb.WriteByte('\n')
}

// linePrefix returns the unified-diff prefix byte for a line type.
func linePrefix(t LineType) byte {
	switch t {
	case LineAdded:
		return '+'
	case LineRemoved:
		return '-'
	default:
		return ' '
	}
}

// GenerateHunkPatch generates a patch for a single hunk.
func GenerateHunkPatch(file string, hunk *DiffHunk) string {
	var sb strings.Builder

	// File header
	writeFileHeader(&sb, file)
	sb.WriteString(hunk.Header)
	sb.WriteByte('\n')

	// Hunk content
	for _, line := range hunk.Lines {
		sb.WriteByte(linePrefix(line.Type))
		sb.WriteString(line.Content)
		sb.WriteByte('\n')
	}

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
	writeFileHeader(&sb, file)

	// Modified hunk header
	newOldCount := hunk.OldCount - deletions
	newNewCount := hunk.NewCount - additions
	sb.WriteString("@@ -")
	sb.WriteString(strconv.Itoa(hunk.OldStart))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(newOldCount))
	sb.WriteString(" +")
	sb.WriteString(strconv.Itoa(hunk.NewStart))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(newNewCount))
	sb.WriteString(" @@\n")

	// Output lines, converting non-selected +/- lines to context
	for _, line := range hunk.Lines {
		if selected[line] {
			// Keep the line as-is
			sb.WriteByte(linePrefix(line.Type))
			sb.WriteString(line.Content)
			sb.WriteByte('\n')
		} else {
			// Convert change lines to context; skip unselected additions.
			switch line.Type {
			case LineContext, LineRemoved:
				sb.WriteByte(' ')
				sb.WriteString(line.Content)
				sb.WriteByte('\n')
			case LineAdded:
				// Skip unselected additions
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
