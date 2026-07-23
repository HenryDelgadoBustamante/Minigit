package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"minigit/internal/filesystem"
	"minigit/internal/object"
)

var (
	ErrDivergentBranches = errors.New("No se puede realizar fast-forward: las ramas han divergido.")
	ErrDetachedHEAD      = errors.New("no se puede realizar un merge en estado detached HEAD")
)

// IsAncestor checks if ancestorHash is a parent or grand-parent of commitHash.
func (r *Repository) IsAncestor(ancestorHash, commitHash string) (bool, error) {
	if ancestorHash == "" || commitHash == "" {
		return false, nil
	}
	if ancestorHash == commitHash {
		return true, nil
	}

	curr := commitHash
	visited := make(map[string]bool)

	for curr != "" && !visited[curr] {
		visited[curr] = true
		if curr == ancestorHash {
			return true, nil
		}

		commitObj, _, err := r.GetCommitByHash(curr)
		if err != nil {
			break
		}

		curr = commitObj.Parent
	}

	return false, nil
}

// MergeFastForward performs a fast-forward merge of targetBranch into the current branch.
func (r *Repository) MergeFastForward(targetBranch string) error {
	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return errors.New("se debe especificar el nombre de la rama a integrar")
	}

	head, err := ReadHEAD(r.Root)
	if err != nil {
		return fmt.Errorf("error al leer HEAD: %w", err)
	}

	if head.Type != HEADTypeBranch {
		return ErrDetachedHEAD
	}

	currentBranch := head.Branch
	if currentBranch == targetBranch {
		return nil // Already on target branch
	}

	currentCommitHash, err := ReadBranchCommit(r.Root, currentBranch)
	if err != nil {
		return fmt.Errorf("error leyendo commit de la rama actual '%s': %w", currentBranch, err)
	}

	targetCommitHash, err := ReadBranchCommit(r.Root, targetBranch)
	if err != nil {
		return fmt.Errorf("rama '%s' no encontrada: %w", targetBranch, err)
	}

	if currentCommitHash == targetCommitHash {
		return nil // Already up-to-date
	}

	// Verify ancestry (HU04)
	isAncestor, err := r.IsAncestor(currentCommitHash, targetCommitHash)
	if err != nil {
		return fmt.Errorf("error al verificar la historia de las ramas: %w", err)
	}

	if !isAncestor {
		return ErrDivergentBranches
	}

	// Perform Fast-Forward (HU03)
	lock, err := AcquireLock(GetIndexPath(r.Root))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	targetCommit, _, err := r.GetCommitByHash(targetCommitHash)
	if err != nil {
		return fmt.Errorf("error cargando commit destino '%s': %w", targetCommitHash[:7], err)
	}

	targetTreeMap, err := r.ReadTreeToMap(targetCommit.Tree)
	if err != nil {
		return fmt.Errorf("error leyendo el árbol del commit destino: %w", err)
	}

	currentHeadTreeMap := make(map[string]object.TreeEntry)
	if currentCommitHash != "" {
		if curCommit, _, err := r.GetCommitByHash(currentCommitHash); err == nil {
			currentHeadTreeMap, _ = r.ReadTreeToMap(curCommit.Tree)
		}
	}

	// Restore files in target commit
	for path, targetEntry := range targetTreeMap {
		raw, _, err := r.Objects.ReadObject(targetEntry.Hash)
		if err != nil {
			return fmt.Errorf("error leyendo blob para %s: %w", path, err)
		}

		blob, err := object.DecodeBlob(raw)
		if err != nil {
			return fmt.Errorf("error decodificando blob para %s: %w", path, err)
		}

		if err := filesystem.SafeWriteFile(r.Root, path, blob.Data, targetEntry.Mode); err != nil {
			return fmt.Errorf("error escribiendo %s: %w", path, err)
		}
	}

	// Remove files missing in target commit
	idx, _ := ReadIndex(r.Root)
	filesToRemove := make(map[string]bool)
	for path := range currentHeadTreeMap {
		if _, inTarget := targetTreeMap[path]; !inTarget {
			filesToRemove[path] = true
		}
	}
	if idx != nil {
		for path := range idx.Entries {
			if _, inTarget := targetTreeMap[path]; !inTarget {
				filesToRemove[path] = true
			}
		}
	}
	for path := range filesToRemove {
		filesystem.SafeRemoveFile(r.Root, path)
	}

	// Update Index
	newIdx := NewIndex()
	for path, entry := range targetTreeMap {
		absPath := filepath.Join(r.Root, filepath.FromSlash(path))
		var size int64
		var modTime time.Time = time.Now()
		if info, err := os.Stat(absPath); err == nil {
			size = info.Size()
			modTime = info.ModTime()
		}
		newIdx.AddOrUpdateEntry(IndexEntry{
			Path:    path,
			Hash:    entry.Hash,
			Size:    size,
			Mode:    entry.Mode,
			ModTime: modTime,
			Deleted: false,
		})
	}

	if err := WriteIndex(r.Root, newIdx); err != nil {
		return fmt.Errorf("error actualizando index durante merge: %w", err)
	}

	// Update current branch ref to targetCommitHash
	if err := WriteBranchCommit(r.Root, currentBranch, targetCommitHash); err != nil {
		return fmt.Errorf("error actualizando la referencia de la rama '%s': %w", currentBranch, err)
	}

	return nil
}
