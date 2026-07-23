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
	if strings.HasPrefix(content, "ref: ") {
		refTarget := strings.TrimPrefix(content, "ref: ")
		branchName := strings.TrimPrefix(refTarget, "refs/heads/")
		return &HEAD{
			Type:   HEADTypeBranch,
			Branch: branchName,
		}, nil
	}

	if len(content) == 64 {
		return &HEAD{
			Type:   HEADTypeDetached,
			Commit: content,
		}, nil
	}

	return nil, fmt.Errorf("%w: content '%s'", ErrInvalidHEAD, content)
}

// WriteHEAD updates .minigit/HEAD atomically.
func WriteHEAD(repoRoot string, head *HEAD) error {
	headPath := filepath.Join(repoRoot, ".minigit", "HEAD")
	var content string

	switch head.Type {
	case HEADTypeBranch:
		branch := head.Branch
		if !strings.HasPrefix(branch, "refs/heads/") {
			branch = "refs/heads/" + branch
		}
		content = fmt.Sprintf("ref: %s\n", branch)
	case HEADTypeDetached:
		if len(head.Commit) != 64 {
			return fmt.Errorf("%w: invalid commit hash length for detached HEAD", ErrInvalidHEAD)
		}
		content = fmt.Sprintf("%s\n", head.Commit)
	default:
		return fmt.Errorf("%w: unknown HEAD type", ErrInvalidHEAD)
	}

	func NewHEAD(branch string) *HEAD {
    return &HEAD{Type: HEADTypeBranch, Branch: branch}
}
}
