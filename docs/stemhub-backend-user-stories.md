# StemHub — Historias de usuario: Back-End

Especificación funcional del Back-End de StemHub para implementación por un agente. Fuente: sección 16 "Modelo funcional" del documento unificado del proyecto (capítulo 6. Back-End). Las pantallas que consumen estos endpoints están en `stemhub-frontend-user-stories.md` y se referencian por su ID (HU-XXX-Fnn).

## Cómo usar este documento

- Cada historia (HU) es una unidad de trabajo independiente. Implementar los **Criterios de aceptación** tal como están escritos: cada fila "Cuando → Espero" es un caso a cubrir con tests (código HTTP y cuerpo de respuesta).
- La columna **Referencias** indica el endpoint (método y ruta) o la HU relacionada. Las rutas abreviadas con `/.../` se completan con la ruta base del recurso indicada en el primer criterio.
- **Estado (matriz de trazabilidad):** "Codificado" significa que ya existe una implementación; verificarla contra los criterios en lugar de reescribirla. "Rechazada" requiere confirmación antes de implementar. "No figura" corresponde a HU agregadas al completar el documento.
- **Puntos a revisar:** dudas, inconsistencias y decisiones pendientes. Si un punto bloquea la implementación, aplicar la opción más conservadora (más restrictiva en permisos), dejarla documentada en el código o en el PR y no inventar reglas de negocio.

## Contexto técnico

- **Lenguaje y API:** Go, API REST con JSON. Rutas en español y en plural (ej. `/proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/comentarios`).
- **Base de datos:** PostgreSQL. Todas las consultas parametrizadas.
- **Autenticación:** delegada en Supabase Auth (IdP) con OAuth 2.0 Authorization Code Flow + PKCE. El IdP emite un JWT firmado con RS256 que contiene el rol como claim; el backend lo valida en cada request con `golang-jwt` y `go-oidc`. StemHub no almacena ni registra contraseñas.
- **Autorización:** RBAC con claims del JWT más la membresía y el rol del usuario en cada proyecto. Siempre se valida en el servidor (HU-SEG-B01).
- **Sesión:** token vencido, invalidado por logout o 45 minutos de inactividad → HTTP 401 (HU-SEG-B02).
- **Almacenamiento de archivos:** Cloudflare R2. El backend devuelve URL firmadas con expiración corta (TTL entre 15 y 60 minutos), nunca la ruta física. Si R2 falla, el resto del sistema (login, navegación, comentarios, versionado) debe seguir operativo.
- **IA:** microservicio en Python (FastAPI + Spleeter) expuesto como API HTTP interna y consumido por el backend en Go. Contenerizado con Docker.
- **Emails:** servicio de emails transaccionales externo. StemHub detecta el evento, decide destinatarios y arma el contenido; el proveedor envía y gestiona rebotes. La verificación de email al registrarse es del IdP.
- **Auditoría:** históricos por entidad mediante triggers de base de datos (alta, modificación, baja) más una tabla general de auditoría para eventos (login, logout, accesos, descargas, cambios de permisos, eliminación de versiones). Cada registro guarda usuario, rol, fecha y hora, acción, entidad afectada y valores modificados. Los logs no son modificables por usuarios comunes.

## Convenciones de API

- **Paginación:** parámetros `page` y `pageSize` (por defecto 1 y 10). Respuesta `{ data, totalItems, totalPages, currentPage }`. Una página fuera de rango devuelve HTTP 200 con `data` vacío.
- **Búsqueda:** parámetro `q`, coincidencia parcial sin distinguir mayúsculas. Los filtros se combinan con AND.
- **Errores:** formato propuesto `{ codigo, mensaje, errores: [{ campo, mensaje }], correlacionId }` (HU-SEG-B06). Nunca exponer detalles técnicos.
- **Códigos HTTP:** 200/201/202/204 éxito; 400 validación; 401 sin token o token inválido; 403 sin permiso o no miembro del proyecto; 404 recurso inexistente; 409 conflicto (duplicados, estado inválido); 410 token vencido; 413 archivo demasiado grande; 415 formato no soportado; 422 operación no procesable; 429 límite de peticiones; 502/503 falla de servicio externo.
- **Campos calculados por el servidor:** autor (desde el JWT), estado inicial, fechas, número de versión, `esActual`, `esPropio`. El cliente no los envía.
- **Duplicados:** las validaciones de nombres y emails únicos son case-insensitive.

## Roles

- **Administrador del sistema:** rol global. Administra usuarios, roles, proyectos y configuraciones generales (ej. géneros musicales). Acceso irrestricto a todos los proyectos.
- **Administrador de proyecto:** creador del proyecto. Al crear un proyecto, el usuario queda asignado con este rol.
- **Productor:** rol principal dentro de un proyecto. Toma decisiones sobre el versionado y tiene control sobre contenido y colaboradores.
- **Músico:** rol colaborador dentro de un proyecto. Aporta feedback y comentarios; no controla la estructura del proyecto ni sus versiones.

Un mismo usuario puede tener roles distintos en proyectos distintos (ej. Productor en uno y Músico en otro): los permisos dentro de un proyecto se resuelven por proyecto.

## Matriz de permisos por rol

| Acción | Administrador | Productor | Músico |
|---|---|---|---|
| Ver Inicio, Mis proyectos, Mis canciones, Detalle de proyecto, Detalle de canción | ✅ | ✅ | ✅ |
| Ver pantalla Stems | ✅ | ✅ | ❌ (solo Productor según HU-VER-F04) |
| Crear proyecto | ✅ | ✅ | ✅ |
| Eliminar proyecto | ✅ | ✅ | ❌ |
| Invitar / quitar colaboradores | ✅ | ✅ | ✅ |
| Crear versión o rama | ✅ | ✅ | ❌ |
| Eliminar versión o rama | ✅ | ✅ | ❌ |
| Ver historial completo de versiones | ✅ | ✅ | ✅ |
| Agregar comentario | ✅ | ✅ | ✅ |
| Eliminar comentario | ✅ cualquiera | ✅ cualquiera | ✅ solo propios |
| Editar comentario | Solo propios | Solo propios | Solo propios |
| Cambiar estado del comentario (Pendiente / Hecho) | ✅ | ✅ | ✅ |
| Administración (usuarios, roles, géneros musicales) | ✅ (Administrador del sistema) | ❌ | ❌ |

> La matriz original no define permisos para Notificaciones, Reportes, Tablero ni Ayuda. Las HU de esos módulos indican su actor; ante dudas, ver "Puntos a revisar" de cada HU.

## Equivalencia de IDs antiguos

Algunas HU referencian IDs de una numeración anterior. Usar esta tabla para resolverlos:

| ID antiguo | Equivale a |
|---|---|
| HU-10 | HU-ABM-01 (Formulario Proyecto) |
| HU-12 | HU-VER-F05 (Canciones del Proyecto) |
| HU-13 | HU-VER-F06 (Acciones sobre una Canción / meatball menú) |
| HU-14 | HU-VER-F12 (Canciones del Proyecto - Manejo de errores) |
| HU-15 | HU-ABM-02 (Formulario Canción) |
| HU-16 | HU-ABM-04 (Indicador de progreso de carga) |
| HU-17 | HU-VER-F07 (Pantalla Detalle de Canción) |
| HU-18 | HU-VER-F08 (Reproducción de pista) |
| HU-19 | HU-VER-F10 (Sidebar de Comentarios) |
| HU-20 | HU-VER-F09 (Sidebar de Versiones) |
| HU-23 | Comparación de Versiones (módulo descartado del alcance definitivo) |
| HU-24 | Ambiguo: la matriz lo usa para "Separar Instrumentos" (HU-IA-F01) y otras HU para el Formulario Nueva Versión (HU-ABM-03) |
| HU-27 | HU-CMP-02 (Snackbar de notificación) |
| PANT-05 | VER-PRD-FORM-01 (Formulario Canción) |

## Índice de historias

| ID | Historia | Módulo | Prioridad | Estado (matriz) |
|---|---|---|---|---|
| HU-ACA-B01 | Registro de usuario | 6.1.1 Autenticación y control de acceso | Alta | No figura |
| HU-ACA-B02 | Verificación de email y reenvío | 6.1.1 Autenticación y control de acceso | Alta | No figura |
| HU-ACA-B03 | Inicio de sesión y emisión de token | 6.1.1 Autenticación y control de acceso | Alta | No figura |
| HU-ACA-B04 | Cierre de sesión | 6.1.1 Autenticación y control de acceso | Alta | No figura |
| HU-ACA-B05 | Recuperación y restablecimiento de contraseña | 6.1.1 Autenticación y control de acceso | Alta | No figura |
| HU-SEG-B03 | ABM Usuario | 6.1.2 Gestión de usuarios, roles y permisos | Alta | No figura |
| HU-SEG-B04 | ABM Rol | 6.1.2 Gestión de usuarios, roles y permisos | Alta | No figura |
| HU-SEG-B01 | Autorización de acceso a recursos protegidos | 6.1.3 Seguridad | Alta | Codificado |
| HU-SEG-B02 | Expiración de sesión por inactividad o token vencido | 6.1.3 Seguridad | Alta | No Iniciado |
| HU-SEG-B05 | Registro de aceptación de términos y condiciones | 6.1.3 Seguridad | Media | No figura |
| HU-SEG-B06 | Validación de datos y manejo controlado de errores | 6.1.3 Seguridad | Alta | No figura |
| HU-AYT-B1 | Middleware de auditoría y trazabilidad | 6.1.4 Auditoría y trazabilidad | Media | No figura |
| HU-ABM-B01 | ABM Proyecto | 6.2.1 Gestión de datos maestros (ABMs) | Alta | No figura |
| HU-ABM-B02 | ABM Canción | 6.2.1 Gestión de datos maestros (ABMs) | Alta | No figura |
| HU-ABM-B03 | ABM Versión | 6.2.1 Gestión de datos maestros (ABMs) | Alta | No figura |
| HU-BUQ-B01 | Búsqueda y filtrado de proyectos | 6.2.2 Búsqueda y filtrado | Media | No Iniciado |
| HU-BUQ-B02 | Búsqueda y filtrado de canciones | 6.2.2 Búsqueda y filtrado | Media | No Iniciado |
| HU-BUQ-B03 | Búsqueda y filtrado de stems | 6.2.2 Búsqueda y filtrado | Media | No Iniciado |
| HU-NOT-B01 | Disparo y envío de notificaciones por email | 6.2.3 Notificaciones y alertas | Media | No figura |
| HU-NOT-B02 | Invitación a un proyecto | 6.2.3 Notificaciones y alertas | Alta | No figura |
| HU-COM-B01 | Obtener comentarios de una versión | 6.2.4 Procesos / Transacciones del negocio | Alta | En Desarrollo |
| HU-COM-B02 | Agregar comentario a una versión | 6.2.4 Procesos / Transacciones del negocio | Alta | Codificado |
| HU-COM-B03 | Modificar comentario | 6.2.4 Procesos / Transacciones del negocio | Media | Codificado |
| HU-COM-B04 | Eliminar comentario | 6.2.4 Procesos / Transacciones del negocio | Media | Codificado |
| HU-VER-B01 | Obtener versiones y detalle de una canción | 6.2.4 Procesos / Transacciones del negocio | Alta | No figura |
| HU-IA-B01 | Procesamiento de separación de pistas | 6.2.4 Procesos / Transacciones del negocio | Alta | No figura |
| HU-PER-B01 | Obtener y actualizar perfil de usuario | 6.2.5 Perfil del usuario | Media | No figura |
| HU-REP-B01 | Obtener listado de reportes | 6.3.1 Reportes e informes | Media | No figura |
| HU-REP-B02 | Generar reporte | 6.3.1 Reportes e informes | Alta | No figura |
| HU-REP-B03 | Exportar reporte a PDF | 6.3.1 Reportes e informes | Alta | No figura |
| HU-DASH-B01 | Obtener indicadores numéricos del Tablero | 6.3.2 Tablero / Dashboard | Alta | No figura |
| HU-DASH-B02 | Obtener datos de los gráficos del Tablero | 6.3.2 Tablero / Dashboard | Alta | No figura |
| HU-CFG-B01 | Obtener listado de géneros musicales | 6.3.3 Configuración y parámetros | Alta | En Desarrollo |
| HU-CFG-B02 | Crear género musical | 6.3.3 Configuración y parámetros | Alta | En Desarrollo |
| HU-CFG-B03 | Editar género musical | 6.3.3 Configuración y parámetros | Media | En Desarrollo |
| HU-CFG-B04 | Desactivar / Activar género musical | 6.3.3 Configuración y parámetros | Media | En Desarrollo |
| HU-CFG-B05 | Obtener géneros activos para selector de proyectos | 6.3.3 Configuración y parámetros | Alta | En Desarrollo |
| HU-AYS-B1 | Obtener el las preguntas y respuestas frecuentes | 6.3.4 Ayuda y soporte al usuario | Baja | No figura |
| HU-AYS-B2 | Obtener el manual de usuario | 6.3.4 Ayuda y soporte al usuario | Baja | No figura |

---

## 6.1 Módulos transversales

### 6.1.1 Autenticación y control de acceso

#### HU-ACA-B01 — Registro de usuario

- **Módulo:** 6.1 Módulos transversales › 6.1.1 Autenticación y control de acceso
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Visitante no registrado
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como visitante sin cuenta, quiero que el sistema exponga un endpoint para registrar un nuevo usuario, para que el frontend pueda crear mi cuenta en el proveedor de identidad y mi perfil en StemHub.

