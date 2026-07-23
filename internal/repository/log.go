package repository

import (
	"errors"
	"fmt"
	"time"

	"minigit/internal/storage"
)

var ErrNoCommits = errors.New("no hay commits registrados en este repositorio")

type CommitLogEntry struct {
	Hash        string
	ParentHash  string
	AuthorName  string
	AuthorEmail string
	Timestamp   time.Time
	Message     string
}

// GetCommitHistory returns the commit history starting from HEAD.
// Traverses commits sequentially backwards to the root commit (empty parent).
func (r *Repository) GetCommitHistory() ([]CommitLogEntry, error) {
	headCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return nil, err
	}

	if headCommitHash == "" {
		return nil, ErrNoCommits
	}

	var entries []CommitLogEntry
	currHash := headCommitHash
	visited := make(map[string]bool)

	for currHash != "" {
		if visited[currHash] {
			return nil, fmt.Errorf("ciclo detectado en el historial de commits en: %s", currHash)
		}
		visited[currHash] = true

		commitObj, fullHash, err := r.GetCommitByHash(currHash)
		if err != nil {
			if errors.Is(err, storage.ErrObjectNotFound) {
				return nil, fmt.Errorf("referencia a commit padre inválida (objeto no encontrado: %s)", currHash)
			}
			if errors.Is(err, storage.ErrCorruptObject) {
				return nil, fmt.Errorf("referencia a commit padre inválida (objeto corrupto: %s)", currHash)
			}
			return nil, fmt.Errorf("fallo al leer el commit %s: %w", currHash, err)
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
