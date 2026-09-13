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

	ExisteComentarioActivoEnVersion(
		codigoComentario int64,
		codigoVersion int64,
	) (bool, error)

	CrearRespuesta(
		codigoComentario int64,
		codigoIntegrante int64,
		texto string,
	) (*dto.RespuestaComentarioResponse, error)

	ObtenerAutorComentario(
		codigoComentario int64,
		codigoVersion int64,
	) (int64, error)

	Modificar(
		codigoComentario int64,
		texto string,
	) error

	DarDeBaja(
		codigoComentario int64,
	) (bool, error)

	ExisteEstadoComentarioActivo(
		estado string,
	) (bool, error)

	CambiarEstado(
		codigoComentario int64,
		codigoVersion int64,
		estado string,
	) (bool, error)
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
			(c.codintegrante = $2) AS espropio,

			cr.codigorespuestacomentario,
			cr.descripcionrespuesta,
			cr.fechahoraaltarespuesta,
			ir.codintegrante,
			ir.nombreintegrante,
			(cr.codintegrante = $2) AS espropiarespuesta

		FROM comentario c

		INNER JOIN integrante i
			ON i.codintegrante = c.codintegrante

		INNER JOIN estadocomentario ec
			ON ec.codestadocom = c.codestadocom

		LEFT JOIN comentariorespuesta cr
			ON cr.codigocomentario = c.codigocomentario
			AND cr.fechahorabajarespuesta IS NULL

		LEFT JOIN integrante ir
			ON ir.codintegrante = cr.codintegrante

		WHERE c.codigocancionversion = $1
		  AND c.fechahorabajacomentario IS NULL

		ORDER BY
			c.fechahoraaltacomentario DESC,
			cr.fechahoraaltarespuesta ASC
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

	indices := make(map[int64]int)

	for rows.Next() {

		var codigoComentario int64
		var textoComentario string
		var estado string
		var tiempoInicio *float64
		var tiempoFin *float64
		var fechaHoraAlta sql.NullTime
		var autorCodigo int64
		var autorNombre string
		var esPropio bool

		var codigoRespuesta sql.NullInt64
		var textoRespuesta sql.NullString
		var fechaRespuesta sql.NullTime
		var codigoAutorRespuesta sql.NullInt64
		var nombreAutorRespuesta sql.NullString
		var esPropiaRespuesta sql.NullBool

		err := rows.Scan(
			&codigoComentario,
			&textoComentario,
			&estado,
			&tiempoInicio,
			&tiempoFin,
			&fechaHoraAlta,
			&autorCodigo,
			&autorNombre,
			&esPropio,

			&codigoRespuesta,
			&textoRespuesta,
			&fechaRespuesta,
			&codigoAutorRespuesta,
			&nombreAutorRespuesta,
			&esPropiaRespuesta,
		)

		if err != nil {
			return nil, err
		}

		indice, existe :=
			indices[codigoComentario]

		if !existe {

			comentario := dto.ComentarioListadoResponse{
				CodigoComentario:     codigoComentario,
				Texto:                textoComentario,
				Estado:               estado,
				TiempoInicioSegundos: tiempoInicio,
				TiempoFinSegundos:    tiempoFin,
				FechaHoraAlta:        fechaHoraAlta.Time,
				Autor: dto.AutorComentarioResponse{
					CodigoIntegrante: autorCodigo,
					Nombre:           autorNombre,
				},
				EsPropio:   esPropio,
				Respuestas: make([]dto.RespuestaComentarioResponse, 0),
			}

			comentarios = append(
				comentarios,
				comentario,
			)

			indice = len(comentarios) - 1
			indices[codigoComentario] = indice
		}

		if codigoRespuesta.Valid {

			respuesta :=
				dto.RespuestaComentarioResponse{
					CodigoRespuesta: codigoRespuesta.Int64,
					Texto:           textoRespuesta.String,
					FechaHoraAlta:   fechaRespuesta.Time,
					Autor: dto.AutorComentarioResponse{
						CodigoIntegrante: codigoAutorRespuesta.Int64,
						Nombre:           nombreAutorRespuesta.String,
					},
					EsPropia: esPropiaRespuesta.Bool,
				}

			comentarios[indice].Respuestas =
				append(
					comentarios[indice].Respuestas,
					respuesta,
				)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comentarios, nil
}

func (r *comentarioRepository) ExisteComentarioActivoEnVersion(
	codigoComentario int64,
	codigoVersion int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM comentario
			WHERE codigocomentario = $1
			  AND codigocancionversion = $2
			  AND fechahorabajacomentario IS NULL
		)
	`,
		codigoComentario,
		codigoVersion,
	).Scan(&existe)

	return existe, err
}

func (r *comentarioRepository) CrearRespuesta(
	codigoComentario int64,
	codigoIntegrante int64,
	texto string,
) (*dto.RespuestaComentarioResponse, error) {

	var respuesta dto.RespuestaComentarioResponse

	err := r.db.QueryRow(`
		INSERT INTO comentariorespuesta (
			codigocomentario,
			codintegrante,
			descripcionrespuesta,
			fechahoraaltarespuesta
		)
		VALUES (
			$1,
			$2,
			$3,
			CURRENT_TIMESTAMP
		)
		RETURNING
			codigorespuestacomentario,
			descripcionrespuesta,
			fechahoraaltarespuesta
	`,
		codigoComentario,
		codigoIntegrante,
		texto,
	).Scan(
		&respuesta.CodigoRespuesta,
		&respuesta.Texto,
		&respuesta.FechaHoraAlta,
	)

	return &respuesta, err
}

func (r *comentarioRepository) ObtenerAutorComentario(
	codigoComentario int64,
	codigoVersion int64,
) (int64, error) {

	var codigoIntegrante int64

	err := r.db.QueryRow(`
		SELECT codintegrante
		FROM comentario
		WHERE codigocomentario = $1
		  AND codigocancionversion = $2
		  AND fechahorabajacomentario IS NULL
	`,
		codigoComentario,
		codigoVersion,
	).Scan(&codigoIntegrante)

	return codigoIntegrante, err
}

func (r *comentarioRepository) Modificar(
	codigoComentario int64,
	texto string,
) error {

	_, err := r.db.Exec(`
		UPDATE comentario
		SET descripcioncomentario = $2
		WHERE codigocomentario = $1
		  AND fechahorabajacomentario IS NULL
	`,
		codigoComentario,
		texto,
	)

	return err
}

func (r *comentarioRepository) DarDeBaja(
	codigoComentario int64,
) (bool, error) {

	resultado, err := r.db.Exec(`
		UPDATE comentario
		SET fechahorabajacomentario = CURRENT_TIMESTAMP
		WHERE codigocomentario = $1
		  AND fechahorabajacomentario IS NULL
	`,
		codigoComentario,
	)

	if err != nil {
		return false, err
	}

	filasAfectadas, err := resultado.RowsAffected()

	if err != nil {
		return false, err
	}

	return filasAfectadas > 0, nil
}

func (r *comentarioRepository) ExisteEstadoComentarioActivo(
	estado string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM estadocomentario
			WHERE LOWER(nombreestadocom) = LOWER($1)
			  AND fechahorabajaestadocom IS NULL
		)
	`,
		estado,
	).Scan(&existe)

	return existe, err
}

func (r *comentarioRepository) CambiarEstado(
	codigoComentario int64,
	codigoVersion int64,
	estado string,
) (bool, error) {

	resultado, err := r.db.Exec(`
		UPDATE comentario
		SET codestadocom = (
			SELECT codestadocom
			FROM estadocomentario
			WHERE LOWER(nombreestadocom) = LOWER($3)
			  AND fechahorabajaestadocom IS NULL
		)
		WHERE codigocomentario = $1
		  AND codigocancionversion = $2
		  AND fechahorabajacomentario IS NULL
	`,
		codigoComentario,
		codigoVersion,
		estado,
	)

	if err != nil {
		return false, err
	}

	filasAfectadas, err := resultado.RowsAffected()

	if err != nil {
		return false, err
	}

	return filasAfectadas > 0, nil
}
