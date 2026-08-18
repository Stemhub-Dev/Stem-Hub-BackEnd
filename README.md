# StemHub Backend

Backend de **StemHub**, plataforma web colaborativa para la gestión y versionado de proyectos musicales.

La API está desarrollada en **Go** utilizando **Gin** y persiste la información en **PostgreSQL**. El entorno de desarrollo puede ejecutarse completamente con **Docker Compose** o utilizando PostgreSQL en Docker y el Backend de forma local con `go run`.

---

## Stack

- Go
- Gin
- PostgreSQL 17
- pgx / `database/sql`
- Docker
- Docker Compose
- GitHub Actions
- SonarQube *(planificado para análisis estático)*
- JWT *(seguridad/autenticación a integrar en las HU correspondientes)*

---

## Arquitectura

El Backend utiliza una arquitectura por capas:

```text
Cliente / Frontend
       ↓
     Router
       ↓
     Handler
       ↓
     Service
       ↓
   Repository
       ↓
   PostgreSQL
```

Responsabilidad de cada capa:

- `router`: definición de endpoints y rutas HTTP.
- `handler`: recibe requests HTTP, interpreta parámetros/body y devuelve respuestas HTTP/JSON.
- `service`: contiene reglas de negocio y validaciones.
- `repository`: acceso y persistencia de datos en PostgreSQL.
- `model`: representación interna de entidades persistidas.
- `dto`: objetos utilizados para entrada y salida de datos de la API.
- `middleware`: autenticación, autorización y controles transversales.
- `database`: conexión con PostgreSQL.

---

## Estructura

```text
.
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── database/
│   ├── dto/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── repository/
│   ├── router/
│   └── service/
│
├── migrations/
├── docs/
├── .github/
│   └── workflows/
│       └── backend-ci.yml
│
├── .dockerignore
├── .env.example
├── .gitignore
├── compose.yaml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

---

## Requisitos

Para **levantar el proyecto completo con Docker**:

- Git
- Docker Desktop

Para **desarrollar el Backend localmente**:

- Git
- Docker Desktop
- Go
- VS Code u otro editor compatible

> PostgreSQL no necesita instalarse localmente: se ejecuta mediante Docker.

---

## Clonar el repositorio

```bash
git clone https://github.com/facu-1538/Stem-Hub-BackEnd.git
cd Stem-Hub-BackEnd
git switch dev
git pull origin dev
```

---

## Variables de entorno

Crear el archivo `.env` a partir de `.env.example`.

### PowerShell

```powershell
Copy-Item .env.example .env
```

### CMD

```cmd
copy .env.example .env
```

Configurar:

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=stemhubdev
DB_USER=stemhub_app
DB_PASSWORD=CAMBIAR_PASSWORD
APP_PORT=8080
```

| Variable | Descripción |
|---|---|
| `DB_HOST` | Host de PostgreSQL. Localmente es `localhost`; dentro de Docker Compose el Backend usa el servicio `postgres`. |
| `DB_PORT` | Puerto de PostgreSQL. |
| `DB_NAME` | Nombre de la base de desarrollo. |
| `DB_USER` | Usuario de PostgreSQL. |
| `DB_PASSWORD` | Contraseña local del usuario de PostgreSQL. |
| `APP_PORT` | Puerto HTTP del Backend. |

> `.env` **no debe subirse a GitHub**.  
> `.env.example` sí debe versionarse, pero nunca debe contener contraseñas reales.

---

## Levantar el proyecto completo con Docker

Validar primero la configuración:

```bash
docker compose config
```

Levantar PostgreSQL y Backend:

```bash
docker compose up --build -d
```

Verificar contenedores:

```bash
docker compose ps
```

Resultado esperado:

```text
stemhub-postgres-dev   Up (healthy)
stemhub-backend        Up
```

Ver logs del Backend:

```bash
docker compose logs backend
```

Detener el entorno sin borrar los datos:

```bash
docker compose down
```

### Importante

No utilizar normalmente:

```bash
docker compose down -v
```

La opción `-v` elimina el volumen de PostgreSQL y, por lo tanto, borra la base DEV local.

---

## Desarrollo local recomendado

Durante el desarrollo activo se recomienda:

```text
PostgreSQL  → Docker
Backend Go  → local
```

Detener solamente el Backend Docker:

```bash
docker compose stop backend
```

Ejecutar el Backend local:

```bash
go run ./cmd/api
```

La API queda disponible en:

```text
http://localhost:8080
```

Cuando se modifica código:

```text
Ctrl + C
go run ./cmd/api
```

### Volver a probar el Backend dentro de Docker

Detener el proceso local de Go y reconstruir el servicio:

```bash
docker compose up --build -d backend
```

Esto reconstruye el Backend sin eliminar la base PostgreSQL ni su volumen.

---

## Base de datos

Entorno de desarrollo:

```text
Motor: PostgreSQL 17
Base: stemhubdev
Usuario: stemhub_app
Schema: public
```

El volumen Docker utilizado para persistencia es:

```text
stemhub_postgres_dev_data
```

Los datos permanecen aunque el contenedor sea detenido o recreado, mientras no se elimine el volumen.

---

## Migraciones

Las migraciones se encuentran en:

```text
migrations/
```

Actualmente el proyecto dispone de:

```text
001 - esquema inicial
002 - datos iniciales
003 - seguridad y propiedad
004 - datos de seguridad
005 - auditoría
```

En una instalación nueva, Docker monta `migrations/` en `/docker-entrypoint-initdb.d` y PostgreSQL ejecuta automáticamente estos scripts cuando inicializa un volumen vacío.

### Importante

Las migraciones ya aplicadas **no deben modificarse**.

Los próximos cambios de base deben agregarse como nuevas migraciones:

```text
006...
007...
008...
```

Si un desarrollador ya tiene un volumen PostgreSQL inicializado, una nueva migración descargada mediante `git pull` **no se ejecuta automáticamente**. Debe aplicarse de forma controlada sobre la base existente.

---

## Conexión opcional con DBeaver

```text
Host: localhost
Puerto: 5432
Database: stemhubdev
Usuario: stemhub_app
Password: valor configurado en .env
Schema: public
```

---

## Comandos Go

### Ejecutar el Backend

```bash
go run ./cmd/api
```

### Formatear código

```bash
go fmt ./...
```

### Análisis estático básico

```bash
go vet ./...
```

### Ejecutar tests

```bash
go test ./...
```

### Verificar compilación

```bash
go build ./...
```

### Descargar dependencias declaradas

```bash
go mod download
```

### Ordenar dependencias

```bash
go mod tidy
```

### Agregar una dependencia nueva

```bash
go get <paquete>
go mod tidy
```

---

## Validación antes de un commit

Antes de realizar un commit se recomienda ejecutar:

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...
```

Si todo finaliza correctamente, revisar:

```bash
git status
```

y recién después preparar el commit.

---

## Flujo Git

Ramas permanentes:

```text
main
test
dev
```

Ramas temporales:

```text
feature/...   → nueva funcionalidad / HU
fix/...       → corrección durante desarrollo
hotfix/...    → corrección urgente desde producción
version/...   → preparación de una versión estable
```

Flujo habitual:

```text
dev
 ↓
feature/...
 ↓
commits
 ↓
push
 ↓
Pull Request
 ↓
pipeline
 ↓
revisión
 ↓
merge a dev
 ↓
eliminar feature
```

No se debe programar directamente sobre `main`, `test` o `dev`.

---

## Crear una nueva feature

Partir siempre de `dev` actualizado:

```bash
git switch dev
git pull origin dev
git switch -c feature/<id-jira>-descripcion
```

Al terminar:

```bash
go fmt ./...
go vet ./...
go test ./...
go build ./...

git status
git add <archivos>
git commit -m "feat: descripcion del cambio"
git push -u origin feature/<id-jira>-descripcion
```

Luego crear un **Pull Request hacia `dev`**.

Después del merge:

```bash
git switch dev
git pull origin dev
git branch -d feature/<id-jira>-descripcion
```

---

## CI - GitHub Actions

El Backend utiliza:

```text
.github/workflows/backend-ci.yml
```

El pipeline se ejecuta ante los eventos definidos en el workflow y valida el proyecto mediante tareas como:

```text
go mod download
go vet ./...
go test ./...
go build ./...
```

El pipeline de GitHub Actions es independiente de Docker Desktop local.

---

## Análisis estático

El analizador seleccionado para el proyecto es:

```text
SonarQube
```

Actualmente `go vet` se utiliza como análisis estático básico de Go. La integración de SonarQube con el pipeline queda prevista para una etapa posterior.

---

## Reglas importantes para el equipo

- No subir `.env`.
- No subir backups locales de PostgreSQL.
- No modificar migraciones ya aplicadas.
- No desarrollar directamente sobre `main`, `test` o `dev`.
- Crear una rama `feature` por HU o funcionalidad.
- Realizar Pull Request hacia `dev`.
- Ejecutar validaciones Go antes de cada commit.
- Los datos locales de prueba no se comparten mediante Git.
- Los cambios de estructura o datos maestros se comparten mediante migraciones.
- Borrar las ramas temporales después de que su PR haya sido mergeado correctamente.
