BEGIN;

-- =========================================================
-- USUARIO
-- =========================================================

CREATE TABLE usuario (
    codigousuario BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email VARCHAR(254) NOT NULL UNIQUE,
    contrasenahash TEXT NOT NULL,
    ultimologin TIMESTAMPTZ,
    fechahorabajausuario TIMESTAMPTZ
);


-- =========================================================
-- INTEGRANTE
-- =========================================================

CREATE TABLE integrante (
    codintegrante BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigousuario BIGINT NOT NULL UNIQUE,
    nombreintegrante VARCHAR(150) NOT NULL,
    descripcionintegrante TEXT,
    fechahorabajaintegrante TIMESTAMPTZ,

    CONSTRAINT fkintegranteusuario
        FOREIGN KEY (codigousuario)
        REFERENCES usuario(codigousuario)
        ON DELETE RESTRICT
);


-- =========================================================
-- ROL
-- =========================================================

CREATE TABLE rol (
    codrol BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombrerol VARCHAR(100) NOT NULL UNIQUE,
    descripcionrol TEXT,
    fechahorabajarol TIMESTAMPTZ
);


-- =========================================================
-- ESTADO PROYECTO
-- =========================================================

CREATE TABLE estadoproyecto (
    codestadoproy BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombreestadoproy VARCHAR(100) NOT NULL UNIQUE,
    descripcionestadoproy TEXT,
    fechahorabajaestadoproy TIMESTAMPTZ
);


-- =========================================================
-- TIPO PROYECTO
-- =========================================================

CREATE TABLE tipoproyecto (
    codtipoproy BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombretipoproy VARCHAR(100) NOT NULL UNIQUE,
    descripciontipoproy TEXT,
    fechahorabajatipoproy TIMESTAMPTZ
);


-- =========================================================
-- GENERO MUSICAL PROYECTO
-- =========================================================

CREATE TABLE generomusicalproyecto (
    codigogeneroproy BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombregeneroproy VARCHAR(100) NOT NULL UNIQUE,
    descripciongeneroproy TEXT,
    fechahorabajageneroproy TIMESTAMPTZ
);


-- =========================================================
-- PROYECTO
-- =========================================================

CREATE TABLE proyecto (
    codigoproyecto BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombreproyecto VARCHAR(150) NOT NULL,
    descripcionproyecto TEXT,
    logoproyecto TEXT,
    codestadoproy BIGINT NOT NULL,
    codtipoproy BIGINT NOT NULL,
    fechahorabajaproyecto TIMESTAMPTZ,

    CONSTRAINT fkproyectoestado
        FOREIGN KEY (codestadoproy)
        REFERENCES estadoproyecto(codestadoproy)
        ON DELETE RESTRICT,

    CONSTRAINT fkproyectotipo
        FOREIGN KEY (codtipoproy)
        REFERENCES tipoproyecto(codtipoproy)
        ON DELETE RESTRICT
);


-- =========================================================
-- PROYECTO - GENERO MUSICAL
-- Relacion N:M
-- =========================================================

CREATE TABLE proyectogeneromusical (
    codigoproyectogeneromusical BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigoproyecto BIGINT NOT NULL,
    codigogeneroproy BIGINT NOT NULL,

    CONSTRAINT fkproyectogeneroproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto(codigoproyecto)
        ON DELETE RESTRICT,

    CONSTRAINT fkproyectogenerogenero
        FOREIGN KEY (codigogeneroproy)
        REFERENCES generomusicalproyecto(codigogeneroproy)
        ON DELETE RESTRICT,

    CONSTRAINT uqproyectogeneromusical
        UNIQUE (codigoproyecto, codigogeneroproy)
);


-- =========================================================
-- INTEGRANTE - PROYECTO
-- =========================================================

CREATE TABLE integranteproyecto (
    codigointegranteproyecto BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codintegrante BIGINT NOT NULL,
    codigoproyecto BIGINT NOT NULL,
    codrol BIGINT NOT NULL,
    fechahoraaltaintegranteproy TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajaintegranteproy TIMESTAMPTZ,

    CONSTRAINT fkintegranteproyectointegrante
        FOREIGN KEY (codintegrante)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT,

    CONSTRAINT fkintegranteproyectoproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto(codigoproyecto)
        ON DELETE RESTRICT,

    CONSTRAINT fkintegranteproyectorol
        FOREIGN KEY (codrol)
        REFERENCES rol(codrol)
        ON DELETE RESTRICT
);

