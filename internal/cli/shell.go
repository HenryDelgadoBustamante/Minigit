package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"minigit/internal/repository"
)

// RunInteractiveShell starts an interactive shell (REPL) like Git Bash when minigit is opened directly.
func RunInteractiveShell() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==========================================================")
	fmt.Println("       MiniGit Console v" + Version + " [Local Version Control]")
	fmt.Println("==========================================================")
	fmt.Println("Escribe comandos directamente (ej: 'estado', 'agregar .', 'comentario \"mensaje\"').")
	fmt.Println("Comandos especiales: 'cd <carpeta>', 'ls', 'cls', 'ayuda', 'salir'.")
	fmt.Println()

	for {
		prompt := getPromptString()
		fmt.Print(prompt)

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		args := parseCommandLine(line)
		if len(args) == 0 {
			continue
		}

		cmd := strings.ToLower(args[0])

		// Handle shell built-ins
		if cmd == "exit" || cmd == "salir" || cmd == "quit" {
			fmt.Println("¡Hasta luego!")
			break
		}

		if cmd == "clear" || cmd == "cls" {
			fmt.Print("\033[H\033[2J")
			continue
		}

		if cmd == "pwd" {
			currDir, _ := os.Getwd()
			fmt.Println(currDir)
			continue
		}

		if cmd == "cd" {
			targetDir := "."
			if len(args) > 1 {
				targetDir = args[1]
			}
			if err := os.Chdir(targetDir); err != nil {
				fmt.Printf("cd error: %v\n", err)
			} else {
				curr, _ := os.Getwd()
				fmt.Printf("Carpeta actual: %s\n", curr)
			}
			continue
		}

		if cmd == "ls" || cmd == "dir" {
			currDir, _ := os.Getwd()
			entries, err := os.ReadDir(currDir)
			if err != nil {
				fmt.Printf("ls error: %v\n", err)
			} else {
				for _, entry := range entries {
					if entry.IsDir() {
						fmt.Printf("  [DIR]  %s\n", entry.Name())
					} else {
						fmt.Printf("         %s\n", entry.Name())
					}
				}
			}
			continue
		}

		// If user typed "minigit <cmd>", strip the leading "minigit"
		if cmd == "minigit" || cmd == "minigit.exe" {
			if len(args) == 1 {
				ShowGeneralHelp()
				continue
			}
			args = args[1:]
		}

		// Dispatch command
		Execute(args)
		fmt.Println()
	}
}

func getPromptString() string {
	currDir, _ := os.Getwd()
	repoRoot, err := repository.DiscoverRepository(currDir)
	if err != nil {
		dirName := filepath.Base(currDir)
		return fmt.Sprintf("minigit [%s]> ", dirName)
	}

	head, err := repository.ReadHEAD(repoRoot)
	branchLabel := "main"
	if err == nil {
		if head.Type == repository.HEADTypeBranch {
			branchLabel = head.Branch
		} else if len(head.Commit) >= 7 {
			branchLabel = head.Commit[:7]
		}
	}

	dirName := filepath.Base(currDir)
	return fmt.Sprintf("minigit [%s] (%s)> ", dirName, branchLabel)
}

// parseCommandLine splits a command line string into arguments, handling quoted strings correctly.
func parseCommandLine(line string) []string {
	var args []string
	var current strings.Builder
	inDoubleQuotes := false
	inSingleQuotes := false

	for i := 0; i < len(line); i++ {
		r := line[i]

		switch r {
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			} else {
				current.WriteByte(r)
			}
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			} else {
				current.WriteByte(r)
			}
		case ' ', '\t':
			if inDoubleQuotes || inSingleQuotes {
				current.WriteByte(r)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteByte(r)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
