package git

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atterpac/jig/components"
)

// LoadGraph loads the commit graph for visualization.
func (r *Repository) LoadGraph(limit int) (*components.GitGraphData, error) {
	headHash := r.HEAD()

	// Format: hash|short_hash|subject|author|timestamp|parents|refs
	format := "%H|%h|%s|%an|%at|%P|%D"

	// Use git log with date order for chronological display
	// HEAD first ensures current branch gets priority in layout
	// Exclude stash refs - they are loaded separately when toggled
	out, err := r.run("log", "HEAD", "--exclude=refs/stash", "--all", "--date-order",
		"--max-count="+strconv.Itoa(limit),
		"--format="+format)
	if err != nil {
		return nil, err
	}

	graph := components.NewGitGraphData()

	// Set current branch so it gets priority (column 0) in layout
	graph.CurrentBranch = r.CurrentBranch()

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 6 {
			continue
		}

		timestamp, _ := strconv.ParseInt(parts[4], 10, 64)
		parents := strings.Fields(parts[5])

		var refs []string
		if len(parts) > 6 && parts[6] != "" {
			refs = parseRefs(parts[6])
		}

		// Determine branch name from refs
		branch := ""
		for _, ref := range refs {
			if !strings.HasPrefix(ref, "origin/") && ref != "HEAD" {
				branch = ref
				break
			}
		}

		commit := &components.GitCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      time.Unix(timestamp, 0),
			Parents:   parents,
			IsMerge:   len(parents) > 1,
			Refs:      refs,
			Branch:    branch,
		}

		// Mark if this is the HEAD commit
		if parts[0] == headHash {
			hasHead := false
			for _, r := range commit.Refs {
				if r == "HEAD" {
					hasHead = true
					break
				}
			}
			if !hasHead {
				commit.Refs = append([]string{"HEAD"}, commit.Refs...)
			}
		}

		graph.AddCommit(commit)
	}

	// Populate ahead/behind counts for branch tips
	r.populateAheadBehind(graph)

	graph.LayoutGraph()
	return graph, nil
}

// populateAheadBehind fetches ahead/behind counts for commits with local branches.
func (r *Repository) populateAheadBehind(graph *components.GitGraphData) {
	type branchQuery struct {
		commit      *components.GitCommit
		localBranch string
		hasRemote   bool
	}

	var queries []branchQuery
	for _, commit := range graph.Commits {
		if len(commit.Refs) == 0 {
			continue
		}

		var localBranch string
		hasRemote := false
		for _, ref := range commit.Refs {
			if strings.HasPrefix(ref, "origin/") {
				hasRemote = true
			} else if ref != "HEAD" && localBranch == "" {
				localBranch = ref
			}
		}

		if localBranch == "" {
			continue
		}

		queries = append(queries, branchQuery{commit, localBranch, hasRemote})
	}

	if len(queries) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(queries))
	for _, q := range queries {
		go func(q branchQuery) {
			defer wg.Done()
			upstream := "origin/" + q.localBranch
			out, err := r.run("rev-list", "--left-right", "--count", q.localBranch+"..."+upstream)
			if err != nil {
				if q.hasRemote {
					q.commit.Ahead = 0
					q.commit.Behind = 0
				}
				return
			}
			parts := strings.Fields(strings.TrimSpace(out))
			if len(parts) >= 2 {
				q.commit.Ahead, _ = strconv.Atoi(parts[0])
				q.commit.Behind, _ = strconv.Atoi(parts[1])
			}
		}(q)
	}
	wg.Wait()
}

