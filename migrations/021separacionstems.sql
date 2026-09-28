BEGIN;

-- =========================================================
-- SEPARACIÓN DE STEMS CON IA ("Separar Pistas")
-- =========================================================
-- La separación la hace stemhub-microservicio-IA (Spleeter) y puede tardar
-- minutos, así que el backend la corre en segundo plano: cada pedido es una
-- fila de separacionstem que el frontend consulta hasta que termina. Los
-- stems resultantes son stems comunes con generadoconia = true (019).

CREATE TABLE separacionstem (
    codseparacionstem BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigocancionversion BIGINT NOT NULL,
    codintegrante BIGINT NOT NULL,
    cantidadstems SMALLINT NOT NULL,
    estadoseparacion VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    mensajeerror TEXT,
    tiempoprocesamientoms INTEGER,
    fechahorasolicitud TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorafin TIMESTAMPTZ,

    CONSTRAINT fkseparacionstemversion
        FOREIGN KEY (codigocancionversion)
        REFERENCES cancionversion(codigocancionversion)
        ON DELETE RESTRICT,

    CONSTRAINT fkseparacionstemintegrante
        FOREIGN KEY (codintegrante)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT,

    CONSTRAINT chkcantidadstems
        CHECK (cantidadstems IN (2, 4, 5)),

    CONSTRAINT chkestadoseparacion
        CHECK (estadoseparacion IN ('PENDIENTE', 'PROCESANDO', 'COMPLETADA', 'ERROR'))
);

-- Una sola separación en curso por versión: dos pedidos simultáneos chocan
-- acá y el segundo se responde con 409.
CREATE UNIQUE INDEX uqseparacionstemencurso
    ON separacionstem (codigocancionversion)
    WHERE estadoseparacion IN ('PENDIENTE', 'PROCESANDO');

CREATE INDEX idxseparacionstemversion
    ON separacionstem (codigocancionversion, fechahorasolicitud DESC);

-- ---------------------------------------------------------
-- CATEGORÍAS POR DEFECTO para los stems que genera Spleeter
-- ---------------------------------------------------------
-- vocals → Voz, drums → Batería, bass → Bajo ya existen (019). piano,
-- other y accompaniment necesitan las suyas.

INSERT INTO categoriastem (nombrecategoriastem, codigoproyecto)
SELECT nombre, NULL
FROM (VALUES ('Piano'), ('Otros')) AS nuevas(nombre)
WHERE NOT EXISTS (
    SELECT 1
    FROM categoriastem c
    WHERE c.codigoproyecto IS NULL
      AND c.fechahorabajacategoriastem IS NULL
      AND lower(btrim(c.nombrecategoriastem)) = lower(nuevas.nombre)
);

COMMIT;
