package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"minigit/internal/object"
)

var (
	ErrEmptyCommitMessage = errors.New("commit message cannot be empty")
	ErrNothingToCommit    = errors.New("nothing to commit, working tree clean")
)

type CommitResult struct {
	Hash      string
	ShortHash string
	Branch    string
	Message   string
}

// Commit creates a new commit from the staged index.
func (r *Repository) Commit(message, authorName, authorEmail string) (*CommitResult, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, ErrEmptyCommitMessage
	}

	lock, err := AcquireLock(GetIndexPath(r.Root))
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	idx, err := ReadIndex(r.Root)
	if err != nil {
		return nil, err
	}

	head, err := r.GetHEAD()
	if err != nil {
		return nil, err
	}

	parentCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return nil, err
	}

	treeHash, err := r.BuildTreeFromIndex(idx)
	if err != nil {
		return nil, fmt.Errorf("failed to build tree object: %w", err)
	}

	// Validate tree existence & structure in Merkle graph
	if err := r.ValidateTreeRecursively(treeHash, nil, 1); err != nil {
		return nil, fmt.Errorf("invalid tree for commit: %w", err)
	}

	// Prevent creating commit if there are no changes relative to HEAD commit
	if parentCommitHash != "" {
		parentType, _, err := r.GetObjectType(parentCommitHash)
		if err != nil {
			return nil, fmt.Errorf("parent commit %s not found: %w", parentCommitHash, err)
		}
		if parentType != object.TypeCommit {
			return nil, fmt.Errorf("%w: parent %s is of type %s", ErrObjectTypeMismatch, parentCommitHash, parentType)
		}

		parentCommit, _, err := r.GetCommitByHash(parentCommitHash)
		if err != nil {
			return nil, err
		}
		if parentCommit.Tree == treeHash {
			return nil, ErrNothingToCommit
		}
	} else if len(idx.Entries) == 0 {
		return nil, ErrNothingToCommit
	}

	if authorName == "" {
		authorName = "MiniGit User"
	}
	if authorEmail == "" {
		authorEmail = "user@minigit.local"
	}

	commitObj := object.NewCommit(treeHash, parentCommitHash, authorName, authorEmail, message, time.Now())
	commitHash, err := r.Objects.WriteObject(commitObj.Serialize())
	if err != nil {
		return nil, fmt.Errorf("failed to save commit object: %w", err)
	}

	// Update branch reference or detached HEAD
	branchName := ""
	if head.Type == HEADTypeBranch {
		branchName = head.Branch
		if err := WriteBranchCommit(r.Root, head.Branch, commitHash); err != nil {
			return nil, fmt.Errorf("failed to update branch ref %s: %w", head.Branch, err)
		}
		appendRefLog(r.Root, filepath.Join(".minigit", "logs", "refs", "heads", head.Branch), commitHash, message)
	} else {
		head.Commit = commitHash
		if err := r.SetHEAD(head); err != nil {
			return nil, fmt.Errorf("failed to update detached HEAD: %w", err)
		}
	}

	appendRefLog(r.Root, filepath.Join(".minigit", "logs", "HEAD.log"), commitHash, message)

	firstLine := strings.Split(message, "\n")[0]
	return &CommitResult{
		Hash:      commitHash,
		ShortHash: commitHash[:7],
		Branch:    branchName,
		Message:   firstLine,
	}, nil
}

func appendRefLog(repoRoot, relLogPath, commitHash, message string) {
	logFile := filepath.Join(repoRoot, relLogPath)
	os.MkdirAll(filepath.Dir(logFile), 0755)

	entry := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339), commitHash, strings.ReplaceAll(message, "\n", " "))
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(entry)
	}
}
