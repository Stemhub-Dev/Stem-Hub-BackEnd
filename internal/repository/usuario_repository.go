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
	) (*model.Usuario, bool, error)
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

func (r *usuarioRepository) Registrar(
	idAutenticacion string,
	email string,
) (*model.Usuario, bool, error) {

	query := `
		INSERT INTO usuario (idautenticacion, email)
		VALUES ($1, $2)
		ON CONFLICT (idautenticacion) DO NOTHING
			RETURNING
			codigousuario,
			idautenticacion,
			email,
			ultimologin,
			fechahorabajausuario
	`

	var usuario model.Usuario

	err := r.db.QueryRow(
		query,
		idAutenticacion,
		email,
	).Scan(
		&usuario.CodigoUsuario,
		&usuario.IDAutenticacion,
		&usuario.Email,
		&usuario.UltimoLogin,
		&usuario.FechaHoraBajaUsuario,
	)

	if err == nil {
		return &usuario, true, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	usuarioExiste, err := r.BuscarPorIDAutenticacion(idAutenticacion)
	if err != nil {
		return nil, false, err
	}

	return usuarioExiste, false, nil
}
