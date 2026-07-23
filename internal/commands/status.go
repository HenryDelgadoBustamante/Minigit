package commands

import (
	"fmt"
	"strings"

	"minigit/internal/repository"
)

// RunStatus returns the formatted status output.
func RunStatus(repo *repository.Repository) (string, error) {
	status, err := repo.GetStatus()
	if err != nil {
		return "", err
	}

	head, err := repo.GetHEAD()
	if err != nil {
		return "", err
	}

	return FormatStatus(status, head), nil
}

// FormatStatus formats the repository status for output display.
func FormatStatus(s *repository.StatusResult, head *repository.HEAD) string {
	var sb strings.Builder

	if head.Type == repository.HEADTypeBranch {
		sb.WriteString(fmt.Sprintf("On branch %s\n", head.Branch))
	} else {
		sb.WriteString(fmt.Sprintf("HEAD detached at %s\n", head.Commit[:7]))
	}

	var stagedAdded, stagedModified, stagedDeleted []string
	for _, f := range s.Staged {
		switch f.Type {
		case repository.StatusAdded:
			stagedAdded = append(stagedAdded, f.Path)
		case repository.StatusModified:
			stagedModified = append(stagedModified, f.Path)
		case repository.StatusDeleted:
			stagedDeleted = append(stagedDeleted, f.Path)
		}
	}

	var unstagedModified, unstagedDeleted []string
	for _, f := range s.Unstaged {
		switch f.Type {
		case repository.StatusModified:
			unstagedModified = append(unstagedModified, f.Path)
		case repository.StatusDeleted:
			unstagedDeleted = append(unstagedDeleted, f.Path)
		}
	}

	hasStaged := len(stagedAdded) > 0 || len(stagedModified) > 0 || len(stagedDeleted) > 0
	hasUnstaged := len(unstagedModified) > 0 || len(unstagedDeleted) > 0
	hasUntracked := len(s.Untracked) > 0

	if hasStaged {
		sb.WriteString("\nChanges to be committed:\n")
		for _, p := range stagedAdded {
			sb.WriteString(fmt.Sprintf("  new file:   %s\n", p))
		}
		for _, p := range stagedModified {
			sb.WriteString(fmt.Sprintf("  modified:   %s\n", p))
		}
		for _, p := range stagedDeleted {
			sb.WriteString(fmt.Sprintf("  deleted:    %s\n", p))
		}
	}

	if hasUnstaged {
		sb.WriteString("\nChanges not staged for commit:\n")
		for _, p := range unstagedModified {
			sb.WriteString(fmt.Sprintf("  modified:   %s\n", p))
		}
package commands

import (
	"fmt"
	"strings"

	"minigit/internal/repository"
)

// RunStatus returns the formatted status output.
func RunStatus(repo *repository.Repository) (string, error) {
	status, err := repo.GetStatus()
	if err != nil {
		return "", err
	}

	head, err := repo.GetHEAD()
	if err != nil {
		return "", err
	}

	return FormatStatus(status, head), nil
}

// FormatStatus formats the repository status for output display.
func FormatStatus(s *repository.StatusResult, head *repository.HEAD) string {
	var sb strings.Builder

	if head.Type == repository.HEADTypeBranch {
		sb.WriteString(fmt.Sprintf("En la rama %s\n", head.Branch))
	} else {
		sb.WriteString(fmt.Sprintf("HEAD separado en %s\n", head.Commit[:7]))
	}

	var stagedAdded, stagedModified, stagedDeleted []string
	for _, f := range s.Staged {
		switch f.Type {
		case repository.StatusAdded:
			stagedAdded = append(stagedAdded, f.Path)
		case repository.StatusModified:
			stagedModified = append(stagedModified, f.Path)
		case repository.StatusDeleted:
			stagedDeleted = append(stagedDeleted, f.Path)
		}
	}

	var unstagedModified, unstagedDeleted []string
	for _, f := range s.Unstaged {
		switch f.Type {
		case repository.StatusModified:
			unstagedModified = append(unstagedModified, f.Path)
		case repository.StatusDeleted:
			unstagedDeleted = append(unstagedDeleted, f.Path)
		}
	}

	hasStaged := len(stagedAdded) > 0 || len(stagedModified) > 0 || len(stagedDeleted) > 0
	hasUnstaged := len(unstagedModified) > 0 || len(unstagedDeleted) > 0
	hasUntracked := len(s.Untracked) > 0

	if hasStaged {
		sb.WriteString("\nCambios preparados para commit:\n")
		for _, p := range stagedAdded {
			sb.WriteString(fmt.Sprintf("  nuevo archivo:   %s\n", p))
		}
		for _, p := range stagedModified {
			sb.WriteString(fmt.Sprintf("  modificado:   %s\n", p))
		}
		for _, p := range stagedDeleted {
			sb.WriteString(fmt.Sprintf("  deleted:    %s\n", p))
		}
	}

	if hasUnstaged {
		sb.WriteString("\nCambios no preparados para commit:\n")
		for _, p := range unstagedModified {
			sb.WriteString(fmt.Sprintf("  modified:   %s\n", p))
		}
		for _, p := range unstagedDeleted {
			sb.WriteString(fmt.Sprintf("  deleted:    %s\n", p))
		}
	}

	if hasUntracked {
		sb.WriteString("\nArchivos no rastreados:\n")
		for _, p := range s.Untracked {
			sb.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	if !hasStaged && !hasUnstaged && !hasUntracked {
		sb.WriteString("El árbol de trabajo está limpio.\n")
	}

	return sb.String()
}
