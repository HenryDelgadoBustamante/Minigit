package repository

import (
	"fmt"
	"time"
)

type CommitLogEntry struct {
	Hash        string
	ParentHash  string
	AuthorName  string
	AuthorEmail string
	Timestamp   time.Time
	Message     string
}

// GetCommitHistory returns the commit history starting from HEAD.
func (r *Repository) GetCommitHistory() ([]CommitLogEntry, error) {
	headCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return nil, err
	}

	if headCommitHash == "" {
		return nil, fmt.Errorf("your current branch does not have any commits yet")
	}

	var entries []CommitLogEntry
	currHash := headCommitHash
	visited := make(map[string]bool)

	for currHash != "" {
		if visited[currHash] {
			return nil, fmt.Errorf("cycle detected in commit history at %s", currHash)
		}
		visited[currHash] = true

		commitObj, fullHash, err := r.GetCommitByHash(currHash)
		if err != nil {
			return nil, fmt.Errorf("failed to read commit %s: %w", currHash, err)
		}

		entries = append(entries, CommitLogEntry{
			Hash:        fullHash,
			ParentHash:  commitObj.Parent,
			AuthorName:  commitObj.AuthorName,
			AuthorEmail: commitObj.AuthorMail,
			Timestamp:   commitObj.CreatedAt,
			Message:     commitObj.Message,
		})

		currHash = commitObj.Parent
	}

	return entries, nil
}
