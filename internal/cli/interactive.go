package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"minigit/internal/commands"
	"minigit/internal/repository"
)

// RunInteractiveMenu runs an interactive terminal menu when launched without arguments (or by double-clicking the EXE).
func RunInteractiveMenu() {
	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreen()
		fmt.Println("==================================================")
		fmt.Println("       MiniGit - Control de Versiones Local       ")
		fmt.Println("==================================================")

		currDir, _ := os.Getwd()
		fmt.Printf("Carpeta actual: %s\n", currDir)

		repoRoot, err := repository.DiscoverRepository(currDir)
		if err == nil {
			fmt.Printf("Estado: Repositorio activo en %s\n", repoRoot)
		} else {
			fmt.Println("Estado: No se ha inicializado un repositorio aquí.")
		}
		fmt.Println("--------------------------------------------------")

		fmt.Println("Selecciona una opción:")
		fmt.Println("  1. Inicializar un nuevo repositorio (minigit inicializar)")
		fmt.Println("  2. Ver estado actual (minigit estado)")
		fmt.Println("  3. Agregar archivos o carpetas (minigit agregar)")
		fmt.Println("  4. Guardar cambios con comentario (minigit comentario)")
		fmt.Println("  5. Ver historial de versiones (minigit historial)")
		fmt.Println("  6. Mostrar detalle de un commit (minigit mostrar)")
		fmt.Println("  7. Cambiar de rama o versión (minigit cambiar)")
		fmt.Println("  8. Crear nueva rama (minigit rama)")
		fmt.Println("  9. Restaurar un archivo (minigit recuperar)")
		fmt.Println("  H. Ver ayuda detallada de comandos")
		fmt.Println("  0. Salir")
		fmt.Println("--------------------------------------------------")
		fmt.Print("Opción (0-9 / H): ")

		input, _ := reader.ReadString('\n')
		opcion := strings.TrimSpace(strings.ToUpper(input))

		if opcion == "0" {
			fmt.Println("¡Hasta luego!")
			break
		}

		fmt.Println()
		switch opcion {
		case "1":
			fmt.Print("Ingresa la ruta de la carpeta a inicializar (o presiona ENTER para la carpeta actual): ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)
			if err := commands.RunInit(target); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "2":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				out, err := commands.RunStatus(repo)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Print(out)
				}
			}

		case "3":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el archivo o carpeta a agregar (ej: '.' para todo, o 'archivo.txt'): ")
				target, _ := reader.ReadString('\n')
				target = strings.TrimSpace(target)
				if target == "" {
					target = "."
				}
				if err := commands.RunAdd(repo, []string{target}); err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Printf("¡Se agregó '%s' correctamente al área de preparación!\n", target)
				}
			}

		case "4":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el comentario/mensaje para esta versión: ")
				msg, _ := reader.ReadString('\n')
				msg = strings.TrimSpace(msg)
				if msg == "" {
					fmt.Println("Error: El comentario no puede estar vacío.")
				} else {
					if err := commands.RunCommit(repo, msg); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
				}
			}

		case "5":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("¿Deseas formato simplificado en una línea? (s/n): ")
				ans, _ := reader.ReadString('\n')
				oneline := strings.HasPrefix(strings.ToLower(strings.TrimSpace(ans)), "s")
				if err := commands.RunLog(repo, oneline); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}

		case "6":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el hash del commit a ver (o ENTER para el último commit): ")
				hashVal, _ := reader.ReadString('\n')
				hashVal = strings.TrimSpace(hashVal)
				if err := commands.RunShow(repo, hashVal); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}

		case "7":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el nombre de la rama o el hash del commit al que deseas cambiar: ")
				target, _ := reader.ReadString('\n')
				target = strings.TrimSpace(target)
				if target == "" {
					fmt.Println("Error: Debes ingresar un destino.")
				} else {
					if err := commands.RunCheckout(repo, target); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
				}
			}

		case "8":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el nombre para la nueva rama (o ENTER para listar ramas): ")
				branchName, _ := reader.ReadString('\n')
				branchName = strings.TrimSpace(branchName)
				if err := commands.RunBranch(repo, branchName); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}

		case "9":
			if repo := getRepoOrPrompt(currDir); repo != nil {
				fmt.Print("Ingresa el nombre del archivo a recuperar: ")
				target, _ := reader.ReadString('\n')
				target = strings.TrimSpace(target)
				if target == "" {
					fmt.Println("Error: Debes especificar la ruta del archivo.")
				} else {
					if err := commands.RunRestore(repo, target, false); err != nil {
						fmt.Printf("Error: %v\n", err)
					} else {
						fmt.Printf("¡El archivo '%s' ha sido restaurado exitosamente!\n", target)
					}
				}
			}

		case "H":
			ShowGeneralHelp()

		default:
			fmt.Println("Opción no válida.")
		}

		pause(reader)
	}
}

func getRepoOrPrompt(currDir string) *repository.Repository {
	repoRoot, err := repository.DiscoverRepository(currDir)
	if err != nil {
		fmt.Println("No se encontró un repositorio .minigit en la carpeta actual ni en carpetas superiores.")
		fmt.Print("Selecciona la opción 1 en el menú para inicializar un repositorio aquí.")
		return nil
	}

	// Make sure relative operations work in discovered root
	if absRoot, err := filepath.Abs(repoRoot); err == nil {
		return repository.OpenRepository(absRoot)
	}
	return repository.OpenRepository(repoRoot)
}

func pause(reader *bufio.Reader) {
	fmt.Println("\n--------------------------------------------------")
	fmt.Print("Presiona ENTER para continuar...")
	reader.ReadString('\n')
}

func clearScreen() {
	// Prints ANSI clear screen escape sequence if supported
	fmt.Print("\033[H\033[2J")
}
