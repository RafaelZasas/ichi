package git

import (
	"strconv"
	"strings"
)

// GetCommitDiff returns the diff for a specific commit.
func (r *Repository) GetCommitDiff(hash string) (string, error) {
	// Use -m --first-parent for merge commits (like stashes)
	out, err := r.run("show", "-m", "--first-parent", "--format=", "--patch", hash)
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetFileDiff returns the diff for a specific file in a commit.
func (r *Repository) GetFileDiff(hash, file string) (string, error) {
	// Use -m --first-parent for merge commits (like stashes)
	out, err := r.run("show", "-m", "--first-parent", "--format=", "--patch", hash, "--", file)
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetWorkingDiff returns the diff of unstaged changes.
func (r *Repository) GetWorkingDiff() (string, error) {
	out, err := r.run("diff")
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetWorkingFileDiff returns the diff of unstaged changes for a specific file.
func (r *Repository) GetWorkingFileDiff(file string) (string, error) {
	out, err := r.run("diff", "--", file)
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetStagedDiff returns the diff of staged changes.
func (r *Repository) GetStagedDiff() (string, error) {
	out, err := r.run("diff", "--cached")
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetStagedFileDiff returns the diff of staged changes for a specific file.
func (r *Repository) GetStagedFileDiff(file string) (string, error) {
	out, err := r.run("diff", "--cached", "--", file)
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetDiffBetween returns the diff between two refs.
func (r *Repository) GetDiffBetween(from, to string) (string, error) {
	out, err := r.run("diff", from+".."+to)
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetDiffStats returns diff statistics between two refs.
func (r *Repository) GetDiffStats(from, to string) (string, error) {
	out, err := r.run("diff", "--stat", from+".."+to)
	if err != nil {
		return "", err
	}
	return out, nil
}

// FileContent returns the content of a file at a specific ref.
func (r *Repository) FileContent(ref, file string) (string, error) {
	out, err := r.run("show", ref+":"+file)
	if err != nil {
		return "", err
	}
	return out, nil
}

// WorkingFileContent returns the content of a file in the working directory.
func (r *Repository) WorkingFileContent(file string) (string, error) {
	out, err := r.run("show", ":"+file)
	if err != nil {
		// File might not be staged, try reading from working tree
		out, err = r.run("show", "HEAD:"+file)
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

// Blame returns blame information for a file.
func (r *Repository) Blame(file string) ([]BlameLine, error) {
	out, err := r.run("blame", "--porcelain", file)
	if err != nil {
		return nil, err
	}

	return parseBlame(out), nil
}

// BlameAtCommit returns blame information for a file at a specific commit.
func (r *Repository) BlameAtCommit(file, hash string) ([]BlameLine, error) {
	out, err := r.run("blame", "--porcelain", hash, "--", file)
	if err != nil {
		return nil, err
	}

	return parseBlame(out), nil
}

// BlameLine represents a single line from git blame.
type BlameLine struct {
	Hash       string // Full commit hash
	ShortHash  string // Short commit hash (7 chars)
	Author     string
	AuthorMail string
	Date       string // Unix timestamp as string
	LineNumber int    // Line number in current file
	OrigLine   int    // Original line number in commit
	Content    string
}

// parseBlame parses git blame --porcelain output.
func parseBlame(output string) []BlameLine {
	var lines []BlameLine
	var currentLine BlameLine
	commitInfo := make(map[string]map[string]string) // cache commit info
	lineNum := 0

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		// Lines starting with tab are content
		if strings.HasPrefix(line, "\t") {
			currentLine.Content = line[1:]
			lineNum++
			if currentLine.LineNumber == 0 {
				currentLine.LineNumber = lineNum
			}
			lines = append(lines, currentLine)
			currentLine = BlameLine{}
			continue
		}

		// Parse header line (hash orig_line final_line [group_lines])
		if len(line) >= 40 && isHexChar(line[0]) && !strings.Contains(line[:40], " ") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				hash := parts[0]
				currentLine.Hash = hash
				currentLine.ShortHash = hash[:7]

				// Use cached info if available
				if info, ok := commitInfo[hash]; ok {
					currentLine.Author = info["author"]
					currentLine.AuthorMail = info["author-mail"]
					currentLine.Date = info["author-time"]
				}
			}
			if len(parts) >= 3 {
				currentLine.OrigLine, _ = strconv.Atoi(parts[1])
				currentLine.LineNumber, _ = strconv.Atoi(parts[2])
			}
			continue
		}

		// Parse metadata
		switch {
		case strings.HasPrefix(line, "author "):
			currentLine.Author = strings.TrimPrefix(line, "author ")
			if commitInfo[currentLine.Hash] == nil {
				commitInfo[currentLine.Hash] = make(map[string]string)
			}
			commitInfo[currentLine.Hash]["author"] = currentLine.Author
		case strings.HasPrefix(line, "author-mail "):
			currentLine.AuthorMail = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
			if commitInfo[currentLine.Hash] != nil {
				commitInfo[currentLine.Hash]["author-mail"] = currentLine.AuthorMail
			}
		case strings.HasPrefix(line, "author-time "):
			currentLine.Date = strings.TrimPrefix(line, "author-time ")
			if commitInfo[currentLine.Hash] != nil {
				commitInfo[currentLine.Hash]["author-time"] = currentLine.Date
			}
		}
	}

	return lines
}

// isHexChar checks if a rune is a hex character.
func isHexChar(r byte) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
