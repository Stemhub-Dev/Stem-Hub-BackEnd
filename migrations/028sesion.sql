BEGIN;

-- =========================================================
-- SESIONES (HU-SEG-B02: logout e inactividad)
-- =========================================================
-- El access token de Supabase sigue siendo válido hasta su "exp" aunque el
-- usuario cierre sesión, y GoTrue no avisa al backend. Cada sesión de GoTrue
-- (claim session_id) se registra acá: el middleware rechaza con 401 las
-- revocadas (POST /auth/logout) y las que pasaron 45 minutos sin uso.

CREATE TABLE IF NOT EXISTS sesion (
    idsesion UUID PRIMARY KEY,
    idautenticacion UUID NOT NULL,
    fechahoraultimaactividad TIMESTAMPTZ NOT NULL,
    fechahorarevocacion TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idxsesionidautenticacion ON sesion (idautenticacion);

COMMIT;