// LoadCommit loads detailed information about a single commit.
func (r *Repository) LoadCommit(hash string) (*CommitDetail, error) {
	// Get commit info with GPG signature
	format := "%H|%h|%s|%B|%an|%ae|%at|%cn|%ce|%ct|%P|%D|%G?|%GK|%GS"
	out, err := r.run("show", "-s", "--format="+format, hash)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(strings.TrimSpace(out), "|", 15)
	if len(parts) < 10 {
		return nil, err
	}

	authorTime, _ := strconv.ParseInt(parts[6], 10, 64)
	committerTime, _ := strconv.ParseInt(parts[9], 10, 64)
	parents := strings.Fields(parts[10])

	var refs []string
	if len(parts) > 11 && parts[11] != "" {
		refs = parseRefs(parts[11])
	}

	// Parse GPG signature status
	gpgStatus := GPGSignature{}
	if len(parts) > 12 && parts[12] != "" {
		switch parts[12] {
		case "G":
			gpgStatus.Signed = true
			gpgStatus.Valid = true
			gpgStatus.TrustLevel = "good"
		case "B":
			gpgStatus.Signed = true
			gpgStatus.Valid = false
			gpgStatus.TrustLevel = "bad"
		case "U":
			gpgStatus.Signed = true
			gpgStatus.Valid = true
			gpgStatus.TrustLevel = "unknown"
		case "X":
			gpgStatus.Signed = true
			gpgStatus.Valid = true
			gpgStatus.TrustLevel = "expired"
		case "Y":
			gpgStatus.Signed = true
			gpgStatus.Valid = true
			gpgStatus.TrustLevel = "expired_key"
		case "R":
			gpgStatus.Signed = true
			gpgStatus.Valid = false
			gpgStatus.TrustLevel = "revoked"
		case "E":
			gpgStatus.Signed = true
			gpgStatus.Valid = false
			gpgStatus.TrustLevel = "missing_key"
		case "N":
			gpgStatus.Signed = false
		}
		if len(parts) > 13 {
			gpgStatus.KeyID = parts[13]
		}
		if len(parts) > 14 {
			gpgStatus.Signer = parts[14]
		}
	}

	// Run independent git commands in parallel
	var (
		stats          CommitStats
		files          []ChangedFile
		parentSubjects []string
		branches       []string
		wg             sync.WaitGroup
	)

	wg.Add(3)

	// Stats + files + numstat (sequential since files feeds into numstat)
	go func() {
		defer wg.Done()
		// Get changed files with name-status and numstat in parallel
		var filesOut, numstatOut, statsOut string
		var innerWg sync.WaitGroup
		innerWg.Add(3)
		go func() { defer innerWg.Done(); statsOut, _ = r.run("show", "-m", "--first-parent", "--stat", "--format=", hash) }()
		go func() { defer innerWg.Done(); filesOut, _ = r.run("show", "-m", "--first-parent", "--name-status", "--format=", hash) }()
		go func() { defer innerWg.Done(); numstatOut, _ = r.run("show", "-m", "--first-parent", "--numstat", "--format=", hash) }()
		innerWg.Wait()
		stats = parseStats(statsOut)
		files = parseChangedFiles(filesOut)
		parseFileNumstat(numstatOut, files)
	}()

	// Parent subjects (all in parallel)
	go func() {
		defer wg.Done()
		parentSubjects = make([]string, len(parents))
		if len(parents) == 0 {
			return
		}
		var pWg sync.WaitGroup
		pWg.Add(len(parents))
		for i, parentHash := range parents {
			go func(idx int, ph string) {
				defer pWg.Done()
				if out, err := r.run("log", "-1", "--format=%s", ph); err == nil {
					parentSubjects[idx] = strings.TrimSpace(out)
				}
			}(i, parentHash)
		}
		pWg.Wait()
	}()

	// Branches containing this commit
	go func() {
		defer wg.Done()
		branchesOut, _ := r.run("branch", "-a", "--contains", hash)
		branches = parseBranchList(branchesOut)
	}()

	wg.Wait()

	return &CommitDetail{
		Hash:           parts[0],
		ShortHash:      parts[1],
		Subject:        parts[2],
		Body:           parts[3],
		Author:         parts[4],
		AuthorEmail:    parts[5],
		AuthorDate:     time.Unix(authorTime, 0),
		Committer:      parts[7],
		CommitterEmail: parts[8],
		CommitterDate:  time.Unix(committerTime, 0),
		Parents:        parents,
		ParentSubjects: parentSubjects,
		Refs:           refs,
		Branches:       branches,
		Stats:          stats,
		Files:          files,
		GPGStatus:      gpgStatus,
	}, nil
}

// CommitDetail contains detailed commit information.
type CommitDetail struct {
	Hash           string
	ShortHash      string
	Subject        string
	Body           string
	Author         string
	AuthorEmail    string
	AuthorDate     time.Time
	Committer      string
	CommitterEmail string
	CommitterDate  time.Time
	Parents        []string
	ParentSubjects []string // Subject lines of parent commits
	Refs           []string
	Branches       []string // Branches containing this commit
	Stats          CommitStats
	Files          []ChangedFile
	GPGStatus      GPGSignature
}

// GPGSignature contains GPG signature verification info.
type GPGSignature struct {
	Signed   bool
	Valid    bool
	KeyID    string
	Signer   string
	TrustLevel string
}

// CommitStats contains commit statistics.
type CommitStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

// ChangedFile represents a file changed in a commit.
type ChangedFile struct {
	Status     FileStatus
	Path       string
	OldPath    string // For renames
	Binary     bool
	Insertions int
	Deletions  int
}

