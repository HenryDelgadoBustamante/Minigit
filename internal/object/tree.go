package object

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TreeEntry represents a directory record linking a file or sub-directory.
type TreeEntry struct {
	Name string
	Path string
	Hash string
	Type string // "blob" or "tree"
	Mode uint32
}

// Tree represents a directory snapshot containing an array of TreeEntry records.
type Tree struct {
	Entries []TreeEntry
}

// NewTree creates a new Tree instance with deterministically sorted entries.
func NewTree(entries []TreeEntry) *Tree {
	sorted := make([]TreeEntry, len(entries))
	copy(sorted, entries)
	sortTreeEntries(sorted)
	return &Tree{Entries: sorted}
}

// sortTreeEntries sorts entries deterministically by Name (alphabetical), then by Hash.
func sortTreeEntries(entries []TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return entries[i].Hash < entries[j].Hash
		}
		return entries[i].Name < entries[j].Name
	})
}

// Type returns ObjectType "tree".
func (t *Tree) Type() ObjectType {
	return TypeTree
}

// Serialize serializes the Tree into deterministic bytes.
// Format per entry: "<mode> <type> <hash> <name>\n"
func (t *Tree) Serialize() []byte {
	sortTreeEntries(t.Entries)
	var buf bytes.Buffer
	for _, entry := range t.Entries {
		line := fmt.Sprintf("%o %s %s %s\n", entry.Mode, entry.Type, entry.Hash, entry.Name)
		buf.WriteString(line)
	}
	return EncodeObject(TypeTree, buf.Bytes())
}

// DecodeTree parses tree payload into a Tree struct and validates each entry.
func DecodeTree(raw []byte) (*Tree, error) {
	objType, _, body, err := DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if objType != TypeTree {
		return nil, fmt.Errorf("%w: expected tree, got %s", ErrTypeMismatch, objType)
	}

	lines := strings.Split(string(body), "\n")
	var entries []TreeEntry

	seenNames := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("%w: malformed tree entry line '%s'", ErrInvalidHeader, line)
		}

		mode, err := strconv.ParseUint(parts[0], 8, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid tree entry mode '%s': %v", ErrInvalidHeader, parts[0], err)
		}

		entryType := parts[1]
		if entryType != string(TypeBlob) && entryType != string(TypeTree) {
			return nil, fmt.Errorf("%w: invalid tree entry type '%s'", ErrInvalidHeader, entryType)
		}

		hash := parts[2]
		if len(hash) != 64 {
			return nil, fmt.Errorf("%w: invalid tree entry hash length '%s'", ErrInvalidHash, hash)
		}
		for _, c := range hash {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return nil, fmt.Errorf("%w: non-hex character in tree entry hash '%s'", ErrInvalidHash, hash)
			}
		}

		name := parts[3]
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
			return nil, fmt.Errorf("%w: invalid entry name '%s'", ErrInvalidEntryName, name)
		}

		if seenNames[name] {
			return nil, fmt.Errorf("%w: duplicate tree entry '%s'", ErrDuplicateEntry, name)
		}
		seenNames[name] = true

		entries = append(entries, TreeEntry{
			Mode: uint32(mode),
			Type: entryType,
			Hash: hash,
			Name: name,
		})
	}

	return NewTree(entries), nil
}
