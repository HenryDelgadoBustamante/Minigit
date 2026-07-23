# Bitácora Técnica de Desarrollo — MiniGit

## Resumen Ejecutivo

La presente bitácora documenta las principales decisiones de diseño, hitos de desarrollo, problemas técnicos enfrentados, soluciones implementadas y lecciones aprendidas durante la construcción de **MiniGit**, un sistema de control de versiones local desarrollado íntegramente en lenguaje **Go** como parte del proyecto universitario de arquitectura de software.

---

## Hitos de Desarrollo y Cronología

| Hito / EPIC | Descripción General | Estado |
| :--- | :--- | :--- |
| **EPIC 01-03** | Arquitectura base del repositorio, módulo de almacenamiento de objetos con zlib y hashing SHA-256. | Completado |
| **EPIC 04-06** | Implementación del staging index (`.minigit/index`), ordenamiento determinista de Trees y serialización de Blobs/Trees/Commits. | Completado |
| **EPIC 07-09** | Gestión de referencias (`HEAD`, `refs/heads/`), creación de ramas, navegación mediante `checkout` y manejo del estado *detached HEAD*. | Completado |
| **EPIC 10-12** | Implementación de comandos CLI (`add`, `commit`, `status`, `log`, `show`, `restore`, `branch`), motor de ignorado `.minigitignore` y bloqueos por concurrencia `.lock`. | Completado |
| **EPIC 13** | Pruebas de integración integrales de la CLI y pruebas unitarias de almacenamiento y repositorios. | Completado |
| **EPIC 14** | Fortalecimiento de la documentación técnica, especificación de formatos de objetos, grafo de Merkle y bitácora técnica. | Completado |
| **EPIC 15** | Funcionalidades complementarias opcionales (`minigit diff`, `minigit merge fast-forward`, resolución por prefijos cortos de hash, detección de renombrados). | Completado |

---

## Decisiones Principales de Diseño

### 1. Uso Exclusivo de la Biblioteca Estándar de Go
- **Decisión**: No utilizar dependencias externas de terceros (`vendor` o paquetes externos `go.mod`).
- **Justificación**: Garantizar máxima portabilidad, compilar un binario estático único sin dependencias y aprender la implementación de algoritmos criptográficos (SHA-256 en `crypto/sha256`), de compresión (`compress/zlib`) y manipulación del sistema de archivos (`os`, `path/filepath`).

### 2. Adopción de SHA-256 en lugar de SHA-1
- **Decisión**: Utilizar checksums SHA-256 de 256 bits (representados en 64 caracteres hexadecimales) en sustitución del histórico SHA-1 utilizado por Git.
- **Justificación**: Proporcionar resistencia criptográfica ante colisiones de hash modernas y asegurar la integridad futura del repositorio.

### 3. Índice en Formato JSON Legible (`.minigit/index`)
- **Decisión**: Implementar el área de preparación (index) serializada como archivo JSON estructurado con indización atómica.
- **Justificación**: Simplificar la inspección del estado de preparación, facilitar las pruebas unitarias y ofrecer transparencia educativa sobre los campos guardados (`path`, `hash`, `size`, `mode`, `mod_time`).

### 4. Escritura Atómica de Archivos mediante Ficheros Temporales y Sincronización a Disco
- **Decisión**: Toda modificación de objetos, index o referencias se realiza escribiendo primero en un archivo temporal (`.tmp`), forzando el volcado físico a disco (`file.Sync()`) y realizando un reemplazo atómico mediante `os.Rename`.
- **Justificación**: Prevenir la corrupción de datos u objetos a medio escribir en caso de fallos inesperados de energía o interrupción del proceso.

---

## Problemas Técnicos Enfrentados y Soluciones Aplicadas

### Problema 1: Inconsistencias en Separadores de Ruta Multiplataforma (Windows vs Linux)
- **Descripción**: En Windows los caminos de archivos utilizan barra invertida (`\`), mientras que en Linux/macOS utilizan barra diagonal (`/`). Esto producía hashes diferentes para la misma estructura de árbol y fallos en la búsqueda en el índice.
- **Solución**: Se implementó la función auxiliar `filesystem.NormalizePath()` que convierte de forma sistemática todas las rutas internas a separadores estilo POSIX (`/`) antes de calcular hashes o escribir entradas en objetos `Tree` e `index`.

### Problema 2: Hashes No Repetibles en Objetos Tree por Variación en el Orden de Archivos
- **Descripción**: Al recorrer el sistema de archivos con funciones cuyo orden no está garantizado, el cuerpo de los objetos `Tree` se generaba en orden aleatorio, produciendo hashes SHA-256 diferentes para directorios con contenidos idénticos.
- **Solución**: Se integró la función `sortTreeEntries()` en `internal/object/tree.go`, la cual ordena determinísticamente todas las entradas de un `Tree` por nombre alfabético ascendente antes de cualquier operación de serialización.

### Problema 3: Manejo Seguro del Estado *Detached HEAD*
- **Descripción**: Al ejecutar `checkout <hash_commit>`, el puntero `HEAD` ya no debe apuntar a una rama (`ref: refs/heads/main`), sino directamente al hash de un commit.
- **Solución**: Se adaptó el parser y escritor en `internal/repository/head.go` para soportar dinámicamente dos tipos de estado (`HEADTypeBranch` y `HEADTypeDetached`), validando estrictamente que el valor detached corresponda a un hash hex válido de 64 caracteres.

### Problema 4: Condiciones de Carrera en Operaciones Concurrentes sobre el Repositorio
- **Descripción**: La ejecución simultánea de múltiples instancias de MiniGit en el mismo directorio podía corruptor el índice o las referencias.
- **Solución**: Se creó la abstracción de bloqueo `RepositoryLock` en `internal/repository/lock.go`, creando un archivo `.minigit/index.lock` durante la escritura y garantizando su liberación mediante patrones `defer lock.Unlock()`.

---

## Lecciones Aprendidas

1. **La inmutabilidad simplifica el control de versiones**: El modelo de almacenamiento por contenido inmutable elimina los problemas de sincronización de estado, ya que los objetos existentes nunca cambian; únicamente se crean nuevos nodos en el Grafo de Merkle.
2. **Las escrituras atómicas son indispensables**: Sin sincronización a disco (`Sync`) y renombrado atómico (`Rename`), los fallos de proceso dejan repositorios corruptos difíciles de recuperar.
3. **El diseño determinista es vital en criptografía**: Para que los checksums coincidan entre diferentes ejecuciones y sistemas operativos, todos los componentes (fechas formateadas en UTC RFC3339, ordenamiento de listas, saltos de línea) deben ser 100% deterministas.
