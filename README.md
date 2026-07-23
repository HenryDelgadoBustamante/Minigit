# MiniGit - Sistema de Control de Versiones Local en Go

**MiniGit** es un sistema de control de versiones local e inmutable desarrollado íntegramente en **Go**, inspirado en la arquitectura interna de Git. Ha sido diseñado con una arquitectura modular, segura, determinista y portable (compatible con Windows, Linux y macOS), utilizando **únicamente la biblioteca estándar de Go** sin dependencias externas.

---

## 📋 Requisitos del Sistema

- **Lenguaje**: Go (versión 1.20 o superior).
- **Sistema Operativo**: Windows, Linux o macOS.
- **Sin dependencias externas**: Compila a un binario ejecutable nativo e independiente.

---

## 🛠 Compilación e Instalación

### Compilación directa con `go build`

```bash
# Compilar binario ejecutable en Linux / macOS
go build -o minigit ./cmd/minigit
```

En Windows (PowerShell / Command Prompt):
```powershell
# Compilar binario ejecutable en Windows
go build -o minigit.exe .\cmd\minigit
```

### Compilación usando `Makefile`

```bash
make build
```

Para ejecutar la suite completa de pruebas unitarias y de integración:
```bash
go test -v ./...
```

---

## 💻 Guía de Uso y Comandos CLI

MiniGit ofrece soporte para comandos en inglés y sus correspondientes alias en español para mayor accesibilidad.

| Comando Principal | Alias en Español | Descripción |
| :--- | :--- | :--- |
| `minigit init [dir]` | `inicializar`, `iniciar` | Inicializa un nuevo repositorio MiniGit en el directorio actual o especificado. |
| `minigit add <archivos...>` | `agregar`, `añadir` | Añade archivos o directorios al área de preparación (*staging index*). |
| `minigit status` | `estado` | Muestra el estado del árbol de trabajo, archivos preparados, modificados o no rastreados. |
| `minigit commit -m "msg"` | `confirmar`, `guardar` | Registra una nueva instantánea del repositorio con el mensaje especificado. |
| `minigit log [--oneline]` | `historial` | Muestra el historial cronológico de commits desde la posición actual de `HEAD`. |
| `minigit show <hash\|prefijo>` | `mostrar`, `ver` | Muestra los metadatos y contenido de un commit, árbol u objeto por hash completo o prefijo corto. |
| `minigit restore [--staged] <file>` | `recuperar`, `restaurar` | Restaura un archivo en el directorio de trabajo o lo saca del área de preparación. |
| `minigit checkout <rama\|hash>` | `cambiar` | Cambia a una rama existente o coloca el repositorio en estado *detached HEAD*. |
| `minigit branch [nombre]` | `rama`, `ramas` | Lista las ramas existentes o crea una nueva rama apuntando al commit actual. |
| `minigit diff <commit1> [commit2]`| `diferencias` | *(Opcional)* Compara cambios de estructura (A, M, D, R) y líneas de contenido (`-` / `+`) entre dos commits. |
| `minigit merge <rama>` | `fusionar`, `unir` | *(Opcional)* Realiza una fusión *fast-forward* de la rama especificada en la rama activa. |
| `minigit help [comando]` | `ayuda` | Muestra la ayuda general o la documentación detallada de un comando específico. |
| `minigit version` | | Muestra la versión actual instalada de MiniGit. |

---

## 💡 Ejemplos Prácticos de Uso

### Flujo de Trabajo Básico (PowerShell / Bash)

```bash
# 1. Inicializar un repositorio
minigit init mi-proyecto
cd mi-proyecto

# 2. Crear archivos
echo "Hola MiniGit" > README.md
mkdir src
echo "package main" > src/main.go

# 3. Consultar el estado inicial
minigit status

# 4. Preparar todos los archivos en el index
minigit add .

# 5. Registrar el primer commit
minigit commit -m "Initial commit: estructura base del proyecto"

# 6. Consultar el historial de confirmaciones
minigit log --oneline

# 7. Crear y cambiar a una nueva rama de desarrollo
minigit branch feature/login
minigit checkout feature/login

# 8. Modificar un archivo y confirmar en la nueva rama
echo "// Nueva funcion de autenticacion" >> src/main.go
minigit add src/main.go
minigit commit -m "feat: agregar autenticacion"
```

### Funciones Complementarias Opcionales (`diff`, `merge`, prefijos corto de hash)

