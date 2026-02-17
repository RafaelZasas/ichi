package remote

import (
	"fmt"
	"strings"
	"time"
)

// Provider abstracts GitHub, GitLab, Bitbucket, etc.
type Provider interface {
	// Identity
	Name() string

	// Pull/Merge Requests
	ListPRs(repo string, opts ListPRsOpts) ([]PullRequest, error)
	GetPR(repo string, id int) (*PullRequest, error)
	CreatePR(repo string, pr *CreatePRInput) (*PullRequest, error)
	UpdatePR(repo string, id int, pr *UpdatePRInput) error
	MergePR(repo string, id int, opts MergeOpts) error
	ClosePR(repo string, id int) error

	// Reviews
	ListReviews(repo string, prID int) ([]Review, error)
	SubmitReview(repo string, prID int, review *SubmitReviewInput) error

	// Comments
	ListComments(repo string, prID int) ([]Comment, error)
	AddComment(repo string, prID int, comment *CommentInput) (*Comment, error)
	AddInlineComment(repo string, prID int, comment *InlineCommentInput) (*Comment, error)
	ResolveComment(repo string, prID int, commentID int64) error

	// Diff
	GetPRDiff(repo string, prID int) (string, error)
	GetPRFiles(repo string, prID int) ([]ChangedFile, error)

	// CI/Status
	GetChecks(repo string, prID int) ([]Check, error)

	// Auth
	Authenticate() error
	CurrentUser() (*User, error)
}

// ListPRsOpts configures PR listing
type ListPRsOpts struct {
	State  PRState // Filter by state (empty = all open)
	Author string  // Filter by author
	Limit  int     // Max results (0 = default)
}

// CreatePRInput for creating a new PR
type CreatePRInput struct {
	Title     string
	Body      string
	Base      string // Target branch
	Head      string // Source branch
	Draft     bool
	Reviewers []string
	Labels    []string
}

// UpdatePRInput for updating a PR
type UpdatePRInput struct {
	Title     *string
	Body      *string
	State     *PRState
	Draft     *bool
	Reviewers []string
	Labels    []string
}

// MergeOpts configures merge behavior
type MergeOpts struct {
	Method        MergeMethod // merge, squash, rebase
	CommitTitle   string
	CommitMessage string
	DeleteBranch  bool
}

// MergeMethod defines how to merge
type MergeMethod string

const (
	MergeMerge  MergeMethod = "merge"
	MergeSquash MergeMethod = "squash"
	MergeRebase MergeMethod = "rebase"
)

// SubmitReviewInput for submitting a review
type SubmitReviewInput struct {
	State    ReviewState
	Body     string
	Comments []InlineCommentInput // Pending inline comments
}

// CommentInput for adding a general comment
type CommentInput struct {
	Body string
}

// InlineCommentInput for adding an inline comment on a diff
type InlineCommentInput struct {
	Path      string
	Line      int
	Side      DiffSide
	Body      string
	InReplyTo *int64
}

// Core types - provider-agnostic

// PullRequest represents a PR/MR
type PullRequest struct {
	ID          int64
	Number      int
	Title       string
	Body        string
	State       PRState
	Author      User
	BaseBranch  string
	HeadBranch  string
	HeadSHA     string
	Draft       bool
	Mergeable   *bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Labels      []string
	Reviewers   []User
	Checks      []Check
	URL         string
	Additions   int
	Deletions   int
	ChangedFiles int
}

// PRState represents the state of a PR
type PRState string

const (
	PROpen   PRState = "open"
	PRClosed PRState = "closed"
	PRMerged PRState = "merged"
	PRAll    PRState = "all"
)

// Review represents a review on a PR
type Review struct {
	ID        int64
	Author    User
	State     ReviewState
	Body      string
	CreatedAt time.Time
}

// ReviewState represents the state of a review
type ReviewState string

const (
	ReviewApproved         ReviewState = "APPROVED"
	ReviewChangesRequested ReviewState = "CHANGES_REQUESTED"
	ReviewCommented        ReviewState = "COMMENTED"
	ReviewPending          ReviewState = "PENDING"
	ReviewDismissed        ReviewState = "DISMISSED"
)

// Comment represents a comment on a PR
type Comment struct {
	ID        int64
	Author    User
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Resolved  bool
	// For inline comments
	Path       string
	Line       int
	OrigLine   int
	Side       DiffSide
	InReplyTo  *int64
	DiffHunk   string
	CommitID   string
}

// DiffSide indicates which side of the diff
type DiffSide string

const (
	SideLeft  DiffSide = "LEFT"
	SideRight DiffSide = "RIGHT"
)

// Check represents a CI check/status
type Check struct {
	Name       string
	Status     CheckStatus
	Conclusion CheckConclusion
	URL        string
	StartedAt  time.Time
	CompletedAt time.Time
}

// CheckStatus represents the status of a check
type CheckStatus string

const (
	CheckQueued     CheckStatus = "queued"
	CheckInProgress CheckStatus = "in_progress"
	CheckCompleted  CheckStatus = "completed"
)

// CheckConclusion represents the conclusion of a completed check
type CheckConclusion string

const (
	CheckSuccess   CheckConclusion = "success"
	CheckFailure   CheckConclusion = "failure"
	CheckNeutral   CheckConclusion = "neutral"
	CheckCancelled CheckConclusion = "cancelled"
	CheckTimedOut  CheckConclusion = "timed_out"
	CheckSkipped   CheckConclusion = "skipped"
)

// ChangedFile represents a file changed in a PR
type ChangedFile struct {
	Path      string
	Status    FileStatus
	Additions int
	Deletions int
	Patch     string
	PrevPath  string // For renames
}

// FileStatus represents the status of a changed file
type FileStatus string

const (
	FileAdded    FileStatus = "added"
	FileModified FileStatus = "modified"
	FileDeleted  FileStatus = "deleted"
	FileRenamed  FileStatus = "renamed"
	FileCopied   FileStatus = "copied"
)

// User represents a user
type User struct {
	ID        int64
	Login     string
	Name      string
	Email     string
	AvatarURL string
}

// Provider registry

var providers = make(map[string]func() Provider)

// Register adds a provider factory
func Register(name string, factory func() Provider) {
	providers[name] = factory
}

// Get returns a provider by name
func Get(name string) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return factory(), nil
}

// DetectFromURL auto-detects provider from git remote URL
func DetectFromURL(remoteURL string) (Provider, string, error) {
	// Extract repo path from URL
	repo := extractRepoPath(remoteURL)

	switch {
	case strings.Contains(remoteURL, "github.com"):
		p, err := Get("github")
		return p, repo, err
	case strings.Contains(remoteURL, "gitlab.com"):
		p, err := Get("gitlab")
		return p, repo, err
	case strings.Contains(remoteURL, "bitbucket.org"):
		p, err := Get("bitbucket")
		return p, repo, err
	default:
		return nil, "", fmt.Errorf("unable to detect provider from: %s", remoteURL)
	}
}

func extractRepoPath(url string) string {
	// Handle SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
	}

	// Handle HTTPS: https://github.com/owner/repo.git
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove host
	parts := strings.SplitN(url, "/", 2)
	if len(parts) < 2 {
		return url
	}
	url = parts[1]

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	return url
}
