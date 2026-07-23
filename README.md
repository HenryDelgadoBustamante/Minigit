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

El proyecto sigue una arquitectura modular en capas dirigidas donde la interfaz de usuario se desacopla completamente de la lógica del dominio:

```text
Usuario
   │
   ▼
CLI
   │
   ▼
Commands
   │
   ▼
Repository
   ├── Working Tree
   ├── Index
   ├── HEAD
   ├── Refs
   ├── Ignore
   └── Object Store
         │
         ├── Object
         ├── Storage
         └── Filesystem
```

### Responsabilidad de cada paquete

- **`cli`**: Parseo de argumentos de línea de comandos, manejo de banderas generales y enrutamiento hacia la capa de comandos.
- **`commands`**: Capa de presentación y adaptación CLI. Valida argumentos de comandos específicos, interpreta opciones de usuario, lee variables de entorno, invoca operaciones del paquete `repository` y da formato visible a la salida o errores.
- **`repository`**: Núcleo del dominio de control de versiones. Coordina las operaciones entre el **Working Tree**, **Index**, **HEAD**, **Refs**, **Ignore** y el **Object Store**.
- **`object`**: Modelos de dominio e inmutabilidad de objetos (`Blob`, `Tree`, `Commit`).
- **`storage`**: Almacenamiento físico de objetos, compresión `zlib`, hashing SHA-256 y escrituras atómicas.
- **`filesystem`**: Operaciones físicas seguras del sistema de archivos, validación estricta contra Path Traversal y recorrido recursivo de directorios.

### Conceptos clave del repositorio

1. **Working Tree (Árbol de Trabajo)**: Los archivos y directorios físicos presentes en el disco del usuario.
2. **Index (Área de Preparación / Staging)**: Caché atómica intermedia (`.minigit/index`) que registra los estados preparados listos para ser capturados en el próximo commit.
3. **HEAD**: Puntero principal que indica el estado actual del repositorio, ya sea hacia una rama activa (`refs/heads/<branch>`) o en estado *detached HEAD* apuntando directamente a un hash de commit.
4. **Refs**: Referencias persistentes en el disco (`.minigit/refs/heads/*`) que asocian nombres de ramas con hashes de commits.
5. **Object Store**: Almacén de contenido inmutable (`.minigit/objects/`) donde cada archivo (`blob`), estructura de directorio (`tree`) e instantánea (`commit`) se guarda comprimido y direccionado por su SHA-256.

---

## 📦 Almacenamiento Direccionable por Contenido (Object Store)

MiniGit implementa un almacenamiento inmutable direccionado por contenido en `.minigit/objects/`.

### 1. Generación de Hashes SHA-256
Cada objeto en MiniGit (`blob`, `tree`, `commit`) se identifica unívocamente mediante un hash **SHA-256** de 64 caracteres hexadecimales en minúsculas. El hash se calcula sobre el contenido binario serializado completo, compuesto por la cabecera estándar de Git y el cuerpo:

```text
<tipo> <tamaño>\x00<contenido>
```

- **Determinismo**: El mismo contenido genera exactamente el mismo hash de 64 caracteres, garantizando desduplicación absoluta.
- **Tipos de objetos**:
  - **Blob**: `blob <tamaño>\x00<datos-del-archivo>`
  - **Tree**: `tree <tamaño>\x00<modo> <tipo> <hash> <nombre>\n...` (ordenado deterministamente).
  - **Commit**: `commit <tamaño>\x00tree <hash>\nparent <hash>\nauthor <nombre> <<email>> <timestamp>\n\n<mensaje>`

### 2. Estructura de Almacenamiento y Desduplicación
Los objetos se persisten físicamente utilizando los dos primeros caracteres del hash como nombre del subdirectorio y los 62 caracteres restantes como el nombre del archivo:

```text
.minigit/objects/
├── 4b/
│   └── 825dc642cb6eb9a060e54bf8d69288fbee4904ce8e32906b3a0f7...
└── e3/
    └── b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b...
```

- **Desduplicación**: Antes de escribir un nuevo objeto, `ObjectStore` verifica si la trayectoria `.minigit/objects/xx/yyyy...` ya existe. Si el archivo ya está en disco, se omite la escritura.
- **Inmutabilidad**: Los objetos se guardan con permisos de solo lectura (`0444`) para prevenir modificaciones accidentales.

### 3. Compresión zlib
Para optimizar el espacio en disco, todo objeto es comprimido mediante `zlib` antes de persistirlo en el sistema de archivos:
- **Escritura**: El payload completo (`cabecera + cuerpo`) pasa por `zlib.Writer` y se escribe de manera atómica (`WriteFileAtomic`).
- **Lectura**: Al recuperar un objeto, se descomprime mediante `zlib.Reader`. Si los datos en disco están truncados o no son un flujo `zlib` válido, se detecta inmediatamente un error de corrupción.

