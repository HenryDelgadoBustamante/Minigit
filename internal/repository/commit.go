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
	ErrParentNotFound     = errors.New("parent commit not found")
	ErrInvalidParentType  = errors.New("parent object is not a commit")
)

type CommitResult struct {
	Hash         string
	ShortHash    string
	Branch       string
	DetachedHEAD bool
	ParentHash   string
	RootTreeHash string
	Message      string
}

// Commit creates a new commit from the staged index using current UTC time.
func (r *Repository) Commit(message, authorName, authorEmail string) (*CommitResult, error) {
	return r.CommitWithTime(message, authorName, authorEmail, time.Now().UTC())
}

// CommitWithTime creates a new commit from the staged index using a specified timestamp (useful for testing).
func (r *Repository) CommitWithTime(message, authorName, authorEmail string, commitTime time.Time) (*CommitResult, error) {
	if strings.TrimSpace(message) == "" {
		return nil, ErrEmptyCommitMessage
	}

	lock, err := AcquireLock(GetIndexPath(r.Root))
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	idx, err := ReadIndex(r.Root)
	if err != nil {
		return nil, fmt.Errorf("cargar index para commit: %w", err)
	}

	// Validate index entries and verify blobs exist in ObjectStore
	for _, entry := range idx.Entries {
		if entry.Deleted {
			continue
		}
		if len(entry.Hash) != 64 {
			return nil, fmt.Errorf("%w: invalid hash length '%s' for %s", ErrCorruptIndex, entry.Hash, entry.Path)
		}
		if !r.Objects.Exists(entry.Hash) {
			return nil, fmt.Errorf("staging error: blob for %s (%s) not found in object store", entry.Path, entry.Hash)
		}
	}

	head, err := r.GetHEAD()
	if err != nil {
		return nil, fmt.Errorf("HEAD invalido: %w", err)
	}

	parentCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return nil, fmt.Errorf("obtener parent commit: %w", err)
	}

	// Validate parent commit existence and object type if parentCommitHash is present
	if parentCommitHash != "" {
		parentType, _, err := r.GetObjectType(parentCommitHash)
		if err != nil {
			return nil, fmt.Errorf("No se puede crear el commit: HEAD apunta a un objeto inexistente (%s): %w", parentCommitHash, ErrParentNotFound)
		}
		if parentType != object.TypeCommit {
			return nil, fmt.Errorf("%w: HEAD apunta a un objeto de tipo %s (%s)", ErrInvalidParentType, parentType, parentCommitHash)
		}
	}

	treeHash, err := r.BuildTreeFromIndex(idx)
	if err != nil {
		return nil, fmt.Errorf("crear tree raiz: %w", err)
	}

	// Validate tree existence & structure in Merkle graph
	if err := r.ValidateTreeRecursively(treeHash, nil, 1); err != nil {
		return nil, fmt.Errorf("invalid tree for commit: %w", err)
	}

	// Prevent creating commit if there are no changes relative to HEAD commit
	if parentCommitHash != "" {
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

	commitObj := object.NewCommit(treeHash, parentCommitHash, authorName, authorEmail, message, commitTime)
	commitHash, err := r.Objects.WriteObject(commitObj.Serialize())
	if err != nil {
		return nil, fmt.Errorf("failed to save commit object: %w", err)
	}

	// Update branch reference or detached HEAD safely
	branchName := ""
	isDetached := false
	if head.Type == HEADTypeBranch {
		branchName = head.Branch
		if err := WriteBranchCommit(r.Root, head.Branch, commitHash); err != nil {
			return nil, fmt.Errorf("failed to update branch ref %s: %w", head.Branch, err)
		}
		appendRefLog(r.Root, filepath.Join(".minigit", "logs", "refs", "heads", head.Branch), commitHash, message)
	} else {
		isDetached = true
		head.Commit = commitHash
		if err := r.SetHEAD(head); err != nil {
			return nil, fmt.Errorf("failed to update detached HEAD: %w", err)
		}
	}

	appendRefLog(r.Root, filepath.Join(".minigit", "logs", "HEAD.log"), commitHash, message)

	firstLine := strings.Split(message, "\n")[0]
	return &CommitResult{
		Hash:         commitHash,
		ShortHash:    commitHash[:7],
		Branch:       branchName,
		DetachedHEAD: isDetached,
		ParentHash:   parentCommitHash,
		RootTreeHash: treeHash,
		Message:      firstLine,
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
