package repository

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type UsuarioRepository interface {
	BuscarPorIDAutenticacion(idAutenticacion string) (*model.Usuario, error)
	Registrar(
		idAutenticacion string,
		email string,
		nombre string,
		descripcion *string,
	) (*model.Usuario, *model.Integrante, bool, error)
}

type usuarioRepository struct {
	db *sql.DB
}

func NewUsuarioRepository(db *sql.DB) UsuarioRepository {
	return &usuarioRepository{
		db: db,
	}
}

func (r *usuarioRepository) BuscarPorIDAutenticacion(idAutenticacion string) (*model.Usuario, error) {
	query := `
		SELECT
			codigousuario,
			idautenticacion,
			email,
			ultimologin,
			fechahorabajausuario
		FROM usuario
		WHERE idautenticacion = $1
	`

	var usuario model.Usuario

	err := r.db.QueryRow(query, idAutenticacion).Scan(
		&usuario.CodigoUsuario,
		&usuario.IDAutenticacion,
		&usuario.Email,
		&usuario.UltimoLogin,
		&usuario.FechaHoraBajaUsuario,
	)

	if err != nil {
		return nil, err
	}

	return &usuario, nil
}

func (r *usuarioRepository) buscarIntegrantePorCodigoUsuario(
	codigoUsuario int64,
) (*model.Integrante, error) {

	query := `
		SELECT
			codintegrante,
			codigousuario,
			nombreintegrante,
			descripcionintegrante,
			fechahorabajaintegrante
		FROM integrante
		WHERE codigousuario = $1
	`

	var integrante model.Integrante

	err := r.db.QueryRow(query, codigoUsuario).Scan(
		&integrante.CodIntegrante,
		&integrante.CodigoUsuario,
		&integrante.NombreIntegrante,
		&integrante.DescripcionIntegrante,
		&integrante.FechaHoraBajaIntegrante,
	)

	if err != nil {
		return nil, err
	}

	return &integrante, nil
}

func (r *usuarioRepository) Registrar(
	idAutenticacion string,
	email string,
	nombre string,
	descripcion *string,
) (*model.Usuario, *model.Integrante, bool, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return nil, nil, false, err
	}
	defer tx.Rollback()

	var usuario model.Usuario

	err = tx.QueryRow(`
		INSERT INTO usuario (idautenticacion, email)
		VALUES ($1, $2)
		ON CONFLICT (idautenticacion) DO NOTHING
		RETURNING
			codigousuario,
			idautenticacion,
			email,
			ultimologin,
			fechahorabajausuario
	`,
		idAutenticacion,
		email,
	).Scan(
		&usuario.CodigoUsuario,
		&usuario.IDAutenticacion,
		&usuario.Email,
		&usuario.UltimoLogin,
		&usuario.FechaHoraBajaUsuario,
	)

	if errors.Is(err, sql.ErrNoRows) {
		usuarioExistente, errBusqueda := r.BuscarPorIDAutenticacion(idAutenticacion)
		if errBusqueda != nil {
			return nil, nil, false, errBusqueda
		}

		integranteExistente, errIntegrante := r.buscarIntegrantePorCodigoUsuario(usuarioExistente.CodigoUsuario)
		if errIntegrante != nil {
			return nil, nil, false, errIntegrante
		}

		return usuarioExistente, integranteExistente, false, nil
	}

	if err != nil {
		return nil, nil, false, err
	}

	var integrante model.Integrante

	err = tx.QueryRow(`
		INSERT INTO integrante (
			codigousuario,
			nombreintegrante,
			descripcionintegrante
		)
		VALUES ($1, $2, $3)
		RETURNING
			codintegrante,
			codigousuario,
			nombreintegrante,
			descripcionintegrante,
			fechahorabajaintegrante
	`,
		usuario.CodigoUsuario,
		nombre,
		descripcion,
	).Scan(
		&integrante.CodIntegrante,
		&integrante.CodigoUsuario,
		&integrante.NombreIntegrante,
		&integrante.DescripcionIntegrante,
		&integrante.FechaHoraBajaIntegrante,
	)

	if err != nil {
		return nil, nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, false, err
	}

	return &usuario, &integrante, true, nil
}