### 4. Recuperación y Resolución de Hashes Cortos
MiniGit permite recuperar objetos proporcionando:
- El hash SHA-256 completo (64 caracteres).
- Un prefijo hexadecimal abreviado (mínimo 4 caracteres). Si el prefijo coincide con un único objeto en el subdirectorio `.minigit/objects/xx/`, se resuelve de forma transparente al hash completo. Si coincide con múltiples objetos, se reporta un error de hash ambiguo.

### 5. Verificación de Integridad y Detección de Errores
- **Detección de objetos inexistentes**: Si se solicita un hash que no está presente en el repositorio, MiniGit retorna un error claro `ErrObjectNotFound` evitando fallos o colapsos del sistema.
- **Detección de objetos corruptos**: Durante cada lectura (`ReadObject`), MiniGit ejecuta una verificación de integridad en dos pasos:
  1. **Descompresión zlib**: Valida el formato y la integridad del archivo comprimido.
  2. **Recálculo de checksum SHA-256**: Recalcula el SHA-256 sobre los datos descomprimidos y los compara contra el hash esperado. Si existe cualquier discrepancia, no se entrega información alterada y se reporta inmediatamente un error de integridad (`ErrCorruptObject`).

### 6. Modelo de Objetos y Grafo de Relaciones

MiniGit organiza los datos del usuario mediante un grafo acíclico dirigido (DAG) de tres objetos inmutables principales:

```text
               ┌──────────────┐
               │    Commit    │  (Autor, fecha, mensaje)
               └──────┬───────┘
                      │ puntos a
                      ▼
               ┌──────────────┐
               │  Tree Raíz   │  (Directorio principal)
               └──────┬───────┘
                      ├───────────────┐
                      ▼               ▼
               ┌──────────────┐┌──────────────┐
               │   Subtree    ││  Blob (file) │  (Contenido crudo)
               └──────┬───────┘└──────────────┘
                      ▼
               ┌──────────────┐
               │  Blob (file) │
               └──────────────┘
```

#### A. Objeto Blob (`blob`)
Almacena únicamente el contenido crudo de un archivo. No conserva metadatos como el nombre o los permisos del archivo (estos son gestionados por el objeto `Tree`).

- **Serialización determinista**:
  ```text
  blob <tamaño_bytes>\x00<contenido_crudo_del_archivo>
  ```
- **Ejemplo**: Un archivo con `"Hola MiniGit"` (12 bytes) se serializa como:
  ```text
  blob 12\x00Hola MiniGit
  ```

#### B. Objeto Tree (`tree`)
Representa la estructura de un directorio. Almacena punteros a objetos `blob` (archivos) y otros `tree` (subdirectorios), asociándoles un nombre de archivo y modo de permisos octal (`100644` para archivos estándar, `100755` para ejecutables, `040000` para subdirectorios).

- **Serialización determinista**: Las entradas se ordenan alfabéticamente por `Name` (y secuencialmente por `Hash` en caso de nombres idénticos).
  ```text
  tree <tamaño_bytes>\x00<modo> <tipo> <hash_sha256> <nombre>\n...
  ```
- **Ejemplo**:
  ```text
  tree 118\x00100644 blob e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 archivo.txt
  100755 blob 4b825dc642cb6eb9a060e54bf8d69288fbee4904ce8e32906b3a0f7902d0eb2d script.sh
  ```

#### C. Objeto Commit (`commit`)
Captura una instantánea del estado del repositorio en un instante del tiempo. Contiene la referencia al `Tree` raíz, la referencia al commit `Parent` (si existe), los datos del autor (`Nombre <email> <RFC3339_timestamp>`) y el mensaje descriptivo.

- **Serialización determinista**: Timestamps formateados estrictamente en UTC RFC3339 (`YYYY-MM-DDTHH:MM:SSZ`).
  ```text
  commit <tamaño_bytes>\x00tree <hash_tree_raiz>
  parent <hash_commit_padre>
  author <Nombre Autor> <<email>> <timestamp_utc_rfc3339>

  <mensaje_del_commit>
  ```
- **Ejemplo**:
  ```text
  commit 215\x00tree 1111111111111111111111111111111111111111111111111111111111111111
  parent 2222222222222222222222222222222222222222222222222222222222222222
  author MiniGit User <user@minigit.local> 2026-07-22T23:45:00Z

  Primer commit de prueba
  ```