**Precondiciones**

El visitante no está autenticado. El proveedor de identidad (IdP) se encuentra disponible.

**Permisos**

Endpoint público. No requiere token de sesión.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición con nombre, email, contraseña y aceptaTerminos válidos | El sistema crea la cuenta en el IdP, crea el perfil del usuario en StemHub con estado 'pendiente de verificación', registra la aceptación de términos (HU-SEG-B05), dispara el email de verificación y responde con HTTP 201 y el objeto creado (id, nombre, email, estado). No se devuelve token de sesión. | POST /auth/registro |
| 2 | Se envía una petición con algún campo obligatorio vacío o ausente | El sistema responde con HTTP 400 indicando los campos requeridos. | POST /auth/registro |
| 3 | El email tiene un formato inválido | El sistema responde con HTTP 400 indicando que el formato del email no es válido. | POST /auth/registro |
| 4 | La contraseña no cumple la política del IdP (menos de 8 caracteres) | El sistema responde con HTTP 400 indicando que la contraseña no cumple los requisitos mínimos. | POST /auth/registro |
| 5 | aceptaTerminos es false o está ausente | El sistema responde con HTTP 400 indicando que debe aceptar los términos y condiciones. | POST /auth/registro |
| 6 | El email ya pertenece a una cuenta existente | El sistema responde con HTTP 409 Conflict sin crear una nueva cuenta. | POST /auth/registro |
| 7 | El IdP no está disponible o responde con error | El sistema responde con HTTP 502 con un mensaje genérico, sin exponer detalles técnicos, y no deja registros parciales en la base de datos. | POST /auth/registro |
| 8 | Se completa el registro | El sistema registra el evento de auditoría 'registro de usuario' (ver HU-AYT-B1). | HU-AYT-B1 |

**Fuera de alcance**

- La verificación del email se cubre en HU-ACA-B02 y el email de bienvenida en HU-NOT-B01.
- El registro mediante cuenta de Google se resuelve por el flujo OAuth 2.0 del IdP y el perfil se crea en el primer inicio de sesión (HU-ACA-B03).

**Notas técnicas**

- StemHub no almacena ni registra contraseñas; solo las transmite al IdP por canal cifrado (HTTPS/TLS 1.2+).
- El rol de sistema por defecto de los usuarios registrados es el de usuario estándar; los roles dentro de cada proyecto se asignan por membresía.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si el alta se realiza a través de este endpoint o directamente desde el frontend hacia el IdP, sincronizando el perfil por webhook o en el primer inicio de sesión.
- [ ] Con desarrollo: Confirmar el límite de peticiones (rate limiting) para este endpoint.

---

#### HU-ACA-B02 — Verificación de email y reenvío

- **Módulo:** 6.1 Módulos transversales › 6.1.1 Autenticación y control de acceso
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Visitante no registrado
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como usuario recién registrado, quiero que el sistema exponga endpoints para confirmar mi email y para reenviar el email de verificación, para activar mi cuenta aunque el enlace original haya vencido o no lo haya recibido.

**Precondiciones**

El usuario posee una cuenta creada con email pendiente de verificación.

**Permisos**

Endpoints públicos. No requieren token de sesión; se protegen mediante el token de verificación y un límite de peticiones.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición de confirmación con un token de verificación válido y vigente | El sistema marca el email como verificado, actualiza el estado del perfil a 'activo', dispara el email de bienvenida y responde con HTTP 200. | POST /auth/verificacion-email/confirmar |
| 2 | Se envía una petición de confirmación con un token vencido | El sistema responde con HTTP 410 Gone indicando que el enlace expiró. | POST /auth/verificacion-email/confirmar |
| 3 | Se envía una petición de confirmación con un token inválido o alterado | El sistema responde con HTTP 400 sin modificar la cuenta. | POST /auth/verificacion-email/confirmar |
| 4 | Se envía una petición de confirmación con un token ya utilizado | El sistema responde con HTTP 409 Conflict indicando que el email ya fue verificado. | POST /auth/verificacion-email/confirmar |
| 5 | Se solicita el reenvío para un email con cuenta pendiente de verificación | El sistema invalida el token anterior, envía un nuevo email de verificación y responde con HTTP 202 Accepted. | POST /auth/verificacion-email/reenviar |
| 6 | Se solicita el reenvío para un email inexistente o ya verificado | El sistema responde con HTTP 202 Accepted sin enviar ningún email, para no revelar si la cuenta existe. | POST /auth/verificacion-email/reenviar |
| 7 | Se supera el límite de reenvíos permitidos en el período configurado | El sistema responde con HTTP 429 Too Many Requests e indica el tiempo de espera. | POST /auth/verificacion-email/reenviar |
| 8 | Se solicita el reenvío con un email de formato inválido o ausente | El sistema responde con HTTP 400. | POST /auth/verificacion-email/reenviar |

**Notas técnicas**

- El envío del email de verificación es responsabilidad del IdP; StemHub solo orquesta la solicitud y actualiza el estado del perfil.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar la vigencia del token de verificación y el límite de reenvíos por período.

---

#### HU-ACA-B03 — Inicio de sesión y emisión de token

- **Módulo:** 6.1 Módulos transversales › 6.1.1 Autenticación y control de acceso
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Visitante no registrado
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como usuario con una cuenta activa, quiero que el sistema valide mis credenciales a través del proveedor de identidad y devuelva un token de sesión, para acceder a los recursos protegidos de StemHub.

**Precondiciones**

El usuario posee una cuenta registrada. El proveedor de identidad (IdP) se encuentra disponible.

**Permisos**

Endpoint público. El token emitido contiene el rol del usuario como claim y es verificado en cada petición por HU-SEG-B01.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envían credenciales correctas de una cuenta activa y verificada | El sistema autentica al usuario mediante el IdP y responde con HTTP 200 con el token de sesión (JWT firmado con RS256 con el rol como claim), su vigencia y los datos básicos del usuario (id, nombre, email, rol). | POST /auth/login |
| 2 | Se envían credenciales incorrectas | El sistema responde con HTTP 401 con un mensaje genérico que no indica si el error corresponde al email o a la contraseña. | POST /auth/login |
| 3 | Se envía un email que no corresponde a ninguna cuenta registrada | El sistema responde con HTTP 401 con el mismo mensaje genérico que para credenciales incorrectas. | POST /auth/login |
| 4 | La cuenta alcanzó el límite de intentos fallidos consecutivos | El sistema responde con HTTP 429 indicando que el acceso está bloqueado temporalmente, según la política configurada en el IdP. | POST /auth/login |
| 5 | La cuenta no tiene el email verificado | El sistema responde con HTTP 403 con el código 'EMAIL_NO_VERIFICADO' para que el frontend ofrezca el reenvío del email de verificación. | POST /auth/login |
| 6 | La cuenta se encuentra suspendida o desactivada | El sistema responde con HTTP 403 con el código 'CUENTA_SUSPENDIDA'. | POST /auth/login |
| 7 | Se envía una petición con el email o la contraseña vacíos | El sistema responde con HTTP 400. | POST /auth/login |
| 8 | Un usuario autenticado mediante Google ingresa por primera vez | El sistema crea su perfil en StemHub a partir de los datos del IdP, con el email marcado como verificado, y responde como en un inicio de sesión exitoso. | GET /auth/callback |
| 9 | Se produce un inicio de sesión exitoso o fallido | El sistema registra el evento de auditoría con el usuario, fecha y hora, IP y resultado (ver HU-AYT-B1). | HU-AYT-B1 |
| 10 | El IdP no está disponible | El sistema responde con HTTP 502 con un mensaje genérico, sin exponer detalles técnicos. | POST /auth/login |

**Fuera de alcance**

- Las reglas de bloqueo, longitud y complejidad de contraseñas son configuradas y aplicadas por el IdP.

**Notas técnicas**

- Se utiliza el flujo Authorization Code con PKCE de OAuth 2.0; el backend valida el JWT con golang-jwt y go-oidc.
- El bloqueo por intentos fallidos se mantiene en el IdP; el backend solo traduce su respuesta a un código HTTP.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si el inicio de sesión con email y contraseña se hace mediante este endpoint o directamente entre el frontend y el IdP (con PKCE), en cuyo caso este endpoint se limita al callback y a la sincronización del perfil.
- [ ] Con desarrollo: Confirmar el tiempo de vigencia del token y la estrategia de renovación (refresh token).

---

#### HU-ACA-B04 — Cierre de sesión

- **Módulo:** 6.1 Módulos transversales › 6.1.1 Autenticación y control de acceso
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Baja

**Descripción**

Como usuario autenticado, quiero que el sistema exponga un endpoint para cerrar mi sesión e invalidar el token activo, para que nadie pueda reutilizarlo desde otro dispositivo o desde el mismo navegador.

**Precondiciones**

El usuario está autenticado con un token de sesión vigente emitido por el IdP.

**Permisos**

Accesible para todos los roles autenticados.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición de cierre de sesión con un token vigente | El sistema invalida la sesión en el IdP y responde con HTTP 204 No Content. | POST /auth/logout |
| 2 | Se realiza una petición a un endpoint protegido con un token invalidado por cierre de sesión | El sistema responde con HTTP 401 (ver HU-SEG-B02). | HU-SEG-B02 |
| 3 | Se envía la petición de cierre de sesión con un token ya vencido o ya invalidado | El sistema responde con HTTP 204 (operación idempotente). | POST /auth/logout |
| 4 | Se envía la petición sin token | El sistema responde con HTTP 401. | POST /auth/logout |
| 5 | Falla la invalidación de la sesión en el IdP | El sistema registra el error en el log, responde con HTTP 502 y el frontend elimina igualmente el token local. | POST /auth/logout |
| 6 | Se cierra la sesión | El sistema registra el evento de auditoría de cierre de sesión con el usuario, fecha y hora (ver HU-AYT-B1). | HU-AYT-B1 |

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si se invalida únicamente la sesión del dispositivo actual o todas las sesiones activas del usuario.

---

#### HU-ACA-B05 — Recuperación y restablecimiento de contraseña

- **Módulo:** 6.1 Módulos transversales › 6.1.1 Autenticación y control de acceso
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Visitante no registrado
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como usuario que olvidó su contraseña, quiero que el sistema exponga endpoints para solicitar un enlace de recuperación y establecer una nueva contraseña, para recuperar el acceso a mi cuenta de forma segura.

**Precondiciones**

El usuario posee una cuenta registrada. Para el restablecimiento, posee un token de recuperación vigente recibido por email.

**Permisos**

Endpoints públicos. El restablecimiento se protege mediante el token de recuperación de un solo uso.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita la recuperación para un email registrado | El sistema solicita al IdP el envío del email de recuperación con un enlace de un solo uso y responde con HTTP 202 Accepted. | POST /auth/recuperar-contrasena |
| 2 | Se solicita la recuperación para un email que no corresponde a ninguna cuenta | El sistema responde con HTTP 202 Accepted sin enviar ningún email, con la misma respuesta que para un email registrado. | POST /auth/recuperar-contrasena |
| 3 | Se solicita la recuperación con un email de formato inválido o ausente | El sistema responde con HTTP 400. | POST /auth/recuperar-contrasena |
| 4 | Se supera el límite de solicitudes de recuperación en el período configurado | El sistema responde con HTTP 429 Too Many Requests. | POST /auth/recuperar-contrasena |
| 5 | Se envía un token válido y una nueva contraseña que cumple la política | El sistema actualiza la contraseña mediante el IdP, invalida el token utilizado y las sesiones activas de la cuenta, y responde con HTTP 204. | POST /auth/restablecer-contrasena |
| 6 | Se envía una nueva contraseña que no cumple la política (menos de 8 caracteres) | El sistema responde con HTTP 400 indicando que la contraseña no cumple los requisitos mínimos. | POST /auth/restablecer-contrasena |
| 7 | Se envía un token vencido, inválido o ya utilizado | El sistema responde con HTTP 410 Gone (vencido o utilizado) o HTTP 400 (inválido) sin modificar la contraseña. | POST /auth/restablecer-contrasena |
| 8 | Se restablece una contraseña | El sistema registra el evento de auditoría (sin incluir la contraseña) y dispara una notificación de seguridad de cambio de credenciales (ver HU-NOT-B01). | HU-AYT-B1 |

**Notas técnicas**

- La contraseña nunca se registra en logs ni en el historial de auditoría.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar la vigencia del token de recuperación y el límite de solicitudes por período.

---

### 6.1.2 Gestión de usuarios, roles y permisos

#### HU-SEG-B03 — ABM Usuario

- **Módulo:** 6.1 Módulos transversales › 6.1.2 Gestión de usuarios, roles y permisos
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador del sistema
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga endpoints para listar, consultar, crear, editar y activar o desactivar usuarios, para que la pantalla de administración de usuarios pueda gestionar las cuentas de la plataforma.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema.

**Permisos**

