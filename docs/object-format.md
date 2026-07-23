# Formato de Objetos en MiniGit

## Introducción

MiniGit utiliza un modelo de almacenamiento orientado a objetos inmutables direccionables por contenido, fuertemente inspirado en la arquitectura interna de Git. Existen tres tipos de objetos principales en el sistema:

- **Blob**: Almacena el contenido crudo de un archivo.
- **Tree**: Almacena la estructura de un directorio (nombres, permisos, tipos y hashes de archivos o subdirectorios).
- **Commit**: Almacena una instantánea del estado del repositorio en un punto del tiempo, junto con metadatos del autor, fecha y referencia al commit padre.

Todos los objetos comparten una estructura de encapsulamiento común basada en un **encabezado estandarizado**, son identificados de forma única mediante un checksum **SHA-256** de 64 caracteres hexadecimales, y son almacenados comprimidos en disco mediante **zlib**.

---

## Encabezado de los Objetos

Cualquier objeto almacenado en MiniGit se construye encapsulando su contenido (body) con un encabezado en texto plano finalizado por un byte nulo (`\x00`).

### Estructura General

```text
<tipo> <tamaño_en_bytes>\x00<cuerpo_del_objeto>
```

- `<tipo>`: Tipo de objeto (`blob`, `tree` o `commit`).
- `<tamaño_en_bytes>`: Longitud decimal en bytes del cuerpo del objeto (excluyendo el encabezado y el byte nulo).
- `\x00`: Byte nulo (`0x00`) actuando como delimitador estricto entre el encabezado y el cuerpo.
- `<cuerpo_del_objeto>`: Carga útil (payload) del objeto codificada según el tipo especifico.

### Proceso de Cálculo de Hash y Almacenamiento

1. Se genera la carga útil del objeto (body).
2. Se antepone el encabezado: `EncodeObject(tipo, body)`.
3. Se calcula la suma de verificación SHA-256 sobre el arreglo completo de bytes (encabezado + byte nulo + body):
   $$\text{Hash} = \text{SHA-256}(\text{encabezado} + \text{"\x00"} + \text{body})$$
4. El payload completo resultante se comprime usando el algoritmo **zlib**.
5. Se almacena en la ruta `.minigit/objects/xx/yyyy...` donde `xx` son los primeros 2 caracteres hex y `yyyy...` los 62 restantes.

---

## Formato del Objeto Blob

Un **Blob** (*Binary Large Object*) almacena únicamente los bytes crudos del contenido de un archivo. No conserva el nombre del archivo, permisos ni fechas de modificación; estos metadatos son responsabilidad del objeto **Tree**.

### Estructura

- **Encabezado**: `blob <tamaño>\x00`
- **Cuerpo**: Bytes binarios o de texto del archivo sin modificaciones.

### Ejemplo de Serialización

Supóngase un archivo de texto con el contenido `"Hola MiniGit\n"` (13 bytes):

#### Contenido del cuerpo:
```text
Hola MiniGit\n
```

#### Payload serializado completo (pre-compresión zlib):
```text
blob 13\x00Hola MiniGit\n
```

#### Representación Hexadecimal:
```text
62 6c 6f 62 20 31 33 00 48 6f 6c 61 20 4d 69 6e 69 47 69 74 0a
|-- "blob 13" --| \x00 |-- "Hola MiniGit\n" ------------------|
```

---

## Formato del Objeto Tree

Un **Tree** representa un directorio. Contiene un listado de entradas, donde cada entrada vincula un archivo o subdirectorio con su nombre, modo de permisos y su hash SHA-256 correspondiente.

### Estructura de las Entradas

Cada entrada dentro del cuerpo de un Tree sigue el siguiente formato textual de una línea por elemento:

```text
<modo> <tipo> <hash_64_hex> <nombre>\n
```