#### D. Deserialización y Validaciones Estrictas
Durante la deserialización (`DecodeBlob`, `DecodeTree`, `DecodeCommit`), MiniGit ejecuta validaciones en cascada:
1. **Encabezado general (`DecodeObject`)**: Verifica el byte delimitador `\x00`, el formato `<tipo> <tamaño>`, que el tipo sea válido (`blob`, `tree`, `commit`), que el tamaño sea un entero no negativo (`size >= 0`), y que coincida exactamente con la longitud real del cuerpo.
2. **Validación específica por tipo**:
   - `DecodeTree` valida que el modo sea octal, el tipo sea `"blob"` o `"tree"`, y los hashes sean de 64 caracteres hex válidos. Rechaza nombres inválidos (`..`, `/`, vacíos) y entradas duplicadas (`ErrDuplicateEntry`).
   - `DecodeCommit` valida la presencia obligatoria de la cabecera `tree`, el formato de autor con timestamp ISO/RFC3339 y la validez de los hashes hexadecimales de 64 caracteres.

### 7. Grafo de Merkle y Propagación de Hashes

MiniGit fortalece la estructura del Grafo de Merkle asegurando la inmutabilidad de los objetos y la propagación exacta de cambios:

```text
Commit B
   │
   ├── parent → Commit A
   └── tree → Root Tree
               ├── blob → README.md
               ├── blob → main.go
               └── tree → docs
                           ├── blob → arquitectura.md
                           └── tree → utils
                                       └── blob → hash.go
```

- **Blobs**: Almacenan exactamente el contenido binario del archivo. Dos archivos idénticos en distintas rutas comparten el mismo hash de Blob.
- **Trees**: Almacenan entradas con nombre, modo, tipo (`blob` o `tree`) y hash SHA-256. Las entradas se serializan en un orden determinista y estable.
- **Commits**: Enlazan la instantánea actual mediante el hash del `tree` raíz y registran la historia mediante el puntero `parent` hacia el commit anterior.
- **Propagación de Hashes**:
  ```text
  Cambio en archivo
        ↓
  Nuevo hash de Blob
        ↓
  Nuevo hash del Tree que lo contiene
        ↓
  Nuevo hash de los Trees padres
        ↓
  Nuevo hash del Commit
  ```
- **Reutilización e Inmutabilidad**: Los objetos existentes no se modifican. Si un archivo o subdirectorio no sufre cambios entre commits, los objetos `tree` reutilizan la referencia al hash existente.

### 8. Flujo del Proceso de Commit

El proceso de creación de instantáneas (commits) en MiniGit sigue una secuencia estrictamente ordenada y desacoplada de la interfaz CLI:

```text
Working Tree
     │
     ▼
    Add
     │
     ▼
    Index (Staging)
     │
     ▼
Creación de Blobs
     │
     ▼
Construcción recursiva de Trees
     │
     ▼
   Tree raíz
     │
     ▼
Objeto Commit
     │
     ▼
Object Store
     │
     ▼
Actualización de la rama activa / HEAD
```

- **Construcción desde el Index**: El commit se construye a partir del contenido preparado en el Index (`.minigit/index`). Los cambios no preparados del Working Tree se omiten.
- **Validación del Mensaje y Autor**: El mensaje no puede estar vacío ni compuesto únicamente de espacios. Los datos del autor se leen desde las variables de entorno `MINIGIT_AUTHOR_NAME` y `MINIGIT_AUTHOR_EMAIL` (con fallback predeterminado si no están definidas).
- **Protección mediante Lock**: Se adquiere un bloqueo exclusivo (`index.lock`) antes de realizar cualquier lectura o modificación crítica.
- **Prevención de Commits sin Cambios**: Se compara el hash del `tree` raíz generado desde el Index contra el `tree` raíz del commit padre. Si los hashes son idénticos, la operación se rechaza con `ErrNothingToCommit`.
- **Persistencia Segura y Reflog**: El objeto `commit` se serializa en UTC RFC3339, se comprime en `zlib` y se almacena en `ObjectStore`. Posteriormente, se actualiza atómicamente la referencia de la rama (`.minigit/refs/heads/<branch>`) o `HEAD`, registrando la transacción en el reflog (`.minigit/logs/...`).

### 9. Historial y Consulta de Objetos (Log y Show)

MiniGit proporciona herramientas avanzadas de navegación en el historial y consulta de objetos almacenados:

#### A. Comando `minigit log`
El comando `log` permite explorar el historial de cambios del repositorio iniciando en la posición actual de `HEAD`:
- **Recorrido de Commits Padre**: Navega secuencialmente desde el commit señalado por `HEAD` siguiendo la referencia `Parent` hasta llegar al commit inicial (cuyo atributo `Parent` es vacío `""`).
- **Seguridad en el Recorrido**: Mantiene un registro de nodos visitados (`visited`) que detecta y aborta inmediatamente cualquier ciclo infinito en la cadena de commits.
- **Validación de Padres**: Detecta si una referencia a un commit padre apunta a un objeto inexistente o corrupto, informando un error claro sin colapsar el programa.
- **Sintaxis**:
  - `minigit log`: Muestra el historial completo (hash, autor, email, fecha formateada RFC1123Z y mensaje).
  - `minigit log --oneline`: Muestra un formato resumido de una sola línea por commit (`<hash_7_chars> (HEAD -> <rama>) <primera_linea_mensaje>`).

