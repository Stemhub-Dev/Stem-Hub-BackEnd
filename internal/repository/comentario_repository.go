package repository

import "database/sql"

type ComentarioRepository interface {
	Crear(
		codigoIntegrante int64,
		codigoVersion int64,
		texto string,
	) (int64, error)
}

type comentarioRepository struct {
	db *sql.DB
}

func NewComentarioRepository(
	db *sql.DB,
) ComentarioRepository {
	return &comentarioRepository{
		db: db,
	}
}

func (r *comentarioRepository) Crear(
	codigoIntegrante int64,
	codigoVersion int64,
	texto string,
) (int64, error) {

	var codigoComentario int64

	err := r.db.QueryRow(`
		INSERT INTO comentario (
			codintegrante,
			codestadocom,
			codigocancionversion,
			descripcioncomentario,
			fechahoraaltacomentario
		)
		SELECT
			$1,
			ec.codestadocom,
			$2,
			$3,
			CURRENT_TIMESTAMP
		FROM estadocomentario ec
		WHERE LOWER(ec.nombreestadocom) = LOWER('Pendiente')
		  AND ec.fechahorabajaestadocom IS NULL
		RETURNING codigocomentario
	`,
		codigoIntegrante,
		codigoVersion,
		texto,
	).Scan(&codigoComentario)

	return codigoComentario, err
}
