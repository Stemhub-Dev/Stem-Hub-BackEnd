BEGIN;

SET search_path TO public;


-- =========================================================
-- 1. CLAVE TECNICA DEL PERMISO
-- =========================================================

ALTER TABLE permiso
ADD COLUMN clavepermiso VARCHAR(100);

-- La clave técnica no puede repetirse.
CREATE UNIQUE INDEX uqclavepermiso
ON permiso (clavepermiso);


-- =========================================================
-- 2. ROLES INICIALES
-- =========================================================

-- Productor ya fue creado en 002.
UPDATE rol
SET
    descripcionrol = 'Rol funcional de productor dentro de un proyecto musical.',
    ambitorol = 'PROYECTO'
WHERE LOWER(nombrerol) = LOWER('Productor');


-- En 002 se creó como "Artista Musical".
-- Lo dejamos con el nombre definitivo.
UPDATE rol
SET
    nombrerol = 'Músico (Artista)',
    descripcionrol = 'Rol funcional de músico o artista dentro de un proyecto musical.',
    ambitorol = 'PROYECTO'
WHERE LOWER(nombrerol) = LOWER('Artista Musical');


-- Por seguridad, lo creamos si no existiera.
INSERT INTO rol (
    nombrerol,
    descripcionrol,
    ambitorol
)
SELECT
    'Músico (Artista)',
    'Rol funcional de músico o artista dentro de un proyecto musical.',
    'PROYECTO'
WHERE NOT EXISTS (
    SELECT 1
    FROM rol
    WHERE LOWER(nombrerol) = LOWER('Músico (Artista)')
);


-- Rol global del sistema.
INSERT INTO rol (
    nombrerol,
    descripcionrol,
    ambitorol
)
SELECT
    'Administrador del sistema',
    'Administrador global de StemHub con acceso a funciones de administración del sistema.',
    'SISTEMA'
WHERE NOT EXISTS (
    SELECT 1
    FROM rol
    WHERE LOWER(nombrerol) = LOWER('Administrador del sistema')
);


-- =========================================================
-- 3. PERMISOS DE SISTEMA
-- =========================================================

INSERT INTO permiso (
    clavepermiso,
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso
)
VALUES
(
    'GESTIONAR_USUARIOS',
    'Gestionar usuarios',
    'Permite administrar los usuarios del sistema.',
    'SISTEMA'
),
(
    'GESTIONAR_ROLES',
    'Gestionar roles',
    'Permite crear, modificar, dar de baja y configurar roles y sus permisos.',
    'SISTEMA'
),
(
    'GESTIONAR_PREGUNTAS_FRECUENTES',
    'Gestionar preguntas frecuentes',
    'Permite administrar las preguntas frecuentes del sistema.',
    'SISTEMA'
),
(
    'CONSULTAR_AUDITORIA',
    'Consultar auditoría',
    'Permite consultar los registros de auditoría del sistema.',
    'SISTEMA'
);


-- =========================================================
-- 4. PERMISOS DE PROYECTO
-- =========================================================

INSERT INTO permiso (
    clavepermiso,
    nombrepermiso,
    descripcionpermiso,
    ambitopermiso
)
VALUES
(
    'GESTIONAR_CANCIONES',
    'Gestionar canciones',
    'Permite realizar operaciones sobre las canciones de un proyecto.',
    'PROYECTO'
),
(
    'GESTIONAR_VERSIONES',
    'Gestionar versiones',
    'Permite administrar versiones de canciones dentro de un proyecto.',
    'PROYECTO'
),
(
    'GESTIONAR_STEMS',
    'Gestionar stems',
    'Permite administrar los stems pertenecientes a versiones de canciones.',
    'PROYECTO'
),
(
    'GESTIONAR_COMENTARIOS',
    'Gestionar comentarios',
    'Permite realizar operaciones relacionadas con los comentarios del proyecto.',
    'PROYECTO'
);


-- =========================================================
-- 5. HACER OBLIGATORIA LA CLAVE DEL PERMISO
-- =========================================================

ALTER TABLE permiso
ALTER COLUMN clavepermiso SET NOT NULL;


-- =========================================================
-- 6. ASIGNAR PERMISOS AL ADMINISTRADOR DEL SISTEMA
-- =========================================================

-- El Administrador del sistema recibe todos
-- los permisos cuyo ámbito sea SISTEMA.

INSERT INTO rolpermiso (
    codrol,
    codigopermiso,
    ambitorolpermiso
)
SELECT
    r.codrol,
    p.codigopermiso,
    'SISTEMA'
FROM rol r
CROSS JOIN permiso p
WHERE LOWER(r.nombrerol) = LOWER('Administrador del sistema')
  AND r.ambitorol = 'SISTEMA'
  AND p.ambitopermiso = 'SISTEMA'
  AND p.fechahorabajapermiso IS NULL;


COMMIT;


