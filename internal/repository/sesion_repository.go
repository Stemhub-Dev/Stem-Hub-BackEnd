package repository

import (
	"database/sql"
	"errors"
	"time"
)

// SesionRepository registra el uso de cada sesión de GoTrue (claim
// session_id del JWT) para poder rechazar las cerradas y las inactivas.
type SesionRepository interface {
	// RegistrarActividad marca la sesión como usada en "ahora" y devuelve
	// false, sin tocarla, si fue revocada, si su último uso es anterior a
	// ahora - inactividadMaxima o si pertenece a otro usuario.
	RegistrarActividad(idSesion, idAutenticacion string, ahora time.Time, inactividadMaxima time.Duration) (bool, error)
	// Revocar cierra la sesión: desde ahí todo token con ese session_id
	// recibe 401, aunque no haya vencido.
	Revocar(idSesion, idAutenticacion string, ahora time.Time) error
}

type sesionRepository struct {
	db *sql.DB
}

func NewSesionRepository(db *sql.DB) SesionRepository {
	return &sesionRepository{db: db}
}

// Un solo statement para que dos requests en paralelo no puedan revivir una
// sesión vencida: si la fila existe y no cumple el WHERE, el UPDATE no se
// aplica y no vuelve ninguna fila. GREATEST evita que una request demorada
// retroceda la marca.
const consultaRegistrarActividadSesion = `
		INSERT INTO sesion (idsesion, idautenticacion, fechahoraultimaactividad)
		VALUES ($1, $2, $3)
		ON CONFLICT (idsesion) DO UPDATE
		SET fechahoraultimaactividad = GREATEST(sesion.fechahoraultimaactividad, EXCLUDED.fechahoraultimaactividad)
		WHERE sesion.fechahorarevocacion IS NULL
			AND sesion.idautenticacion = EXCLUDED.idautenticacion
			AND sesion.fechahoraultimaactividad > $3 - make_interval(secs => $4)
		RETURNING idsesion
`

func (r *sesionRepository) RegistrarActividad(
	idSesion, idAutenticacion string,
	ahora time.Time,
	inactividadMaxima time.Duration,
) (bool, error) {

	var id string

	err := r.db.QueryRow(
		consultaRegistrarActividadSesion,
		idSesion,
		idAutenticacion,
		ahora,
		inactividadMaxima.Seconds(),
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// Si la sesión nunca pasó por el backend se crea ya revocada, para que un
// token emitido antes del logout tampoco sirva después.
const consultaRevocarSesion = `
		INSERT INTO sesion (idsesion, idautenticacion, fechahoraultimaactividad, fechahorarevocacion)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (idsesion) DO UPDATE
		SET fechahorarevocacion = COALESCE(sesion.fechahorarevocacion, EXCLUDED.fechahorarevocacion)
		WHERE sesion.idautenticacion = EXCLUDED.idautenticacion
`

func (r *sesionRepository) Revocar(idSesion, idAutenticacion string, ahora time.Time) error {
	_, err := r.db.Exec(consultaRevocarSesion, idSesion, idAutenticacion, ahora)
	return err
}
