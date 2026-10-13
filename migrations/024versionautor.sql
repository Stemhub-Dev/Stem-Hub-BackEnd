BEGIN;

-- =========================================================
-- AUTOR DE LA VERSIÓN (HU-VER-F08)
-- =========================================================
-- La pantalla Canción muestra quién subió cada versión. El esquema no lo
-- guardaba, así que se agrega el integrante que la subió. Las versiones
-- anteriores a esta migración quedan sin autor (NULL): no hay de dónde
-- reconstruirlo, porque la tabla auditoria no se llena.

ALTER TABLE cancionversion
    ADD COLUMN IF NOT EXISTS codintegrantealtaversion BIGINT;

ALTER TABLE cancionversion
    DROP CONSTRAINT IF EXISTS fkcancionversionintegrantealta;

ALTER TABLE cancionversion
    ADD CONSTRAINT fkcancionversionintegrantealta
        FOREIGN KEY (codintegrantealtaversion)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT;

COMMIT;
