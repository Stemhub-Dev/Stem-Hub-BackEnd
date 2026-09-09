BEGIN;

-- =========================================================
-- CANCION VERSION: ARCHIVO UNICO EN LUGAR DE WAV/MP3 SEPARADOS
-- =========================================================
-- Cada version corresponde a UN archivo de audio subido por el usuario,
-- no wav y mp3 simultaneamente. Se reemplazan las dos columnas por
-- una URL/key de objeto generica + el formato del archivo.

ALTER TABLE cancionversion
    ADD COLUMN urlarchivocancionver TEXT,
    ADD COLUMN formatoarchivocancionver VARCHAR(10);

ALTER TABLE cancionversion
    ADD CONSTRAINT chkformatoarchivocancionver
    CHECK (formatoarchivocancionver IN ('mp3', 'wav', 'flac'));

UPDATE cancionversion
SET urlarchivocancionver = COALESCE(urlversionwavcancionver, urlversionmp3cancionver),
    formatoarchivocancionver = CASE
        WHEN urlversionwavcancionver IS NOT NULL THEN 'wav'
        WHEN urlversionmp3cancionver IS NOT NULL THEN 'mp3'
    END
WHERE urlversionwavcancionver IS NOT NULL
   OR urlversionmp3cancionver IS NOT NULL;

ALTER TABLE cancionversion
    DROP COLUMN urlversionwavcancionver,
    DROP COLUMN urlversionmp3cancionver;

COMMIT;
