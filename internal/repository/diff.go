package repository

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"minigit/internal/object"
)

type ChangeType string

const (
	ChangeAdded    ChangeType = "A"
	ChangeModified ChangeType = "M"
	ChangeDeleted  ChangeType = "D"
	ChangeRenamed  ChangeType = "R"
)

type FileChange struct {
	Type    ChangeType
	Path    string
	OldPath string
	OldHash string
	NewHash string
}

type CommitDiffResult struct {
	Commit1Hash string
	Commit2Hash string
	Changes     []FileChange
}

// CompareCommits compares two commits (or references) and returns structural file changes (A, M, D, R).
func (r *Repository) CompareCommits(commit1Ref, commit2Ref string) (*CommitDiffResult, error) {
	hash1, err := r.ResolveObjectID(commit1Ref)
	if err != nil {
		return nil, fmt.Errorf("error resolviendo commit '%s': %w", commit1Ref, err)
	}

	hash2, err := r.ResolveObjectID(commit2Ref)
	if err != nil {
		return nil, fmt.Errorf("error resolviendo commit '%s': %w", commit2Ref, err)
	}

	commit1, _, err := r.GetCommitByHash(hash1)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar el commit '%s': %w", hash1[:7], err)
	}

	commit2, _, err := r.GetCommitByHash(hash2)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar el commit '%s': %w", hash2[:7], err)
	}

	map1, err := r.ReadTreeToMap(commit1.Tree)
	if err != nil {
		return nil, fmt.Errorf("error leyendo el árbol del commit %s: %w", hash1[:7], err)
	}

	map2, err := r.ReadTreeToMap(commit2.Tree)
	if err != nil {
		return nil, fmt.Errorf("error leyendo el árbol del commit %s: %w", hash2[:7], err)
	}

	var changes []FileChange
	deletedMap := make(map[string]object.TreeEntry)
	addedMap := make(map[string]object.TreeEntry)

	// Identify deleted and modified
	for path, entry1 := range map1 {
		entry2, inMap2 := map2[path]
		if !inMap2 {
			deletedMap[path] = entry1
		} else if entry1.Hash != entry2.Hash {
			changes = append(changes, FileChange{
				Type:    ChangeModified,
				Path:    path,
				OldHash: entry1.Hash,
				NewHash: entry2.Hash,
			})
		}
	}

	// Identify added
	for path, entry2 := range map2 {
		if _, inMap1 := map1[path]; !inMap1 {
			addedMap[path] = entry2
		}
	}

	// Basic Rename Detection (HU07): match deleted entry with added entry having identical hash
	renamedDeleted := make(map[string]bool)
	renamedAdded := make(map[string]bool)

	for delPath, delEntry := range deletedMap {
		for addPath, addEntry := range addedMap {
			if !renamedAdded[addPath] && delEntry.Hash == addEntry.Hash && delEntry.Hash != "" {
				changes = append(changes, FileChange{
					Type:    ChangeRenamed,
					Path:    addPath,
					OldPath: delPath,
					OldHash: delEntry.Hash,
					NewHash: addEntry.Hash,
				})
				renamedDeleted[delPath] = true
				renamedAdded[addPath] = true
				break
			}
		}
	}

	// Add remaining deleted files
	for delPath, delEntry := range deletedMap {
		if !renamedDeleted[delPath] {
			changes = append(changes, FileChange{
				Type:    ChangeDeleted,
				Path:    delPath,
				OldHash: delEntry.Hash,
			})
		}
	}

	// Add remaining added files
	for addPath, addEntry := range addedMap {
		if !renamedAdded[addPath] {
			changes = append(changes, FileChange{
				Type:    ChangeAdded,
				Path:    addPath,
				NewHash: addEntry.Hash,
			})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return &CommitDiffResult{
		Commit1Hash: hash1,
		Commit2Hash: hash2,
		Changes:     changes,
	}, nil
}

// GetBlobContent reads blob raw data by hash. Returns empty slice if hash is empty.
func (r *Repository) GetBlobContent(hash string) ([]byte, error) {
	if hash == "" {
		return []byte{}, nil
	}
	raw, _, err := r.Objects.ReadObject(hash)
	if err != nil {
		return nil, err
	}
	blob, err := object.DecodeBlob(raw)
	if err != nil {
		return nil, err
	}
	return blob.Data, nil
}

// GetFileDiffLines computes line-by-line diff between two byte payloads.
// Returns diff lines prefixed with '-' or '+', and a boolean indicating if binary.
func GetFileDiffLines(oldData, newData []byte) ([]string, bool) {
	if isBinary(oldData) || isBinary(newData) {
		return nil, true
	}

	oldLines := splitLines(string(oldData))
	newLines := splitLines(string(newData))

	var diffLines []string

	// Simple Myers-like or LCS/line-comparison diff
	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			i++
			j++
		} else {
			// Lookahead to find match
			matchInNew := -1
			for nj := j + 1; nj < len(newLines) && nj < j+10; nj++ {
				if oldLines[i] == newLines[nj] {
					matchInNew = nj
					break
				}
			}
			matchInOld := -1
			for oi := i + 1; oi < len(oldLines) && oi < i+10; oi++ {
				if oldLines[oi] == newLines[j] {
					matchInOld = oi
					break
				}
			}

			if matchInNew != -1 && (matchInOld == -1 || matchInNew-j <= matchInOld-i) {
				for ; j < matchInNew; j++ {
					diffLines = append(diffLines, "+ "+newLines[j])
				}
			} else if matchInOld != -1 {
				for ; i < matchInOld; i++ {
					diffLines = append(diffLines, "- "+oldLines[i])
				}
			} else {
				diffLines = append(diffLines, "- "+oldLines[i])
				diffLines = append(diffLines, "+ "+newLines[j])
				i++
				j++
			}
		}
	}

	for ; i < len(oldLines); i++ {
		diffLines = append(diffLines, "- "+oldLines[i])
	}
	for ; j < len(newLines); j++ {
		diffLines = append(diffLines, "+ "+newLines[j])
	}

	return diffLines, false
}

func isBinary(data []byte) bool {
	checkLen := len(data)
	if checkLen > 8000 {
		checkLen = 8000
	}
	return bytes.IndexByte(data[:checkLen], 0) != -1
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
