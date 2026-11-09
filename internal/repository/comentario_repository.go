package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type ComentarioRepository interface {
	Crear(
		codigoUsuario int64,
		codigoIntegrante int64,
		codigoVersion int64,
		codStem *int64,
		texto string,
		tiempoInicioSegundos *float64,
		tiempoFinSegundos *float64,
	) (int64, error)

	ExisteStemActivoEnVersion(
		codStem int64,
		codigoVersion int64,
	) (bool, error)

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

	// Autor de una respuesta activa del comentario indicado, que a su vez
	// tiene que estar activo en la versión.
	ObtenerAutorRespuesta(
		codigoRespuesta int64,
		codigoComentario int64,
		codigoVersion int64,
	) (int64, error)

	DarDeBajaRespuesta(
		codigoRespuesta int64,
	) (bool, error)

	Modificar(
		codigoUsuario int64,
		codigoComentario int64,
		texto string,
	) error

	DarDeBaja(
		codigoUsuario int64,
		codigoComentario int64,
	) (bool, error)

	ExisteEstadoComentarioActivo(
		estado string,
	) (bool, error)

	CambiarEstado(
		codigoUsuario int64,
		codigoComentario int64,
		codigoVersion int64,
		estado string,
	) (bool, error)

	// Nombre de la canción y número de la versión, para darle contexto al
	// resumen de comentarios con IA.
	ObtenerContextoVersion(
		codigoVersion int64,
	) (nombreCancion string, numeroVersion int, err error)
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
	codigoUsuario int64,
	codigoIntegrante int64,
	codigoVersion int64,
	codStem *int64,
	texto string,
	tiempoInicioSegundos *float64,
	tiempoFinSegundos *float64,
) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var codigoComentario int64

	err = tx.QueryRow(`
		INSERT INTO comentario (
			codintegrante,
			codestadocom,
			codigocancionversion,
			descripcioncomentario,
			tiempoiniciosegundos,
			tiempofinsegundos,
			codstem,
			fechahoraaltacomentario
		)
		SELECT
			$1,
			ec.codestadocom,
			$2,
			$3,
			$4,
			$5,
			$6,
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
		codStem,
	).Scan(&codigoComentario)
	if err != nil {
		return 0, err
	}

	err = RegistrarAltaComentarioTx(
		tx,
		codigoUsuario,
		codigoComentario,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return codigoComentario, nil
}

func (r *comentarioRepository) ExisteStemActivoEnVersion(
	codStem int64,
	codigoVersion int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM stem
			WHERE codstem = $1
			  AND codigocancionversion = $2
			  AND fechahorabajastem IS NULL
		)
	`,
		codStem,
		codigoVersion,
	).Scan(&existe)

	return existe, err
}

