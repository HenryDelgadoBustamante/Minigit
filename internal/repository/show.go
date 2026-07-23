package repository

import (
	"errors"
	"fmt"
	"sort"
	"strings"

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

// InspectObject inspects any object (Blob, Tree, or Commit) given a full hash or short prefix (or "HEAD").
func (r *Repository) InspectObject(hashPrefix string) (*ObjectInspectResult, error) {
	hashPrefix = strings.TrimSpace(hashPrefix)
	if hashPrefix == "" || strings.EqualFold(hashPrefix, "HEAD") {
		headHash, err := r.GetHeadCommitHash()
		if err != nil || headHash == "" {
			return nil, ErrNoCommits
		}
		hashPrefix = headHash
	}

	raw, fullHash, err := r.Objects.ReadObject(hashPrefix)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, fmt.Errorf("No se encontró el objeto solicitado: %s", hashPrefix)
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

// GetCommitDiff retrieves a commit by prefix, resolving default hash, and computes diff against its parent commit.
func (r *Repository) GetCommitDiff(hashPrefix string) (*CommitDiff, *object.Commit, string, error) {
	if hashPrefix == "" || strings.EqualFold(hashPrefix, "HEAD") {
		headHash, err := r.GetHeadCommitHash()
		if err != nil {
			return nil, nil, "", err
		}
		if headHash == "" {
			return nil, nil, "", ErrNoCommits
		}
		hashPrefix = headHash
	}

	commitObj, fullHash, err := r.GetCommitByHash(hashPrefix)
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
