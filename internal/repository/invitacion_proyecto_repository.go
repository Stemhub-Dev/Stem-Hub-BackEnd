package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type InvitacionProyectoRepository interface {
	// CrearOReemplazarPendiente hace UPDATE si ya existe una invitación
	// pendiente para (codigoProyecto, email) sin aceptar/cancelar, o
	// INSERT si no existe. Devuelve la invitación resultante con su nuevo
	// token.
	CrearOReemplazarPendiente(
		codigoProyecto int64,
		emailInvitado string,
		codRol int64,
		ambitoRol string,
		codIntegranteInvito int64,
		token string,
		expiracion time.Time,
	) (*model.InvitacionProyecto, error)

	BuscarPorToken(token string) (*model.InvitacionProyecto, error)

	BuscarDetallePorToken(token string) (*model.InvitacionDetalle, error)

	ListarPendientesPorProyecto(
		codigoProyecto int64,
	) ([]model.InvitacionProyecto, error)

	BuscarPendientePorID(
		codigoInvitacionProy int64,
		codigoProyecto int64,
	) (*model.InvitacionProyecto, error)

	// ListarPendientesPorEmail trae las invitaciones pendientes y vigentes
	// (ni aceptadas, ni canceladas, ni vencidas) para el email del usuario
	// que consulta su propia bandeja de notificaciones.
	ListarPendientesPorEmail(
		email string,
	) ([]model.InvitacionPendienteUsuario, error)

	MarcarCancelada(codigoInvitacionProy int64) error

	// Aceptar es atómico: inserta en integranteproyecto y marca la
	// invitación como aceptada en una sola transacción, para que nunca
	// quede uno de los dos sin el otro si algo falla a mitad de camino.
	Aceptar(
		codigoInvitacionProy int64,
		codIntegranteAceptante int64,
		codigoProyecto int64,
		codRol int64,
		ambitoRol string,
	) error
}

type invitacionProyectoRepository struct {
	db *sql.DB
}

func NewInvitacionProyectoRepository(db *sql.DB) InvitacionProyectoRepository {
	return &invitacionProyectoRepository{
		db: db,
	}
}

const columnasInvitacionProyecto = `
	codigoinvitacionproy, codigoproyecto, emailinvitado, codrol,
	ambitorol, codintegranteinvito, tokeninvitacion,
	fechahoraaltainvitacionproy, fechahoraexpiracioninvitacion,
	fechahoraaceptacioninvitacion, fechahorabajainvitacionproy
`

