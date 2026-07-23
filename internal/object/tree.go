package object

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TreeEntry struct {
	Name string
	Path string
	Hash string
	Type string // "blob" or "tree"
	Mode uint32
}

type Tree struct {
	Entries []TreeEntry
}

func NewTree(entries []TreeEntry) *Tree {
	sorted := make([]TreeEntry, len(entries))
	copy(sorted, entries)
	sortTreeEntries(sorted)
	return &Tree{Entries: sorted}
}

func sortTreeEntries(entries []TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return entries[i].Hash < entries[j].Hash
		}
		return entries[i].Name < entries[j].Name
	})
}

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

// DecodeTree parses tree payload into a Tree struct.
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

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("%w: malformed tree entry: %s", ErrInvalidHeader, line)
		}

		mode, err := strconv.ParseUint(parts[0], 8, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid tree entry mode: %v", ErrInvalidHeader, err)
		}

		entries = append(entries, TreeEntry{
			Mode: uint32(mode),
			Type: parts[1],
			Hash: parts[2],
			Name: parts[3],
		})
	}

	return NewTree(entries), nil
}
