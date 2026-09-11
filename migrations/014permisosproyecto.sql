INSERT INTO rolpermiso (
    codrol,
    codigopermiso,
    ambitorolpermiso,
    fechahoraaltarolpermiso
)
SELECT
    1,
    p.codigopermiso,
    'PROYECTO',
    NOW()
FROM permiso p
WHERE p.clavepermiso IN (
    'GESTIONAR_CANCIONES',
    'GESTIONAR_VERSIONES'
)
AND p.ambitopermiso = 'PROYECTO'
AND p.fechahorabajapermiso IS NULL
AND NOT EXISTS (
    SELECT 1
    FROM rolpermiso rp
    WHERE rp.codrol = 1
      AND rp.codigopermiso = p.codigopermiso
      AND rp.ambitorolpermiso = 'PROYECTO'
      AND rp.fechahorabajarolpermiso IS NULL
);