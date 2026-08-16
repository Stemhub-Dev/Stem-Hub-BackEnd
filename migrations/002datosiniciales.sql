BEGIN;

SET search_path TO public;

-- =========================================================
-- ROLES
-- =========================================================

INSERT INTO rol (
    nombrerol,
    descripcionrol
)
VALUES
    ('Productor', 'Integrante responsable de tareas de producción dentro del proyecto.'),
    ('Artista Musical', 'Integrante que participa musicalmente dentro del proyecto.')
ON CONFLICT (nombrerol) DO NOTHING;


-- =========================================================
-- ESTADOS DE COMENTARIO
-- =========================================================

INSERT INTO estadocomentario (
    nombreestadocom,
    descripcionestadocom
)
VALUES
    ('Pendiente', 'Comentario pendiente de revisión o resolución.'),
    ('Resuelto', 'Comentario que ya fue atendido o resuelto.')
ON CONFLICT (nombreestadocom) DO NOTHING;


-- =========================================================
-- ESTADOS DE PROYECTO
-- =========================================================

INSERT INTO estadoproyecto (
    nombreestadoproy,
    descripcionestadoproy
)
VALUES
    ('Sin Empezar', 'El proyecto fue creado pero todavía no se inició su desarrollo.'),
    ('En Desarrollo', 'El proyecto se encuentra actualmente en desarrollo.'),
    ('Terminado', 'El proyecto se encuentra finalizado.')
ON CONFLICT (nombreestadoproy) DO NOTHING;


-- =========================================================
-- TIPOS DE PROYECTO
-- =========================================================

INSERT INTO tipoproyecto (
    nombretipoproy,
    descripciontipoproy
)
VALUES
    ('Single', 'Proyecto musical correspondiente a un lanzamiento individual.'),
    ('EP', 'Proyecto musical de extensión intermedia compuesto por varias canciones.'),
    ('Album', 'Proyecto musical correspondiente a un álbum.')
ON CONFLICT (nombretipoproy) DO NOTHING;


-- =========================================================
-- GENEROS MUSICALES INICIALES
-- =========================================================

INSERT INTO generomusicalproyecto (
    nombregeneroproy,
    descripciongeneroproy
)
VALUES
    ('Rock', 'Género musical Rock.'),
    ('Rap', 'Género musical Rap.'),
    ('Pop', 'Género musical Pop.'),
    ('Jazz', 'Género musical Jazz.')
ON CONFLICT (nombregeneroproy) DO NOTHING;

COMMIT;