Solo accesible para el rol Administrador del sistema. Cualquier otro rol recibe HTTP 403.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado de usuarios sin parámetros | El sistema responde con HTTP 200 y un objeto paginado (data, totalItems, totalPages, currentPage), ordenado por fecha de alta descendente. Cada objeto incluye: id, nombre, email, rolSistema, estado (activo / inactivo / pendiente), fechaAlta. | GET /usuarios?page=1&pageSize=10 |
| 2 | Se solicita el listado con parámetro de búsqueda (q), estado o rol | El sistema devuelve únicamente los usuarios cuyo nombre o email contiene el texto (sin distinguir mayúsculas) y que coinciden con los filtros, aplicados en conjunto (AND). | GET /usuarios?q=&estado=&rol= |
| 3 | Se solicita el detalle de un usuario existente | El sistema responde con HTTP 200 y el objeto del usuario. | GET /usuarios/{id} |
| 4 | Se solicita el detalle de un usuario inexistente | El sistema responde con HTTP 404. | GET /usuarios/{id} |
| 5 | Se envía una petición de creación con nombre, email y rol válidos | El sistema crea el usuario, dispara la invitación por email para que defina su contraseña y responde con HTTP 201 y el objeto creado. | POST /usuarios |
| 6 | Se envía una petición de creación con campos obligatorios vacíos o rol inexistente | El sistema responde con HTTP 400. | POST /usuarios |
| 7 | Se envía un email que ya pertenece a otro usuario (sin distinguir mayúsculas) | El sistema responde con HTTP 409 Conflict. | POST /usuarios |
| 8 | Se envía una petición de edición válida (nombre, rol de sistema) | El sistema actualiza el usuario y responde con HTTP 200 y el objeto actualizado. El email no es modificable. | PUT /usuarios/{id} |
| 9 | Se desactiva un usuario activo | El sistema cambia el estado a inactivo, invalida sus sesiones activas y responde con HTTP 200. El usuario no puede iniciar sesión (HU-ACA-B03). | PATCH /usuarios/{id}/estado |
| 10 | Se activa un usuario inactivo | El sistema cambia el estado a activo y responde con HTTP 200. | PATCH /usuarios/{id}/estado |
| 11 | Se intenta desactivar la propia cuenta o al último Administrador del sistema activo | El sistema responde con HTTP 409 Conflict sin modificar el estado. | PATCH /usuarios/{id}/estado |
| 12 | El usuario no está autenticado o no tiene rol de Administrador del sistema | El sistema responde con HTTP 401 o 403 según corresponda. | Todos los endpoints de /usuarios |
| 13 | Se crea, edita o cambia el estado de un usuario | El sistema registra el evento de auditoría con el valor anterior y el nuevo valor (ver HU-AYT-B1). | HU-AYT-B1 |

**Fuera de alcance**

- No se contempla la eliminación física de usuarios; se conserva el registro para la trazabilidad.

**Notas técnicas**

- La comparación de emails duplicados debe ser case-insensitive.
- La creación y la invitación se delegan en el IdP; el backend sincroniza el perfil local.

**Puntos a revisar**

- [ ] Con PO: Confirmar si el Administrador puede crear usuarios y cambiar el rol de sistema de un usuario existente.

---

#### HU-SEG-B04 — ABM Rol

- **Módulo:** 6.1 Módulos transversales › 6.1.2 Gestión de usuarios, roles y permisos
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador del sistema
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga endpoints para gestionar roles y sus permisos asociados, para que la pantalla de administración de roles pueda persistir la configuración de acceso de la plataforma.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema.

**Permisos**

Solo accesible para el rol Administrador del sistema. Cualquier otro rol recibe HTTP 403.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado de roles | El sistema responde con HTTP 200 y un array de roles. Cada objeto incluye: id, nombre, descripcion, estado, cantidadPermisos. | GET /roles |
| 2 | Se solicita el detalle de un rol existente | El sistema responde con HTTP 200 con el rol y la lista de permisos asociados (gestionar usuarios, roles, proyectos, canciones, versiones, comentarios y géneros musicales). | GET /roles/{id} |
| 3 | Se envía una petición de creación con nombre válido y no existente | El sistema crea el rol y responde con HTTP 201 y el objeto creado. | POST /roles |
| 4 | Se envía una petición de creación con el nombre vacío o ausente | El sistema responde con HTTP 400 indicando que el nombre es requerido. | POST /roles |
| 5 | Se envía un nombre de rol que ya existe (sin distinguir mayúsculas) | El sistema responde con HTTP 409 Conflict. | POST /roles |
| 6 | Se envía una petición de edición con descripción y permisos válidos | El sistema actualiza el rol y sus permisos asociados y responde con HTTP 200 y el objeto actualizado. | PUT /roles/{id} |
| 7 | Se intenta quitar permisos críticos al rol Administrador del sistema | El sistema responde con HTTP 409 Conflict sin modificar el rol, para evitar dejar la plataforma sin administración. | PUT /roles/{id} |
| 8 | Se solicita la eliminación de un rol sin usuarios ni proyectos asociados | El sistema elimina el rol y responde con HTTP 204. | DELETE /roles/{id} |
| 9 | Se solicita la eliminación de un rol asociado a usuarios o proyectos activos | El sistema responde con HTTP 409 Conflict indicando que el rol se encuentra en uso. | DELETE /roles/{id} |
| 10 | El ID no corresponde a ningún rol existente | El sistema responde con HTTP 404. | GET / PUT / DELETE /roles/{id} |
| 11 | El usuario no está autenticado o no tiene rol de Administrador del sistema | El sistema responde con HTTP 401 o 403 según corresponda. | Todos los endpoints de /roles |
| 12 | Se crea, edita o elimina un rol | El sistema registra el evento de auditoría con el valor anterior y el nuevo valor (ver HU-AYT-B1). | HU-AYT-B1 |

**Notas técnicas**

- Los permisos del rol se reflejan como claims en el token emitido por el IdP; los cambios se aplican en los siguientes inicios de sesión o renovaciones de token.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si los cambios de permisos deben aplicarse de inmediato a las sesiones activas o al renovar el token.

---

### 6.1.3 Seguridad

#### HU-SEG-B01 — Autorización de acceso a recursos protegidos

- **Módulo:** 6.1 Módulos transversales › 6.1.3 Seguridad
- **Estado (matriz de trazabilidad):** Codificado
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como miembro autenticado de la plataforma, quiero que el sistema valide mis permisos antes de procesar cualquier operación sobre un recurso, para que solo pueda ejecutar las acciones que corresponden a mi rol dentro de cada proyecto.

**Precondiciones**

El usuario está autenticado y el sistema recibió un token de sesión válido emitido por el IdP.

**Permisos**

Aplica a todos los roles. El sistema verifica en cada request que el token sea válido y que el rol declarado en él esté habilitado para la operación solicitada.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se realiza una petición a cualquier endpoint protegido sin token de sesión | El sistema responde con HTTP 401. | — |
| 2 | Se realiza una petición con un token cuya firma no es válida o fue alterado | El sistema responde con HTTP 401. | Todos los endpoints protegidos del sistema |
| 3 | Se realiza una petición con un token válido pero el rol declarado no tiene permiso para esa operación | El sistema responde con HTTP 403 sin ejecutar la operación solicitada. | Matriz de permisos — DOC UNIFICADO sección 14.1.2 |
| 4 | Un Músico envía una petición para crear una versión | El sistema responde con HTTP 403. | Matriz de permisos — DOC UNIFICADO sección 14.1.2 |

**Puntos a revisar**

- [ ] La referencia "DOC UNIFICADO sección 14.1.2" corresponde a la Matriz de permisos por rol incluida en este archivo.

---

#### HU-SEG-B02 — Expiración de sesión por inactividad o token vencido

- **Módulo:** 6.1 Módulos transversales › 6.1.3 Seguridad
- **Estado (matriz de trazabilidad):** No Iniciado
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como miembro autenticado de la plataforma, quiero que el sistema detecte cuando mi sesión ya no es válida y rechace mis peticiones, para evitar accesos no autorizados desde sesiones abandonadas o tokens comprometidos.

**Precondiciones**

El usuario tenía una sesión activa. El token emitido por el IdP fue utilizado previamente para acceder al sistema.

**Permisos**

Aplica a todos los roles

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se realiza una petición con un token cuya fecha de expiración ya venció | El sistema responde con HTTP 401 sin procesar la operación solicitada. | — |
| 2 | Se realiza una petición con un token que fue invalidado mediante cierre de sesión explícito | El sistema responde con HTTP 401. | — |
| 3 | El usuario estuvo inactivo durante 45 minutos y luego intenta realizar una operación | El sistema rechaza la petición con HTTP 401, dado que el token ya no debe considerarse vigente para ese período de inactividad. | — |
| 4 | Se realiza una petición con un token válido y vigente | El sistema procesa la petición normalmente. | — |

---

#### HU-SEG-B05 — Registro de aceptación de términos y condiciones

- **Módulo:** 6.1 Módulos transversales › 6.1.3 Seguridad
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Media
- **Complejidad:** Baja

**Descripción**

Como usuario de la plataforma, quiero que el sistema registre la versión de los términos y condiciones que acepté junto con la fecha y hora, para dejar constancia de mi aceptación y que se me solicite nuevamente cuando cambien.

**Precondiciones**

El usuario completó el registro o inició sesión y tiene un token de sesión válido.

**Permisos**

Accesible para todos los roles autenticados sobre su propia cuenta.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicitan los términos y condiciones vigentes | El sistema responde con HTTP 200 con el identificador de versión, el texto y la fecha de vigencia. | GET /terminos-condiciones/vigente |
| 2 | Se envía la aceptación de la versión vigente | El sistema registra la aceptación (usuarioId, versión, fecha y hora, IP) y responde con HTTP 201. | POST /usuarios/me/terminos-condiciones |
| 3 | Se envía la aceptación de una versión que no es la vigente | El sistema responde con HTTP 409 Conflict indicando la versión vigente. | POST /usuarios/me/terminos-condiciones |
| 4 | El usuario inicia sesión y su última aceptación corresponde a una versión anterior | La respuesta del inicio de sesión indica que debe aceptar nuevamente los términos (terminosPendientes = true). | POST /auth/login |
| 5 | El usuario no está autenticado | El sistema responde con HTTP 401. | POST /usuarios/me/terminos-condiciones |
| 6 | Se registra una aceptación | El registro de aceptación es inmutable y no puede ser modificado ni eliminado por usuarios comunes (ver HU-AYT-B1). | HU-AYT-B1 |

**Puntos a revisar**

- [ ] Con PO / Legales: Definir el versionado de los términos y condiciones y cuándo se considera obligatoria una nueva aceptación.

---

#### HU-SEG-B06 — Validación de datos y manejo controlado de errores

- **Módulo:** 6.1 Módulos transversales › 6.1.3 Seguridad
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como sistema, quiero validar en el backend todos los datos y archivos recibidos y responder los errores de forma controlada, para evitar datos inconsistentes, usos indebidos y la exposición de información técnica.

**Precondiciones**

El sistema recibe una petición a cualquier endpoint que acepta datos o archivos.

**Permisos**

Aplica a todos los endpoints y a todos los roles.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se recibe una petición con campos obligatorios vacíos, ausentes o con tipo de dato incorrecto | El sistema responde con HTTP 400 con un cuerpo estandarizado que identifica cada campo con error y un mensaje claro, sin ejecutar la operación. | Todos los endpoints |
| 2 | Se recibe un texto que supera la longitud máxima del campo | El sistema responde con HTTP 400 indicando el máximo permitido. | Todos los endpoints |
| 3 | Se recibe un archivo con formato o tipo de contenido no permitido | El sistema responde con HTTP 415 Unsupported Media Type. La validación se basa en el contenido real del archivo y no únicamente en su extensión. | Endpoints de carga de archivos |
| 4 | Se recibe un archivo que supera el tamaño máximo permitido | El sistema responde con HTTP 413 Payload Too Large. | Endpoints de carga de archivos |
| 5 | Se recibe un texto con caracteres de control o código potencialmente malicioso | El sistema rechaza o sanea el valor antes de procesarlo y todas las consultas a la base de datos se ejecutan con parámetros, nunca concatenando valores del usuario. | Todos los endpoints |
| 6 | Se recibe un identificador de recurso con formato inválido | El sistema responde con HTTP 400. | Endpoints con {id} |
| 7 | Ocurre un error interno no controlado | El sistema responde con HTTP 500 con un mensaje genérico y un código de correlación. No se expone información técnica (rutas internas, consultas SQL, nombres de tablas, trazas de pila). El detalle se registra únicamente en el log del servidor. | Todos los endpoints |
| 8 | Falla un servicio externo (IdP, almacenamiento de objetos, microservicio de IA, proveedor de emails) | El sistema responde con HTTP 502 o 503 con un mensaje genérico, mantiene operativas las funcionalidades que no dependen del servicio afectado y registra el incidente. | Todos los endpoints |

**Fuera de alcance**

- Las reglas específicas de validación de cada recurso se definen en la HU de cada endpoint.

**Notas técnicas**

- Los códigos de error siguen un formato estándar: { codigo, mensaje, errores: [{ campo, mensaje }], correlacionId }.
- Los límites de tamaño y los formatos permitidos deben ser configurables.

**Puntos a revisar**

- [ ] Con PO: Definir los formatos de audio permitidos y el tamaño máximo por archivo.
- [ ] Con desarrollo: Definir los límites de peticiones (rate limiting) por endpoint.

---

### 6.1.4 Auditoría y trazabilidad

#### HU-AYT-B1 — Middleware de auditoría y trazabilidad

- **Módulo:** 6.1 Módulos transversales › 6.1.4 Auditoría y trazabilidad
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** El sistema
- **Prioridad:** Media
- **Complejidad:** Alta

**Descripción**

Como sistema, quiero interceptar y registrar automáticamente cad acción relevante que realice un usuario autenticado, para mantener un historial completo y confiable de todo lo ocurrido

**Precondiciones**

Este creado el middleware global de auditoria

**Permisos**

