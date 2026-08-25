package repository

import "database/sql"

type UsuarioRolRepository interface {
	ExisteUsuarioActivo(codigoUsuario int64) (bool, error)
	ExisteRolActivo(codRol int64) (bool, error)
	TieneRolActivo(codigoUsuario int64, codRol int64) (bool, error)
	AsignarRol(codigoUsuario int64, codRol int64) error
}

type usuarioRolRepository struct {
	db *sql.DB
}

func NewUsuarioRolRepository(db *sql.DB) UsuarioRolRepository {
	return &usuarioRolRepository{
		db: db,
	}
}

func (r *usuarioRolRepository) ExisteUsuarioActivo(
	codigoUsuario int64,
) (bool, error) {

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM usuario
			WHERE codigousuario = $1
			  AND fechahorabajausuario IS NULL
		)
	`

	var existe bool

	err := r.db.QueryRow(
		query,
		codigoUsuario,
	).Scan(&existe)

	return existe, err
}

func (r *usuarioRolRepository) ExisteRolActivo(
	codRol int64,
) (bool, error) {

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM rol
			WHERE codrol = $1
			  AND fechahorabajarol IS NULL
		)
	`

	var existe bool

	err := r.db.QueryRow(
		query,
		codRol,
	).Scan(&existe)

	return existe, err
}

func (r *usuarioRolRepository) TieneRolActivo(
	codigoUsuario int64,
	codRol int64,
) (bool, error) {

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM usuariorol
			WHERE codigousuario = $1
			  AND codrol = $2
			  AND fechahorabajausuariorol IS NULL
		)
	`

	var existe bool

	err := r.db.QueryRow(
		query,
		codigoUsuario,
		codRol,
	).Scan(&existe)

	return existe, err
}

func (r *usuarioRolRepository) AsignarRol(
	codigoUsuario int64,
	codRol int64,
) error {

	query := `
		INSERT INTO usuariorol (
			codigousuario,
			codrol,
			ambitorol,
			fechahoraaltausuariorol
		)
		SELECT
			$1,
			r.codrol,
			r.ambitorol,
			CURRENT_TIMESTAMP
		FROM rol r
		WHERE r.codrol = $2
		  AND r.fechahorabajarol IS NULL
	`

	_, err := r.db.Exec(
		query,
		codigoUsuario,
		codRol,
	)

	return err
}
