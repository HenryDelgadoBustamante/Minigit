package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"minigit/internal/storage"
)

var ErrInvalidHEAD = errors.New("invalid HEAD reference")

type HEADType string

const (
	HEADTypeBranch   HEADType = "branch"
	HEADTypeDetached HEADType = "detached"
)

type HEAD struct {
	Type   HEADType
	Branch string // e.g. "main" or "refs/heads/main"
	Commit string // 64 hex character SHA-256 commit hash
}

// ReadHEAD reads and parses .minigit/HEAD.
func ReadHEAD(repoRoot string) (*HEAD, error) {
	headPath := filepath.Join(repoRoot, ".minigit", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return nil, fmt.Errorf("reading HEAD failed: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil, fmt.Errorf("%w: HEAD file is empty", ErrInvalidHEAD)
	}

	if strings.HasPrefix(content, "ref: ") {
		refTarget := strings.TrimPrefix(content, "ref: ")
		if refTarget == "" {
			return nil, fmt.Errorf("%w: ref: target is empty", ErrInvalidHEAD)
		}
		if !strings.HasPrefix(refTarget, "refs/heads/") {
			return nil, fmt.Errorf("%w: invalid ref format '%s'", ErrInvalidHEAD, refTarget)
		}
		branchName := strings.TrimPrefix(refTarget, "refs/heads/")
		if strings.TrimSpace(branchName) == "" {
			return nil, fmt.Errorf("%w: branch name is empty", ErrInvalidHEAD)
		}
		return &HEAD{
			Type:   HEADTypeBranch,
			Branch: branchName,
		}, nil
	}

	if len(content) == 64 {
		for _, c := range content {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return nil, fmt.Errorf("%w: invalid hex character in detached HEAD", ErrInvalidHEAD)
			}
		}
		return &HEAD{
			Type:   HEADTypeDetached,
			Commit: strings.ToLower(content),
		}, nil
	}

	return nil, fmt.Errorf("%w: content '%s'", ErrInvalidHEAD, content)
}

// WriteHEAD updates .minigit/HEAD atomically.
func WriteHEAD(repoRoot string, head *HEAD) error {
	if head == nil {
		return fmt.Errorf("%w: cannot write nil HEAD", ErrInvalidHEAD)
	}

	headPath := filepath.Join(repoRoot, ".minigit", "HEAD")
	var content string

	switch head.Type {
	case HEADTypeBranch:
		branch := head.Branch
		if branch == "" {
			return fmt.Errorf("%w: branch name cannot be empty", ErrInvalidHEAD)
		}
		if !strings.HasPrefix(branch, "refs/heads/") {
			branch = "refs/heads/" + branch
		}
		content = fmt.Sprintf("ref: %s\n", branch)
	case HEADTypeDetached:
		if len(head.Commit) != 64 {
			return fmt.Errorf("%w: invalid commit hash length for detached HEAD (got %d, expected 64)", ErrInvalidHEAD, len(head.Commit))
		}
		for _, c := range head.Commit {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("%w: detached HEAD contains invalid hex character", ErrInvalidHEAD)
			}
		}
		content = fmt.Sprintf("%s\n", head.Commit)
	default:
		return fmt.Errorf("%w: unknown HEAD type '%s'", ErrInvalidHEAD, head.Type)
	}

	if err := storage.WriteFileAtomic(headPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write HEAD: %w", err)
	}
	return nil
}

func NewHEAD(branch string) *HEAD {
	return &HEAD{Type: HEADTypeBranch, Branch: branch}
}