- `<modo>`: Permisos del sistema de archivos formateados en octal (ejemplo: `100644` para archivos regulares, `100755` para ejecutables, `40000` para subdirectorios/trees).
- `<tipo>`: Tipo de objeto referenciado (`blob` o `tree`).
- `<hash_64_hex>`: Hash SHA-256 de 64 caracteres en hexadecimal del objeto referenciado.
- `<nombre>`: Nombre simple del archivo o subdirectorio (sin rutas relativas ni slashes `/`).
- `\n`: Salto de línea (`0x0A`) delimitador de cada entrada.

> [!NOTE]
> **Ordenamiento Determinista**: Para garantizar que un directorio con el mismo contenido genere exactamente el mismo hash SHA-256, las entradas de un `Tree` se ordenan estrictamente por nombre de forma alfabética ascendente.

### Ejemplo de Serialización

Dado un directorio que contiene el archivo `main.go` (blob `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`) y la carpeta `docs` (tree `1f82c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a939`):

#### Cuerpo del Tree (ordenado alfabéticamente por nombre):
```text
40000 tree 1f82c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a939 docs
100644 blob e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 main.go
```
*(Tamaño del cuerpo: 154 bytes)*

#### Payload serializado completo (pre-compresión zlib):
```text
tree 154\x0040000 tree 1f82c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a939 docs\n100644 blob e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 main.go\n
```

---

## Formato del Objeto Commit

Un **Commit** representa un punto en el historial del repositorio. Contiene punteros al árbol raíz de archivos (`tree`), al commit padre (`parent`), metadatos del autor y el mensaje explicativo de la confirmación.

### Estructura

El cuerpo del commit se compone de líneas de encabezado seguidas por dos saltos de línea (`\n\n`) y el mensaje descriptivo:

```text
tree <hash_tree_raiz_64_hex>\n
[parent <hash_commit_padre_64_hex>\n]
author <nombre_autor> <<email_autor>> <timestamp_RFC3339>\n\n
<mensaje_del_commit>
```

- `tree`: Hash SHA-256 del `Tree` que representa la raíz del área de trabajo en el momento de la confirmación.
- `parent`: (Opcional) Hash SHA-256 del commit anterior. El primer commit del repositorio (commit inicial) no incluye la línea `parent`.
- `author`: Identificación del creador expresada como `Nombre <email> TimestampUTC` formateado según **RFC3339** (ej: `2026-07-23T13:40:00Z`).
- `\n\n`: Doble salto de línea obligatorio que separa los metadatos de la cabecera del cuerpo del mensaje.
- `<mensaje_del_commit>`: Texto explicativo introducido por el usuario.

### Ejemplo de Serialización

#### Commit Inicial (sin padre):
```text
tree 4f82c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a939
author Desarrollador <dev@minigit.org> 2026-07-23T13:40:00Z

Primer commit del proyecto
```
*(Tamaño del cuerpo: 161 bytes)*

##### Payload completo serializado:
```text
commit 161\x00tree 4f82c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a939\nauthor Desarrollador <dev@minigit.org> 2026-07-23T13:40:00Z\n\nPrimer commit del proyecto
```

#### Commit Secundario (con padre):
```text
tree a912c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482b123
parent b812c2d7310185960459a936a71537248e3e44923769c8491c107f9eb482a988
author Desarrollador <dev@minigit.org> 2026-07-23T14:00:00Z

Agregar nueva funcionalidad
```

---

## Resumen Comparativo de Objetos

| Tipo de Objeto | Responsabilidad Principal | Encabezado | Delimitador Interno |
| :--- | :--- | :--- | :--- |
| **Blob** | Contenido puro de archivos | `blob <size>\x00` | N/A (datos crudos) |
| **Tree** | Estructura de directorios y nombres | `tree <size>\x00` | Salto de línea (`\n`) por entrada |
| **Commit** | Metadatos e historial del repositorio | `commit <size>\x00` | Doble salto de línea (`\n\n`) antes del mensaje |
