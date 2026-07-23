package repository

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"minigit/internal/object"
	"minigit/internal/storage"
)

type Repository struct {
	Root    string
	Objects *storage.ObjectStore
	Ignore  *IgnoreMatcher
}

// OpenRepository initializes a Repository struct for an existing repo root.
func OpenRepository(repoRoot string) *Repository {
	objectsDir := filepath.Join(repoRoot, ".minigit", "objects")
	return &Repository{
		Root:    repoRoot,
		Objects: storage.NewObjectStore(objectsDir),
		Ignore:  NewIgnoreMatcher(repoRoot),
	}
}

// GetHEAD returns current HEAD state.
func (r *Repository) GetHEAD() (*HEAD, error) {
	return ReadHEAD(r.Root)
}

// SetHEAD sets HEAD state.
func (r *Repository) SetHEAD(head *HEAD) error {
	return WriteHEAD(r.Root, head)
}

// GetHeadCommitHash resolves current commit hash HEAD points to (empty string if initial commit with no commits yet).
func (r *Repository) GetHeadCommitHash() (string, error) {
	head, err := r.GetHEAD()
	if err != nil {
		return "", err
	}

	if head.Type == HEADTypeDetached {
		return head.Commit, nil
	}

	commitHash, err := ReadBranchCommit(r.Root, head.Branch)
	if err != nil {
		if errorsIs(err, ErrBranchNotFound) {
			return "", nil // Initial state, branch ref does not exist yet
		}
		return "", err
	}

	return commitHash, nil
}

// BuildTreeFromIndex builds hierarchical tree objects from staged index entries and returns the root tree hash.
func (r *Repository) BuildTreeFromIndex(idx *Index) (string, error) {
	// Filter active (non-deleted) entries
	entries := idx.SortedEntries()
	activeEntries := make([]IndexEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Deleted {
			activeEntries = append(activeEntries, e)
		}
	}

	if len(activeEntries) == 0 {
		// Empty root tree
		emptyTree := object.NewTree(nil)
		return r.Objects.WriteObject(emptyTree.Serialize())
	}

	return r.buildSubTree("", activeEntries)
}

func (r *Repository) buildSubTree(prefix string, entries []IndexEntry) (string, error) {
	// Direct file entries under current prefix vs subdirectories
	dirEntriesMap := make(map[string][]IndexEntry)
	var treeEntries []object.TreeEntry

	for _, entry := range entries {
		relPath := entry.Path
		if prefix != "" {
			relPath = strings.TrimPrefix(relPath, prefix+"/")
		}

		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) == 1 {
			// Direct file
			treeEntries = append(treeEntries, object.TreeEntry{
				Name: parts[0],
				Path: entry.Path,
				Hash: entry.Hash,
				Type: "blob",
				Mode: entry.Mode,
			})
		} else {
			// Directory
			dirName := parts[0]
			dirEntriesMap[dirName] = append(dirEntriesMap[dirName], entry)
		}
	}

	// Process subdirectories
	var dirNames []string
	for dirName := range dirEntriesMap {
		dirNames = append(dirNames, dirName)
	}
	sort.Strings(dirNames)

	for _, dirName := range dirNames {
		subPrefix := dirName
		if prefix != "" {
			subPrefix = prefix + "/" + dirName
		}
		subHash, err := r.buildSubTree(subPrefix, dirEntriesMap[dirName])
		if err != nil {
			return "", err
		}

		treeEntries = append(treeEntries, object.TreeEntry{
			Name: dirName,
			Path: subPrefix,
			Hash: subHash,
			Type: "tree",
			Mode: 0755,
		})
	}

	treeObj := object.NewTree(treeEntries)
	return r.Objects.WriteObject(treeObj.Serialize())
}

var (
	ErrObjectTypeMismatch  = errors.New("object type mismatch in Merkle graph")
	ErrCyclicTreeReference = errors.New("cyclic tree reference detected")
	ErrExcessiveTreeDepth  = errors.New("excessive tree depth limit exceeded")
)

const maxTreeDepth = 100

// GetObjectType reads the object header to inspect its actual ObjectType without decompressing large payloads.
func (r *Repository) GetObjectType(hash string) (object.ObjectType, []byte, error) {
	objTypeStr, _, err := r.Objects.ReadObjectType(hash)
	if err != nil {
		return "", nil, err
	}
	return object.ObjectType(objTypeStr), nil, nil
}

