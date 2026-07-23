package repository

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type IgnorePattern struct {
	Pattern string
	Negated bool
	IsDir   bool
}

type IgnoreMatcher struct {
	patterns []IgnorePattern
}

func NewIgnoreMatcher(repoRoot string) *IgnoreMatcher {
	m := &IgnoreMatcher{
		patterns: []IgnorePattern{
			{Pattern: ".minigit", IsDir: true},
			{Pattern: ".git", IsDir: true},
		},
	}
	m.LoadIgnoreFile(filepath.Join(repoRoot, ".minigitignore"))
	return m
}

func (m *IgnoreMatcher) LoadIgnoreFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		negated := false
		if strings.HasPrefix(line, "!") {
			negated = true
			line = line[1:]
		}

		isDir := false
		if strings.HasSuffix(line, "/") {
			isDir = true
			line = line[:len(line)-1]
		}

		line = strings.TrimPrefix(line, "/")
		m.patterns = append(m.patterns, IgnorePattern{
			Pattern: line,
			Negated: negated,
			IsDir:   isDir,
		})
	}
}

// IsIgnored checks if a normalized relative path (using '/') should be ignored.
func (m *IgnoreMatcher) IsIgnored(relPath string, isDir bool) bool {
	norm := strings.TrimPrefix(relPath, "/")
	if norm == ".minigit" || strings.HasPrefix(norm, ".minigit/") || norm == ".git" || strings.HasPrefix(norm, ".git/") {
		return true
	}

	ignored := false
	for _, p := range m.patterns {
		if p.IsDir && !isDir {
			continue
		}

		matched := matchPathPattern(p.Pattern, norm, isDir)
		if matched {
			if p.Negated {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

func matchPathPattern(pattern, normPath string, isDir bool) bool {
	// Exact match or basename match
	base := path.Base(normPath)

	if pattern == normPath || pattern == base {
		return true
	}

	// Match glob against full path or basename
	if matched, _ := path.Match(pattern, normPath); matched {
		return true
	}
	if matched, _ := path.Match(pattern, base); matched {
		return true
	}

	// Subpath match (e.g. pattern "build" matching "build/output.txt")
	if strings.HasPrefix(normPath, pattern+"/") {
		return true
	}

	return false
}
