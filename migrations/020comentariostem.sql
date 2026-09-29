BEGIN;

-- =========================================================
-- COMENTARIO: COMENTARIOS DE UN STEM
-- =========================================================
-- Un comentario sigue perteneciendo a la versión (codigocancionversion), y
-- opcionalmente a uno de sus stems. NULL = comentario de la versión
-- completa. Así responder, editar, cambiar de estado y eliminar funcionan
-- igual para ambos.
--
-- RESTRICT como el resto del esquema: al eliminar un stem (borrado real)
-- el repositorio borra antes sus comentarios y las respuestas.

ALTER TABLE comentario
    ADD COLUMN codstem BIGINT;

ALTER TABLE comentario
    ADD CONSTRAINT fkcomentariostem
        FOREIGN KEY (codstem)
        REFERENCES stem(codstem)
        ON DELETE RESTRICT;

CREATE INDEX idxcomentariostem
    ON comentario (codstem)
    WHERE codstem IS NOT NULL;

COMMIT;
