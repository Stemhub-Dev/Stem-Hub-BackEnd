BEGIN;

-- =========================================================
-- FECHA Y AUTOR DEL ALTA DE UN STEM (Inicio: "Mientras no estabas")
-- =========================================================
-- El feed de novedades del Inicio y el indicador de stems necesitan saber
-- cuándo y quién subió cada stem, y el esquema no lo guardaba (igual que
-- pasaba con las versiones antes de 024versionautor.sql).
--
-- Stems existentes: los generados con IA toman la fecha de fin y el
-- solicitante de la última separación completada de su versión; el resto,
-- la fecha de su versión y sin autor (no hay de dónde reconstruirlo).

ALTER TABLE stem
    ADD COLUMN IF NOT EXISTS fechahoraaltastem TIMESTAMPTZ;

ALTER TABLE stem
    ADD COLUMN IF NOT EXISTS codintegrantealtastem BIGINT;

ALTER TABLE stem
    DROP CONSTRAINT IF EXISTS fkstemintegrantealta;

ALTER TABLE stem
    ADD CONSTRAINT fkstemintegrantealta
        FOREIGN KEY (codintegrantealtastem)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT;

UPDATE stem s
SET (fechahoraaltastem, codintegrantealtastem) = (
    SELECT sep.fechahorafin, sep.codintegrante
    FROM separacionstem sep
    WHERE sep.codigocancionversion = s.codigocancionversion
      AND sep.estadoseparacion = 'COMPLETADA'
      AND sep.fechahorafin IS NOT NULL
    ORDER BY sep.fechahorafin DESC
    LIMIT 1
)
WHERE s.generadoconia
  AND s.fechahoraaltastem IS NULL;

UPDATE stem s
SET fechahoraaltastem = cv.fechahoraaltaversion
FROM cancionversion cv
WHERE cv.codigocancionversion = s.codigocancionversion
  AND s.fechahoraaltastem IS NULL;

ALTER TABLE stem
    ALTER COLUMN fechahoraaltastem SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE stem
    ALTER COLUMN fechahoraaltastem SET NOT NULL;

COMMIT;
