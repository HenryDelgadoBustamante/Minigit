package cli

import (
	"fmt"
	"strings"
)

const Version = "1.0.0"

func ShowGeneralHelp() {
	helpText := `MiniGit - Sistema de Control de Versiones Local (v` + Version + `)

Uso:
  minigit <comando> [argumentos]
  minigit [--help | --version]

Comandos disponibles (Inglés y Español):
  init | inicializar [directorio]         Inicializa un nuevo repositorio MiniGit
  add  | agregar <archivo|dir|.>           Prepara archivos en el área de ensayo
  status | estado                         Muestra el estado del árbol de trabajo
  commit | comentario -m "msg" | "msg"    Registra una instantánea de cambios
  log | historial [--oneline]             Muestra el historial de commits
  show | mostrar <hash>                   Muestra detalles de un commit y archivos modificados
  restore | recuperar | restaurar <file>  Restaura archivos del espacio de trabajo o index
  checkout | cambiar <hash|rama>          Cambia de rama o restaura el estado a un commit
  branch | rama [nombre]                  Lista o crea ramas
  help | ayuda [comando]                  Muestra la ayuda de un comando
  version                                 Muestra la versión actual

Ejemplos en español:
  minigit inicializar mi-proyecto
  minigit agregar .
  minigit estado
  minigit comentario "Primer commit"
  minigit historial --oneline
  minigit recuperar archivo.txt
  minigit rama nueva-rama
  minigit cambiar nueva-rama
`
	fmt.Print(helpText)
}

func ShowCommandHelp(command string) {
	switch strings.ToLower(command) {
	case "init", "inicializar", "iniciar":
		fmt.Println(`Uso: minigit init [directorio]
     minigit inicializar [directorio]

Inicializa un nuevo repositorio MiniGit. Crea la estructura de carpetas .minigit.

Ejemplos:
  minigit inicializar
  minigit inicializar mi-proyecto`)

	case "add", "agregar", "añadir", "anadir":
		fmt.Println(`Uso: minigit add <archivo|directorio|.>
     minigit agregar <archivo|directorio|.>

Añade contenido de archivos al área de preparación (index).

Ejemplos:
  minigit agregar main.go
  minigit agregar src/
  minigit agregar .`)

	case "status", "estado":
		fmt.Println(`Uso: minigit status
     minigit estado

Muestra el estado del directorio de trabajo, archivos preparados, modificados y no rastreados.`)

	case "commit", "comentario", "confirmar", "guardar":
		fmt.Println(`Uso: minigit commit -m "mensaje"
     minigit comentario "mensaje"
     minigit confirmar -m "mensaje"

Guarda el contenido del index en un nuevo commit con un mensaje descriptivo.

Ejemplos:
  minigit comentario "Mi primer commit"
  minigit commit -m "Soluciona error de compilación"`)

	case "log", "historial":
		fmt.Println(`Uso: minigit log [--oneline]
     minigit historial [--oneline]

Muestra el historial de commits desde HEAD.

Opciones:
  --oneline    Muestra cada commit en una sola línea`)

	case "show", "mostrar", "ver":
		fmt.Println(`Uso: minigit show <hash>
     minigit mostrar <hash>

Muestra información detallada de un commit y el resumen de sus cambios.`)

	case "restore", "recuperar", "restaurar":
		fmt.Println(`Uso: minigit restore [--staged] <archivo>
     minigit recuperar [--staged] <archivo>

Restaura archivos en el directorio de trabajo o en el área de preparación.

Ejemplos:
  minigit recuperar archivo.txt
  minigit recuperar --staged archivo.txt`)

	case "checkout", "cambiar":
		fmt.Println(`Uso: minigit checkout <hash-o-rama>
     minigit cambiar <hash-o-rama>

Cambia a la rama especificada o restaura el área de trabajo al estado del commit indicado.`)

	case "branch", "rama", "ramas":
		fmt.Println(`Uso: minigit branch [nombre-rama]
     minigit rama [nombre-rama]

Lista las ramas existentes (marcando la activa con *) o crea una nueva rama desde HEAD.`)

	case "version":
		fmt.Printf("MiniGit versión %s\n", Version)

	default:
		fmt.Printf("Comando desconocido '%s'. Ejecuta 'minigit ayuda' para ver los comandos disponibles.\n", command)
	}
}