func (r *comentarioRepository) ListarPorVersion(
	codigoVersion int64,
	codigoIntegranteActual int64,
) ([]dto.ComentarioListadoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			c.codigocomentario,
			c.codstem,
			c.descripcioncomentario,
			ec.nombreestadocom,
			c.tiempoiniciosegundos,
			c.tiempofinsegundos,
			c.fechahoraaltacomentario,
			i.codintegrante,
			i.nombreintegrante,
			i.urlavatarintegrante,
			(c.codintegrante = $2) AS espropio,

			cr.codigorespuestacomentario,
			cr.descripcionrespuesta,
			cr.fechahoraaltarespuesta,
			ir.codintegrante,
			ir.nombreintegrante,
			ir.urlavatarintegrante,
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
		var codStem sql.NullInt64
		var textoComentario string
		var estado string
		var tiempoInicio *float64
		var tiempoFin *float64
		var fechaHoraAlta sql.NullTime
		var autorCodigo int64
		var autorNombre string
		var autorAvatar sql.NullString
		var esPropio bool

		var codigoRespuesta sql.NullInt64
		var textoRespuesta sql.NullString
		var fechaRespuesta sql.NullTime
		var codigoAutorRespuesta sql.NullInt64
		var nombreAutorRespuesta sql.NullString
		var avatarAutorRespuesta sql.NullString
		var esPropiaRespuesta sql.NullBool

		err := rows.Scan(
			&codigoComentario,
			&codStem,
			&textoComentario,
			&estado,
			&tiempoInicio,
			&tiempoFin,
			&fechaHoraAlta,
			&autorCodigo,
			&autorNombre,
			&autorAvatar,
			&esPropio,

			&codigoRespuesta,
			&textoRespuesta,
			&fechaRespuesta,
			&codigoAutorRespuesta,
			&nombreAutorRespuesta,
			&avatarAutorRespuesta,
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
				CodStem:              punteroSiValido(codStem),
				Texto:                textoComentario,
				Estado:               estado,
				TiempoInicioSegundos: tiempoInicio,
				TiempoFinSegundos:    tiempoFin,
				FechaHoraAlta:        fechaHoraAlta.Time,
				Autor: dto.AutorComentarioResponse{
					CodigoIntegrante: autorCodigo,
					Nombre:           autorNombre,
					AvatarObjectKey:  punteroTextoSiValido(autorAvatar),
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
						AvatarObjectKey:  punteroTextoSiValido(avatarAutorRespuesta),
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

func (r *comentarioRepository) ObtenerAutorRespuesta(
	codigoRespuesta int64,
	codigoComentario int64,
	codigoVersion int64,
) (int64, error) {

	var codigoIntegrante int64

	err := r.db.QueryRow(`
		SELECT cr.codintegrante
		FROM comentariorespuesta cr
		INNER JOIN comentario c
			ON c.codigocomentario = cr.codigocomentario
		WHERE cr.codigorespuestacomentario = $1
		  AND cr.codigocomentario = $2
		  AND c.codigocancionversion = $3
		  AND cr.fechahorabajarespuesta IS NULL
		  AND c.fechahorabajacomentario IS NULL
	`,
		codigoRespuesta,
		codigoComentario,
		codigoVersion,
	).Scan(&codigoIntegrante)

	return codigoIntegrante, err
}

func (r *comentarioRepository) DarDeBajaRespuesta(
	codigoRespuesta int64,
) (bool, error) {

	resultado, err := r.db.Exec(`
		UPDATE comentariorespuesta
		SET fechahorabajarespuesta = CURRENT_TIMESTAMP
		WHERE codigorespuestacomentario = $1
		  AND fechahorabajarespuesta IS NULL
	`,
		codigoRespuesta,
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

func (r *comentarioRepository) Modificar(
	codigoUsuario int64,
	codigoComentario int64,
	texto string,
) error {
	actualizado, err := r.cambiarConAuditoria(
		codigoUsuario,
		codigoComentario,
		nil,
		"MODIFICACION",
		texto,
	)
	if err != nil {
		return err
	}

	if !actualizado {
		return sql.ErrNoRows
	}

	return nil
}

func (r *comentarioRepository) DarDeBaja(
	codigoUsuario int64,
	codigoComentario int64,
) (bool, error) {
	return r.cambiarConAuditoria(
		codigoUsuario,
		codigoComentario,
		nil,
		"BAJA",
		"",
	)
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
	codigoUsuario int64,
	codigoComentario int64,
	codigoVersion int64,
	estado string,
) (bool, error) {
	return r.cambiarConAuditoria(
		codigoUsuario,
		codigoComentario,
		&codigoVersion,
		"CAMBIO_ESTADO",
		estado,
	)
}

func punteroTextoSiValido(valor sql.NullString) *string {
	if !valor.Valid {
		return nil
	}
	return &valor.String
}

func punteroSiValido(valor sql.NullInt64) *int64 {
	if !valor.Valid {
		return nil
	}
	return &valor.Int64
}

func (r *comentarioRepository) ObtenerContextoVersion(
	codigoVersion int64,
) (string, int, error) {

	var nombreCancion string
	var numeroVersion int

	err := r.db.QueryRow(`
		SELECT
			c.nombrecancion,
			cv.numeroversion
		FROM cancionversion cv
		JOIN cancion c
		  ON c.codigocancion = cv.codigocancion
		WHERE cv.codigocancionversion = $1
	`,
		codigoVersion,
	).Scan(&nombreCancion, &numeroVersion)

	return nombreCancion, numeroVersion, err
}

type contextoAuditoriaComentario struct {
	CodigoProyecto int64
	NombreProyecto string
	Texto          string
	Estado         string
	Datos          json.RawMessage
}

func obtenerContextoComentarioTx(
	tx *sql.Tx,
	codigoComentario int64,
	codigoVersion *int64,
) (*contextoAuditoriaComentario, error) {
	var contexto contextoAuditoriaComentario
	var datos string

	err := tx.QueryRow(`
		SELECT
			c.codigoproyecto,
			p.nombreproyecto,
			com.descripcioncomentario,
			ec.nombreestadocom,
			jsonb_build_object(
				'codigoCancion', c.codigocancion,
				'nombreCancion', c.nombrecancion,
				'codigoCancionVersion', cv.codigocancionversion,
				'numeroVersion', cv.numeroversion,
				'codStem', com.codstem,
				'nombreStem', s.nombrestem,
				'texto', com.descripcioncomentario,
				'estado', ec.nombreestadocom,
				'tiempoInicioSegundos', com.tiempoiniciosegundos,
				'tiempoFinSegundos', com.tiempofinsegundos
			)::text
		FROM comentario com
		JOIN cancionversion cv
		  ON cv.codigocancionversion = com.codigocancionversion
		JOIN cancion c
		  ON c.codigocancion = cv.codigocancion
		JOIN proyecto p
		  ON p.codigoproyecto = c.codigoproyecto
		JOIN estadocomentario ec
		  ON ec.codestadocom = com.codestadocom
		LEFT JOIN stem s
		  ON s.codstem = com.codstem
		WHERE com.codigocomentario = $1
		  AND com.fechahorabajacomentario IS NULL
		  AND (
			$2::bigint IS NULL
			OR com.codigocancionversion = $2
		  )
		FOR UPDATE OF com
	`, codigoComentario, codigoVersion).Scan(
		&contexto.CodigoProyecto,
		&contexto.NombreProyecto,
		&contexto.Texto,
		&contexto.Estado,
		&datos,
	)
	if err != nil {
		return nil, err
	}

	contexto.Datos = json.RawMessage(datos)
	return &contexto, nil
}

func (r *comentarioRepository) cambiarConAuditoria(
	codigoUsuario int64,
	codigoComentario int64,
	codigoVersion *int64,
	evento string,
	valor string,
) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	anterior, err := obtenerContextoComentarioTx(
		tx,
		codigoComentario,
		codigoVersion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	contexto := map[string]any{
		"nombreProyecto": anterior.NombreProyecto,
	}

	switch evento {
	case "MODIFICACION":
		if anterior.Texto == valor {
			return true, tx.Commit()
		}

		_, err = tx.Exec(`
			UPDATE comentario
			SET descripcioncomentario = $2
			WHERE codigocomentario = $1
			  AND fechahorabajacomentario IS NULL
		`, codigoComentario, valor)

	case "CAMBIO_ESTADO":
		if strings.EqualFold(anterior.Estado, valor) {
			return true, tx.Commit()
		}

		_, err = tx.Exec(`
			UPDATE comentario
			SET codestadocom = (
				SELECT codestadocom
				FROM estadocomentario
				WHERE LOWER(nombreestadocom) = LOWER($2)
				  AND fechahorabajaestadocom IS NULL
			)
			WHERE codigocomentario = $1
			  AND fechahorabajacomentario IS NULL
		`, codigoComentario, valor)

	case "BAJA":
		var fechaBaja time.Time

		err = tx.QueryRow(`
			UPDATE comentario
			SET fechahorabajacomentario = CURRENT_TIMESTAMP
			WHERE codigocomentario = $1
			  AND fechahorabajacomentario IS NULL
			RETURNING fechahorabajacomentario
		`, codigoComentario).Scan(&fechaBaja)

		contexto["fechaBaja"] = fechaBaja
		contexto["estadoAnterior"] = anterior.Datos

	default:
		return false, fmt.Errorf(
			"evento de comentario no soportado: %s",
			evento,
		)
	}

	if err != nil {
		return false, err
	}

	if evento != "BAJA" {
		posterior, err := obtenerContextoComentarioTx(
			tx,
			codigoComentario,
			codigoVersion,
		)
		if err != nil {
			return false, err
		}

		contexto["anterior"] = anterior.Datos
		contexto["posterior"] = posterior.Datos
	}

	err = RegistrarAuditoriaTx(tx, RegistroAuditoria{
		CodigoUsuario:  codigoUsuario,
		Entidad:        "comentario",
		CodigoRegistro: codigoComentario,
		Accion:         "UPDATE",
		Evento:         evento,
		CodigoProyecto: &anterior.CodigoProyecto,
		Contexto:       contexto,
	})
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}
