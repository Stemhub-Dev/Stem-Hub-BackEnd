# Mejoras y definiciones pendientes - StemHub

## Género Musical

### M-001 - Edición de descripción
- HU relacionada: HU-CFG-B03
- Estado: Sin Hacer
- Detectado durante: Desarrollo Backend
- Descripción:
  Actualmente la HU permite modificar únicamente el nombre del género.
  Evaluar permitir también la modificación de la descripción para evitar
  inconsistencias entre nombre y descripción.
- Requiere definición de: PO
- Prioridad sugerida: Baja

### M-002 - Creación descripción en Género Musical
- HU relacionada: HU-CFG-B02
- Estado: Sin Hacer
- Detectado durante: Desarrollo Backend
- Descripción:
  Actualmente la HU permite dar de alta el nombre pero la descripción va a quedar vacía.
  Evaluar permitir también el alta de la descripción.
- Requiere definición de: PO
- Prioridad sugerida: Baja

## Mejoras pendientes - Seguridad y autorización

### M-S-001 Validación de ámbito al asignar roles

Actualmente el endpoint de asignación de roles a usuarios está pensado para roles de ámbito SISTEMA, por ejemplo Administrador.

Como mejora, se deberá validar explícitamente el ámbito del rol seleccionado antes de realizar la asociación.

- Los roles de ámbito SISTEMA podrán asignarse mediante `usuariorol`.
- Los roles de ámbito PROYECTO, como Productor o Artista, deberán asignarse únicamente dentro del contexto de un proyecto mediante `integranteproyecto`.
- Si se intenta asignar un rol de PROYECTO desde el endpoint de roles de sistema, el backend deberá devolver un mensaje claro, por ejemplo:

`El rol seleccionado pertenece al ámbito PROYECTO y debe asignarse dentro de un proyecto.`

Esta validación permitirá mantener separada la autorización global del sistema de la autorización específica de cada proyecto.

### M-S-002 Parametrización de permisos

Mantener la autorización basada en permisos y no en nombres de roles hardcodeados.

Ejemplo:

- `CONSULTAR_GENEROS`: permite consultar géneros musicales.
- `GESTIONAR_GENEROS`: permite realizar alta, modificación y cambio de estado de géneros musicales.

Los roles obtienen sus permisos mediante `rolpermiso`, por lo que agregar, quitar o modificar permisos de un rol no debe requerir cambios en el código backend.

---

# Estado actual – Administración / Seguridad StemHub

## Rama Backend

`feature/administracion-seguridad`

## Rama Frontend

`administracion-seguridad`

---

# 1. Estructura general de Administración

Se definió una única pantalla:

`/administracion`

con cuatro pestañas principales:

- Usuarios
- Roles y permisos
- Auditoría
- Parametrizaciones

La idea es evitar crear muchas pantallas independientes y centralizar la administración del sistema.

---

# 2. Roles y permisos

## Modelo de seguridad

Se mantiene la separación por ámbito:

### SISTEMA

Ejemplo:

- Administrador

Los roles SISTEMA se asignan mediante:

`usuariorol`

### PROYECTO

Ejemplos:

- Productor
- Artista

Los roles PROYECTO se asignan mediante:

`integranteproyecto`

Un usuario puede ser, por ejemplo:

- Administrador del sistema
- Productor en Proyecto A
- Artista en Proyecto B
- Productor en Proyecto C

El rol de proyecto no pertenece globalmente al usuario, sino a su participación en un proyecto determinado.

---

# 3. Permisos

Los permisos existentes actualmente son:

## SISTEMA

- GESTIONAR_USUARIOS
- GESTIONAR_ROLES
- GESTIONAR_PREGUNTAS_FRECUENTES
- CONSULTAR_AUDITORIA
- GESTIONAR_GENEROS
- CONSULTAR_GENEROS

## PROYECTO

- GESTIONAR_CANCIONES
- GESTIONAR_VERSIONES
- GESTIONAR_STEMS
- GESTIONAR_COMENTARIOS

La autorización no debe depender directamente del nombre del rol.

Ejemplo:

No:

`si rol == Productor`

Sí:

`¿el rol que tiene el usuario en este proyecto posee GESTIONAR_VERSIONES?`

Esto permite modificar permisos desde Administración sin tener que cambiar código cada vez.

Importante:

En las operaciones de proyecto existe actualmente la regla:

`esPropietario || tienePermiso`

Por lo tanto, se deberá revisar posteriormente en qué operaciones el propietario mantiene privilegios aunque su rol no tenga el permiso correspondiente.

---

# 4. Backend realizado – Roles y permisos

Ya se implementó:

`GET /permisos`

Devuelve los permisos activos.

Ya se implementó:

`GET /roles/:id/permisos`

Devuelve:

- rol
- ámbito
- todos los permisos compatibles con ese ámbito
- `asignado: true/false`

Ejemplo:

```json
{
  "codigoRol": 1,
  "nombreRol": "Administrador",
  "ambitoRol": "SISTEMA",
  "permisos": [
    {
      "codigoPermiso": 2,
      "clavePermiso": "GESTIONAR_ROLES",
      "nombrePermiso": "Gestionar roles",
      "descripcionPermiso": "...",
      "asignado": true
    }
  ]
}