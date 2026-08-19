BEGIN;

SET search_path TO public;

CREATE TABLE auditoria (
    codigoauditoria BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    codigousuario BIGINT NOT NULL,

    entidad VARCHAR(100) NOT NULL,

    codigoregistro BIGINT NOT NULL,

    accion VARCHAR(10) NOT NULL,

    fechahoraauditoria TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fkauditoriausuario
        FOREIGN KEY (codigousuario)
        REFERENCES usuario(codigousuario)
        ON DELETE RESTRICT,

    CONSTRAINT chkaccionauditoria
        CHECK (accion IN ('INSERT', 'UPDATE', 'DELETE'))
);

CREATE INDEX idxauditoriausuario
ON auditoria (codigousuario);

CREATE INDEX idxauditoriaentidadregistro
ON auditoria (entidad, codigoregistro);

COMMIT;