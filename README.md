# MiniGit - Local Version Control System CLI in Go

**MiniGit** es un sistema de control de versiones local desarrollado íntegramente en **Go**, inspirado en la arquitectura conceptual de Git. Ha sido diseñado de forma modular, segura, sin dependencias externas innecesarias (usando únicamente la biblioteca estándar de Go) y portable para Windows, Linux y macOS.

---

## 🚀 Características principales

- **Almacenamiento de objetos inmutables**: Blobs, Trees y Commits con compresión `zlib` y direccionamiento por contenido mediante hashing **SHA-256**.
- **Serialización determinista**: Ordenamiento de árboles y fechas en UTC (RFC3339) para garantizar la irrepetibilidad exacta de checksums.
- **Área de preparación (Staging Index)**: Control de estado persistente atómico en `.minigit/index`.
- **Ramas y HEAD**: Punteros a ramas (`refs/heads/*`) y soporte para estado `detached HEAD`.
- **Seguridad rigurosa**:
  - Rechazo estricto de trayectorias con salto de directorio (`..`), rutas absolutas y bytes nulos.
  - Bloqueo de concurrencia mediante archivos `.lock`.
  - Escrituras atómicas con sincronización a disco (`Sync`).
  - Verificación de integridad SHA-256 al leer cualquier objeto.
- **Reglas `.minigitignore`**: Ignorado inteligente con soporte para comodines (`*`, `?`), comentarios, extensiones y negaciones (`!`).
- **Pruebas integrales**: Cobertura completa de casos unitarios e integración CLI.

---

## 📋 Requisitos

- **Go**: Versión 1.20 o superior.

---

## 🛠 Compilación e Instalación

### Compilación usando `go build`

```bash
# Compilar el binario executable
go build -o minigit ./cmd/minigit
```

En Windows (PowerShell / Command Prompt):
```powershell
go build -o minigit.exe .\cmd\minigit
```

### Compilación usando `Makefile`

```bash
make build
```

---

## 💻 Uso de Comandos

| Comando | Descripción |
| :--- | :--- |
| `minigit init [dir]` | Inicializa un nuevo repositorio MiniGit. |
| `minigit add <file\|dir\|.>` | Añade archivos o directorios al área de preparación (index). |
| `minigit status` | Muestra el estado del árbol de trabajo, archivos preparados y no rastreados. |
| `minigit commit -m "mensaje"` | Guarda una nueva instantánea (commit) de los archivos preparados. |
| `minigit log [--oneline]` | Muestra el historial de commits desde HEAD. |
| `minigit show <hash>` | Muestra la información de un commit y el resumen de sus cambios. |
| `minigit restore [--staged] <file>` | Restaura un archivo en el directorio de trabajo o en el index desde HEAD. |
| `minigit checkout <hash\|rama>` | Cambia de rama o restaura el área de trabajo a un commit anterior. |
| `minigit branch [nombre]` | Lista las ramas existentes o crea una nueva rama desde HEAD. |
| `minigit help [comando]` | Muestra la ayuda general o detallada de un comando. |
| `minigit version` | Muestra la versión actual de la herramienta. |

---

## 💡 Ejemplo Completo de Flujo

### En Linux / macOS (Bash)

```bash
# 1. Inicializar repositorio
./minigit init proyecto-demo
cd proyecto-demo

# 2. Crear archivos
echo "Hola mundo" > saludo.txt
echo "log data" > app.log

# 3. Consultar estado
../minigit status

# 4. Preparar archivos
../minigit add .

# 5. Confirmar cambios
../minigit commit -m "Primer commit del proyecto"

# 6. Consultar historial
../minigit log --oneline

# 7. Crear y cambiar a nueva rama
../minigit branch feature
../minigit checkout feature

# 8. Modificar archivo
echo "Nueva línea en feature" >> saludo.txt
../minigit add saludo.txt
../minigit commit -m "Actualización en rama feature"

# 9. Regresar a rama principal
../minigit checkout main
```

### En Windows (PowerShell)

```powershell
# 1. Inicializar repositorio
..\minigit.exe init proyecto-demo
cd proyecto-demo

# 2. Crear archivos
Set-Content -Path saludo.txt -Value "Hola mundo"

# 3. Preparar y hacer commit
..\minigit.exe add .
..\minigit.exe commit -m "Primer commit en Windows"

# 4. Ver historial y estado
..\minigit.exe log
..\minigit.exe status
```

---

## 🏗 Arquitectura Interna

El proyecto sigue una estructura limpia en capas dirigidas (Command Handler → Repository Domain → Object/Storage Domain → Filesystem):

```text
minigit/
├── cmd/
│   └── minigit/
│       └── main.go          # Punto de entrada principal
├── internal/
│   ├── cli/                 # Parseo de argumentos y enrutamiento CLI
│   ├── commands/            # Implementación de comandos (init, add, commit, etc.)
│   ├── repository/          # Lógica de dominio, index, refs, HEAD, ignore y locks
│   ├── object/              # Modelos de objetos inmutables (blob, tree, commit)
│   ├── storage/             # Almacenamiento, compresión zlib, hashing SHA-256 y escrituras atómicas
│   └── filesystem/          # Validaciones de seguridad de rutas y caminante de directorios
├── minigit_integration_test.go # Pruebas de integración CLI de extremo a extremo
├── Makefile
├── README.md
├── LICENSE
└── .gitignore
```

---

## 📦 Formato de Objetos y Hashing

Cada objeto se almacena comprimido en `.minigit/objects/ab/cdef1234...` con la cabecera:

```text
<tipo> <tamaño>\x00<contenido>
```

- **Blob**: `blob <tamaño>\x00<datos-del-archivo>`
- **Tree**: `tree <tamaño>\x00<modo> <tipo> <hash> <nombre>\n...` (ordenado alfabéticamente de forma determinista).
- **Commit**: `commit <tamaño>\x00tree <hash>\nparent <hash>\nauthor <nombre> <<email>> <timestamp>\n\n<mensaje>`

---

## 🔒 Seguridad e Integridad

1. **Rutas seguras**: Previene vulnerabilidades de Path Traversal bloqueando referencias `..`, rutas absolutas y symlinks dirigidos fuera de la raíz del repositorio.
2. **Escrituras atómicas**: La modificación de archivos críticos (index, HEAD, refs) escribe primero en un archivo temporal en el mismo directorio, ejecuta `.Sync()` y realiza un renombrado atómico (`os.Rename`).
3. **Control de concurrencia**: Crea archivos `.lock` de manera exclusiva (`O_EXCL`) para evitar escrituras simultáneas corruptas.
4. **Verificación checksum**: Se recalcula el SHA-256 de cada objeto al ser leído; cualquier discrepancia devuelve un error descriptivo de corrupción.

---

## 🧪 Pruebas

Para ejecutar la suite completa de pruebas unitarias e integración:

```bash
make check
```

O manualmente:

```bash
gofmt -s -w .
go vet ./...
go test -v ./...
```

---

## ⚖️ Diferencias respecto a Git

- **Hash Standard**: MiniGit utiliza **SHA-256** (64 hex chars) en lugar de SHA-1.
- **Index**: El índice de MiniGit utiliza una estructura JSON estructurada y atómica determinista.
- **Configuración de Autor**: Se configura mediante variables de entorno (`MINIGIT_AUTHOR_NAME`, `MINIGIT_AUTHOR_EMAIL`) o valores predeterminados explícitos.
