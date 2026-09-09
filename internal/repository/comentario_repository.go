package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type ComentarioRepository interface {
	Crear(
		codigoIntegrante int64,
		codigoVersion int64,
		texto string,
		tiempoInicioSegundos *float64,
		tiempoFinSegundos *float64,
	) (int64, error)

	ListarPorVersion(
		codigoVersion int64,
		codigoIntegranteActual int64,
	) ([]dto.ComentarioListadoResponse, error)
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
	tiempoInicioSegundos *float64,
	tiempoFinSegundos *float64,
) (int64, error) {

	var codigoComentario int64

	err := r.db.QueryRow(`
		INSERT INTO comentario (
			codintegrante,
			codestadocom,
			codigocancionversion,
			descripcioncomentario,
			tiempoiniciosegundos,
			tiempofinsegundos,
			fechahoraaltacomentario
		)
		SELECT
			$1,
			ec.codestadocom,
			$2,
			$3,
			$4,
			$5,
			CURRENT_TIMESTAMP
		FROM estadocomentario ec
		WHERE LOWER(ec.nombreestadocom) = LOWER('Pendiente')
		  AND ec.fechahorabajaestadocom IS NULL
		RETURNING codigocomentario
	`,
		codigoIntegrante,
		codigoVersion,
		texto,
		tiempoInicioSegundos,
		tiempoFinSegundos,
	).Scan(&codigoComentario)

	return codigoComentario, err
}

func (r *comentarioRepository) ListarPorVersion(
	codigoVersion int64,
	codigoIntegranteActual int64,
) ([]dto.ComentarioListadoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			c.codigocomentario,
			c.descripcioncomentario,
			ec.nombreestadocom,
			c.tiempoiniciosegundos,
			c.tiempofinsegundos,
			c.fechahoraaltacomentario,
			i.codintegrante,
			i.nombreintegrante,
			(c.codintegrante = $2) AS espropio
		FROM comentario c
		INNER JOIN integrante i
			ON i.codintegrante = c.codintegrante
		INNER JOIN estadocomentario ec
			ON ec.codestadocom = c.codestadocom
		WHERE c.codigocancionversion = $1
		  AND c.fechahorabajacomentario IS NULL
		ORDER BY c.fechahoraaltacomentario DESC
	`,
		codigoVersion,
		codigoIntegranteActual,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	comentarios := make(
		[]dto.ComentarioListadoResponse,
		0,
	)

	for rows.Next() {

		var comentario dto.ComentarioListadoResponse

		err := rows.Scan(
			&comentario.CodigoComentario,
			&comentario.Texto,
			&comentario.Estado,
			&comentario.TiempoInicioSegundos,
			&comentario.TiempoFinSegundos,
			&comentario.FechaHoraAlta,
			&comentario.Autor.CodigoIntegrante,
			&comentario.Autor.Nombre,
			&comentario.EsPropio,
		)

		if err != nil {
			return nil, err
		}

		comentarios = append(
			comentarios,
			comentario,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comentarios, nil
}
