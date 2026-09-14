BEGIN;

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
INNER JOIN permiso p
    ON p.clavepermiso = 'GESTIONAR_COMENTARIOS'
   AND p.ambitopermiso = 'PROYECTO'
   AND p.fechahorabajapermiso IS NULL
WHERE r.nombrerol = 'Productor'
  AND r.ambitorol = 'PROYECTO'
  AND r.fechahorabajarol IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM rolpermiso rp
      WHERE rp.codrol = r.codrol
        AND rp.codigopermiso = p.codigopermiso
        AND rp.ambitorolpermiso = 'PROYECTO'
        AND rp.fechahorabajarolpermiso IS NULL
  );

COMMIT;