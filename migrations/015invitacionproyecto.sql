BEGIN;

SET search_path TO public;

-- =========================================================
-- INVITACION A PROYECTO
-- =========================================================

CREATE TABLE invitacionproyecto (
    codigoinvitacionproy BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigoproyecto BIGINT NOT NULL,
    emailinvitado VARCHAR(254) NOT NULL,
    codrol BIGINT NOT NULL,
    ambitorol VARCHAR(20) NOT NULL DEFAULT 'PROYECTO',
    codintegranteinvito BIGINT NOT NULL,
    tokeninvitacion TEXT NOT NULL,
    fechahoraaltainvitacionproy TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahoraexpiracioninvitacion TIMESTAMPTZ NOT NULL,
    fechahoraaceptacioninvitacion TIMESTAMPTZ,
    fechahorabajainvitacionproy TIMESTAMPTZ,

    CONSTRAINT fkinvitacionproyectoproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto (codigoproyecto)
        ON DELETE RESTRICT,

    -- Esta columna es intencionalmente fija (mismo mecanismo que
    -- integranteproyecto): nos permite impedir asignar un rol de ámbito
    -- SISTEMA a una invitación de proyecto mediante una FK de PostgreSQL.
    CONSTRAINT chkinvitacionproyectoambito
        CHECK (ambitorol = 'PROYECTO'),

    CONSTRAINT fkinvitacionproyectorol
        FOREIGN KEY (codrol, ambitorol)
        REFERENCES rol (codrol, ambitorol)
        ON DELETE RESTRICT,

    CONSTRAINT fkinvitacionproyectointegrante
        FOREIGN KEY (codintegranteinvito)
        REFERENCES integrante (codintegrante)
        ON DELETE RESTRICT,

    CONSTRAINT uqinvitacionproyectotoken
        UNIQUE (tokeninvitacion)
);

-- Un email solo puede tener una invitación PENDIENTE activa por proyecto.
-- "Pendiente" = ni aceptada ni dada de baja (cancelada o reemplazada por
-- una reinvitación). Al reinvitar, el repository hace UPDATE sobre la fila
-- existente (nuevo token, nueva expiración) en vez de INSERT, así que este
-- índice nunca compite consigo mismo, incluso si la invitación existente
-- ya venció.
CREATE UNIQUE INDEX uqinvitacionproyectopendiente
ON invitacionproyecto (codigoproyecto, LOWER(emailinvitado))
WHERE fechahoraaceptacioninvitacion IS NULL
  AND fechahorabajainvitacionproy IS NULL;

-- Listado de invitaciones pendientes por proyecto (el owner las ve/cancela).
CREATE INDEX ixinvitacionproyectoproyecto
ON invitacionproyecto (codigoproyecto)
WHERE fechahoraaceptacioninvitacion IS NULL
  AND fechahorabajainvitacionproy IS NULL;

COMMIT;
