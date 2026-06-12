package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/atterpac/ichi/internal/git"
	"github.com/atterpac/ichi/internal/selection"
)

// tokenRe matches {{token}} placeholders, including ones with a colon argument
// such as {{env:EDITOR}}, and an optional "?default" suffix marking the token
// as optional (e.g. {{1?}} or {{1?HEAD}}). For an optional token, a missing
// value expands to the default (empty when none is given) instead of erroring.
var tokenRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+(?::[a-zA-Z0-9_]+)?)(\?[^}]*)?\}\}`)

// unquotedTokens resolve to command words (not data) and are substituted
// verbatim instead of being shell-escaped.
var unquotedTokens = map[string]bool{
	"editor": true,
}

// Expand resolves {{template}} tokens in tmpl against the current selection,
// repository, and command arguments. Data-derived values are shell-escaped so
// they are safe inside the "sh -c" string. A token whose value is required but
// empty (e.g. {{commitHash}} with nothing selected) returns an error.
func Expand(tmpl string, sel *selection.Context, repo *git.Repository, args []string) (string, error) {
	var resolveErr error

	out := tokenRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		groups := tokenRe.FindStringSubmatch(match)
		name := groups[1]
		// groups[2] is the "?default" suffix when present; the leading '?'
		// marks the token optional and the remainder is its default value.
		optional := groups[2] != ""
		def := strings.TrimPrefix(groups[2], "?")

		val, ok, err := resolveToken(name, sel, repo, args)
		if err != nil {
			if optional {
				val = def
			} else {
				if resolveErr == nil {
					resolveErr = err
				}
				return match
			}
		} else if !ok {
			// Unknown token: leave it untouched rather than silently dropping.
			return match
		}
		if unquotedTokens[name] || strings.HasPrefix(name, "env:") {
			return val
		}
		return shellQuote(val)
	})

	if resolveErr != nil {
		return "", resolveErr
	}
	return out, nil
}

// resolveToken returns the value for a single token name. The bool reports
// whether the token is recognized; an error is returned when a recognized
// token requires a selection that is absent.
func resolveToken(name string, sel *selection.Context, repo *git.Repository, args []string) (string, bool, error) {
	// Positional arguments: {{1}}, {{2}}, ... and {{args}}.
	if name == "args" {
		return strings.Join(args, " "), true, nil
	}
	if n, err := strconv.Atoi(name); err == nil {
		if n >= 1 && n <= len(args) {
			return args[n-1], true, nil
		}
		return "", false, fmt.Errorf("command needs argument %d", n)
	}

	// Environment variables: {{env:NAME}}.
	if strings.HasPrefix(name, "env:") {
		return os.Getenv(strings.TrimPrefix(name, "env:")), true, nil
	}

	switch name {
	// Commit tokens.
	case "commitHash":
		return commitField(sel, func(c commitVals) string { return c.hash })
	case "shortHash":
		return commitField(sel, func(c commitVals) string { return c.short })
	case "commitMessage":
		return commitField(sel, func(c commitVals) string { return c.message })
	case "commitSubject":
		return commitField(sel, func(c commitVals) string { return c.subject })
	case "author":
		return commitField(sel, func(c commitVals) string { return c.author })

	// Branch tokens.
	case "branch":
		if sel == nil || !sel.HasBranch() {
			return "", true, fmt.Errorf("command needs a selected branch")
		}
		return sel.Branch.Name, true, nil
	case "branchUpstream":
		if sel == nil || !sel.HasBranch() {
			return "", true, fmt.Errorf("command needs a selected branch")
		}
		return sel.Branch.Upstream, true, nil
	case "currentBranch":
		return repo.CurrentBranch(), true, nil

	// File tokens.
	case "file", "filePath":
		if sel == nil || !sel.HasFile() {
			return "", true, fmt.Errorf("command needs a selected file")
		}
		return sel.File.Path, true, nil
	case "fileName":
		if sel == nil || !sel.HasFile() {
			return "", true, fmt.Errorf("command needs a selected file")
		}
		return filepath.Base(sel.File.Path), true, nil
	case "fileDir":
		if sel == nil || !sel.HasFile() {
			return "", true, fmt.Errorf("command needs a selected file")
		}
		return filepath.Dir(sel.File.Path), true, nil

	// Stash tokens.
	case "stash":
		if sel == nil || !sel.HasStash() {
			return "", true, fmt.Errorf("command needs a selected stash")
		}
		return fmt.Sprintf("stash@{%d}", sel.Stash.Index), true, nil
	case "stashIndex":
		if sel == nil || !sel.HasStash() {
			return "", true, fmt.Errorf("command needs a selected stash")
		}
		return strconv.Itoa(sel.Stash.Index), true, nil
	case "stashMessage":
		if sel == nil || !sel.HasStash() {
			return "", true, fmt.Errorf("command needs a selected stash")
		}
		return sel.Stash.Message, true, nil

	// Smart ref: commit, else branch, else stash.
	case "ref":
		if sel != nil && sel.HasCommit() {
			return sel.Commit.Hash, true, nil
		}
		if sel != nil && sel.HasBranch() {
			return sel.Branch.Name, true, nil
		}
		if sel != nil && sel.HasStash() {
			return fmt.Sprintf("stash@{%d}", sel.Stash.Index), true, nil
		}
		return "", true, fmt.Errorf("command needs a selected commit, branch, or stash")

	// Repo / environment.
	case "repoRoot":
		return repo.Path(), true, nil
	case "view":
		if sel == nil {
			return "", true, nil
		}
		return sel.ViewName, true, nil
	case "editor":
		ed := os.Getenv("EDITOR")
		if ed == "" {
			ed = "vi"
		}
		return ed, true, nil
	}

	return "", false, nil
}

type commitVals struct {
	hash, short, message, subject, author string
}

func commitField(sel *selection.Context, pick func(commitVals) string) (string, bool, error) {
	if sel == nil || !sel.HasCommit() {
		return "", true, fmt.Errorf("command needs a selected commit")
	}
	c := sel.Commit
	subject := c.Message
	if i := strings.IndexByte(subject, '\n'); i >= 0 {
		subject = subject[:i]
	}
	return pick(commitVals{
		hash:    c.Hash,
		short:   c.ShortHash,
		message: c.Message,
		subject: subject,
		author:  c.Author,
	}), true, nil
}

// shellQuote wraps s in single quotes, escaping any embedded single quotes, so
// it is safe as a single argument inside an "sh -c" string.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
