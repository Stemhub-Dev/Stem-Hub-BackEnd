BEGIN;

SET search_path TO public;

-- =========================================================
-- 1. AMBITO DEL ROL
-- =========================================================

ALTER TABLE rol
ADD COLUMN ambitorol VARCHAR(20);

-- Los roles que ya existen actualmente
-- (Productor y Artista Musical) son roles de proyecto.
UPDATE rol
SET ambitorol = 'PROYECTO'
WHERE ambitorol IS NULL;

ALTER TABLE rol
ALTER COLUMN ambitorol SET NOT NULL;

ALTER TABLE rol
ADD CONSTRAINT chkambitorol
CHECK (ambitorol IN ('SISTEMA', 'PROYECTO'));

-- Necesario para poder utilizar una FK compuesta
-- que valide codigo + ambito.
ALTER TABLE rol
ADD CONSTRAINT uqrolcodigoambito
UNIQUE (codrol, ambitorol);

-- El nombre del rol no puede repetirse aunque cambien mayúsculas.
CREATE UNIQUE INDEX uqnombrerol
ON rol (LOWER(nombrerol));


-- =========================================================
-- 2. PROPIEDAD DEL PROYECTO
-- =========================================================

ALTER TABLE integranteproyecto
ADD COLUMN espropietario BOOLEAN NOT NULL DEFAULT FALSE;

-- Solo puede existir un propietario ACTIVO por proyecto.
CREATE UNIQUE INDEX uqpropietarioproyectoactivo
ON integranteproyecto (codigoproyecto)
WHERE espropietario = TRUE
  AND fechahorabajaintegranteproy IS NULL;


-- =========================================================
-- 3. IMPEDIR ROLES DE SISTEMA DENTRO DE UN PROYECTO
-- =========================================================

-- Esta columna es intencionalmente fija.
-- Nos permite hacer cumplir la regla directamente
-- mediante una FK de PostgreSQL.
ALTER TABLE integranteproyecto
ADD COLUMN ambitorol VARCHAR(20) NOT NULL DEFAULT 'PROYECTO';

ALTER TABLE integranteproyecto
ADD CONSTRAINT chkintegranteproyectoambito
CHECK (ambitorol = 'PROYECTO');

-- Eliminamos la FK simple creada en la migración 001.
ALTER TABLE integranteproyecto
DROP CONSTRAINT IF EXISTS fkintegranteproyectorol;

-- La reemplazamos por una FK que controla rol + ámbito.
ALTER TABLE integranteproyecto
ADD CONSTRAINT fkintegranteproyectorol
FOREIGN KEY (codrol, ambitorol)
REFERENCES rol (codrol, ambitorol)
ON DELETE RESTRICT;


-- =========================================================
-- 4. PERMISO
-- =========================================================

CREATE TABLE permiso (
    codigopermiso BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombrepermiso VARCHAR(150) NOT NULL,
    descripcionpermiso TEXT,
    ambitopermiso VARCHAR(20) NOT NULL,
    fechahorabajapermiso TIMESTAMPTZ,

    CONSTRAINT chkambitopermiso
        CHECK (ambitopermiso IN ('SISTEMA', 'PROYECTO')),

    CONSTRAINT uqpermisocodigoambito
        UNIQUE (codigopermiso, ambitopermiso)
);

-- Evita "Crear Proyecto" y "crear proyecto" como permisos diferentes.
CREATE UNIQUE INDEX uqnombrepermiso
ON permiso (LOWER(nombrepermiso));


-- =========================================================
-- 5. ROL - PERMISO
-- =========================================================

CREATE TABLE rolpermiso (
    codigorolpermiso BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codrol BIGINT NOT NULL,
    codigopermiso BIGINT NOT NULL,
    ambitorolpermiso VARCHAR(20) NOT NULL,
    fechahoraaltarolpermiso TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajarolpermiso TIMESTAMPTZ,

    CONSTRAINT chkambitorolpermiso
        CHECK (ambitorolpermiso IN ('SISTEMA', 'PROYECTO')),

    -- El rol debe pertenecer al mismo ámbito.
    CONSTRAINT fkrolpermisorol
        FOREIGN KEY (codrol, ambitorolpermiso)
        REFERENCES rol (codrol, ambitorol)
        ON DELETE RESTRICT,

    -- El permiso también debe pertenecer al mismo ámbito.
    CONSTRAINT fkrolpermisopermiso
        FOREIGN KEY (codigopermiso, ambitorolpermiso)
        REFERENCES permiso (codigopermiso, ambitopermiso)
        ON DELETE RESTRICT
);

-- Un mismo permiso no puede estar asignado dos veces
-- de forma activa al mismo rol.
CREATE UNIQUE INDEX uqrolpermisoactivo
ON rolpermiso (codrol, codigopermiso)
WHERE fechahorabajarolpermiso IS NULL;


-- =========================================================
-- 6. USUARIO - ROL GLOBAL DEL SISTEMA
-- =========================================================

CREATE TABLE usuariorol (
    codigousuariorol BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigousuario BIGINT NOT NULL,
    codrol BIGINT NOT NULL,

    -- En esta tabla solamente pueden existir roles SISTEMA.
    ambitorol VARCHAR(20) NOT NULL DEFAULT 'SISTEMA',

    fechahoraaltausuariorol TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajausuariorol TIMESTAMPTZ,

    CONSTRAINT chkusuariorolambito
        CHECK (ambitorol = 'SISTEMA'),

    CONSTRAINT fkusuariorolusuario
        FOREIGN KEY (codigousuario)
        REFERENCES usuario (codigousuario)
        ON DELETE RESTRICT,

    CONSTRAINT fkusuariorolrol
        FOREIGN KEY (codrol, ambitorol)
        REFERENCES rol (codrol, ambitorol)
        ON DELETE RESTRICT
);

-- Por ahora un usuario puede tener como máximo
-- un rol global activo.
CREATE UNIQUE INDEX uqusuariorolactivo
ON usuariorol (codigousuario)
WHERE fechahorabajausuariorol IS NULL;

COMMIT;