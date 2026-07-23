package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"minigit/internal/object"
	"minigit/internal/repository"
)

var (
	ErrEmptyCommitMessage = errors.New("commit message cannot be empty")
	ErrNothingToCommit    = errors.New("nothing to commit, working tree clean")
)

// RunCommit creates a new commit from the staged index.
func RunCommit(repo *repository.Repository, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return ErrEmptyCommitMessage
	}

	lock, err := repository.AcquireLock(repository.GetIndexPath(repo.Root))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	idx, err := repository.ReadIndex(repo.Root)
	if err != nil {
		return err
	}

	head, err := repo.GetHEAD()
	if err != nil {
		return err
	}

	parentCommitHash, err := repo.GetHeadCommitHash()
	if err != nil {
		return err
	}

	treeHash, err := repo.BuildTreeFromIndex(idx)
	if err != nil {
		return fmt.Errorf("failed to build tree object: %w", err)
	}

	// Prevent creating commit if there are no changes relative to HEAD commit
	if parentCommitHash != "" {
		parentCommit, _, err := repo.GetCommitByHash(parentCommitHash)
		if err != nil {
			return err
		}
		if parentCommit.Tree == treeHash {
			return ErrNothingToCommit
		}
	} else if len(idx.Entries) == 0 {
		return ErrNothingToCommit
	}

	authorName := os.Getenv("MINIGIT_AUTHOR_NAME")
	if authorName == "" {
		authorName = "MiniGit User"
	}
	authorEmail := os.Getenv("MINIGIT_AUTHOR_EMAIL")
	if authorEmail == "" {
		authorEmail = "user@minigit.local"
	}

	commitObj := object.NewCommit(treeHash, parentCommitHash, authorName, authorEmail, message, time.Now())
	commitHash, err := repo.Objects.WriteObject(commitObj.Serialize())
	if err != nil {
		return fmt.Errorf("failed to save commit object: %w", err)
	}

	// Update branch reference or detached HEAD
	if head.Type == repository.HEADTypeBranch {
		if err := repository.WriteBranchCommit(repo.Root, head.Branch, commitHash); err != nil {
			return fmt.Errorf("failed to update branch ref %s: %w", head.Branch, err)
		}
		appendRefLog(repo.Root, filepath.Join(".minigit", "logs", "refs", "heads", head.Branch), commitHash, message)
	} else {
		head.Commit = commitHash
		if err := repo.SetHEAD(head); err != nil {
			return fmt.Errorf("failed to update detached HEAD: %w", err)
		}
	}

	appendRefLog(repo.Root, filepath.Join(".minigit", "logs", "HEAD.log"), commitHash, message)

	shortHash := commitHash[:7]
	firstLine := strings.Split(message, "\n")[0]
	if head.Type == repository.HEADTypeBranch {
		fmt.Printf("[%s %s] %s\n", head.Branch, shortHash, firstLine)
	} else {
		fmt.Printf("[detached HEAD %s] %s\n", shortHash, firstLine)
	}

	return nil
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