```bash
# Comparación entre dos commits (diferencias de estructura A, M, D, R y líneas + / -)
minigit diff a8f42c 1f82c2

# Inspección de objeto usando un prefijo corto de hash (ejemplo: primeros 6 caracteres)
minigit show a8f42c

# Fusión Fast-Forward de la rama feature/login a la rama main
minigit checkout main
minigit merge feature/login
# Salida esperada: Fast-forward realizado correctamente.
```

---

## 🚀 Funcionalidades Complementarias (EPIC 15)

MiniGit incluye las siguientes mejoras complementarias opcionales desarrolladas para optimizar la experiencia de uso:

1. **Comparación de Commits (`minigit diff`)**:
   - Compara los árboles raíz de dos commits.
   - Identifica archivos agregados (`A`), modificados (`M`), eliminados (`D`) y renombrados (`R`).
   - Muestra diferencias línea por línea (`-` / `+`) para archivos de texto y advierte si los archivos son binarios.
2. **Detección Básica de Renombrados**:
   - Si un archivo eliminado y uno agregado poseen exactamente el mismo hash SHA-256 de Blob, MiniGit lo clasifica automáticamente como un renombrado: `R  anterior.txt -> nuevo.txt`.
3. **Fusión Fast-Forward (`minigit merge`)**:
   - Permite integrar ramas cuando la rama destino es descendiente directa de la rama actual.
   - Verifica la cadena de ancestría antes de mover punteros.
   - **Rechazo seguro de fusiones divergentes**: Si las ramas han divergido (contienen commits independientes), la operación es rechazada sin alterar el área de trabajo ni el índice (`No se puede realizar fast-forward: las ramas han divergido.`).
4. **Resolución de Objetos mediante Prefijos de Hash**:
   - Acepta identificadores prefijados de al menos 4 caracteres hexadecimales.
   - Si el prefijo coincide con un único objeto, lo resuelve automáticamente.
   - En caso de múltiples coincidencias, genera un error explícito de ambigüedad (`El prefijo indicado es ambiguo`).

---

## 📁 Estructura Interna del Repositorio (`.minigit`)

```text
.minigit/
├── config            # Archivo de configuración local del repositorio
├── HEAD              # Referencia al commit actual o a la rama activa
├── index             # Estado persistente del área de preparación (JSON)
├── logs/             # Directorio para registros de actividades y depuración
├── objects/          # Almacenamiento direccionable por contenido (Blobs, Trees, Commits)
│   ├── 0a/
│   ├── 4f/
│   └── ...
└── refs/
    └── heads/        # Punteros a ramas del repositorio (ej: main, dev)
        ├── main
        └── feature
```

---

## 🏗 Arquitectura y Almacenamiento Interno

Consulte la documentación en la carpeta `docs/`:
- 📄 [Formato de Objetos (Blob, Tree, Commit)](file:///c:/Users/ROBERT/Documents/sipan/IX/lenguaje%20taller/Minigit/docs/object-format.md)
- 📄 [Grafo de Merkle e Integridad Crypto](file:///c:/Users/ROBERT/Documents/sipan/IX/lenguaje%20taller/Minigit/docs/merkle-graph.md)
- 📄 [Bitácora Técnica de Desarrollo](file:///c:/Users/ROBERT/Documents/sipan/IX/lenguaje%20taller/Minigit/docs/technical-log.md)

---

## ⚖️ Diferencias entre MiniGit y Git

| Característica | Git | MiniGit |
| :--- | :--- | :--- |
| **Algoritmo de Hash** | SHA-1 / SHA-256 | **SHA-256 (Nativo)** |
| **Formato del Index** | Binario estricto (`.git/index`) | **JSON estructurado** |
| **Almacenamiento** | Loose Objects + Packfiles binarios | **Loose Objects comprimidos con zlib** |
| **Fusión de Ramas** | Merge de 3 vías, rebase, resolución de conflictos | **Fusión Fast-Forward (Rechaza divergencias)** |
| **Sistemas Remotos** | `push`, `pull`, `fetch`, `clone` | **Operaciones 100% locales** |
| **Lenguaje de Desarrollo** | C / Shell / Perl | **Go 100% Estándar** |

---

## 📄 Licencia y Créditos

Proyecto desarrollado con fines académicos para la asignatura de Taller de Lenguajes de Programación. Distribuido bajo la Licencia **MIT**.
