BEGIN;

-- Crear el permiso para administrar géneros musicales.
INSERT INTO permiso (
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso,
    clavepermiso
)
SELECT
    'Gestionar géneros musicales',
    'Permite crear, modificar y eliminar géneros musicales del sistema',
    'SISTEMA',
    'GESTIONAR_GENEROS'
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso
    WHERE clavepermiso = 'GESTIONAR_GENEROS'
);

-- Asignar el permiso al rol Administrador.
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
    ON p.clavepermiso = 'GESTIONAR_GENEROS'
WHERE r.nombrerol = 'Administrador'
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


BEGIN;

INSERT INTO permiso (
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso,
    clavepermiso
)
SELECT
    'Consultar géneros musicales',
    'Permite consultar géneros musicales del sistema',
    'SISTEMA',
    'CONSULTAR_GENEROS'
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso
    WHERE clavepermiso = 'CONSULTAR_GENEROS'
);

COMMIT;