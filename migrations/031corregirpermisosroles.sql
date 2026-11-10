BEGIN;

-- =========================================================
-- CORRECCIÓN DE ASIGNACIONES DE 029mejoraspermisos.sql
-- =========================================================
-- 029 tenía dos huecos:
--   1. Buscaba el rol 'Artista', pero desde 004datosseguridadl.sql se llama
--      'Músico (Artista)'. El músico no recibía los permisos granulares y
--      además perdía los GESTIONAR_* dados de baja: quedaba sin permisos.
--   2. Creaba EDITAR_VERSION y ELIMINAR_VERSION sin asignarlos a ningún rol,
--      así que nadie podía editar ni dar de baja versiones.
-- Idempotente: solo inserta asignaciones activas que falten.

-- =========================================================
-- 1. PERMISOS COMUNES DEL MÚSICO (mismos que 029 daba a 'Artista')
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
WHERE LOWER(r.nombrerol) = LOWER('Músico (Artista)')
  AND r.ambitorol = 'PROYECTO'
  AND r.fechahorabajarol IS NULL
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
-- 2. EDITAR/ELIMINAR VERSIÓN PARA EL PRODUCTOR
-- =========================================================
-- Igual que CREAR_VERSION, exclusivos del productor.

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
      'EDITAR_VERSION',
      'ELIMINAR_VERSION'
  )
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.fechahorabajarolpermiso IS NULL
  );

COMMIT;
