BEGIN;

-- 007permisosgeneros.sql asignó GESTIONAR_GENEROS buscando el rol por
-- r.nombrerol = 'Administrador', pero el rol real se llama 'Administrador
-- del sistema' (004datosseguridadl.sql), así que esa asignación nunca se
-- aplicó. Se repara acá con el nombre correcto; es idempotente por el
-- mismo NOT EXISTS que ya usan las demás migraciones de rolpermiso.
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
   AND p.ambitopermiso = 'SISTEMA'
   AND p.fechahorabajapermiso IS NULL
WHERE LOWER(r.nombrerol) = LOWER('Administrador del sistema')
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
