BEGIN;

-- =========================================================
-- RESPUESTAS A COMENTARIOS
-- =========================================================

CREATE TABLE comentariorespuesta (
    codigorespuestacomentario BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    codigocomentario BIGINT NOT NULL,
    codintegrante BIGINT NOT NULL,

    descripcionrespuesta TEXT NOT NULL,

    fechahoraaltarespuesta TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajarespuesta TIMESTAMPTZ,

    CONSTRAINT fkcomentariorespuestacomentario
        FOREIGN KEY (codigocomentario)
        REFERENCES comentario(codigocomentario)
        ON DELETE RESTRICT,

    CONSTRAINT fkcomentariorespuestaintegrante
        FOREIGN KEY (codintegrante)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT,

    CONSTRAINT chkcomentariorespuestatexto
        CHECK (LENGTH(TRIM(descripcionrespuesta)) > 0)
);

CREATE INDEX idxcomentariorespuestacomentario
    ON comentariorespuesta (codigocomentario);

CREATE INDEX idxcomentariorespuestaintegrante
    ON comentariorespuesta (codintegrante);

COMMIT;