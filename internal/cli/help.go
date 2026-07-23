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
  show | mostrar <hash>                   Muestra detalles de un commit, objeto o prefijo
  restore | recuperar | restaurar <file>  Restaura archivos del espacio de trabajo o index
  checkout | cambiar <hash|rama>          Cambia de rama o restaura el estado a un commit
  branch | rama [nombre]                  Lista o crea ramas
  diff | diferencias <commit1> <commit2>  Compara cambios estructurales y líneas entre commits
  merge | fusionar <rama>                 Realiza fusión fast-forward de la rama especificada
  web | ui | visor [puerto]               Inicia el visor web local de solo lectura
  help | ayuda [comando]                  Muestra la ayuda de un comando
  version                                 Muestra la versión actual

Ejemplos en español:
  minigit inicializar mi-proyecto
  minigit agregar .
  minigit estado
  minigit comentario "Primer commit"
  minigit historial --oneline
  minigit diferencias HEAD~1 HEAD
  minigit fusionar feature-login
  minigit web 8080
`
	fmt.Print(helpText)
}

func ShowCommandHelp(command string) {
	switch strings.ToLower(command) {
	case "init", "inicializar", "iniciar":
		fmt.Println(`Uso: minigit init [directorio]
     minigit inicializar [directorio]

Inicializa un nuevo repositorio MiniGit. Crea la estructura de carpetas .minigit.`)

	case "add", "agregar", "añadir", "anadir":
		fmt.Println(`Uso: minigit add <archivo|directorio|.>
     minigit agregar <archivo|directorio|.>

Añade contenido de archivos al área de preparación (index).`)

	case "status", "estado":
		fmt.Println(`Uso: minigit status
     minigit estado

Muestra el estado del directorio de trabajo, archivos preparados, modificados y no rastreados.`)

	case "commit", "comentario", "confirmar", "guardar":
		fmt.Println(`Uso: minigit commit -m "mensaje"
     minigit comentario "mensaje"

Guarda el contenido del index en un nuevo commit con un mensaje descriptivo.`)

	case "log", "historial":
		fmt.Println(`Uso: minigit log [--oneline]
     minigit historial [--oneline]

Muestra el historial de commits desde HEAD.`)

	case "show", "mostrar", "ver":
		fmt.Println(`Uso: minigit show <hash-o-prefijo>
     minigit mostrar <hash-o-prefijo>

Muestra información detallada de un objeto (Blob, Tree o Commit) mediante su hash o prefijo.`)

	case "restore", "recuperar", "restaurar":
		fmt.Println(`Uso: minigit restore [--staged] <archivo>
     minigit recuperar [--staged] <archivo>

Restaura archivos en el directorio de trabajo o en el área de preparación.`)

	case "checkout", "cambiar":
		fmt.Println(`Uso: minigit checkout <hash-o-rama>
     minigit cambiar <hash-o-rama>

Cambia a la rama especificada o restaura el área de trabajo al estado del commit indicado.`)

	case "branch", "rama", "ramas":
		fmt.Println(`Uso: minigit branch [nombre-rama]
     minigit rama [nombre-rama]

Lista las ramas existentes o crea una nueva rama desde HEAD.`)

	case "diff", "diferencias":
		fmt.Println(`Uso: minigit diff <commit-1> [commit-2]
     minigit diferencias <commit-1> [commit-2]

Compara dos commits e identifica archivos agregados (A), modificados (M), eliminados (D) y renombrados (R), además de mostrar las líneas agregadas (+) y eliminadas (-).`)

	case "merge", "fusionar", "unir":
		fmt.Println(`Uso: minigit merge <rama-destino>
     minigit fusionar <rama-destino>

Realiza una fusión fast-forward de la rama especificada en la rama actual. Si las ramas han divergido, la operación será rechazada.`)

	case "web", "ui", "visor":
		fmt.Println(`Uso: minigit web [puerto]
     minigit ui [puerto]
     minigit visor [puerto]

Inicia un servidor web local y de solo lectura para inspeccionar el estado, historial, ramas, archivos y diferencias del repositorio mediante una interfaz gráfica. Por defecto utiliza el puerto 8080.`)

	case "version":
		fmt.Printf("MiniGit versión %s\n", Version)

	default:
		fmt.Printf("Comando desconocido '%s'. Ejecuta 'minigit ayuda' para ver los comandos disponibles.\n", command)
	}
}
