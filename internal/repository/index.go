package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"minigit/internal/filesystem"
	"minigit/internal/storage"
)

var ErrCorruptIndex = errors.New("corrupt index file")

type IndexEntry struct {
	Path    string    `json:"path"`
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	Mode    uint32    `json:"mode"`
	ModTime time.Time `json:"mod_time"`
	Deleted bool      `json:"deleted,omitempty"`
}

type Index struct {
	Entries map[string]IndexEntry `json:"entries"`
}

func NewIndex() *Index {
	return &Index{
		Entries: make(map[string]IndexEntry),
	}
}

func GetIndexPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".minigit", "index")
}

// ReadIndex loads the index from .minigit/index.
func ReadIndex(repoRoot string) (*Index, error) {
	indexPath := GetIndexPath(repoRoot)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewIndex(), nil
		}
		return nil, fmt.Errorf("reading index failed: %w", err)
	}

	if len(data) == 0 {
		return NewIndex(), nil
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptIndex, err)
	}

	if idx.Entries == nil {
		idx.Entries = make(map[string]IndexEntry)
	}

	return &idx, nil
}

// WriteIndex saves the index to .minigit/index atomically.
func WriteIndex(repoRoot string, idx *Index) error {
	indexPath := GetIndexPath(repoRoot)

	if idx.Entries == nil {
		idx.Entries = make(map[string]IndexEntry)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index failed: %w", err)
	}

	return storage.WriteFileAtomic(indexPath, data, 0644)
}

// AddOrUpdateEntry stages a file entry into the index.
func (idx *Index) AddOrUpdateEntry(entry IndexEntry) {
	entry.Path = filesystem.NormalizePath(entry.Path)
	idx.Entries[entry.Path] = entry
}

// RemoveEntry stages a file deletion or removes it from index.
func (idx *Index) RemoveEntry(path string) {
	norm := filesystem.NormalizePath(path)
	delete(idx.Entries, norm)
}

// SortedEntries returns index entries sorted by path.
func (idx *Index) SortedEntries() []IndexEntry {
	var paths []string
	for p := range idx.Entries {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	entries := make([]IndexEntry, 0, len(paths))
	for _, p := range paths {
		entries = append(entries, idx.Entries[p])
	}
	return entries
}