Sin definir (ver Puntos a revisar).

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | El usuario realiza una acción en cualquier módulo del sistema | El middleware global de auditoría intercepte automáticamente la petición y envía los datos duplicados y genere un evento de auditoría y guarde: usuario_id, rol, acción, modulo, entidad_id, ip, fecha y resultado. | — |
| 2 | Ocurra un error durante el proceso de registro de auditoría. | El sistema realice reintentos automáticos antes de registrar definitivamente el error | — |
| 3 | Cuando se agoten los intentos de registros | El sistema lo guarda en los logs y crea una cola de errores para su posterior reprocesamiento | — |

**Puntos a revisar**

- [ ] La ficha no tiene Nombre ni Permisos definidos.
- [ ] El documento plantea la auditoría mediante triggers de base de datos (históricos por entidad) más una tabla general de auditoría; esta HU define un middleware. Confirmar si se implementan ambos niveles.
- [ ] Los registros de auditoría no deben poder modificarse ni eliminarse por usuarios comunes.

---

## 6.2 Módulos funcionales

### 6.2.1 Gestión de datos maestros (ABMs)

#### HU-ABM-B01 — ABM Proyecto

- **Módulo:** 6.2 Módulos funcionales › 6.2.1 Gestión de datos maestros (ABMs)
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como usuario autenticado, quiero que el sistema exponga endpoints para crear, consultar, editar y eliminar proyectos, para que el formulario y el detalle de proyecto puedan persistir y mostrar la información del proyecto.

**Precondiciones**

El usuario está autenticado. Para consultar, editar o eliminar, es miembro del proyecto.

**Permisos**

Crear: cualquier usuario autenticado. Consultar: miembros del proyecto. Editar y eliminar: Administrador del sistema, Administrador de proyecto y Productor. El Músico recibe HTTP 403 al editar o eliminar.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición de creación con nombre, tipo y género válidos | El sistema crea el proyecto, asigna al usuario autenticado como Administrador de proyecto y responde con HTTP 201 y el objeto creado (id, nombre, descripcion, tipo, estado, generoId, portadaUrl, fechaCreacion). | POST /proyectos |
| 2 | Se envía una petición de creación con el nombre vacío o ausente | El sistema responde con HTTP 400 indicando que el nombre es requerido. | POST /proyectos |
| 3 | Se envía un tipo distinto de EP, Álbum o Single | El sistema responde con HTTP 400 indicando que el tipo no es válido. | POST /proyectos |
| 4 | Se envía un género inexistente o inactivo | El sistema responde con HTTP 400 indicando que el género no es válido (ver HU-CFG-B05). | POST /proyectos |
| 5 | Se envía una portada con formato no permitido o mayor a 5 MB | El sistema responde con HTTP 415 o HTTP 413 según corresponda, sin crear el proyecto. | POST /proyectos |
| 6 | Se solicita el detalle de un proyecto del que el usuario es miembro | El sistema responde con HTTP 200 con el proyecto, el rol del usuario en el proyecto y la cantidad de canciones. | GET /proyectos/{proyectoId} |
| 7 | Se envía una petición de edición válida por un Administrador de proyecto o Productor | El sistema actualiza el proyecto y responde con HTTP 200 y el objeto actualizado. | PUT /proyectos/{proyectoId} |
| 8 | Un Músico intenta editar o eliminar el proyecto | El sistema responde con HTTP 403 sin ejecutar la operación. | PUT / DELETE /proyectos/{proyectoId} |
| 9 | Se solicita la eliminación de un proyecto por un rol habilitado | El sistema elimina el proyecto junto con sus canciones, versiones, comentarios, stems y archivos asociados, y responde con HTTP 204. | DELETE /proyectos/{proyectoId} |
| 10 | El proyectoId no corresponde a ningún proyecto existente | El sistema responde con HTTP 404. | GET / PUT / DELETE /proyectos/{proyectoId} |
| 11 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | GET / PUT / DELETE /proyectos/{proyectoId} |
| 12 | El usuario no está autenticado | El sistema responde con HTTP 401. | Todos los endpoints |
| 13 | Se crea, edita o elimina un proyecto | El sistema registra el evento de auditoría y el histórico de la entidad con el valor anterior y el nuevo valor (ver HU-AYT-B1). | HU-AYT-B1 |

**Fuera de alcance**

- El listado y la búsqueda de proyectos se cubren en HU-BUQ-B01.
- La invitación y baja de colaboradores se cubre en HU-NOT-B02.

**Notas técnicas**

- El campo estado del proyecto se establece automáticamente al crear; no debe ser enviado por el cliente.
- La eliminación de archivos en el almacenamiento de objetos debe ejecutarse de forma que una falla parcial no deje registros huérfanos.

**Puntos a revisar**

- [ ] Con PO: Confirmar si la eliminación de un proyecto es permanente (hard delete) o aplica baja lógica.
- [ ] Con desarrollo: Definir el ID como UUID o entero autoincremental.

---

#### HU-ABM-B02 — ABM Canción

- **Módulo:** 6.2 Módulos funcionales › 6.2.1 Gestión de datos maestros (ABMs)
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como Administrador de proyecto o Productor, quiero que el sistema exponga endpoints para crear, consultar, editar y eliminar canciones de un proyecto, para que los formularios y el detalle de proyecto puedan gestionar el contenido musical.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto al que pertenece la canción.

**Permisos**

Crear, editar y eliminar: Administrador del sistema, Administrador de proyecto y Productor. Consultar: cualquier miembro del proyecto. El Músico recibe HTTP 403 al crear, editar o eliminar.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición de creación (multipart) con nombre, género y archivo de audio válidos | El sistema almacena el audio en el almacenamiento de objetos, crea la canción y su versión inicial como versión actual, y responde con HTTP 201 y el objeto creado (id, nombre, descripcion, generoId, portadaUrl, estado, versionActual). | POST /proyectos/{proyectoId}/canciones |
| 2 | Se envía una petición con el nombre vacío o ausente | El sistema responde con HTTP 400 indicando que el nombre es requerido. | POST /proyectos/{proyectoId}/canciones |
| 3 | Se envía una petición sin archivo de audio | El sistema responde con HTTP 400 indicando que el audio es requerido. | POST /proyectos/{proyectoId}/canciones |
| 4 | Se envía un archivo de audio con formato no permitido o que supera el tamaño máximo | El sistema responde con HTTP 415 o HTTP 413 según corresponda, sin crear la canción. | POST /proyectos/{proyectoId}/canciones |
| 5 | Falla el almacenamiento del audio | El sistema no crea la canción ni la versión (operación atómica), responde con HTTP 502 con un mensaje genérico y registra el incidente. | POST /proyectos/{proyectoId}/canciones |
| 6 | Se solicita el detalle de una canción existente | El sistema responde con HTTP 200 con los datos de la canción y su versión actual. | GET /proyectos/{proyectoId}/canciones/{cancionId} |
| 7 | Se envía una petición de edición válida (nombre, descripción, género, portada) | El sistema actualiza la canción y responde con HTTP 200 y el objeto actualizado. | PUT /proyectos/{proyectoId}/canciones/{cancionId} |
| 8 | Se solicita la eliminación de una canción | El sistema elimina la canción junto con sus versiones, comentarios, stems y archivos asociados, y responde con HTTP 204. | DELETE /proyectos/{proyectoId}/canciones/{cancionId} |
| 9 | Un Músico intenta crear, editar o eliminar una canción | El sistema responde con HTTP 403. | POST / PUT / DELETE |
| 10 | El proyectoId o el cancionId no corresponden a un recurso existente | El sistema responde con HTTP 404. | Todos los endpoints |
| 11 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | Todos los endpoints |
| 12 | El usuario no está autenticado | El sistema responde con HTTP 401. | Todos los endpoints |
| 13 | Se crea, edita o elimina una canción | El sistema registra el evento de auditoría y el histórico de la entidad (ver HU-AYT-B1). | HU-AYT-B1 |

**Fuera de alcance**

- El listado y la búsqueda de canciones se cubren en HU-BUQ-B02.
- La obtención de versiones se cubre en HU-VER-B01 y la creación de nuevas versiones en HU-ABM-B03.

**Notas técnicas**

- La URL de acceso al audio nunca es la ruta física del archivo; se utilizan URL firmadas con expiración (ver HU-VER-B01).
- La creación de la canción y de su versión inicial debe ejecutarse en una única transacción.

**Puntos a revisar**

- [ ] Con PO: Confirmar si el audio es obligatorio al crear la canción y la etapa asignada a la versión inicial.
- [ ] Con PO: Confirmar si la eliminación de canciones es permanente o aplica baja lógica.

---

#### HU-ABM-B03 — ABM Versión

- **Módulo:** 6.2 Módulos funcionales › 6.2.1 Gestión de datos maestros (ABMs)
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como Administrador de proyecto o Productor, quiero que el sistema exponga endpoints para crear, editar y eliminar versiones de una canción, para que el formulario de nueva versión pueda registrar la evolución de la canción.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto. La canción con el ID indicado existe.

**Permisos**

Crear, editar y eliminar versión: Administrador del sistema, Administrador de proyecto y Productor (según la Matriz de permisos por rol). El Músico recibe HTTP 403.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición de creación (multipart) con audio, etapa y descripción de cambios válidos | El sistema almacena el audio, genera automáticamente el siguiente número de versión, marca la nueva versión como actual (esActual = true) y la anterior como no actual, y responde con HTTP 201 y el objeto creado (id, numeroVersion, etapa, instrumentos, descripcion, tags, estado, fechaCarga, autor, esActual). | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 2 | Se envía una petición sin archivo de audio | El sistema responde con HTTP 400 indicando que el audio es requerido. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 3 | Se envía una petición con etapa inexistente o descripción de cambios vacía | El sistema responde con HTTP 400 indicando el campo con error. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 4 | Se envía un audio con formato no permitido o que supera el tamaño máximo | El sistema responde con HTTP 415 o HTTP 413 según corresponda, sin crear la versión. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 5 | Se envían tags duplicados | El sistema conserva una única ocurrencia de cada tag. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 6 | Falla el almacenamiento del audio | El sistema no crea la versión ni modifica la versión actual (operación atómica), responde con HTTP 502 y registra el incidente. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 7 | Se crea la versión correctamente | El sistema dispara el evento 'Nueva versión subida' para notificar a los demás miembros del proyecto (ver HU-NOT-B01). | HU-NOT-B01 |
| 8 | Se envía una petición de edición de etapa, descripción, instrumentos o tags | El sistema actualiza los metadatos de la versión (no el audio) y responde con HTTP 200 y el objeto actualizado. | PUT /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId} |
| 9 | Se solicita la eliminación de una versión que no es la única de la canción | El sistema elimina la versión y su audio. Si era la versión actual, la versión inmediata anterior pasa a ser la actual. Responde con HTTP 204. | DELETE /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId} |
| 10 | Se solicita la eliminación de la única versión de una canción | El sistema responde con HTTP 409 Conflict indicando que la canción debe conservar al menos una versión. | DELETE /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId} |
| 11 | Un Músico intenta crear, editar o eliminar una versión | El sistema responde con HTTP 403. | POST / PUT / DELETE |
| 12 | El cancionId o el versionId no corresponden a un recurso existente | El sistema responde con HTTP 404. | Todos los endpoints |
| 13 | El usuario no pertenece al proyecto o no está autenticado | El sistema responde con HTTP 403 o HTTP 401 según corresponda. | Todos los endpoints |
| 14 | Se crea, edita o elimina una versión | El sistema registra el evento de auditoría con el usuario que realizó la acción y la fecha y hora (ver HU-AYT-B1). | HU-AYT-B1 |

**Fuera de alcance**

- La obtención del listado y del detalle de versiones se cubre en HU-VER-B01.
- El árbol de ramas de versiones no forma parte de este endpoint.

**Notas técnicas**

- El número de versión sigue el formato vX.Y.Z (ej. v1.1.0) y se asigna en el backend; el cliente no lo envía.
- El campo esActual debe ser true únicamente para la última versión cargada de la canción.
- El autor se obtiene del token JWT; el cliente no envía el userId.

**Puntos a revisar**

- [ ] Con PO: Definir la regla de numeración (qué incrementa X, Y o Z según la etapa o el tipo de cambio).
- [ ] Con PO: Confirmar si el audio de una versión puede reemplazarse una vez creada.
- [ ] Con desarrollo: Confirmar si la eliminación de versiones es permanente o aplica baja lógica.

---

### 6.2.2 Búsqueda y filtrado

#### HU-BUQ-B01 — Búsqueda y filtrado de proyectos

- **Módulo:** 6.2 Módulos funcionales › 6.2.2 Búsqueda y filtrado
- **Estado (matriz de trazabilidad):** No Iniciado
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como usuario autenticado, quiero que el sistema exponga un endpoint para buscar y filtrar mis proyectos, para que la pantalla Mis Proyectos pueda mostrar únicamente los resultados que coincidan con el texto y los filtros aplicados.

**Precondiciones**

El usuario está autenticado.

**Permisos**

