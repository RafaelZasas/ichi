package app

import (
	"errors"
	"regexp"
	"strings"

	"github.com/atterpac/dado/validators"
)

// Git-specific validators

// BranchNameValidator validates git branch names.
func BranchNameValidator() validators.Validator {
	// Git branch name rules:
	// - Cannot start with a dot
	// - Cannot contain: ~ ^ : \ ? * [ space
	// - Cannot end with .lock
	// - Cannot have consecutive dots
	// - Cannot have @ followed by {
	invalidChars := regexp.MustCompile(`[~^:\\\?*\[\s]`)
	consecutiveDots := regexp.MustCompile(`\.\.`)
	atBrace := regexp.MustCompile(`@\{`)

	return func(value any) error {
		s, ok := value.(string)
		if !ok || s == "" {
			return nil
		}

		if strings.HasPrefix(s, ".") {
			return errors.New("branch name cannot start with a dot")
		}

		if strings.HasPrefix(s, "-") {
			return errors.New("branch name cannot start with a hyphen")
		}

		if strings.HasSuffix(s, ".lock") {
			return errors.New("branch name cannot end with .lock")
		}

		if invalidChars.MatchString(s) {
			return errors.New("branch name contains invalid characters (~ ^ : \\ ? * [ space)")
		}

		if consecutiveDots.MatchString(s) {
			return errors.New("branch name cannot contain consecutive dots")
		}

		if atBrace.MatchString(s) {
			return errors.New("branch name cannot contain @{")
		}

		return nil
	}
}

// CommitMessageValidator validates commit messages.
func CommitMessageValidator() validators.Validator {
	return validators.All(
		validators.Required(),
		validators.MinLength(1),
		validators.MaxLength(72), // First line should be 72 chars max
	)
}

// ConventionalCommitValidator validates conventional commit format.
func ConventionalCommitValidator() validators.Validator {
	// Format: type(scope)?: description
	pattern := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9-]+\))?!?: .+`)

	return validators.All(
		validators.Required(),
		func(value any) error {
			s, ok := value.(string)
			if !ok || s == "" {
				return nil
			}

			// Only validate first line
			firstLine := strings.Split(s, "\n")[0]
			if !pattern.MatchString(firstLine) {
				return errors.New("commit message should follow conventional commits format: type(scope): description")
			}
			return nil
		},
	)
}

// StashMessageValidator validates stash messages.
func StashMessageValidator() validators.Validator {
	return validators.MaxLength(100)
}

// TagNameValidator validates git tag names.
func TagNameValidator() validators.Validator {
	invalidChars := regexp.MustCompile(`[~^:\\\?*\[\s]`)

	return func(value any) error {
		s, ok := value.(string)
		if !ok || s == "" {
			return nil
		}

		if strings.HasPrefix(s, "-") {
			return errors.New("tag name cannot start with a hyphen")
		}

		if invalidChars.MatchString(s) {
			return errors.New("tag name contains invalid characters")
		}

		return nil
	}
}

// RemoteNameValidator validates git remote names.
func RemoteNameValidator() validators.Validator {
	validName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	return func(value any) error {
		s, ok := value.(string)
		if !ok || s == "" {
			return nil
		}

		if !validName.MatchString(s) {
			return errors.New("remote name must start with a letter and contain only letters, numbers, hyphens, and underscores")
		}

		return nil
	}
}