// escanearFilas recorre rows aplicando scan a cada una y devuelve el slice
// resultante. Centraliza el manejo de errores y el cierre de rows para que
// cada query de listado solo tenga que declarar su SELECT y su Scan.
func escanearFilas[T any](rows *sql.Rows, scan func(*sql.Rows, *T) error) ([]T, error) {

	defer rows.Close()

	resultado := make([]T, 0)

	for rows.Next() {

		var item T

		if err := scan(rows, &item); err != nil {
			return nil, err
		}

		resultado = append(resultado, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resultado, nil
}

func escanearInvitacion(row *sql.Row, invitacion *model.InvitacionProyecto) error {
	return row.Scan(
		&invitacion.CodigoInvitacionProy,
		&invitacion.CodigoProyecto,
		&invitacion.EmailInvitado,
		&invitacion.CodRol,
		&invitacion.AmbitoRol,
		&invitacion.CodIntegranteInvito,
		&invitacion.TokenInvitacion,
		&invitacion.FechaHoraAltaInvitacionProy,
		&invitacion.FechaHoraExpiracion,
		&invitacion.FechaHoraAceptacion,
		&invitacion.FechaHoraBajaInvitacionProy,
	)
}

func (r *invitacionProyectoRepository) CrearOReemplazarPendiente(
	codigoProyecto int64,
	emailInvitado string,
	codRol int64,
	ambitoRol string,
	codIntegranteInvito int64,
	token string,
	expiracion time.Time,
) (*model.InvitacionProyecto, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	var invitacion model.InvitacionProyecto

	row := tx.QueryRow(`
		UPDATE invitacionproyecto
		SET codrol = $3,
		    ambitorol = $4,
		    codintegranteinvito = $5,
		    tokeninvitacion = $6,
		    fechahoraaltainvitacionproy = CURRENT_TIMESTAMP,
		    fechahoraexpiracioninvitacion = $7
		WHERE codigoproyecto = $1
		  AND LOWER(emailinvitado) = LOWER($2)
		  AND fechahoraaceptacioninvitacion IS NULL
		  AND fechahorabajainvitacionproy IS NULL
		RETURNING `+columnasInvitacionProyecto,
		codigoProyecto, emailInvitado, codRol, ambitoRol,
		codIntegranteInvito, token, expiracion,
	)

	err = escanearInvitacion(row, &invitacion)

	if errors.Is(err, sql.ErrNoRows) {

		row := tx.QueryRow(`
			INSERT INTO invitacionproyecto (
				codigoproyecto, emailinvitado, codrol, ambitorol,
				codintegranteinvito, tokeninvitacion,
				fechahoraexpiracioninvitacion
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING `+columnasInvitacionProyecto,
			codigoProyecto, emailInvitado, codRol, ambitoRol,
			codIntegranteInvito, token, expiracion,
		)

		err = escanearInvitacion(row, &invitacion)
	}

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &invitacion, nil
}

func (r *invitacionProyectoRepository) BuscarPorToken(
	token string,
) (*model.InvitacionProyecto, error) {

	var invitacion model.InvitacionProyecto

	row := r.db.QueryRow(`
		SELECT `+columnasInvitacionProyecto+`
		FROM invitacionproyecto
		WHERE tokeninvitacion = $1
	`, token)

	if err := escanearInvitacion(row, &invitacion); err != nil {
		return nil, err
	}

	return &invitacion, nil
}

func (r *invitacionProyectoRepository) BuscarDetallePorToken(
	token string,
) (*model.InvitacionDetalle, error) {

	var detalle model.InvitacionDetalle

	err := r.db.QueryRow(`
		SELECT
			ip.emailinvitado,
			p.nombreproyecto,
			i.nombreintegrante,
			r.nombrerol,
			(ip.fechahoraexpiracioninvitacion < CURRENT_TIMESTAMP
				AND ip.fechahoraaceptacioninvitacion IS NULL) AS vencida,
			ip.fechahoraaceptacioninvitacion IS NOT NULL AS yaaceptada,
			ip.fechahorabajainvitacionproy IS NOT NULL AS cancelada
		FROM invitacionproyecto ip
		JOIN proyecto p
		  ON p.codigoproyecto = ip.codigoproyecto
		JOIN integrante i
		  ON i.codintegrante = ip.codintegranteinvito
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		WHERE ip.tokeninvitacion = $1 AND p.fechahorabajaproyecto IS NULL
	`,
		token,
	).Scan(
		&detalle.EmailInvitado,
		&detalle.NombreProyecto,
		&detalle.NombreIntegranteInvito,
		&detalle.NombreRol,
		&detalle.Vencida,
		&detalle.YaAceptada,
		&detalle.Cancelada,
	)

	if err != nil {
		return nil, err
	}

	return &detalle, nil
}

func (r *invitacionProyectoRepository) ListarPendientesPorProyecto(
	codigoProyecto int64,
) ([]model.InvitacionProyecto, error) {

	rows, err := r.db.Query(`
		SELECT `+columnasInvitacionProyecto+`
		FROM invitacionproyecto
		WHERE codigoproyecto = $1
		  AND fechahoraaceptacioninvitacion IS NULL
		  AND fechahorabajainvitacionproy IS NULL
		ORDER BY fechahoraaltainvitacionproy DESC
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	return escanearFilas(rows, func(rows *sql.Rows, invitacion *model.InvitacionProyecto) error {
		return rows.Scan(
			&invitacion.CodigoInvitacionProy,
			&invitacion.CodigoProyecto,
			&invitacion.EmailInvitado,
			&invitacion.CodRol,
			&invitacion.AmbitoRol,
			&invitacion.CodIntegranteInvito,
			&invitacion.TokenInvitacion,
			&invitacion.FechaHoraAltaInvitacionProy,
			&invitacion.FechaHoraExpiracion,
			&invitacion.FechaHoraAceptacion,
			&invitacion.FechaHoraBajaInvitacionProy,
		)
	})
}

func (r *invitacionProyectoRepository) BuscarPendientePorID(
	codigoInvitacionProy int64,
	codigoProyecto int64,
) (*model.InvitacionProyecto, error) {

	var invitacion model.InvitacionProyecto

	row := r.db.QueryRow(`
		SELECT `+columnasInvitacionProyecto+`
		FROM invitacionproyecto
		WHERE codigoinvitacionproy = $1
		  AND codigoproyecto = $2
		  AND fechahoraaceptacioninvitacion IS NULL
		  AND fechahorabajainvitacionproy IS NULL
	`, codigoInvitacionProy, codigoProyecto)

	if err := escanearInvitacion(row, &invitacion); err != nil {
		return nil, err
	}

	return &invitacion, nil
}

func (r *invitacionProyectoRepository) ListarPendientesPorEmail(
	email string,
) ([]model.InvitacionPendienteUsuario, error) {

	rows, err := r.db.Query(`
		SELECT
			ip.codigoinvitacionproy,
			ip.tokeninvitacion,
			p.nombreproyecto,
			i.nombreintegrante,
			r.nombrerol,
			ip.fechahoraexpiracioninvitacion
		FROM invitacionproyecto ip
		JOIN proyecto p
		  ON p.codigoproyecto = ip.codigoproyecto
		JOIN integrante i
		  ON i.codintegrante = ip.codintegranteinvito
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		WHERE LOWER(ip.emailinvitado) = LOWER($1)
		  AND ip.fechahoraaceptacioninvitacion IS NULL
		  AND ip.fechahorabajainvitacionproy IS NULL
		  AND ip.fechahoraexpiracioninvitacion > CURRENT_TIMESTAMP
		  AND p.fechahorabajaproyecto IS NULL
		ORDER BY ip.fechahoraaltainvitacionproy DESC
	`,
		email,
	)

	if err != nil {
		return nil, err
	}

	return escanearFilas(rows, func(rows *sql.Rows, invitacion *model.InvitacionPendienteUsuario) error {
		return rows.Scan(
			&invitacion.CodigoInvitacionProy,
			&invitacion.TokenInvitacion,
			&invitacion.NombreProyecto,
			&invitacion.NombreIntegranteInvito,
			&invitacion.NombreRol,
			&invitacion.FechaHoraExpiracion,
		)
	})
}

func (r *invitacionProyectoRepository) MarcarCancelada(
	codigoInvitacionProy int64,
) error {

	_, err := r.db.Exec(`
		UPDATE invitacionproyecto
		SET fechahorabajainvitacionproy = CURRENT_TIMESTAMP
		WHERE codigoinvitacionproy = $1
	`, codigoInvitacionProy)

	return err
}

func (r *invitacionProyectoRepository) Aceptar(
	codigoInvitacionProy int64,
	codIntegranteAceptante int64,
	codigoProyecto int64,
	codRol int64,
	ambitoRol string,
) error {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO integranteproyecto (
			codintegrante,
			codigoproyecto,
			codrol,
			fechahoraaltaintegranteproy,
			espropietario,
			ambitorol
		)
		VALUES (
			$1,
			$2,
			$3,
			CURRENT_TIMESTAMP,
			FALSE,
			$4
		)
	`,
		codIntegranteAceptante,
		codigoProyecto,
		codRol,
		ambitoRol,
	)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE invitacionproyecto
		SET fechahoraaceptacioninvitacion = CURRENT_TIMESTAMP
		WHERE codigoinvitacionproy = $1
	`, codigoInvitacionProy)

	if err != nil {
		return err
	}

	return tx.Commit()
}
