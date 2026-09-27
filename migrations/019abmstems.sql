BEGIN;

-- =========================================================
-- ABM DE STEMS (HU-ABM-04-01/02/03)
-- =========================================================
-- Los stems pasan a cargarse de a uno desde el Detalle Canción (o con
-- "Separar Pistas"), no al crear la versión, y siempre pertenecen a una
-- categoría. Mismo criterio de archivo único que cancionversion (010).

-- ---------------------------------------------------------
-- CATEGORIA STEM
-- ---------------------------------------------------------
-- codigoproyecto NULL = categoría por defecto, disponible en todos los
-- proyectos. Las que crea el usuario pertenecen a un proyecto.

CREATE TABLE categoriastem (
    codcategoriastem BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombrecategoriastem VARCHAR(100) NOT NULL,
    codigoproyecto BIGINT,
    fechahoraaltacategoriastem TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajacategoriastem TIMESTAMPTZ,

    CONSTRAINT fkcategoriastemproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto(codigoproyecto)
        ON DELETE RESTRICT,

    CONSTRAINT chknombrecategoriastem
        CHECK (btrim(nombrecategoriastem) <> '')
);

-- Una categoría de proyecto no puede repetir el nombre (sin distinguir
-- mayúsculas) de otra del mismo proyecto. Que no choque con una por
-- defecto lo valida el service, un índice no puede cruzar ambos casos.
CREATE UNIQUE INDEX uqcategoriastemnombre
    ON categoriastem (COALESCE(codigoproyecto, 0), lower(btrim(nombrecategoriastem)))
    WHERE fechahorabajacategoriastem IS NULL;

INSERT INTO categoriastem (nombrecategoriastem, codigoproyecto)
VALUES
    ('Batería', NULL),
    ('Bajo', NULL),
    ('Guitarra', NULL),
    ('Voz', NULL);

-- ---------------------------------------------------------
-- STEM
-- ---------------------------------------------------------

-- Los stems cargados con la versión (antes de este cambio) no tienen
-- categoría: se dan de baja para que no aparezcan en la nueva sección.
UPDATE stem
SET fechahorabajastem = CURRENT_TIMESTAMP
WHERE fechahorabajastem IS NULL;

ALTER TABLE stem
    ADD COLUMN codcategoriastem BIGINT,
    ADD COLUMN urlarchivostem TEXT,
    ADD COLUMN formatoarchivostem VARCHAR(10),
    ADD COLUMN nombrearchivostem VARCHAR(255),
    ADD COLUMN generadoconia BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE stem
    ADD CONSTRAINT fkstemcategoriastem
        FOREIGN KEY (codcategoriastem)
        REFERENCES categoriastem(codcategoriastem)
        ON DELETE RESTRICT,

    ADD CONSTRAINT chkformatoarchivostem
        CHECK (formatoarchivostem IN ('mp3', 'wav', 'flac')),

    -- Todo stem activo tiene categoría y archivo. Los dados de baja arriba
    -- quedan exceptuados.
    ADD CONSTRAINT chkstemactivocompleto
        CHECK (
            fechahorabajastem IS NOT NULL
            OR (
                codcategoriastem IS NOT NULL
                AND urlarchivostem IS NOT NULL
                AND formatoarchivostem IS NOT NULL
                AND nombrearchivostem IS NOT NULL
            )
        );

CREATE UNIQUE INDEX uqstemnombreversion
    ON stem (codigocancionversion, lower(btrim(nombrestem)))
    WHERE fechahorabajastem IS NULL;

-- ---------------------------------------------------------
-- PERMISO: el Productor gestiona los stems
-- ---------------------------------------------------------

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
    ON p.clavepermiso = 'GESTIONAR_STEMS'
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
