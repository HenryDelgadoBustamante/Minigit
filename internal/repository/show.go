package repository

import (
	"errors"
	"fmt"
	"sort"

	"minigit/internal/object"
	"minigit/internal/storage"
)

type CommitDiff struct {
	Added    []string
	Modified []string
	Deleted  []string
}

type ObjectInspectResult struct {
	Type       object.ObjectType
	FullHash   string
	BlobData   []byte
	Tree       *object.Tree
	Commit     *object.Commit
	CommitDiff *CommitDiff
}

// InspectObject inspects any object (Blob, Tree, or Commit) given a full hash, short prefix, branch, or "HEAD".
func (r *Repository) InspectObject(hashPrefix string) (*ObjectInspectResult, error) {
	fullHash, err := r.ResolveObjectID(hashPrefix)
	if err != nil {
		if errors.Is(err, ErrAmbiguousPrefix) {
			return nil, fmt.Errorf("El prefijo indicado es ambiguo: %s", hashPrefix)
		}
		if errors.Is(err, ErrObjectNotFound) {
			return nil, fmt.Errorf("No se encontró el objeto solicitado: %s", hashPrefix)
		}
		return nil, err
	}

	raw, fullHash, err := r.Objects.ReadObject(fullHash)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, fmt.Errorf("No se encontró ningún objeto con el prefijo indicado: %s", hashPrefix)
		}
		if errors.Is(err, storage.ErrCorruptObject) || errors.Is(err, storage.ErrInvalidHashFormat) {
			return nil, fmt.Errorf("Objeto corrupto: %w", err)
		}
		return nil, fmt.Errorf("error al leer el objeto %s: %w", hashPrefix, err)
	}

	objType, _, _, err := object.DecodeObject(raw)
	if err != nil {
		return nil, fmt.Errorf("Objeto corrupto: %w", err)
	}

	switch objType {
	case object.TypeBlob:
		blobObj, err := object.DecodeBlob(raw)
		if err != nil {
			return nil, fmt.Errorf("Objeto corrupto: %w", err)
		}
		return &ObjectInspectResult{
			Type:     object.TypeBlob,
			FullHash: fullHash,
			BlobData: blobObj.Data,
		}, nil

	case object.TypeTree:
		treeObj, err := object.DecodeTree(raw)
		if err != nil {
			return nil, fmt.Errorf("Objeto corrupto: %w", err)
		}
		return &ObjectInspectResult{
			Type:     object.TypeTree,
			FullHash: fullHash,
			Tree:     treeObj,
		}, nil

	case object.TypeCommit:
		commitObj, err := object.DecodeCommit(raw)
		if err != nil {
			return nil, fmt.Errorf("Objeto corrupto: %w", err)
		}

		diff, _, _, _ := r.GetCommitDiff(fullHash)
		return &ObjectInspectResult{
			Type:       object.TypeCommit,
			FullHash:   fullHash,
			Commit:     commitObj,
			CommitDiff: diff,
		}, nil

	default:
		return nil, fmt.Errorf("Objeto corrupto: tipo de objeto desconocido '%s'", objType)
	}
}

// GetCommitDiff retrieves a commit by prefix/ref and computes diff against its parent commit.
func (r *Repository) GetCommitDiff(hashPrefix string) (*CommitDiff, *object.Commit, string, error) {
	fullHash, err := r.ResolveObjectID(hashPrefix)
	if err != nil {
		return nil, nil, "", err
	}

	commitObj, fullHash, err := r.GetCommitByHash(fullHash)
	if err != nil {
		return nil, nil, "", err
	}

	currentTreeMap, err := r.ReadTreeToMap(commitObj.Tree)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to read commit tree: %w", err)
	}

	var parentTreeMap map[string]object.TreeEntry
	if commitObj.Parent != "" {
		parentCommit, _, err := r.GetCommitByHash(commitObj.Parent)
		if err == nil {
			parentTreeMap, _ = r.ReadTreeToMap(parentCommit.Tree)
		}
	}

	if parentTreeMap == nil {
		parentTreeMap = make(map[string]object.TreeEntry)
	}

	diff := &CommitDiff{}

	for path, currentEntry := range currentTreeMap {
		parentEntry, inParent := parentTreeMap[path]
		if !inParent {
			diff.Added = append(diff.Added, path)
		} else if parentEntry.Hash != currentEntry.Hash {
			diff.Modified = append(diff.Modified, path)
		}
	}

	for path := range parentTreeMap {
		if _, inCurrent := currentTreeMap[path]; !inCurrent {
			diff.Deleted = append(diff.Deleted, path)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Modified)
	sort.Strings(diff.Deleted)

	return diff, commitObj, fullHash, nil
}