Accesible para todos los roles autenticados. El sistema devuelve únicamente los proyectos en los que el usuario participa como creador o colaborador.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado de proyectos sin parámetros | El sistema responde con HTTP 200 y un array con todos los proyectos del usuario autenticado, ordenados por fecha de última modificación descendente. Cada objeto incluye: id, nombre, tipo (EP / Álbum / Single), estado, cantidadCanciones, portadaUrl, fechaUltimaModificacion, rolDelUsuario. | GET /proyectos?page=1&pageSize=10 |
| 2 | Se solicita con parámetro de búsqueda por texto (q) | El sistema devuelve únicamente los proyectos cuyo nombre contiene el texto ingresado (búsqueda parcial, sin distinción de mayúsculas). El totalItems refleja el total filtrado. . | — |
| 3 | Se solicita filtrando por estado | El sistema devuelve únicamente los proyectos cuyo estado coincide con el valor indicado. Valores posibles: En Progreso, Maquetación, Mezclado, Publicado. | — |
| 4 | Se solicita filtrando por tipo | El sistema devuelve únicamente los proyectos cuyo tipo coincide con el valor indicado. Valores posibles: EP, Álbum, Single. | — |
| 5 | Se combinan búsqueda por texto y uno o más filtros | El sistema aplica todos los criterios de forma simultánea (AND). El totalItems refleja el total tras aplicar todos los criterios. | — |
| 6 | Se solicita con page y pageSize | El sistema responde con HTTP 200 y la página solicitada del subconjunto filtrado. | GET /proyectos?page=2&pageSize=10 |
| 7 | Se solicita sin los parámetros page y pageSize | El sistema aplica valores por defecto: page=1, pageSize=10. | — |
| 8 | Se solicita una página que supera el total disponible | El sistema responde con HTTP 200 y data vacío, con currentPage igual al valor solicitado y totalPages correcto. | — |
| 9 | El usuario no está autenticado | El sistema responde con HTTP 401. | — |

---

#### HU-BUQ-B02 — Búsqueda y filtrado de canciones

- **Módulo:** 6.2 Módulos funcionales › 6.2.2 Búsqueda y filtrado
- **Estado (matriz de trazabilidad):** No Iniciado
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como usuario autenticado, quiero que el sistema exponga un endpoint para buscar, filtrar y paginar mis canciones, para que la pantalla Mis Canciones pueda mostrar únicamente los resultados que coincidan con el texto y los filtros aplicados.

**Precondiciones**

El usuario está autenticado.

**Permisos**

Accesible para todos los roles autenticados. El sistema devuelve únicamente las canciones pertenecientes a proyectos en los que el usuario participa.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado sin parámetros | El sistema responde con HTTP 200 y un objeto paginado con data, totalItems, totalPages, currentPage. Orden por defecto: fecha de última modificación descendente. Cada objeto incluye: id, nombre, estado, etapaVersionado, proyectoId, proyectoNombre. | GET /canciones?page=1&pageSize=10 |
| 2 | Se solicita con parámetro de búsqueda por texto (q) | El sistema devuelve únicamente las canciones cuyo nombre contiene el texto ingresado (búsqueda parcial, sin distinción de mayúsculas). El totalItems refleja el total filtrado. | — |
| 3 | Se solicita filtrando por estado de la canción | El sistema devuelve únicamente las canciones cuyo estado coincide con el valor indicado. Valores posibles: Actual, Maqueta, Mezclada, En Revisión. | — |
| 4 | Se solicita filtrando por etapa de versionado | El sistema devuelve únicamente las canciones cuya etapa de versionado coincide con el valor indicado. Valores posibles: Maquetación, Composición, Mezcla. | — |
| 5 | Se combinan búsqueda por texto y uno o más filtros | El sistema aplica todos los criterios de forma simultánea (AND). El totalItems refleja el total tras aplicar todos los criterios. | — |
| 6 | Se solicita con page y pageSize | El sistema responde con HTTP 200 y la página solicitada del subconjunto filtrado. | GET /canciones?page=2&pageSize=10 |
| 7 | Se solicita sin los parámetros page y pageSize | El sistema aplica valores por defecto: page=1, pageSize=10. | — |
| 8 | Se solicita una página que supera el total disponible | El sistema responde con HTTP 200 y data vacío, con currentPage igual al valor solicitado y totalPages correcto. | — |

---

#### HU-BUQ-B03 — Búsqueda y filtrado de stems

- **Módulo:** 6.2 Módulos funcionales › 6.2.2 Búsqueda y filtrado
- **Estado (matriz de trazabilidad):** No Iniciado
- **Actor:** Productor
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como productor, quiero que el sistema exponga un endpoint para buscar, filtrar y paginar los stems de una versión, para que la pantalla Stems pueda mostrar únicamente los resultados que coincidan con el texto y los filtros aplicados.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto al que pertenece la versión.

**Permisos**

Accesible únicamente para el rol Productor dentro del proyecto. Retorna HTTP 403 si el usuario no tiene ese rol en el proyecto.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado sin parámetros | El sistema responde con HTTP 200 y un objeto paginado con data, totalItems, totalPages, currentPage. Orden por defecto: fecha de creación descendente. Cada objeto incluye: id, nombre, tipoInstrumento, bpm, duracionSegundos, archivoUrl. | /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems?page=1&pageSize=10 |
| 2 | Se solicita con parámetro de búsqueda por texto (q) | El sistema devuelve únicamente los stems cuyo nombre contiene el texto ingresado (búsqueda parcial, sin distinción de mayúsculas). | — |
| 3 | Se solicita filtrando por tipo de instrumento | El sistema devuelve únicamente los stems cuyo tipoInstrumento coincide con el valor indicado. Valores posibles: Batería, Bajo, Guitarra, Voz, Sintes. | — |
| 4 | Se solicita filtrando por BPM | El sistema devuelve únicamente los stems cuyo BPM se encuentra dentro del rango indicado (bpmMin, bpmMax). | — |
| 5 | Se solicita filtrando por duración | El sistema devuelve únicamente los stems cuya duración se encuentra dentro del rango indicado (duracionMin, duracionMax) expresado en segundos. | — |
| 6 | Se combinan búsqueda por texto y uno o más filtros | El sistema aplica todos los criterios de forma simultánea (AND). El totalItems refleja el total tras aplicar todos los criterios. | — |
| 7 | Se solicita con page y pageSize | El sistema responde con HTTP 200 y la página solicitada del subconjunto filtrado. | — |
| 8 | Se solicita sin los parámetros page y pageSize | El sistema aplica valores por defecto: page=1, pageSize=10. | — |

**Puntos a revisar**

- [ ] El endpoint está acotado a una versión, pero la pantalla Stems (HU-VER-F04) lista stems de todos los proyectos del usuario. Definir si se necesita un endpoint a nivel usuario.

---

### 6.2.3 Notificaciones y alertas

#### HU-NOT-B01 — Disparo y envío de notificaciones por email

- **Módulo:** 6.2 Módulos funcionales › 6.2.3 Notificaciones y alertas
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** El sistema
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como sistema, quiero detectar los eventos relevantes del proyecto y enviar la notificación correspondiente por email a los destinatarios que corresponda, para mantener informados a los usuarios sin que deban ingresar permanentemente a la plataforma.

**Precondiciones**

Existe un servicio de emails transaccionales externo configurado. El evento ocurrió sobre una entidad existente y los destinatarios poseen cuentas verificadas.

**Permisos**

Proceso interno del sistema. No es invocado directamente por los usuarios.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Un usuario confirma su email | El sistema envía el email de 'Bienvenida' al usuario, con un resumen de funcionalidades y un acceso directo a la plataforma. | Evento: Bienvenida |
| 2 | Se crea una invitación a un proyecto (HU-NOT-B02) | El sistema envía el email de 'Invitación a un proyecto' al email invitado, con el nombre del proyecto, el rol asignado, quién invita y el enlace de aceptación. | Evento: Invitación a un proyecto |
| 3 | Se crea una nueva versión (HU-ABM-B03) | El sistema envía el email de 'Nueva versión subida' al Productor y a los Músicos del proyecto, excepto al autor de la versión. | Evento: Nueva versión subida |
| 4 | Se agrega un comentario dirigido a un usuario (HU-COM-B02) | El sistema envía el email de 'Comentario recibido' al usuario destinatario del comentario. | Evento: Comentario recibido |
| 5 | El destinatario es el mismo usuario que ejecutó la acción | El sistema no envía la notificación. | Regla de negocio |
| 6 | El destinatario no tiene la cuenta verificada | El sistema no envía la notificación. | Regla de negocio |
| 7 | Se construye el contenido del email | El contenido identifica claramente el proyecto, la canción cuando corresponde, el evento que lo originó y contiene un enlace directo al recurso. | Regla de negocio |
| 8 | Falla el envío al proveedor de emails | La falla no interrumpe ni revierte la operación original. El sistema reintenta el envío de forma automática y, agotados los reintentos, registra el error en el log. | Resiliencia |
| 9 | El proveedor informa un rebote o un error de entrega | La gestión del rebote es responsabilidad del proveedor externo; StemHub registra el evento para su seguimiento. | Responsabilidad por capa |
| 10 | Se envía una notificación | El sistema registra el evento (tipo, destinatario, fecha y hora, resultado) para su trazabilidad. | HU-AYT-B1 |

**Fuera de alcance**

- La verificación del email al registrarse y la gestión de rebotes son responsabilidad del proveedor externo.
- No se contempla la configuración de preferencias de notificación por usuario.

**Notas técnicas**

- El envío se realiza de forma asíncrona (cola o evento) para no demorar la respuesta de los endpoints que originan el evento.
- El backend decide a quién notificar y construye el contenido; el proveedor externo realiza el envío y la entrega.

**Puntos a revisar**

- [ ] Con PO: Confirmar si las notificaciones de seguridad (cambio de contraseña, cambio de rol, incorporación a un proyecto) forman parte de esta HU.
- [ ] Con desarrollo: Definir el proveedor de emails transaccionales y la estrategia de reintentos.
- [ ] El evento "Comentario recibido" requiere definir a quién va dirigido un comentario: el modelo de comentario actual no tiene destinatario (ver HU-COM-B02).

---

#### HU-NOT-B02 — Invitación a un proyecto

- **Módulo:** 6.2 Módulos funcionales › 6.2.3 Notificaciones y alertas
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga endpoints para invitar colaboradores por email o por enlace y para aceptar la invitación, para incorporarlos al proyecto con el rol asignado.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto.

**Permisos**

Invitar: miembros del proyecto con permiso de invitación según la Matriz de permisos por rol. Solo se pueden asignar los roles Productor o Músico. Aceptar: cualquier usuario autenticado con una invitación válida.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una invitación por email con un email válido y un rol Productor o Músico | El sistema crea la invitación con un token de un solo uso y vencimiento, dispara el email de invitación (ver HU-NOT-B01) y responde con HTTP 201 y el objeto creado (id, email, rol, estado = pendiente, fechaVencimiento). | POST /proyectos/{proyectoId}/invitaciones |
| 2 | Se solicita un enlace de invitación con un rol válido | El sistema genera el enlace con token de un solo uso y vencimiento y responde con HTTP 201 con la URL de invitación. | POST /proyectos/{proyectoId}/invitaciones/enlace |
| 3 | Se envía un email con formato inválido o un rol distinto de Productor o Músico | El sistema responde con HTTP 400. | POST /proyectos/{proyectoId}/invitaciones |
| 4 | El email invitado ya es miembro del proyecto | El sistema responde con HTTP 409 Conflict indicando que el usuario ya es colaborador. | POST /proyectos/{proyectoId}/invitaciones |
| 5 | Existe una invitación pendiente para el mismo email y proyecto | El sistema reutiliza la invitación existente, renueva su vencimiento y reenvía el email. | POST /proyectos/{proyectoId}/invitaciones |
| 6 | Un usuario autenticado acepta una invitación válida y vigente | El sistema agrega al usuario como miembro del proyecto con el rol de la invitación, marca la invitación como aceptada (no reutilizable) y responde con HTTP 200 con el proyecto. | POST /invitaciones/{token}/aceptar |
| 7 | El token de invitación está vencido o fue revocado | El sistema responde con HTTP 410 Gone sin agregar al usuario al proyecto. | POST /invitaciones/{token}/aceptar |
| 8 | El token de invitación no existe o ya fue utilizado | El sistema responde con HTTP 404 o HTTP 409 según corresponda. | POST /invitaciones/{token}/aceptar |
| 9 | El usuario invitado no tiene cuenta en StemHub | Tras registrarse y verificar su email, el sistema permite aceptar la invitación con el mismo token. | HU-ACA-B01 |
| 10 | El usuario que invita no pertenece al proyecto | El sistema responde con HTTP 403. | POST /proyectos/{proyectoId}/invitaciones |
| 11 | El proyectoId no corresponde a un proyecto existente | El sistema responde con HTTP 404. | Todos los endpoints |
| 12 | El usuario no está autenticado | El sistema responde con HTTP 401. | Todos los endpoints |

**Fuera de alcance**

- La baja de colaboradores del proyecto no se cubre en esta HU.

**Notas técnicas**

- El token de invitación debe ser aleatorio, no predecible y almacenarse hasheado.
- El vencimiento de la invitación debe ser configurable (valor sugerido: 7 días).

**Puntos a revisar**

- [ ] Con PO: HU-NOT-F02 indica que solo el dueño del proyecto puede invitar, mientras que la Matriz de permisos habilita también a Productor y Músico. Confirmar la regla definitiva.
- [ ] Con PO: Definir una HU para la acción 'quitar colaboradores', contemplada en la Matriz de permisos.

---

### 6.2.4 Procesos / Transacciones del negocio

#### HU-COM-B01 — Obtener comentarios de una versión

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga un endpoint que devuelva todos los comentarios asociados a una versión de una canción, para que la pantalla de detalle pueda mostrarlos ordenados correctamente.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto al que pertenece la canción.

**Permisos**

