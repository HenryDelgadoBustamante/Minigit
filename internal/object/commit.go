package object

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type Commit struct {
	Tree       string
	Parent     string
	AuthorName string
	AuthorMail string
	Message    string
	CreatedAt  time.Time
}

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

func (c *Commit) Type() ObjectType {
	return TypeCommit
}

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
		return nil, fmt.Errorf("%w: malformed commit header", ErrInvalidHeader)
	}

	headerLines := strings.Split(content[:headerEnd], "\n")
	message := content[headerEnd+2:]

	commit := &Commit{
		Message: message,
	}

	for _, line := range headerLines {
		if strings.HasPrefix(line, "tree ") {
			commit.Tree = strings.TrimPrefix(line, "tree ")
		} else if strings.HasPrefix(line, "parent ") {
			commit.Parent = strings.TrimPrefix(line, "parent ")
		} else if strings.HasPrefix(line, "author ") {
			rawAuthor := strings.TrimPrefix(line, "author ")
			// Parse: Author Name <email@example.com> 2026-07-22T16:50:00Z
			emailStart := strings.Index(rawAuthor, "<")
			emailEnd := strings.Index(rawAuthor, ">")
			if emailStart != -1 && emailEnd != -1 && emailEnd > emailStart {
				commit.AuthorName = strings.TrimSpace(rawAuthor[:emailStart])
				commit.AuthorMail = rawAuthor[emailStart+1 : emailEnd]
				timeStr := strings.TrimSpace(rawAuthor[emailEnd+1:])
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					commit.CreatedAt = parsedTime.UTC()
				}
			}
		}
	}

	if commit.Tree == "" {
		return nil, fmt.Errorf("%w: commit missing tree reference", ErrInvalidHeader)
	}

	return commit, nil
}
