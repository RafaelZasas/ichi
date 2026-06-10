package git

import (
	"strings"
)

// StatusEntry represents a file in the working tree status.
type StatusEntry struct {
	Path        string
	OldPath     string     // For renames
	IndexStatus FileStatus // Status in index (staged)
	WorkStatus  FileStatus // Status in working tree (unstaged)
	IsStaged    bool
	IsUntracked bool
	IsConflict  bool
}

// Status returns the working tree status.
func (r *Repository) Status() ([]StatusEntry, error) {
	// Use porcelain v2 for machine-readable output
	out, err := r.run("status", "--porcelain=v2", "--untracked-files=all")
	if err != nil {
		// Fallback to v1 format
		return r.statusV1()
	}

	return parseStatusV2(out), nil
}

// StagedFiles returns only staged files.
func (r *Repository) StagedFiles() ([]StatusEntry, error) {
	entries, err := r.Status()
	if err != nil {
		return nil, err
	}

	var staged []StatusEntry
	for _, entry := range entries {
		if entry.IsStaged {
			staged = append(staged, entry)
		}
	}
	return staged, nil
}

// UnstagedFiles returns only unstaged files (modified but not staged).
func (r *Repository) UnstagedFiles() ([]StatusEntry, error) {
	entries, err := r.Status()
	if err != nil {
		return nil, err
	}

	var unstaged []StatusEntry
	for _, entry := range entries {
		if !entry.IsStaged && !entry.IsUntracked {
			unstaged = append(unstaged, entry)
		}
	}
	return unstaged, nil
}

// UntrackedFiles returns only untracked files.
func (r *Repository) UntrackedFiles() ([]StatusEntry, error) {
	entries, err := r.Status()
	if err != nil {
		return nil, err
	}

	var untracked []StatusEntry
	for _, entry := range entries {
		if entry.IsUntracked {
			untracked = append(untracked, entry)
		}
	}
	return untracked, nil
}

// ConflictFiles returns files with merge conflicts.
func (r *Repository) ConflictFiles() ([]StatusEntry, error) {
	entries, err := r.Status()
	if err != nil {
		return nil, err
	}

	var conflicts []StatusEntry
	for _, entry := range entries {
		if entry.IsConflict {
			conflicts = append(conflicts, entry)
		}
	}
	return conflicts, nil
}

// parseStatusV2 parses git status --porcelain=v2 output.
func parseStatusV2(output string) []StatusEntry {
	// Pre-size to line count; SplitSeq avoids Split's intermediate slice alloc.
	entries := make([]StatusEntry, 0, strings.Count(output, "\n")+1)

	for line := range strings.SplitSeq(output, "\n") {
		if line == "" {
			continue
		}

		// Ordinary changed entries: 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
		// Renamed/copied entries: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <X><score> <path><tab><origPath>
		// Unmerged entries: u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
		// Untracked: ? <path>
		// Ignored: ! <path>

		if strings.HasPrefix(line, "? ") {
			entries = append(entries, StatusEntry{
				Path:        line[2:],
				WorkStatus:  FileUntracked,
				IsUntracked: true,
			})
			continue
		}

		if strings.HasPrefix(line, "! ") {
			entries = append(entries, StatusEntry{
				Path:       line[2:],
				WorkStatus: FileIgnored,
			})
			continue
		}

		if strings.HasPrefix(line, "u ") {
			// Path follows the 10th space (avoids a strings.Fields slice alloc).
			if path := fieldAfter(line, 10); path != "" {
				entries = append(entries, StatusEntry{
					Path:        path,
					IndexStatus: FileConflict,
					WorkStatus:  FileConflict,
					IsConflict:  true,
				})
			}
			continue
		}

		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			// XY status is the fixed 2-byte field at offset 2.
			if len(line) < 4 {
				continue
			}
			entry := StatusEntry{
				IndexStatus: parseStatusChar(line[2]),
				WorkStatus:  parseStatusChar(line[3]),
			}

			if line[0] == '2' {
				// Rename/copy - path is after the score field
				// Format: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <X><score> <path><tab><origPath>
				tabIdx := strings.Index(line, "\t")
				if tabIdx > 0 {
					entry.Path = line[strings.LastIndex(line[:tabIdx], " ")+1 : tabIdx]
					entry.OldPath = line[tabIdx+1:]
				} else if p := fieldAfter(line, 9); p != "" {
					entry.Path = p
				}
			} else {
				// Path is everything after the 8th space.
				entry.Path = fieldAfter(line, 8)
			}
			if entry.Path == "" {
				continue
			}

			// Determine if staged
			entry.IsStaged = entry.IndexStatus != FileUnchanged && entry.IndexStatus != FileUntracked

			entries = append(entries, entry)
		}
	}

	return entries
}

// fieldAfter returns the remainder of line after the nth single space (1-indexed),
// or "" if there are fewer than n. Slices instead of allocating via strings.Fields.
func fieldAfter(line string, n int) string {
	idx := 0
	for ; n > 0; n-- {
		sp := strings.IndexByte(line[idx:], ' ')
		if sp < 0 {
			return ""
		}
		idx += sp + 1
	}
	return line[idx:]
}

// statusV1 is a fallback using the simpler porcelain v1 format.
func (r *Repository) statusV1() ([]StatusEntry, error) {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	var entries []StatusEntry

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}

		xy := line[:2]
		path := line[3:]

		entry := StatusEntry{
			Path:        path,
			IndexStatus: parseStatusChar(xy[0]),
			WorkStatus:  parseStatusChar(xy[1]),
		}

		// Handle renames (indicated by ->)
		if idx := strings.Index(path, " -> "); idx > 0 {
			entry.OldPath = path[:idx]
			entry.Path = path[idx+4:]
		}

		// Determine special states
		entry.IsStaged = xy[0] != ' ' && xy[0] != '?'
		entry.IsUntracked = xy[0] == '?' && xy[1] == '?'
		entry.IsConflict = xy[0] == 'U' || xy[1] == 'U' ||
			(xy[0] == 'A' && xy[1] == 'A') ||
			(xy[0] == 'D' && xy[1] == 'D')

		entries = append(entries, entry)
	}

	return entries, nil
}

// parseStatusChar converts a status character to FileStatus.
func parseStatusChar(c byte) FileStatus {
	switch c {
	case 'M':
		return FileModified
	case 'A':
		return FileAdded
	case 'D':
		return FileDeleted
	case 'R':
		return FileRenamed
	case 'C':
		return FileCopied
	case '?':
		return FileUntracked
	case '!':
		return FileIgnored
	case 'U':
		return FileConflict
	default:
		return FileUnchanged
	}
}