// SearchCommits searches commits by message, author, or hash.
func (r *Repository) SearchCommits(query string, limit int) ([]*components.GitCommit, error) {
	format := "%H|%h|%s|%an|%at|%P"
	out, err := r.run("log", "--all",
		"--max-count="+strconv.Itoa(limit),
		"--grep="+query,
		"--format="+format)
	if err != nil {
		// Try searching by hash
		out, err = r.run("log", "--all",
			"--max-count="+strconv.Itoa(limit),
			"--format="+format,
			query)
		if err != nil {
			return nil, err
		}
	}

	var commits []*components.GitCommit
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 5 {
			continue
		}

		timestamp, _ := strconv.ParseInt(parts[4], 10, 64)
		parents := strings.Fields(parts[5])

		commit := &components.GitCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Message:   parts[2],
			Author:    parts[3],
			Date:      time.Unix(timestamp, 0),
			Parents:   parents,
			IsMerge:   len(parents) > 1,
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// LoadStashes loads stash entries as GitCommits.
func (r *Repository) LoadStashes() ([]*components.GitCommit, error) {
	// Get all stash info in a single command
	out, err := r.run("stash", "list", "--format=%gd|%H|%s|%at|%an")
	if err != nil {
		return nil, nil // No stashes or stash not supported
	}

	var stashes []*components.GitCommit
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		// Get stash details
		stashRef := parts[0] // e.g., stash@{0}
		hash := parts[1]
		message := parts[2]

		// Get short hash
		shortHash := hash
		if len(hash) > 7 {
			shortHash = hash[:7]
		}

		// Get the stash timestamp
		timestampOut, _ := r.run("show", "-s", "--format=%at", stashRef)
		timestamp, _ := strconv.ParseInt(strings.TrimSpace(timestampOut), 10, 64)

		// Get author
		authorOut, _ := r.run("show", "-s", "--format=%an", stashRef)
		author := strings.TrimSpace(authorOut)

		stash := &components.GitCommit{
			Hash:      hash,
			ShortHash: shortHash,
			Message:   message,
			Author:    author,
			Date:      time.Unix(timestamp, 0),
			IsStash:   true,
			Refs:      []string{stashRef},
		}

		stashes = append(stashes, stash)
	}

	return stashes, nil
}

// parseRefs parses the ref string from git log.
func parseRefs(refStr string) []string {
	if refStr == "" {
		return nil
	}

	var refs []string
	for _, ref := range strings.Split(refStr, ", ") {
		ref = strings.TrimSpace(ref)
		ref = strings.TrimPrefix(ref, "HEAD -> ")
		ref = strings.TrimPrefix(ref, "tag: ")
		// Skip stash-related refs
		if ref != "" && !strings.Contains(ref, "stash") {
			refs = append(refs, ref)
		}
	}
	return refs
}

// parseStats parses git show --stat output.
func parseStats(output string) CommitStats {
	var stats CommitStats
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for summary line like "3 files changed, 10 insertions(+), 5 deletions(-)"
		if strings.Contains(line, "files changed") || strings.Contains(line, "file changed") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "file" || part == "files" {
					if i > 0 {
						stats.FilesChanged, _ = strconv.Atoi(parts[i-1])
					}
				}
				if strings.HasPrefix(part, "insertion") {
					if i > 0 {
						stats.Insertions, _ = strconv.Atoi(parts[i-1])
					}
				}
				if strings.HasPrefix(part, "deletion") {
					if i > 0 {
						stats.Deletions, _ = strconv.Atoi(parts[i-1])
					}
				}
			}
			break
		}
	}
	return stats
}

// parseChangedFiles parses git show --name-status output.
func parseChangedFiles(output string) []ChangedFile {
	var files []ChangedFile
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		file := ChangedFile{
			Path: parts[1],
		}

		switch parts[0][0] {
		case 'A':
			file.Status = FileAdded
		case 'D':
			file.Status = FileDeleted
		case 'M':
			file.Status = FileModified
		case 'R':
			file.Status = FileRenamed
			if len(parts) >= 3 {
				file.OldPath = parts[1]
				file.Path = parts[2]
			}
		case 'C':
			file.Status = FileCopied
			if len(parts) >= 3 {
				file.OldPath = parts[1]
				file.Path = parts[2]
			}
		}

		files = append(files, file)
	}
	return files
}

// parseFileNumstat parses git show --numstat output and updates file stats.
func parseFileNumstat(output string, files []ChangedFile) {
	fileMap := make(map[string]*ChangedFile)
	for i := range files {
		fileMap[files[i].Path] = &files[i]
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Format: insertions deletions filepath
		// Binary files show "-" for both
		insertions, errIns := strconv.Atoi(parts[0])
		deletions, errDel := strconv.Atoi(parts[1])
		path := parts[2]

		// Handle renames which show as "old => new"
		if len(parts) > 3 {
			path = parts[len(parts)-1]
		}

		if file, ok := fileMap[path]; ok {
			if errIns == nil {
				file.Insertions = insertions
			} else {
				file.Binary = true
			}
			if errDel == nil {
				file.Deletions = deletions
			}
		}
	}
}

// parseBranchList parses git branch --contains output.
func parseBranchList(output string) []string {
	var branches []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove leading * for current branch
		line = strings.TrimPrefix(line, "* ")
		// Remove remotes/ prefix for cleaner display
		line = strings.TrimPrefix(line, "remotes/")

		// Skip HEAD references
		if strings.Contains(line, "HEAD") {
			continue
		}

		branches = append(branches, line)
	}
	return branches
}