Accesible para Administrador de proyecto, Productor y Músico, siempre que sean miembros del proyecto. Retorna 403 si el usuario no pertenece al proyecto.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicitan los comentarios de una versión existente | El sistema responde con HTTP 200 y un array de comentarios ordenados de más reciente a menos reciente. Cada objeto incluye: id, texto, timestampAudio (segundos), estado (pendiente / hecho), fechaCreacion, autor (id y nombre), esPropio (boolean según el usuario autenticado). | GET /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/comentarios |
| 2 | La versión no tiene comentarios | El sistema responde con HTTP 200 y un array vacío []. | GET /.../{versionId}/comentarios |
| 3 | El versionId no corresponde a ninguna versión existente | El sistema responde con HTTP 404. | GET /.../{versionId}/comentarios |
| 4 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | GET /.../{versionId}/comentarios |
| 5 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /.../{versionId}/comentarios |

**Fuera de alcance**

- La paginación de comentarios no se contempla en esta versión.
- El campo esPropio se calcula en backend comparando el autor del comentario con el userId del token; no lo envía el cliente.

**Notas técnicas**

- El campo esPropio permite al frontend saber si debe mostrar el menú meatball (•••) sin lógica de comparación en el cliente.
- Confirmar con desarrollo si timestampAudio se almacena en segundos enteros o con decimales.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar estructura exacta del objeto autor en la respuesta.
- [ ] Con PO: Confirmar si se requiere ordenamiento configurable (más reciente primero es el default de HU-19).

---

#### HU-COM-B02 — Agregar comentario a una versión

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** Codificado
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga un endpoint para crear un nuevo comentario sobre una versión de una canción, incluyendo el timestamp de audio al que hace referencia, para registrar feedback sobre un momento específico de la pista.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto. La versión con el ID indicado existe.

**Permisos**

Accesible para Administrador de proyecto, Productor y Músico, siempre que sean miembros del proyecto.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición con texto y timestampAudio válidos | El sistema persiste el comentario con estado 'pendiente' y lo asocia al usuario autenticado como autor. Responde con HTTP 201 y el objeto creado (id, texto, timestampAudio, estado, fechaCreacion, autor). | POST /.../{versionId}/comentarios |
| 2 | Se envía una petición con el campo texto vacío o ausente | El sistema responde con HTTP 400 indicando que el campo texto es requerido. | POST /.../{versionId}/comentarios |
| 3 | El texto supera los 200 caracteres | El sistema responde con HTTP 400 indicando que el texto excede la longitud máxima permitida. | POST /.../{versionId}/comentarios |
| 4 | El versionId no corresponde a ninguna versión existente | El sistema responde con HTTP 404. | POST /.../{versionId}/comentarios |
| 5 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | POST /.../{versionId}/comentarios |
| 6 | El usuario no está autenticado | El sistema responde con HTTP 401. | POST /.../{versionId}/comentarios |

**Fuera de alcance**

- El disparo de notificaciones por comentario recibido se maneja en el módulo de Notificaciones, no en este endpoint.

**Notas técnicas**

- El estado se establece en 'pendiente' automáticamente al crear; no debe ser enviado por el cliente.
- El autor se obtiene del token JWT; el cliente no envía el userId.
- El campo timestampAudio puede ser null si el comentario no está anclado a un momento específico del audio. Confirmar con PO.

**Puntos a revisar**

- [ ] Con PO: Confirmar si timestampAudio es obligatorio o puede ser null.
- [ ] Con desarrollo: Confirmar si el backend dispara la notificación de 'comentario recibido' de forma síncrona o mediante un evento asíncrono.
- [ ] El evento de notificación "Comentario recibido" se envía al "usuario que recibe el comentario", pero el comentario no tiene un destinatario (solo autor, texto, timestamp y estado). Definir el destinatario (ej. autor de la versión o menciones).

---

#### HU-COM-B03 — Modificar comentario

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** Codificado
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga un endpoint para modificar un comentario existente, ya sea para corregir su texto o para cambiar su estado entre 'pendiente' y 'hecho', respetando las reglas de permisos definidas por rol.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto. El comentario con el ID indicado existe.

**Permisos**

Edición de texto: solo el autor del comentario puede modificar el texto. Cambio de estado: accesible para todos los roles miembros del proyecto, sin distinción de autoría.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | El autor envía una petición para editar el texto con un valor válido | El sistema actualiza el texto del comentario y responde con HTTP 200 y el objeto actualizado. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 2 | Un usuario que no es el autor intenta editar el texto del comentario | El sistema responde con HTTP 403. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 3 | Se envía el campo texto vacío | El sistema responde con HTTP 400. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 4 | El texto supera los 200 caracteres | El sistema responde con HTTP 400 indicando que el texto excede la longitud máxima permitida. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 5 | Cualquier miembro del proyecto envía una petición para cambiar el estado a 'hecho' sobre un comentario en estado 'pendiente' | El sistema actualiza el estado del comentario a 'hecho' y responde con HTTP 200 y el objeto actualizado. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 6 | Cualquier miembro del proyecto envía una petición para cambiar el estado a 'pendiente' sobre un comentario en estado 'hecho' | El sistema actualiza el estado del comentario a 'pendiente' y responde con HTTP 200 y el objeto actualizado. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 7 | El comentarioId no corresponde a ningún comentario existente | El sistema responde con HTTP 404. | PATCH /.../{versionId}/comentarios/{comentarioId} |
| 8 | El usuario no está autenticado | El sistema responde con HTTP 401. | PATCH /.../{versionId}/comentarios/{comentarioId} |

**Fuera de alcance**

- La edición de timestampAudio no está contemplada; solo se permite modificar el texto o el estado.

**Notas técnicas**

- El endpoint recibe un body con los campos opcionales texto y estado. El backend aplica las reglas de permisos según el campo que se intente modificar.
- La verificación de autoría para edición de texto se realiza comparando el userId del token con el autor almacenado en el comentario.
- Confirmar con PO si el cambio de estado debe registrar quién lo realizó y cuándo.

**Puntos a revisar**

- [ ] Con PO: Confirmar si editar un comentario debe registrar historial de cambios o solo persiste el valor actual.
- [ ] Con desarrollo: Confirmar si texto y estado se modifican en el mismo endpoint PATCH o en endpoints separados.

---

#### HU-COM-B04 — Eliminar comentario

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** Codificado
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Media
- **Complejidad:** Baja

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga un endpoint para eliminar un comentario, respetando las reglas de permisos definidas por rol, para mantener la sección de comentarios ordenada.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto.

**Permisos**

Músico: solo puede eliminar sus propios comentarios. Productor y Administrador de proyecto: pueden eliminar cualquier comentario del proyecto. (Según la matriz de permisos del DOC UNIFICADO sección 14.1.2.)

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Un Músico envía una petición para eliminar su propio comentario | El sistema elimina el comentario y responde con HTTP 204. | DELETE /.../{versionId}/comentarios/{comentarioId} |
| 2 | Un Productor o Administrador de proyecto envía una petición para eliminar cualquier comentario del proyecto | El sistema elimina el comentario y responde con HTTP 204. | DELETE /.../{versionId}/comentarios/{comentarioId} |
| 3 | Un Músico intenta eliminar un comentario que no le pertenece | El sistema responde con HTTP 403. | DELETE /.../{versionId}/comentarios/{comentarioId} |
| 4 | El comentarioId no corresponde a ningún comentario existente | El sistema responde con HTTP 404. | DELETE /.../{versionId}/comentarios/{comentarioId} |
| 5 | El usuario no está autenticado | El sistema responde con HTTP 401. | DELETE /.../{versionId}/comentarios/{comentarioId} |

**Fuera de alcance**

- La eliminación es permanente (hard delete). No se contempla soft delete para comentarios en esta versión.

**Notas técnicas**

- La lógica de autorización verifica: (1) si el usuario es el autor del comentario, o (2) si el usuario tiene rol Productor o Administrador de proyecto dentro del proyecto.

**Puntos a revisar**

- [ ] Con PO: Confirmar si la eliminación de comentarios debe ser permanente o aplicar soft delete.

---

#### HU-VER-B01 — Obtener versiones y detalle de una canción

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga endpoints para obtener el listado de versiones de una canción y el detalle completo de una versión específica, incluyendo su URL de audio firmada y sus notas, para que la pantalla de detalle de canción pueda mostrar el historial y cargar la versión seleccionada.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto al que pertenece la canción.

**Permisos**

Accesible para Administrador de proyecto, Productor y Músico, siempre que sean miembros del proyecto. Retorna 403 si el usuario no pertenece al proyecto.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado de versiones de una canción existente | El sistema responde con HTTP 200 y un array de versiones. Cada objeto incluye: id, numeroVersion (ej. v1.0.0), etapa, tipo, estado, fechaCarga, autor (id y nombre), esActual (boolean). Las versiones se devuelven ordenadas por número de versión ascendente. | GET /proyectos/{proyectoId}/canciones/{cancionId}/versiones |
| 2 | La canción no tiene versiones cargadas | El sistema responde con HTTP 200 y un array vacío []. | GET /.../{cancionId}/versiones |
| 3 | Se solicita el detalle de una versión específica | El sistema responde con HTTP 200 con el objeto completo de la versión: id, numeroVersion, etapa, tipo, estado, fechaCarga, autor (id y nombre), notasVersion (texto), tags (array de strings), urlAudioFirmada (URL temporal con expiración corta), esActual (boolean). | GET /.../{cancionId}/versiones/{versionId} |
| 4 | El cancionId no corresponde a ninguna canción existente | El sistema responde con HTTP 404. | GET /.../{cancionId}/versiones |
| 5 | El versionId no corresponde a ninguna versión existente | El sistema responde con HTTP 404. | GET /.../{cancionId}/versiones/{versionId} |
| 6 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | GET /.../{cancionId}/versiones o GET /.../{versionId} |
| 7 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /.../{cancionId}/versiones o GET /.../{versionId} |

**Fuera de alcance**

- La creación de nuevas versiones corresponde a la HU del Formulario Nueva Versión (HU-ABM-03) y no se cubre aquí.
- La URL de audio nunca es la ruta física del archivo en disco. El backend genera una URL firmada con expiración antes de devolverla (ver medida preventiva de riesgo en DOC UNIFICADO sección 15.6.4).

**Notas técnicas**

- El campo esActual debe ser true únicamente para la última versión cargada de la canción.
- La urlAudioFirmada debe generarse en el momento del request con un TTL de entre 15 y 60 minutos. Una vez vencida, el cliente debe volver a solicitar el detalle para obtener una URL válida.
- El agrupamiento por etapa que muestra HU-20 puede realizarse en el frontend sobre la respuesta plana del listado. Confirmar con desarrollo.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si el agrupamiento por etapa se realiza en backend (respuesta ya agrupada) o en frontend (respuesta plana).
- [ ] Con desarrollo: Definir el TTL exacto de la URL firmada.
- [ ] Con PO: Confirmar la lista de etapas posibles (Maquetación, Composición, etc.) y si son configurables o fijas.
- [ ] Con PO: Confirmar si el campo notasVersion puede estar vacío o es obligatorio al crear una versión.
- [ ] En el documento original la ficha tenía el nombre "Obtener listado de proyectos" por error; se corrigió.
- [ ] Referencia a ID antiguo HU-20 → HU-VER-F09 (Sidebar de Versiones).

---

#### HU-IA-B01 — Procesamiento de separación de pistas

- **Módulo:** 6.2 Módulos funcionales › 6.2.4 Procesos / Transacciones del negocio
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto, Productor, Músico
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como miembro de un proyecto, quiero que el sistema exponga endpoints para solicitar la separación de una versión en pistas individuales (stems) y consultar su estado, para que la pantalla Separar Instrumentos pueda mostrar y descargar los stems resultantes.

**Precondiciones**

El usuario está autenticado y es miembro del proyecto. La versión indicada existe y tiene una pista de audio cargada. El microservicio de separación de pistas se encuentra disponible.

**Permisos**

Accesible para Administrador de proyecto, Productor y Músico, siempre que sean miembros del proyecto. Retorna 403 si el usuario no pertenece al proyecto.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita la separación de una versión que no fue procesada previamente | El sistema crea el proceso de separación con estado 'en_proceso', delega el procesamiento al microservicio de IA y responde con HTTP 202 Accepted con el identificador del proceso (id, estado). | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 2 | Se solicita la separación de una versión que ya fue procesada correctamente | El sistema no vuelve a ejecutar el procesamiento y responde con HTTP 200 con los stems existentes. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 3 | Se solicita la separación de una versión que ya tiene un proceso en curso | El sistema responde con HTTP 409 Conflict con el identificador del proceso en curso, sin iniciar uno nuevo. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 4 | Se consulta el estado de la separación de una versión | El sistema responde con HTTP 200 con el estado del proceso (pendiente / en_proceso / completado / error). | GET /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 5 | El procesamiento finaliza correctamente | El sistema almacena cada stem en el almacenamiento de objetos asociado a la versión de origen, actualiza el estado a 'completado' y la consulta de estado devuelve la lista de stems (id, nombre, tipoInstrumento, duracionSegundos, urlDescargaFirmada con expiración corta). | GET /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 6 | El microservicio de IA falla, no responde o supera el tiempo máximo de procesamiento | El sistema actualiza el estado a 'error', registra el incidente y la consulta de estado devuelve un mensaje genérico sin detalles técnicos. El usuario puede solicitar nuevamente la separación. | GET /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 7 | La versión no tiene pista de audio cargada | El sistema responde con HTTP 422 Unprocessable Entity. | POST /proyectos/{proyectoId}/canciones/{cancionId}/versiones/{versionId}/stems/separacion |
| 8 | El versionId no corresponde a ninguna versión existente | El sistema responde con HTTP 404. | POST / GET .../{versionId}/stems/separacion |
| 9 | El usuario no pertenece al proyecto | El sistema responde con HTTP 403. | POST / GET .../{versionId}/stems/separacion |
| 10 | El usuario no está autenticado | El sistema responde con HTTP 401. | POST / GET .../{versionId}/stems/separacion |

