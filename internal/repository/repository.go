package repository

import (
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

// ReadTreeToMap recursively traverses a tree object and maps relative path -> TreeEntry.
func (r *Repository) ReadTreeToMap(treeHash string) (map[string]object.TreeEntry, error) {
	result := make(map[string]object.TreeEntry)
	if treeHash == "" {
		return result, nil
	}

	err := r.traverseTree("", treeHash, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) traverseTree(prefix string, treeHash string, result map[string]object.TreeEntry) error {
	raw, _, err := r.Objects.ReadObject(treeHash)
	if err != nil {
		return fmt.Errorf("failed to read tree %s: %w", treeHash, err)
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

		if entry.Type == "blob" {
			entry.Path = relPath
			result[relPath] = entry
		} else if entry.Type == "tree" {
			if err := r.traverseTree(relPath, entry.Hash, result); err != nil {
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
