package repository

import (
	"errors"
	"fmt"
	"strings"

	"minigit/internal/storage"
)

var (
	ErrAmbiguousPrefix = errors.New("el prefijo indicado coincide con más de un objeto")
	ErrObjectNotFound  = errors.New("no se encontró ningún objeto con el prefijo indicado")
	ErrInvalidIdentifier = errors.New("identificador de objeto inválido")
)

// ResolveObjectID resolves a 64-character hash, short prefix, branch name, or "HEAD" into a full 64-character SHA-256 hash.
func (r *Repository) ResolveObjectID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "HEAD") {
		headHash, err := r.GetHeadCommitHash()
		if err != nil || headHash == "" {
			return "", ErrNoCommits
		}
		return headHash, nil
	}

	// 2. Handle branch name
	if BranchExists(r.Root, value) {
		commitHash, err := ReadBranchCommit(r.Root, value)
		if err == nil && commitHash != "" {
			return commitHash, nil
		}
	}

	// 3. Handle hash or prefix via ObjectStore
	fullHash, err := r.Objects.ResolveHash(value)
	if err != nil {
		if errors.Is(err, storage.ErrAmbiguousHash) {
			return "", fmt.Errorf("%w: %s", ErrAmbiguousPrefix, value)
		}
		if errors.Is(err, storage.ErrObjectNotFound) {
			return "", fmt.Errorf("%w: %s", ErrObjectNotFound, value)
		}
		if errors.Is(err, storage.ErrInvalidHashFormat) {
			return "", fmt.Errorf("%w: %s", ErrInvalidIdentifier, value)
		}
		return "", fmt.Errorf("error al resolver el identificador '%s': %w", value, err)
	}

	return fullHash, nil
}
