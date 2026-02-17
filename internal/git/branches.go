package git

import (
	"strings"
)

// Branch represents a git branch.
type Branch struct {
	Name       string
	IsRemote   bool
	IsCurrent  bool
	IsTracking bool
	Upstream   string
	Ahead      int
	Behind     int
	LastCommit string // Short hash of last commit
	LastMsg    string // Subject of last commit
}

// ListBranches returns all branches (local and remote).
func (r *Repository) ListBranches() ([]Branch, error) {
	// Get local branches with tracking info
	// Format: refname:short|upstream:short|push:track|objectname:short|subject|HEAD
	format := "%(refname:short)|%(upstream:short)|%(upstream:track)|%(objectname:short)|%(subject)|%(HEAD)"
	out, err := r.run("for-each-ref", "--format="+format, "refs/heads/")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 5 {
			continue
		}

		branch := Branch{
			Name:       parts[0],
			Upstream:   parts[1],
			IsTracking: parts[1] != "",
			LastCommit: parts[3],
			LastMsg:    parts[4],
			IsCurrent:  len(parts) > 5 && parts[5] == "*",
		}

		// Parse tracking info [ahead N, behind M]
		if parts[2] != "" {
			parseTrackingInfo(parts[2], &branch)
		}

		branches = append(branches, branch)
	}

	// Get remote branches
	remoteOut, err := r.run("for-each-ref", "--format=%(refname:short)|%(objectname:short)|%(subject)", "refs/remotes/")
	if err == nil {
		for _, line := range strings.Split(remoteOut, "\n") {
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, "|", 3)
			if len(parts) < 3 {
				continue
			}

			// Skip HEAD pointer
			if strings.HasSuffix(parts[0], "/HEAD") {
				continue
			}

			branch := Branch{
				Name:       parts[0],
				IsRemote:   true,
				LastCommit: parts[1],
				LastMsg:    parts[2],
			}
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

// ListLocalBranches returns only local branches.
func (r *Repository) ListLocalBranches() ([]Branch, error) {
	format := "%(refname:short)|%(upstream:short)|%(upstream:track)|%(objectname:short)|%(subject)|%(HEAD)"
	out, err := r.run("for-each-ref", "--format="+format, "refs/heads/")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 5 {
			continue
		}

		branch := Branch{
			Name:       parts[0],
			Upstream:   parts[1],
			IsTracking: parts[1] != "",
			LastCommit: parts[3],
			LastMsg:    parts[4],
			IsCurrent:  len(parts) > 5 && parts[5] == "*",
		}

		if parts[2] != "" {
			parseTrackingInfo(parts[2], &branch)
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// ListRemoteBranches returns only remote branches.
func (r *Repository) ListRemoteBranches() ([]Branch, error) {
	out, err := r.run("for-each-ref", "--format=%(refname:short)|%(objectname:short)|%(subject)", "refs/remotes/")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		// Skip HEAD pointer
		if strings.HasSuffix(parts[0], "/HEAD") {
			continue
		}

		branch := Branch{
			Name:       parts[0],
			IsRemote:   true,
			LastCommit: parts[1],
			LastMsg:    parts[2],
		}
		branches = append(branches, branch)
	}

	return branches, nil
}

// CreateBranch creates a new branch.
func (r *Repository) CreateBranch(name string) error {
	_, err := r.run("branch", name)
	return err
}

// CreateBranchAt creates a new branch at a specific commit.
func (r *Repository) CreateBranchAt(name, ref string) error {
	_, err := r.run("branch", name, ref)
	return err
}

// DeleteBranch deletes a branch.
func (r *Repository) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.run("branch", flag, name)
	return err
}

// DeleteRemoteBranch deletes a remote branch.
func (r *Repository) DeleteRemoteBranch(remote, branch string) error {
	_, err := r.run("push", remote, "--delete", branch)
	return err
}

// RenameBranch renames a branch.
func (r *Repository) RenameBranch(oldName, newName string) error {
	_, err := r.run("branch", "-m", oldName, newName)
	return err
}

// SetUpstream sets the upstream tracking branch.
func (r *Repository) SetUpstream(local, remote string) error {
	_, err := r.run("branch", "-u", remote, local)
	return err
}

// MergeBranch merges a branch into the current branch.
func (r *Repository) MergeBranch(branch string) error {
	_, err := r.run("merge", branch)
	return err
}

// RebaseBranch rebases the current branch onto another.
func (r *Repository) RebaseBranch(onto string) error {
	_, err := r.run("rebase", onto)
	return err
}

// parseTrackingInfo parses "[ahead N, behind M]" format.
func parseTrackingInfo(info string, branch *Branch) {
	info = strings.Trim(info, "[]")
	parts := strings.Split(info, ", ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ahead ") {
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				n := 0
				for _, ch := range fields[1] {
					if ch >= '0' && ch <= '9' {
						n = n*10 + int(ch-'0')
					}
				}
				branch.Ahead = n
			}
		}
		if strings.HasPrefix(part, "behind ") {
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				n := 0
				for _, ch := range fields[1] {
					if ch >= '0' && ch <= '9' {
						n = n*10 + int(ch-'0')
					}
				}
				branch.Behind = n
			}
		}
	}
}

// Tag represents a git tag.
type Tag struct {
	Name    string
	Hash    string
	Message string
	Tagger  string
	IsAnnotated bool
}

// ListTags returns all tags.
func (r *Repository) ListTags() ([]Tag, error) {
	out, err := r.run("tag", "-l", "--format=%(refname:short)|%(objectname:short)|%(contents:subject)|%(taggername)|%(objecttype)")
	if err != nil {
		return nil, err
	}

	var tags []Tag
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 2 {
			continue
		}

		tag := Tag{
			Name: parts[0],
			Hash: parts[1],
		}
		if len(parts) > 2 {
			tag.Message = parts[2]
		}
		if len(parts) > 3 {
			tag.Tagger = parts[3]
		}
		if len(parts) > 4 {
			tag.IsAnnotated = parts[4] == "tag"
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

// CreateTag creates a new tag.
func (r *Repository) CreateTag(name, ref, message string) error {
	if message != "" {
		_, err := r.run("tag", "-a", name, ref, "-m", message)
		return err
	}
	_, err := r.run("tag", name, ref)
	return err
}

// DeleteTag deletes a tag.
func (r *Repository) DeleteTag(name string) error {
	_, err := r.run("tag", "-d", name)
	return err
}

// PushTag pushes a tag to a remote.
func (r *Repository) PushTag(remote, tag string) error {
	_, err := r.run("push", remote, tag)
	return err
}
