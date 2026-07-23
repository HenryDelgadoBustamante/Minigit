package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"minigit/internal/commands"
	"minigit/internal/repository"
)

// Execute runs the CLI application with command-line arguments.
func Execute(args []string) int {
	if len(args) == 0 {
		RunInteractiveShell()
		return 0
	}

	cmd := strings.ToLower(args[0])

	switch cmd {
	case "--help", "-help", "-h":
		ShowGeneralHelp()
		return 0

	case "--version", "-version", "-v", "version":
		fmt.Printf("minigit versión %s\n", Version)
		return 0

	case "help", "ayuda":
		if len(args) > 1 {
			ShowCommandHelp(args[1])
		} else {
			ShowGeneralHelp()
		}
		return 0

	case "init", "inicializar", "iniciar":
		targetDir := ""
		if len(args) > 1 {
			if args[1] == "--help" || args[1] == "-h" {
				ShowCommandHelp("init")
				return 0
			}
			targetDir = args[1]
		}
		if err := commands.RunInit(targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	// Commands that require an existing repository
	repoRoot, err := repository.DiscoverRepository(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	repo := repository.OpenRepository(repoRoot)

	switch cmd {
	case "add", "agregar", "añadir", "anadir":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			ShowCommandHelp("add")
			return 0
		}
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: nada especificado, nada añadido")
			return 1
		}
		if err := commands.RunAdd(repo, args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "status", "estado":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			ShowCommandHelp("status")
			return 0
		}
		out, err := commands.RunStatus(repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		fmt.Print(out)
		return 0

	case "commit", "comentario", "confirmar", "guardar":
		commitFlags := flag.NewFlagSet("commit", flag.ContinueOnError)
		msgFlag := commitFlags.String("m", "", "Mensaje de commit")
		help := commitFlags.Bool("help", false, "Mostrar ayuda")

		if err := commitFlags.Parse(args[1:]); err != nil {
			return 1
		}

		if *help {
			ShowCommandHelp("commit")
			return 0
		}

		commitMsg := *msgFlag
		if commitMsg == "" && len(commitFlags.Args()) > 0 {
			commitMsg = commitFlags.Args()[0]
		}

		if commitMsg == "" {
			fmt.Fprintln(os.Stderr, "error: el mensaje de commit es obligatorio (ej: minigit comentario \"mensaje\" o minigit commit -m \"mensaje\")")
			return 1
		}

		if err := commands.RunCommit(repo, commitMsg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "log", "historial":
		logFlags := flag.NewFlagSet("log", flag.ContinueOnError)
		oneline := logFlags.Bool("oneline", false, "Una sola línea por commit")
		help := logFlags.Bool("help", false, "Mostrar ayuda")

		if err := logFlags.Parse(args[1:]); err != nil {
			return 1
		}

		if *help {
			ShowCommandHelp("log")
			return 0
		}

		if err := commands.RunLog(repo, *oneline); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "show", "mostrar", "ver":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			ShowCommandHelp("show")
			return 0
		}
		hashPrefix := ""
		if len(args) > 1 {
			hashPrefix = args[1]
		}
		if err := commands.RunShow(repo, hashPrefix); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "restore", "recuperar", "restaurar":
		restoreFlags := flag.NewFlagSet("restore", flag.ContinueOnError)
		staged := restoreFlags.Bool("staged", false, "Restaurar index de preparación")
		help := restoreFlags.Bool("help", false, "Mostrar ayuda")

		if err := restoreFlags.Parse(args[1:]); err != nil {
			return 1
		}

		if *help {
			ShowCommandHelp("restore")
			return 0
		}

		remaining := restoreFlags.Args()
		if len(remaining) == 0 {
			fmt.Fprintln(os.Stderr, "error: se requiere la ruta de archivo para recuperar")
			return 1
		}

		if err := commands.RunRestore(repo, remaining[0], *staged); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "checkout", "cambiar":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			ShowCommandHelp("checkout")
			return 0
		}
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: se requiere el nombre de la rama o el hash del commit")
			return 1
		}
		if err := commands.RunCheckout(repo, args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	case "branch", "rama", "ramas":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			ShowCommandHelp("branch")
			return 0
		}
		targetBranch := ""
		if len(args) > 1 {
			targetBranch = args[1]
		}
		if err := commands.RunBranch(repo, targetBranch); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "error: comando desconocido '%s'. Ejecuta 'minigit ayuda' para ver los comandos disponibles.\n", cmd)
		return 1
	}
}