---

### 6.2.5 Perfil del usuario

#### HU-PER-B01 — Obtener y actualizar perfil de usuario

- **Módulo:** 6.2 Módulos funcionales › 6.2.5 Perfil del usuario
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Media
- **Complejidad:** Baja

**Descripción**

Como usuario autenticado, quiero que el sistema exponga endpoints para obtener y actualizar mi perfil, para que la pantalla Perfil pueda mostrar y guardar mi nombre, descripción y foto.

**Precondiciones**

El usuario está autenticado con un token de sesión válido.

**Permisos**

Accesible para todos los roles. Cada usuario solo puede consultar y modificar su propio perfil.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el perfil del usuario autenticado | El sistema responde con HTTP 200 con id, nombre, email, descripcion, avatarUrl y rol. El email es de solo lectura. | GET /usuarios/me |
| 2 | Se envía una petición con un nombre válido y, opcionalmente, descripción y avatar | El sistema elimina los espacios al inicio y al final de los textos, actualiza el perfil y responde con HTTP 200 y el objeto actualizado. | PUT /usuarios/me |
| 3 | Se envía una petición con el nombre vacío o ausente | El sistema responde con HTTP 400 indicando que el nombre es requerido. | PUT /usuarios/me |
| 4 | Se envía una descripción vacía | El sistema acepta la petición, dado que la descripción es opcional. | PUT /usuarios/me |
| 5 | Se envía un avatar con formato distinto de JPG, PNG o WEBP | El sistema responde con HTTP 415 con el mensaje: 'El avatar debe ser una imagen JPG, PNG o WEBP.' y conserva el avatar anterior. | PUT /usuarios/me |
| 6 | Se envía un avatar de más de 5 MB | El sistema responde con HTTP 413 con el mensaje: 'El avatar no puede superar los 5 MB.' y conserva el avatar anterior. | PUT /usuarios/me |
| 7 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET / PUT /usuarios/me |
| 8 | Se actualiza el perfil | El sistema registra el evento de auditoría con el valor anterior y el nuevo valor (ver HU-AYT-B1). | HU-AYT-B1 |

**Notas técnicas**

- El usuario se identifica a partir del token JWT; el cliente no envía el userId.
- La imagen de avatar se almacena en el almacenamiento de objetos y se devuelve mediante URL firmada.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar las longitudes máximas de nombre y descripción.

---

## 6.3 Módulos de información y control

### 6.3.1 Reportes e informes

#### HU-REP-B01 — Obtener listado de reportes

- **Módulo:** 6.3 Módulos de información y control › 6.3.1 Reportes e informes
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto
- **Prioridad:** Media
- **Complejidad:** Media

**Descripción**

Como usuario administrador de proyecto, quiero que el sistema exponga un endpoint, para que el frontend pueda listar los reportes predeterminados por el sistema y el usuario autenticado.

**Precondiciones**

Usuario autenticado, con permisos definidos por endpoint HU-SEG-B01

**Permisos**

Accesible para Administrador de proyecto y Administrador del sistema. Retorna HTTP 403 para los demás roles.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicitan los reportes de un proyecto | El sistema responde con HTTP 200 y un array de objetos reporte predefinidos:<br>Actividad por proyecto<br>Historial de versiones por canción<br>Participación de colaboradores<br>Estado de canciones<br>Cada objeto cuenta con: id_reporte, tipo_reporte, funcion_ejecucion_reporte | GET report/reportes |
| 2 | El usuario no tiene rol de Administrador de proyecto | El sistema responde con HTTP 403. | GET report/reportes |
| 3 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET report/reportes |

**Puntos a revisar**

- [ ] La ruta "GET report/reportes" no sigue el formato del resto de endpoints (/recurso). Confirmar la ruta definitiva.

---

#### HU-REP-B02 — Generar reporte

- **Módulo:** 6.3 Módulos de información y control › 6.3.1 Reportes e informes
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como Administrador de proyecto, quiero que el sistema exponga un endpoint para ejecutar un reporte predefinido aplicando filtros, para que el frontend pueda mostrar la vista previa de los resultados.

**Precondiciones**

El usuario está autenticado con rol de Administrador de proyecto y es miembro del proyecto sobre el que se genera el reporte. El reporte existe en el listado de HU-REP-B01.

**Permisos**

Accesible para Administrador de proyecto y Administrador del sistema. Retorna HTTP 403 si el usuario no es Administrador del proyecto indicado.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita la ejecución de un reporte existente con proyecto y filtros válidos | El sistema responde con HTTP 200 con: nombre del reporte, fechaGeneracion, filtrosAplicados, columnas (id y etiqueta) y filas con los resultados. | GET /report/reportes/{idReporte}/ejecutar?proyectoId=&fechaDesde=&fechaHasta=&usuarioIds=&estado= |
| 2 | Se ejecuta el reporte 'Actividad por proyecto' | Cada fila incluye: canción, versión actual, fecha de última modificación y usuario responsable. | GET /report/reportes/{idReporte}/ejecutar |
| 3 | Se ejecuta el reporte 'Historial de versiones por canción' | Cada fila incluye: versión, fecha, autor y etapa, ordenadas por fecha. | GET /report/reportes/{idReporte}/ejecutar |
| 4 | Se ejecuta el reporte 'Participación de colaboradores' | Cada fila incluye: usuario, cantidad de versiones subidas y cantidad de comentarios realizados dentro del proyecto. | GET /report/reportes/{idReporte}/ejecutar |
| 5 | Se ejecuta el reporte 'Estado de canciones' | Cada fila incluye: etapa de versionado y cantidad de canciones en esa etapa. | GET /report/reportes/{idReporte}/ejecutar |
| 6 | Los filtros no devuelven resultados | El sistema responde con HTTP 200 con las columnas y un array de filas vacío []. | GET /report/reportes/{idReporte}/ejecutar |
| 7 | fechaDesde es posterior a fechaHasta | El sistema responde con HTTP 400 indicando que el rango de fechas no es válido. | GET /report/reportes/{idReporte}/ejecutar |
| 8 | Falta el proyectoId o algún parámetro tiene formato inválido | El sistema responde con HTTP 400 indicando el parámetro con error. | GET /report/reportes/{idReporte}/ejecutar |
| 9 | El idReporte no corresponde a ningún reporte existente | El sistema responde con HTTP 404. | GET /report/reportes/{idReporte}/ejecutar |
| 10 | El usuario no tiene permisos sobre el proyecto indicado | El sistema responde con HTTP 403. | GET /report/reportes/{idReporte}/ejecutar |
| 11 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /report/reportes/{idReporte}/ejecutar |

**Notas técnicas**

- Los resultados se calculan exclusivamente con datos de proyectos a los que el usuario tiene acceso.
- Se recomienda paginar o limitar la cantidad máxima de filas devueltas para evitar respuestas de gran tamaño.

**Puntos a revisar**

- [ ] Con PO: Confirmar los valores posibles del filtro Estado para cada reporte.
- [ ] Con desarrollo: Confirmar si la ejecución del reporte es síncrona o requiere un proceso asíncrono para volúmenes altos.

---

#### HU-REP-B03 — Exportar reporte a PDF

- **Módulo:** 6.3 Módulos de información y control › 6.3.1 Reportes e informes
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Administrador de proyecto
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como Administrador de proyecto, quiero que el sistema exponga un endpoint para exportar un reporte generado a formato PDF, para que el frontend pueda descargarlo y compartirlo o archivarlo fuera de la plataforma.

**Precondiciones**

El usuario está autenticado con rol de Administrador de proyecto y el reporte con los filtros indicados devuelve al menos un resultado.

**Permisos**

Accesible para Administrador de proyecto y Administrador del sistema. Retorna HTTP 403 si el usuario no es Administrador del proyecto indicado.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita la exportación a PDF de un reporte con filtros válidos y resultados | El sistema responde con HTTP 200 con Content-Type application/pdf y Content-Disposition con el nombre de archivo [nombre_reporte]_[fecha_generacion].pdf. | GET /report/reportes/{idReporte}/exportar?formato=pdf&proyectoId=... |
| 2 | Se genera el PDF | El documento incluye: encabezado con el nombre del reporte y la fecha de generación, filtros aplicados, tabla con los mismos datos que la vista previa y pie de página con el nombre del proyecto. | GET /report/reportes/{idReporte}/exportar |
| 3 | Se solicita la exportación de un reporte sin resultados | El sistema responde con HTTP 422 Unprocessable Entity indicando que no hay datos para exportar. | GET /report/reportes/{idReporte}/exportar |
| 4 | Se solicita un formato distinto de pdf | El sistema responde con HTTP 400 indicando que el formato no es soportado. | GET /report/reportes/{idReporte}/exportar |
| 5 | Los parámetros de filtrado son inválidos (ej. fechaDesde posterior a fechaHasta) | El sistema responde con HTTP 400. | GET /report/reportes/{idReporte}/exportar |
| 6 | El idReporte no corresponde a ningún reporte existente | El sistema responde con HTTP 404. | GET /report/reportes/{idReporte}/exportar |
| 7 | El usuario no tiene permisos sobre el proyecto indicado o no está autenticado | El sistema responde con HTTP 403 o HTTP 401 según corresponda. | GET /report/reportes/{idReporte}/exportar |
| 8 | Ocurre un error al generar el PDF | El sistema responde con HTTP 500 con un mensaje genérico, sin exponer detalles técnicos, y registra el error en el log. | GET /report/reportes/{idReporte}/exportar |
| 9 | Se exporta un reporte | El sistema registra el evento de auditoría con el usuario, el reporte y los filtros aplicados (ver HU-AYT-B1). | HU-AYT-B1 |

**Notas técnicas**

- El PDF debe generarse con los mismos datos de HU-REP-B02, evitando inconsistencias entre la vista previa y el archivo exportado.

**Puntos a revisar**

- [ ] Con UX: Definir la plantilla visual del PDF (logo, tipografía, paginación).
- [ ] Con desarrollo: Definir la librería de generación de PDF y el límite de filas por documento.

---

### 6.3.2 Tablero / Dashboard

#### HU-DASH-B01 — Obtener indicadores numéricos del Tablero

- **Módulo:** 6.3 Módulos de información y control › 6.3.2 Tablero / Dashboard
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Media

**Descripción**

Como usuario autenticado, quiero que el sistema exponga un endpoint que devuelva los indicadores numéricos de mi actividad, para que el Tablero pueda mostrar el estado general de mis proyectos.

**Precondiciones**

El usuario está autenticado.

**Permisos**

Accesible para todos los roles autenticados. El sistema calcula los indicadores únicamente sobre los proyectos en los que el usuario participa.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicitan los indicadores sin parámetros | El sistema responde con HTTP 200 con un objeto con los indicadores globales del usuario. Cada indicador incluye: valor y variacion respecto al período anterior: totalProyectosActivos, totalCanciones, versionesUltimos30Dias, comentariosPendientes y promedioVersionesPorCancion. | GET /tablero/indicadores |
| 2 | Se solicitan los indicadores filtrando por proyectoId | El sistema calcula todos los indicadores únicamente sobre el proyecto indicado. | GET /tablero/indicadores?proyectoId={id} |
| 3 | El usuario no tiene proyectos ni datos asociados | El sistema responde con HTTP 200 con todos los valores en 0 y las variaciones en 0. | GET /tablero/indicadores |
| 4 | El proyectoId no corresponde a ningún proyecto existente | El sistema responde con HTTP 404. | GET /tablero/indicadores?proyectoId={id} |
| 5 | El usuario no pertenece al proyecto indicado | El sistema responde con HTTP 403. | GET /tablero/indicadores?proyectoId={id} |
| 6 | El proyectoId tiene un formato inválido | El sistema responde con HTTP 400. | GET /tablero/indicadores?proyectoId={id} |
| 7 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /tablero/indicadores |

**Notas técnicas**

- El promedio de versiones por canción se calcula como total de versiones sobre total de canciones, y es 0 si no hay canciones.
- Los comentarios pendientes son aquellos con estado 'pendiente' en las versiones de los proyectos del usuario.

**Puntos a revisar**

- [ ] Con PO: Definir el período de la variación (semanal u otro) y el criterio de proyecto activo.
- [ ] Con desarrollo: Evaluar el uso de consultas agregadas o vistas materializadas para optimizar el cálculo.

---

#### HU-DASH-B02 — Obtener datos de los gráficos del Tablero

- **Módulo:** 6.3 Módulos de información y control › 6.3.2 Tablero / Dashboard
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** Todos los roles
- **Prioridad:** Alta
- **Complejidad:** Alta

**Descripción**

Como usuario autenticado, quiero que el sistema exponga endpoints que devuelvan los datos de cada gráfico del Tablero, para que cada gráfico pueda cargarse y reintentarse de forma independiente.

**Precondiciones**

El usuario está autenticado.

**Permisos**

