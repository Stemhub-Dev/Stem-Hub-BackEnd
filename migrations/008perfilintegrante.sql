BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS ux_integrante_codigousuario
ON integrante (codigousuario);

COMMIT;