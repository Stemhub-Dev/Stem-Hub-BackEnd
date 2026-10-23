BEGIN;

-- =========================================================
-- 1. PERMISOS GRANULARES DE PROYECTO
-- =========================================================

INSERT INTO permiso (
    clavepermiso,
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso
)
VALUES

-- PROYECTO
(
    'VER_PROYECTO',
    'Ver proyecto',
    'Permite consultar la información de un proyecto.',
    'PROYECTO'
),
(
    'EDITAR_PROYECTO',
    'Editar proyecto',
    'Permite editar un proyecto cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'ELIMINAR_PROYECTO',
    'Eliminar proyecto',
    'Permite dar de baja un proyecto cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),

-- INTEGRANTES
(
    'VER_INTEGRANTES',
    'Ver integrantes',
    'Permite consultar los integrantes de un proyecto.',
    'PROYECTO'
),
(
    'CAMBIAR_ROL_INTEGRANTE',
    'Cambiar rol de integrante',
    'Permite cambiar el rol de un integrante cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'QUITAR_INTEGRANTE',
    'Quitar integrante',
    'Permite quitar integrantes cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),

-- INVITACIONES
(
    'INVITAR_INTEGRANTE',
    'Invitar integrante',
    'Permite invitar integrantes cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'VER_INVITACIONES_PROYECTO',
    'Ver invitaciones del proyecto',
    'Permite consultar invitaciones pendientes cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'CANCELAR_INVITACION',
    'Cancelar invitación',
    'Permite cancelar invitaciones cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),

-- CANCIONES
(
    'VER_CANCION',
    'Ver canción',
    'Permite consultar canciones de un proyecto.',
    'PROYECTO'
),
(
    'CREAR_CANCION',
    'Crear canción',
    'Permite crear canciones dentro de un proyecto.',
    'PROYECTO'
),
(
    'EDITAR_CANCION',
    'Editar canción',
    'Permite editar canciones dentro de un proyecto.',
    'PROYECTO'
),
(
    'ELIMINAR_CANCION',
    'Eliminar canción',
    'Permite dar de baja canciones dentro de un proyecto.',
    'PROYECTO'
),

-- VERSIONES
(
    'VER_VERSION',
    'Ver versión',
    'Permite consultar versiones y reproducir su audio.',
    'PROYECTO'
),
(
    'CREAR_VERSION',
    'Crear versión',
    'Permite cargar una nueva versión de una canción.',
    'PROYECTO'
),

-- STEMS
(
    'VER_STEM',
    'Ver stem',
    'Permite consultar y reproducir stems dentro de una versión.',
    'PROYECTO'
),
(
    'VER_STEMS_GLOBAL',
    'Ver listado global de stems',
    'Permite acceder al listado global de stems de los proyectos habilitados.',
    'PROYECTO'
),
(
    'CREAR_STEM',
    'Crear stem',
    'Permite cargar stems dentro de una versión.',
    'PROYECTO'
),
(
    'EDITAR_STEM',
    'Editar stem',
    'Permite modificar stems.',
    'PROYECTO'
),
(
    'ELIMINAR_STEM',
    'Eliminar stem',
    'Permite eliminar stems.',
    'PROYECTO'
),
(
    'SEPARAR_STEMS_IA',
    'Separar stems con IA',
    'Permite solicitar la separación de pistas mediante inteligencia artificial.',
    'PROYECTO'
),

-- COMENTARIOS
(
    'VER_COMENTARIOS',
    'Ver comentarios',
    'Permite consultar comentarios de una versión.',
    'PROYECTO'
),
(
    'CREAR_COMENTARIO',
    'Crear comentario',
    'Permite agregar comentarios a una versión.',
    'PROYECTO'
),
(
    'RESPONDER_COMENTARIO',
    'Responder comentario',
    'Permite responder comentarios.',
    'PROYECTO'
),
(
    'EDITAR_COMENTARIO_PROPIO',
    'Editar comentario propio',
    'Permite modificar comentarios propios.',
    'PROYECTO'
),
(
    'ELIMINAR_COMENTARIO_PROPIO',
    'Eliminar comentario propio',
    'Permite eliminar comentarios o respuestas propias.',
    'PROYECTO'
),
(
    'ELIMINAR_CUALQUIER_COMENTARIO',
    'Eliminar cualquier comentario',
    'Permite eliminar comentarios o respuestas de otros integrantes.',
    'PROYECTO'
),
(
    'CAMBIAR_ESTADO_COMENTARIO',
    'Cambiar estado de comentario',
    'Permite cambiar el estado pendiente o resuelto de un comentario.',
    'PROYECTO'
),
(
    'RESUMIR_COMENTARIOS_IA',
    'Resumir comentarios con IA',
    'Permite generar un resumen de comentarios mediante inteligencia artificial.',
    'PROYECTO'
),

-- REPORTES
(
    'VER_REPORTES',
    'Ver reportes',
    'Permite consultar los reportes disponibles cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'GENERAR_REPORTE',
    'Generar reporte',
    'Permite generar reportes cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
),
(
    'EXPORTAR_REPORTE',
    'Exportar reporte',
    'Permite exportar reportes cuando además se cumplen las reglas de propiedad.',
    'PROYECTO'
)

ON CONFLICT DO NOTHING;


-- =========================================================
-- 2. PERMISOS COMUNES A PRODUCTOR Y MÚSICO (ARTISTA)
-- =========================================================

INSERT INTO rolpermiso (
    codrol,
    codigopermiso,
    ambitorolpermiso,
    fechahoraaltarolpermiso
)
SELECT
    r.codrol,
    p.codigopermiso,
    'PROYECTO',
    CURRENT_TIMESTAMP
FROM rol r
CROSS JOIN permiso p
WHERE r.ambitorol = 'PROYECTO'
  AND r.fechahorabajarol IS NULL
  AND LOWER(r.nombrerol) IN (
      LOWER('Productor'),
      LOWER('Artista')
  )
  AND p.ambitopermiso = 'PROYECTO'
  AND p.fechahorabajapermiso IS NULL
  AND p.clavepermiso IN (
      'VER_PROYECTO',
      'EDITAR_PROYECTO',
      'ELIMINAR_PROYECTO',

      'VER_INTEGRANTES',
      'CAMBIAR_ROL_INTEGRANTE',
      'QUITAR_INTEGRANTE',

      'INVITAR_INTEGRANTE',
      'VER_INVITACIONES_PROYECTO',
      'CANCELAR_INVITACION',

      'VER_CANCION',
      'VER_VERSION',

      'VER_STEM',
      'SEPARAR_STEMS_IA',

      'VER_COMENTARIOS',
      'CREAR_COMENTARIO',
      'RESPONDER_COMENTARIO',
      'EDITAR_COMENTARIO_PROPIO',
      'ELIMINAR_COMENTARIO_PROPIO',
      'CAMBIAR_ESTADO_COMENTARIO',
      'RESUMIR_COMENTARIOS_IA',

      'VER_REPORTES',
      'GENERAR_REPORTE',
      'EXPORTAR_REPORTE'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.fechahorabajarolpermiso IS NULL
  );


-- =========================================================
-- 3. PERMISOS EXCLUSIVOS DEL PRODUCTOR
-- =========================================================

INSERT INTO rolpermiso (
    codrol,
    codigopermiso,
    ambitorolpermiso,
    fechahoraaltarolpermiso
)
SELECT
    r.codrol,
    p.codigopermiso,
    'PROYECTO',
    CURRENT_TIMESTAMP
FROM rol r
CROSS JOIN permiso p
WHERE LOWER(r.nombrerol) = LOWER('Productor')
  AND r.ambitorol = 'PROYECTO'
  AND r.fechahorabajarol IS NULL
  AND p.ambitopermiso = 'PROYECTO'
  AND p.fechahorabajapermiso IS NULL
  AND p.clavepermiso IN (
      'CREAR_CANCION',
      'EDITAR_CANCION',
      'ELIMINAR_CANCION',

      'CREAR_VERSION',

      'VER_STEMS_GLOBAL',
      'CREAR_STEM',
      'EDITAR_STEM',
      'ELIMINAR_STEM',

      'ELIMINAR_CUALQUIER_COMENTARIO'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.fechahorabajarolpermiso IS NULL
  );


-- =========================================================
-- 4. RETIRAR ASIGNACIONES DE LOS PERMISOS GENERALES ANTERIORES
-- =========================================================

UPDATE rolpermiso rp
SET fechahorabajarolpermiso = CURRENT_TIMESTAMP
FROM permiso p
WHERE p.codigopermiso = rp.codigopermiso
  AND rp.fechahorabajarolpermiso IS NULL
  AND p.clavepermiso IN (
      'GESTIONAR_CANCIONES',
      'GESTIONAR_VERSIONES',
      'GESTIONAR_STEMS',
      'GESTIONAR_COMENTARIOS'
  );


-- =========================================================
-- 6. PERMISOS VERSIÓN
-- =========================================================

INSERT INTO permiso (
    clavepermiso,
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso
)
SELECT
    nuevo.clave,
    nuevo.nombre,
    nuevo.descripcion,
    'PROYECTO'
FROM (
    VALUES
        (
            'EDITAR_VERSION',
            'Editar versiones',
            'Permite editar las notas de las versiones de canciones del proyecto.'
        ),
        (
            'ELIMINAR_VERSION',
            'Eliminar versiones',
            'Permite dar de baja versiones de canciones del proyecto.'
        )
) AS nuevo(clave, nombre, descripcion)
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso p
    WHERE p.clavepermiso = nuevo.clave
);


-- =========================================================
-- 6. DAR DE BAJA LOS PERMISOS GENERALES ANTERIORES
-- =========================================================

UPDATE permiso
SET fechahorabajapermiso = CURRENT_TIMESTAMP
WHERE fechahorabajapermiso IS NULL
  AND clavepermiso IN (
      'GESTIONAR_CANCIONES',
      'GESTIONAR_VERSIONES',
      'GESTIONAR_STEMS',
      'GESTIONAR_COMENTARIOS'
  );

COMMIT;