// ValidateTreeRecursively validates the integrity, structure, object types, and absence of cycles in a tree graph.
func (r *Repository) ValidateTreeRecursively(treeHash string, visited map[string]bool, depth int) error {
	if visited == nil {
		visited = make(map[string]bool)
	}
	dummyMap := make(map[string]object.TreeEntry)
	return r.traverseTree("", treeHash, dummyMap, visited, depth)
}

// ValidateCommitGraph validates that a commit points to a valid tree and valid parent commit.
func (r *Repository) ValidateCommitGraph(commitHash string) error {
	commitObj, _, err := r.GetCommitByHash(commitHash)
	if err != nil {
		return fmt.Errorf("get commit %s: %w", commitHash, err)
	}

	visited := make(map[string]bool)
	if err := r.ValidateTreeRecursively(commitObj.Tree, visited, 1); err != nil {
		return fmt.Errorf("commit %s tree %s invalid: %w", commitHash, commitObj.Tree, err)
	}

	if commitObj.Parent != "" {
		parentType, _, err := r.GetObjectType(commitObj.Parent)
		if err != nil {
			return fmt.Errorf("commit %s parent %s not found: %w", commitHash, commitObj.Parent, err)
		}
		if parentType != object.TypeCommit {
			return fmt.Errorf("%w: commit %s parent %s is of type %s", ErrObjectTypeMismatch, commitHash, commitObj.Parent, parentType)
		}
	}

	return nil
}

// ReadTreeToMap recursively traverses a tree object and maps relative path -> TreeEntry.
func (r *Repository) ReadTreeToMap(treeHash string) (map[string]object.TreeEntry, error) {
	result := make(map[string]object.TreeEntry)
	if treeHash == "" {
		return result, nil
	}

	visited := make(map[string]bool)
	err := r.traverseTree("", treeHash, result, visited, 1)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) traverseTree(prefix string, treeHash string, result map[string]object.TreeEntry, visited map[string]bool, depth int) error {
	if depth > maxTreeDepth {
		return fmt.Errorf("%w: depth %d at %s", ErrExcessiveTreeDepth, depth, treeHash)
	}
	if visited[treeHash] {
		return fmt.Errorf("%w: %s", ErrCyclicTreeReference, treeHash)
	}
	visited[treeHash] = true
	defer func() { visited[treeHash] = false }()

	raw, _, err := r.Objects.ReadObject(treeHash)
	if err != nil {
		return fmt.Errorf("failed to read tree %s: %w", treeHash, err)
	}

	objType, _, _, err := object.DecodeObject(raw)
	if err != nil {
		return fmt.Errorf("failed to decode object %s: %w", treeHash, err)
	}
	if objType != object.TypeTree {
		return fmt.Errorf("%w: expected tree for %s, got %s", ErrObjectTypeMismatch, treeHash, objType)
	}

	treeObj, err := object.DecodeTree(raw)
	if err != nil {
		return fmt.Errorf("failed to decode tree %s: %w", treeHash, err)
	}

	for _, entry := range treeObj.Entries {
		relPath := entry.Name
		if prefix != "" {
			relPath = prefix + "/" + entry.Name
		}

		targetType, _, err := r.GetObjectType(entry.Hash)
		if err != nil {
			return fmt.Errorf("referenced object %s (%s) in tree %s not found: %w", entry.Hash, relPath, treeHash, err)
		}

		if entry.Type == "blob" {
			if targetType != object.TypeBlob {
				return fmt.Errorf("%w: entry '%s' declared as blob points to %s (%s)", ErrObjectTypeMismatch, relPath, targetType, entry.Hash)
			}
			entry.Path = relPath
			result[relPath] = entry
		} else if entry.Type == "tree" {
			if targetType != object.TypeTree {
				return fmt.Errorf("%w: entry '%s' declared as tree points to %s (%s)", ErrObjectTypeMismatch, relPath, targetType, entry.Hash)
			}
			if err := r.traverseTree(relPath, entry.Hash, result, visited, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCommitByHash loads and decodes a commit object by full or short hash.
func (r *Repository) GetCommitByHash(hash string) (*object.Commit, string, error) {
	raw, fullHash, err := r.Objects.ReadObject(hash)
	if err != nil {
		return nil, "", err
	}

	commitObj, err := object.DecodeCommit(raw)
	if err != nil {
		return nil, "", fmt.Errorf("object %s is not a valid commit: %w", fullHash, err)
	}

	return commitObj, fullHash, nil
}

func errorsIs(err, target error) bool {
	return strings.Contains(err.Error(), target.Error())
}
