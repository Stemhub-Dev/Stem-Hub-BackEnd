-------------------------------------------------------

BEGIN;

ALTER TABLE public.auditoria
    ADD COLUMN evento VARCHAR(30),
    ADD COLUMN codigoproyecto BIGINT,
    ADD COLUMN contexto JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE public.auditoria
    ADD CONSTRAINT chkcontextoauditoria
    CHECK (jsonb_typeof(contexto) = 'object');

CREATE INDEX idxauditoriaproyectofecha
    ON public.auditoria (
        codigoproyecto,
        fechahoraauditoria DESC,
        codigoauditoria DESC
    );

COMMIT;

----------------------------------------------
