BEGIN;

-- =========================================================
-- INVITACIONES POR ENLACE
-- =========================================================
-- HU-NOT-B02 #2 / HU-NOT-F02 #3: además de invitar por email, el
-- propietario puede generar un enlace copiable con un rol. Esa invitación
-- no tiene email: emailinvitado queda NULL y la acepta el primer usuario
-- que abra el enlace (sigue siendo de un solo uso y con vencimiento).
--
-- uqinvitacionproyectopendiente no cambia: los NULL no chocan entre sí,
-- así que puede haber varios enlaces pendientes por proyecto.

ALTER TABLE invitacionproyecto
    ALTER COLUMN emailinvitado DROP NOT NULL;

COMMIT;
