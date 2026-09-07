BEGIN;

-- =========================================================
-- RENOMBRAR TIPOS DE PROYECTO EXISTENTES
-- =========================================================

UPDATE tipoproyecto
SET
    nombretipoproy = 'Album (LP)',
    descripciontipoproy = 'Proyecto musical correspondiente a un álbum de larga duración.'
WHERE LOWER(nombretipoproy) = LOWER('Album');

UPDATE tipoproyecto
SET
    nombretipoproy = 'Extended Play (EP)',
    descripciontipoproy = 'Proyecto musical de extensión intermedia compuesto por varias canciones.'
WHERE LOWER(nombretipoproy) = LOWER('EP');

COMMIT;


BEGIN;

-- =========================================================
-- PERMISO PARA CONSULTAR TIPOS DE PROYECTO
-- =========================================================

INSERT INTO permiso (
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso,
    clavepermiso
)
SELECT
    'Consultar tipos de proyecto',
    'Permite consultar los tipos de proyecto del sistema',
    'SISTEMA',
    'CONSULTAR_TIPOS_PROYECTO'
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso
    WHERE clavepermiso = 'CONSULTAR_TIPOS_PROYECTO'
);

-- Asignar el permiso al rol Administrador del sistema.
INSERT INTO rolpermiso (
    codrol,
    codigopermiso,
    ambitorolpermiso,
    fechahoraaltarolpermiso
)
SELECT
    r.codrol,
    p.codigopermiso,
    'SISTEMA',
    CURRENT_TIMESTAMP
FROM rol r
INNER JOIN permiso p
    ON p.clavepermiso = 'CONSULTAR_TIPOS_PROYECTO'
WHERE LOWER(r.nombrerol) = LOWER('Administrador del sistema')
  AND r.ambitorol = 'SISTEMA'
  AND r.fechahorabajarol IS NULL
  AND p.fechahorabajapermiso IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.fechahorabajarolpermiso IS NULL
  );

COMMIT;