#### B. Comando `minigit show` (Consulta de Objetos)
El comando `show` permite inspeccionar cualquier objeto direccionable por contenido mediante su hash de 64 caracteres o un prefijo abreviado (por defecto `HEAD` si se omite el argumento):

1. **Objeto Blob (`blob`)**: Descomprime y muestra en consola el contenido crudo original del archivo sin alterar saltos de línea ni perder caracteres.
2. **Objeto Tree (`tree`)**: Descomprime y muestra la lista de entradas en formato octal con su tipo, hash y nombre:
   ```text
   100644 blob e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855	archivo.txt
   040000 tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904ce8e32906b3a0f7902d0eb2d	src
   ```
3. **Objeto Commit (`commit`)**: Muestra la cabecera del commit (`tree`, `parent`), los metadatos del autor, fecha, mensaje y el resumen de cambios introducidos respecto al commit padre (`+` agregados, `M` modificados, `-` eliminados).

#### C. Catálogo de Mensajes de Error Frecuentes
- **Objeto no encontrado**:
  ```text
  No se encontró el objeto solicitado: 0000000000000000000000000000000000000000000000000000000000000000
  ```
- **Objeto corrupto**:
  ```text
  Objeto corrupto: zlib decompression failed / checksum mismatch
  ```
- **Repositorio sin commits**:
  ```text
  no hay commits registrados en este repositorio
  ```
- **Referencia a padre inválida**:
  ```text
  referencia a commit padre inválida (objeto no encontrado: <hash>)
  ```

---

## 🔒 Seguridad e Integridad

1. **Rutas seguras**: Previene vulnerabilidades de Path Traversal bloqueando referencias `..`, rutas absolutas y symlinks dirigidos fuera de la raíz del repositorio.
2. **Escrituras atómicas**: La modificación de archivos críticos (index, HEAD, refs) escribe primero en un archivo temporal en el mismo directorio, ejecuta `.Sync()` y realiza un renombrado atómico (`os.Rename`).
3. **Control de concurrencia**: Crea archivos `.lock` de manera exclusiva (`O_EXCL`) para evitar escrituras simultáneas corruptas.
4. **Verificación checksum**: Se recalcula el SHA-256 de cada objeto al ser leído; cualquier discrepancia devuelve un error descriptivo de corrupción.

---

## 📈 Mejoras Recientes

### Ramas y Navegación
- **Listado de ramas mejorado**: Muestra el hash corto de cada rama junto al nombre.
- **Validación de nombres**: Rechaza nombres vacíos, con caracteres inválidos o duplicados.
- **Checkout seguro**: Detecta cambios locales antes de cambiar de rama para evitar pérdida de datos.
- **Restauración automática**: El directorio de trabajo y el índice se sincronizan correctamente al cambiar de rama.
- **HEAD robusto**: Validación mejorada contra archivos corruptos o vacíos.

### Seguridad de Archivos
- **Protección de `.minigit`**: Bloqueo explícito de operaciones sobre la carpeta interna del repositorio.
- **Validación de symlinks**: Detecta y rechaza enlaces simbólicos que apuntan fuera del repositorio.
- **Detección de bytes nulos**: Previene errores del sistema de archivos por rutas malformadas.
- **Mensajes de error claros**: Cada tipo de violación de seguridad muestra un mensaje descriptivo.

### Integridad y Recuperación
- **Limpieza automática**: Al abrir un repositorio, se eliminan archivos temporales y locks abandonados.
- **Detección de locks huérfanos**: Identifica bloqueos de procesos que ya no existen y los libera.
- **Sincronización de directorio**: Las escrituras atómicas incluyen `Sync()` del directorio para persistencia garantizada.
- **Verificación de objetos vacíos**: Detecta y rechaza objetos corruptos o truncados al leerlos.

### Pruebas y Calidad
- **Pruebas Round Trip**: Validación completa de serialización y deserialización de objetos.
- **Estabilidad de hashes**: Verificación de que el mismo contenido siempre genera el mismo hash.
- **Pruebas de corrupción**: Simulación de objetos dañados para verificar la detección de errores.
- **Flujo completo**: Pruebas de integración que cubren init, add, commit, branch, checkout y restore.
- **Código estandarizado**: Formateo consistente con `gofmt` y análisis estático con `go vet`.

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
