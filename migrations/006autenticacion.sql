BEGIN;

ALTER TABLE usuario
ADD COLUMN idautenticacion UUID NOT NULL UNIQUE;

ALTER TABLE usuario
DROP COLUMN contrasenahash;

COMMIT;