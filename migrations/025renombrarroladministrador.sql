BEGIN;

-- =========================================================
-- RENOMBRAR EL ROL DE SISTEMA A "Administrador"
-- =========================================================
-- El backend reconoce al administrador por el nombre "Administrador"
-- (model.NombreRolAdministradorSistema), pero 004datosseguridadl.sql crea
-- el rol como "Administrador del sistema" y ninguna migración lo
-- renombraba: en una base nueva (y en prod) nadie era administrador, y la
-- 022 creaba sus permisos sin asignárselos a ningún rol.
--
-- Idempotente: en una base donde el rol ya se renombró a mano no cambia
-- nada, y los permisos solo se asignan si faltan.

UPDATE rol
SET nombrerol = 'Administrador'
WHERE LOWER(nombrerol) = LOWER('Administrador del sistema')
  AND ambitorol = 'SISTEMA'
  AND NOT EXISTS (
      SELECT 1
      FROM rol
      WHERE LOWER(nombrerol) = LOWER('Administrador')
  );

-- Los dos permisos de 022permisosparametrizacionproyecto.sql, que no se
-- asignaron donde el rol todavía tenía el nombre viejo.
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
    ON p.clavepermiso IN ('GESTIONAR_ESTADOS_PROYECTO', 'GESTIONAR_TIPOS_PROYECTO')
   AND p.ambitopermiso = 'SISTEMA'
   AND p.fechahorabajapermiso IS NULL
WHERE LOWER(r.nombrerol) = LOWER('Administrador')
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
