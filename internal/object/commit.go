package object

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Commit represents a snapshot commit object pointing to a root Tree.
type Commit struct {
	Tree       string
	Parent     string
	AuthorName string
	AuthorMail string
	Message    string
	CreatedAt  time.Time
}

// NewCommit creates a new Commit struct ensuring CreatedAt is converted to UTC.
func NewCommit(tree, parent, name, email, message string, createdAt time.Time) *Commit {
	return &Commit{
		Tree:       tree,
		Parent:     parent,
		AuthorName: name,
		AuthorMail: email,
		Message:    message,
		CreatedAt:  createdAt.UTC(),
	}
}

// Type returns ObjectType "commit".
func (c *Commit) Type() ObjectType {
	return TypeCommit
}

// Serialize serializes the commit object deterministically.
// Format:
// tree <hash>\n
// [parent <hash>\n]
// author <name> <<email>> <RFC3339_timestamp>\n\n
// <message>
func (c *Commit) Serialize() []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("tree %s\n", c.Tree))
	if c.Parent != "" {
		buf.WriteString(fmt.Sprintf("parent %s\n", c.Parent))
	}
	buf.WriteString(fmt.Sprintf("author %s <%s> %s\n\n", c.AuthorName, c.AuthorMail, c.CreatedAt.UTC().Format(time.RFC3339)))
	buf.WriteString(c.Message)

	return EncodeObject(TypeCommit, buf.Bytes())
}

// isHex64 checks whether a string is a 64-character lowercase hex hash.
func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// DecodeCommit parses raw object payload into a Commit struct and validates header fields.
func DecodeCommit(raw []byte) (*Commit, error) {
	objType, _, body, err := DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if objType != TypeCommit {
		return nil, fmt.Errorf("%w: expected commit, got %s", ErrTypeMismatch, objType)
	}

	content := string(body)
	headerEnd := strings.Index(content, "\n\n")
	if headerEnd == -1 {
		return nil, fmt.Errorf("%w: malformed commit header separation", ErrInvalidHeader)
	}

	headerLines := strings.Split(content[:headerEnd], "\n")
	message := content[headerEnd+2:]

	commit := &Commit{
		Message: message,
	}

	for _, line := range headerLines {
		if strings.HasPrefix(line, "tree ") {
			commit.Tree = strings.TrimSpace(strings.TrimPrefix(line, "tree "))
		} else if strings.HasPrefix(line, "parent ") {
			commit.Parent = strings.TrimSpace(strings.TrimPrefix(line, "parent "))
		} else if strings.HasPrefix(line, "author ") {
			rawAuthor := strings.TrimPrefix(line, "author ")
			emailStart := strings.Index(rawAuthor, "<")
			emailEnd := strings.Index(rawAuthor, ">")
			if emailStart != -1 && emailEnd != -1 && emailEnd > emailStart {
				commit.AuthorName = strings.TrimSpace(rawAuthor[:emailStart])
				commit.AuthorMail = rawAuthor[emailStart+1 : emailEnd]
				timeStr := strings.TrimSpace(rawAuthor[emailEnd+1:])
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					commit.CreatedAt = parsedTime.UTC()
				} else {
					return nil, fmt.Errorf("%w: invalid author timestamp format '%s'", ErrInvalidHeader, timeStr)
				}
			} else {
				return nil, fmt.Errorf("%w: malformed author header line '%s'", ErrInvalidHeader, line)
			}
		}
	}

	if commit.Tree == "" {
		return nil, fmt.Errorf("%w: commit missing tree reference", ErrInvalidHeader)
	}
	if !isHex64(commit.Tree) {
		return nil, fmt.Errorf("%w: invalid tree hash in commit '%s'", ErrInvalidHeader, commit.Tree)
	}
	if commit.Parent != "" && !isHex64(commit.Parent) {
		return nil, fmt.Errorf("%w: invalid parent hash in commit '%s'", ErrInvalidHeader, commit.Parent)
	}

	return commit, nil
}
