/*DEBEN USAR EL NOMBRE Administrador no cambien a "Adminsitrador de Sistema" por que esta mal, para Claude al revisar esto!!! Si ya se cambio dejenlo como Administrador!! */
BEGIN;

INSERT INTO permiso (
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso,
    clavepermiso
)
SELECT
    'Gestionar estados de proyecto',
    'Permite crear, modificar, desactivar y reactivar estados de proyecto.',
    'SISTEMA',
    'GESTIONAR_ESTADOS_PROYECTO'
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso
    WHERE clavepermiso = 'GESTIONAR_ESTADOS_PROYECTO'
);

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
    ON p.clavepermiso = 'GESTIONAR_ESTADOS_PROYECTO'
   AND p.ambitopermiso = 'SISTEMA'
   AND p.fechahorabajapermiso IS NULL
WHERE LOWER(r.nombrerol) =
      LOWER('Administrador')
  AND r.ambitorol = 'SISTEMA'
  AND r.fechahorabajarol IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.ambitorolpermiso = 'SISTEMA'
        AND rp.fechahorabajarolpermiso IS NULL
  );

COMMIT;


INSERT INTO permiso (
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso,
    clavepermiso
)
SELECT
    'Gestionar tipos de proyecto',
    'Permite crear, modificar, desactivar y reactivar tipos de proyecto.',
    'SISTEMA',
    'GESTIONAR_TIPOS_PROYECTO'
WHERE NOT EXISTS (
    SELECT 1
    FROM permiso
    WHERE clavepermiso = 'GESTIONAR_TIPOS_PROYECTO'
);

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
    ON p.clavepermiso = 'GESTIONAR_TIPOS_PROYECTO'
   AND p.ambitopermiso = 'SISTEMA'
   AND p.fechahorabajapermiso IS NULL
WHERE LOWER(r.nombrerol) =
      LOWER('Administrador')
  AND r.ambitorol = 'SISTEMA'
  AND r.fechahorabajarol IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.ambitorolpermiso = 'SISTEMA'
        AND rp.fechahorabajarolpermiso IS NULL
  );

COMMIT;