-- Un integrante solo puede tener un rol activo dentro de un mismo proyecto.
CREATE UNIQUE INDEX uqintegranteproyectoactivo
ON integranteproyecto (codintegrante, codigoproyecto)
WHERE fechahorabajaintegranteproy IS NULL;


-- =========================================================
-- CANCION
-- =========================================================

CREATE TABLE cancion (
    codigocancion BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigoproyecto BIGINT NOT NULL,
    nombrecancion VARCHAR(200) NOT NULL,
    fechahorabajacancion TIMESTAMPTZ,

    CONSTRAINT fkcancionproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto(codigoproyecto)
        ON DELETE RESTRICT
);


-- =========================================================
-- CANCION VERSION
-- =========================================================

CREATE TABLE cancionversion (
    codigocancionversion BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigocancion BIGINT NOT NULL,
    numeroversion INTEGER NOT NULL,
    fechahoraaltaversion TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajaversion TIMESTAMPTZ,
    urlversionwavcancionver TEXT,
    urlversionmp3cancionver TEXT,

    CONSTRAINT fkcancionversioncancion
        FOREIGN KEY (codigocancion)
        REFERENCES cancion(codigocancion)
        ON DELETE RESTRICT,

    CONSTRAINT uqcancionversion
        UNIQUE (codigocancion, numeroversion),

    CONSTRAINT chknumeroversion
        CHECK (numeroversion > 0)
);


-- =========================================================
-- STEM
-- =========================================================

CREATE TABLE stem (
    codstem BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigocancionversion BIGINT NOT NULL,
    nombrestem VARCHAR(150) NOT NULL,
    descripcionstem TEXT,
    urlversionwav TEXT,
    urlversionmp3 TEXT,
    fechahorabajastem TIMESTAMPTZ,

    CONSTRAINT fkstemcancionversion
        FOREIGN KEY (codigocancionversion)
        REFERENCES cancionversion(codigocancionversion)
        ON DELETE RESTRICT
);


-- =========================================================
-- ESTADO COMENTARIO
-- =========================================================

CREATE TABLE estadocomentario (
    codestadocom BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombreestadocom VARCHAR(100) NOT NULL UNIQUE,
    descripcionestadocom TEXT,
    fechahorabajaestadocom TIMESTAMPTZ
);


-- =========================================================
-- COMENTARIO
-- =========================================================

CREATE TABLE comentario (
    codigocomentario BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codintegrante BIGINT NOT NULL,
    codestadocom BIGINT NOT NULL,
    codigoproyecto BIGINT,
    codigocancionversion BIGINT,
    descripcioncomentario TEXT NOT NULL,
    fechahoraaltacomentario TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fechahorabajacomentario TIMESTAMPTZ,

    CONSTRAINT fkcomentariointegrante
        FOREIGN KEY (codintegrante)
        REFERENCES integrante(codintegrante)
        ON DELETE RESTRICT,

    CONSTRAINT fkcomentarioestado
        FOREIGN KEY (codestadocom)
        REFERENCES estadocomentario(codestadocom)
        ON DELETE RESTRICT,

    CONSTRAINT fkcomentarioproyecto
        FOREIGN KEY (codigoproyecto)
        REFERENCES proyecto(codigoproyecto)
        ON DELETE RESTRICT,

    CONSTRAINT fkcomentariocancionversion
        FOREIGN KEY (codigocancionversion)
        REFERENCES cancionversion(codigocancionversion)
        ON DELETE RESTRICT,

    -- Un comentario pertenece a un proyecto O a una version,
    -- nunca a ambos y nunca a ninguno.
    CONSTRAINT chkcomentariodestino
        CHECK (
            (codigoproyecto IS NOT NULL AND codigocancionversion IS NULL)
            OR
            (codigoproyecto IS NULL AND codigocancionversion IS NOT NULL)
        )
);


-- =========================================================
-- PREGUNTA FRECUENTE
-- =========================================================

CREATE TABLE preguntafrecuente (
    codigopreguntafrecuente BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    funcionalidadprincipal VARCHAR(150) NOT NULL,
    preguntafrecuente TEXT NOT NULL,
    respuestafrecuente TEXT NOT NULL,
    fechahorabajapreguntafrecuente TIMESTAMPTZ
);

COMMIT;