Accesible para todos los roles autenticados. El sistema calcula los datos únicamente sobre los proyectos en los que el usuario participa.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicitan las versiones subidas por proyecto | El sistema responde con HTTP 200 y un array de objetos (proyectoId, nombre, cantidadVersiones) ordenado de mayor a menor cantidad. | GET /tablero/graficos/versiones-por-proyecto |
| 2 | Se solicita la distribución de canciones por etapa de versionado | El sistema responde con HTTP 200 y un array de objetos (etapa, cantidad), incluyendo las etapas con valor 0. | GET /tablero/graficos/canciones-por-etapa |
| 3 | Se solicita la actividad temporal | El sistema responde con HTTP 200 y un array de objetos (periodo, versiones, comentarios) para los últimos 3 meses, agrupados por semana o por mes según el parámetro agrupacion (por defecto, semanal). | GET /tablero/graficos/actividad?agrupacion=semanal |
| 4 | Se envía un valor de agrupacion distinto de semanal o mensual | El sistema responde con HTTP 400. | GET /tablero/graficos/actividad |
| 5 | Se solicita cualquier gráfico filtrando por proyectoId | El sistema calcula los datos únicamente sobre el proyecto indicado. | GET /tablero/graficos/*?proyectoId={id} |
| 6 | No existen datos para el gráfico solicitado | El sistema responde con HTTP 200 y un array vacío []. | GET /tablero/graficos/* |
| 7 | El proyectoId no corresponde a ningún proyecto existente | El sistema responde con HTTP 404. | GET /tablero/graficos/*?proyectoId={id} |
| 8 | El usuario no pertenece al proyecto indicado | El sistema responde con HTTP 403. | GET /tablero/graficos/*?proyectoId={id} |
| 9 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /tablero/graficos/* |

**Notas técnicas**

- Cada gráfico se obtiene mediante un endpoint independiente para que una falla en uno no afecte al resto del Tablero.

**Puntos a revisar**

- [ ] Con PO: Confirmar si el período de actividad (3 meses) debe ser configurable por el usuario.

---

### 6.3.3 Configuración y parámetros

#### HU-CFG-B01 — Obtener listado de géneros musicales

- **Módulo:** 6.3 Módulos de información y control › 6.3.3 Configuración y parámetros
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Administrador del sistema
- **Prioridad:** Alta
- **Complejidad:** Baja

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga un endpoint que devuelva el listado completo de géneros musicales, para que la pantalla de administración pueda mostrarlo correctamente.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema. El token JWT contiene el claim de rol ADM.

**Permisos**

Solo accesible para el rol Administrador del sistema. El endpoint debe retornar 403 para cualquier otro rol.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | El Administrador solicita el listado de géneros | El sistema responde con HTTP 200 y un array de géneros. Cada objeto incluye: id, nombre, activo (boolean). | GET /configuracion/generos |
| 2 | No existen géneros configurados | El sistema responde con HTTP 200 y un array vacío []. | GET /configuracion/generos |
| 3 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /configuracion/generos |
| 4 | El usuario está autenticado con un rol distinto al Administrador del sistema | El sistema responde con HTTP 403. | GET /configuracion/generos |

**Fuera de alcance**

- El endpoint GET /configuracion/generos?activo=true utilizado por el Formulario Proyecto (HU-10) se documenta en HU-CFG-B05.
- La paginación no se contempla en esta versión dado el bajo volumen esperado del catálogo.

**Notas técnicas**

- El endpoint debe validar el token JWT y verificar el claim de rol antes de devolver los datos.
- Confirmar con desarrollo si el listado se ordena alfabéticamente por nombre o por fecha de creación.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar estructura exacta del objeto de respuesta.
- [ ] Con PO: Confirmar si se requiere ordenamiento específico del listado.
- [ ] Referencia a ID antiguo HU-10 → HU-ABM-01 (Formulario Proyecto).

---

#### HU-CFG-B02 — Crear género musical

- **Módulo:** 6.3 Módulos de información y control › 6.3.3 Configuración y parámetros
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Administrador del sistema
- **Prioridad:** Alta
- **Complejidad:** Baja

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga un endpoint para crear un nuevo género musical, para que la interfaz de administración pueda persistirlo en el catálogo.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema.

**Permisos**

Solo accesible para el rol Administrador del sistema.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición con un nombre válido y no existente en el catálogo | El sistema persiste el nuevo género con activo=true y responde con HTTP 201 y el objeto creado (id, nombre, activo). | POST /configuracion/generos |
| 2 | Se envía una petición con el campo nombre vacío o ausente | El sistema responde con HTTP 400 indicando que el campo nombre es requerido. | POST /configuracion/generos |
| 3 | Se envía un nombre que ya existe en el catálogo (sin distinguir mayúsculas/minúsculas) | El sistema responde con HTTP 409 Conflict. | POST /configuracion/generos |
| 4 | El usuario no está autenticado o el rol no es Administrador del sistema | El sistema responde con HTTP 401 o 403 según corresponda. | POST /configuracion/generos |

**Fuera de alcance**

- La validación de longitud máxima del nombre debe estar tanto en frontend (HU-CFG-F02) como en backend.

**Notas técnicas**

- La comparación de nombres duplicados debe ser case-insensitive.
- El campo activo se establece en true automáticamente al crear; no debe ser enviado por el cliente.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar longitud máxima del campo nombre en la base de datos.
- [ ] Con desarrollo: Definir si el ID se genera como UUID o entero autoincremental.

---

#### HU-CFG-B03 — Editar género musical

- **Módulo:** 6.3 Módulos de información y control › 6.3.3 Configuración y parámetros
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Administrador del sistema
- **Prioridad:** Media
- **Complejidad:** Baja

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga un endpoint para modificar el nombre de un género musical existente, para que la interfaz de administración pueda persistir los cambios.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema. El género con el ID indicado existe en el catálogo.

**Permisos**

Solo accesible para el rol Administrador del sistema.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición con un ID válido y un nombre nuevo no existente en el catálogo | El sistema actualiza el nombre del género y responde con HTTP 200 y el objeto actualizado (id, nombre, activo). | PUT /configuracion/generos/{id} |
| 2 | Se envía una petición con el campo nombre vacío o ausente | El sistema responde con HTTP 400. | PUT /configuracion/generos/{id} |
| 3 | El nuevo nombre ya pertenece a otro género del catálogo (distinto al que se edita) | El sistema responde con HTTP 409 Conflict. | PUT /configuracion/generos/{id} |
| 4 | El ID indicado no corresponde a ningún género existente | El sistema responde con HTTP 404. | PUT /configuracion/generos/{id} |
| 5 | El usuario no está autenticado o el rol no es Administrador del sistema | El sistema responde con HTTP 401 o 403 según corresponda. | PUT /configuracion/generos/{id} |

**Fuera de alcance**

- El renombrado de un género no actualiza de forma retroactiva los proyectos que lo tengan asignado. El nombre persiste asociado al ID.

**Notas técnicas**

- La validación de unicidad debe excluir el propio género que se está editando.
- La comparación de nombres duplicados debe ser case-insensitive.

**Puntos a revisar**

- [ ] Con PO: Confirmar si el renombrado debe reflejarse en la visualización de proyectos existentes o no.

---

#### HU-CFG-B04 — Desactivar / Activar género musical

- **Módulo:** 6.3 Módulos de información y control › 6.3.3 Configuración y parámetros
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Administrador del sistema
- **Prioridad:** Media
- **Complejidad:** Baja

**Descripción**

Como Administrador del sistema, quiero que el sistema exponga un endpoint para cambiar el estado activo/inactivo de un género musical, para que la interfaz pueda aplicar la baja lógica sin eliminar el registro.

**Precondiciones**

El usuario está autenticado con rol de Administrador del sistema. El género con el ID indicado existe en el catálogo.

**Permisos**

Solo accesible para el rol Administrador del sistema.

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se envía una petición para desactivar un género Activo | El sistema actualiza el campo activo a false y responde con HTTP 200 y el objeto actualizado. | PATCH /configuracion/generos/{id} |
| 2 | Se envía una petición para activar un género Inactivo | El sistema actualiza el campo activo a true y responde con HTTP 200 y el objeto actualizado. | PATCH /configuracion/generos/{id} |
| 3 | El ID indicado no corresponde a ningún género existente | El sistema responde con HTTP 404. | PATCH /configuracion/generos/{id} |
| 4 | El usuario no está autenticado o el rol no es Administrador del sistema | El sistema responde con HTTP 401 o 403 según corresponda. | PATCH /configuracion/generos/{id} |

**Fuera de alcance**

- Este endpoint no elimina el registro de la base de datos (soft delete). La eliminación física no está prevista en el alcance actual.

**Notas técnicas**

- El body de la petición incluye únicamente { activo: boolean }.
- Los géneros con activo=false deben ser excluidos del endpoint GET /configuracion/generos?activo=true utilizado por el Formulario Proyecto (HU-CFG-B05).

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si el campo activo aplica lógica de filtrado automático en otras consultas del sistema.

---

#### HU-CFG-B05 — Obtener géneros activos para selector de proyectos

- **Módulo:** 6.3 Módulos de información y control › 6.3.3 Configuración y parámetros
- **Estado (matriz de trazabilidad):** En Desarrollo
- **Actor:** Cualquier usuario autenticado
- **Prioridad:** Alta
- **Complejidad:** Baja

**Descripción**

Como usuario autenticado, quiero que el sistema exponga un endpoint que devuelva únicamente los géneros musicales activos, para que el selector del Formulario Proyecto (HU-10) y del Formulario Canción solo muestre opciones válidas.

**Precondiciones**

El usuario está autenticado con cualquier rol.

**Permisos**

Sin definir (ver Puntos a revisar).

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Se solicita el listado de géneros activos | El sistema responde con HTTP 200 y un array con solo los géneros cuyo campo activo es true. Cada objeto incluye: id y nombre. | GET /configuracion/generos?activo=true |
| 2 | No existen géneros activos configurados | El sistema responde con HTTP 200 y un array vacío []. | GET /configuracion/generos?activo=true |
| 3 | El usuario no está autenticado | El sistema responde con HTTP 401. | GET /configuracion/generos?activo=true |

**Fuera de alcance**

- Este endpoint no expone el campo activo ni las acciones de administración. Es de solo lectura y devuelve únicamente id y nombre.

**Notas técnicas**

- Este endpoint es distinto al de HU-CFG-B01: es accesible por todos los roles autenticados y solo devuelve géneros activos.
- Puede implementarse como el mismo endpoint con query param activo=true, o como ruta separada. Confirmar con desarrollo.

**Puntos a revisar**

- [ ] Con desarrollo: Confirmar si se implementa como query param o ruta separada.
- [ ] Con desarrollo: Confirmar que HU-10 (Formulario Proyecto) y HU-15 (Formulario Canción) consumen este endpoint.
- [ ] Referencia a ID antiguo HU-10 → HU-ABM-01 (Formulario Proyecto).

---

### 6.3.4 Ayuda y soporte al usuario

#### HU-AYS-B1 — Obtener el las preguntas y respuestas frecuentes

- **Módulo:** 6.3 Módulos de información y control › 6.3.4 Ayuda y soporte al usuario
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** El sistema
- **Prioridad:** Baja
- **Complejidad:** Media

**Descripción**

Como sistema, quiero que el sistema exponga un endpoint para obtener el listado de los preguntas frecuentes, para que el front end pueda listar las preguntas .

**Precondiciones**

El usuario esté autorizado y el manual esté cargado

**Permisos**

Sin definir (ver Puntos a revisar).

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Haga una petición al endpoint de preguntas frecuentes | El sistema responde con HTTP 200 y un array de Preguntas Frecuentes. Cada objeto incluye:ID, preguntaFrecuente, respuestaPreguntaFreceutnte, funcionalidadPrincipal | — |
| 2 | El sistema no tiene Preguntas frecuentes cargadas | El sistema responde con HTTP 200 y un array vacío []. | — |
| 3 | El preguntaFrecuenteid no corresponde a ninguna pregunta frecuente existente | El sistema responde con HTTP 404. | — |

**Puntos a revisar**

- [ ] La ficha no tiene Nombre ni Permisos, ni ruta de endpoint definida.
- [ ] La precondición menciona "el manual esté cargado"; para esta HU debería ser "las preguntas frecuentes estén cargadas".

---

#### HU-AYS-B2 — Obtener el manual de usuario

- **Módulo:** 6.3 Módulos de información y control › 6.3.4 Ayuda y soporte al usuario
- **Estado (matriz de trazabilidad):** No figura en la matriz de trazabilidad
- **Actor:** El sistema
- **Prioridad:** Baja
- **Complejidad:** Media

**Descripción**

Como sistema, quiero que el sistema exponga un endpoint para obtener el manual de usuario, para que el frontend pueda descargarlo en el navegador .

**Precondiciones**

El usuario esté autorizado y el manual esté cargado

**Permisos**

Sin definir (ver Puntos a revisar).

**Criterios de aceptación**

| # | Cuando | Espero | Referencias |
|---|---|---|---|
| 1 | Haga una petición al endpoint de manual de usuario | El sistema responde con HTTP 200 y el manual de usuario | — |
| 2 | El sistema no tiene Preguntas frecuentes cargadas | El sistema responde con HTTP 200 y un array vacío []. | — |
| 3 | El preguntaFrecuenteid no corresponde a ninguna pregunta frecuente existente | El sistema responde con HTTP 404. | — |

**Puntos a revisar**

- [ ] La ficha no tiene Nombre ni Permisos, ni ruta de endpoint definida.
- [ ] Los criterios 2 y 3 están copiados de HU-AYS-B1 (preguntas frecuentes). Redefinir: manual inexistente (404) y formato de entrega (PDF descargable).

---
