-- =========================================================
-- COMENTARIO: RANGO DE TIEMPO
-- =========================================================

ALTER TABLE comentario
    ADD COLUMN tiempoiniciosegundos NUMERIC(10,3),
    ADD COLUMN tiempofinsegundos NUMERIC(10,3);

ALTER TABLE comentario
    ADD CONSTRAINT chkcomentariorango
    CHECK (
        (tiempoiniciosegundos IS NULL AND tiempofinsegundos IS NULL)
        OR (tiempoiniciosegundos IS NOT NULL AND tiempofinsegundos IS NOT NULL
            AND tiempofinsegundos >= tiempoiniciosegundos)
    );
