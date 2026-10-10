BEGIN;

-- =========================================================
-- ETAPA DE VERSIONADO (HU-DASH-B02)
-- =========================================================
-- El Tablero muestra la distribución de canciones por etapa de versionado,
-- pero el esquema no la modelaba. Se agrega como catálogo (mismo criterio
-- que estadoproyecto/tipoproyecto) y como referencia opcional desde la
-- versión: la etapa de una canción es la de su versión actual.
--
-- La lista definitiva de etapas está pendiente con PO (el módulo de
-- Versionado habla de Idea, Demo, Borrador, Mezcla y Master; las pantallas
-- y la HU del Tablero muestran Maquetación, Composición y Mezcla). Se carga
-- la de las pantallas; cambiarla es actualizar el catálogo, no el esquema.
--
-- codetapaversion es NULL en las versiones existentes y en las que se creen
-- hasta que el alta de versión pida la etapa: el Tablero las cuenta como
-- "Sin etapa".

CREATE TABLE etapaversionado (
    codetapaversion BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombreetapaversion VARCHAR(100) NOT NULL UNIQUE,
    descripcionetapaversion TEXT,
    ordenetapaversion INTEGER NOT NULL,
    fechahorabajaetapaversion TIMESTAMPTZ,

    CONSTRAINT chkordenetapaversion
        CHECK (ordenetapaversion > 0)
);

INSERT INTO etapaversionado (
    nombreetapaversion,
    descripcionetapaversion,
    ordenetapaversion
)
VALUES
    ('Maquetación', 'Primera maqueta o demo de la canción.', 1),
    ('Composición', 'La canción se está componiendo y arreglando.', 2),
    ('Mezcla', 'La canción está en mezcla.', 3)
ON CONFLICT (nombreetapaversion) DO NOTHING;

ALTER TABLE cancionversion
    ADD COLUMN codetapaversion BIGINT;

ALTER TABLE cancionversion
    ADD CONSTRAINT fkcancionversionetapa
        FOREIGN KEY (codetapaversion)
        REFERENCES etapaversionado(codetapaversion)
        ON DELETE RESTRICT;

COMMIT;
