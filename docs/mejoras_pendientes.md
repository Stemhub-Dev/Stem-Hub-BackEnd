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