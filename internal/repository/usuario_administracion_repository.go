package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type UsuarioAdministracionRepository interface {
	Listar() ([]dto.UsuarioAdministracionResponse, error)
	ObtenerPorID(codigoUsuario int64) (*dto.UsuarioAdministracionResponse, error)
	ObtenerRolesSistema(codigoUsuario int64) (*dto.UsuarioRolesResponse, error)
	ObtenerProyectosPorUsuario(codigoUsuario int64) (*dto.UsuarioProyectosResponse, error)
	ActualizarAdministracion(codigoUsuario int64, esAdministradorSistema bool, activo bool) (*dto.ActualizarAdministracionUsuarioResponse, error)
	CambiarEstado(codigoUsuario int64, activo bool) (*dto.CambiarEstadoUsuarioResponse, error)
}

type usuarioAdministracionRepository struct {
	db *sql.DB
}

func NewUsuarioAdministracionRepository(
	db *sql.DB,
) UsuarioAdministracionRepository {
	return &usuarioAdministracionRepository{
		db: db,
	}
}

func (r *usuarioAdministracionRepository) Listar() (
	[]dto.UsuarioAdministracionResponse,
	error,
) {
	const query = `
		SELECT
			u.codigousuario,
			i.codintegrante,
			u.email,
			i.nombreintegrante,
			i.descripcionintegrante,
			u.ultimologin,
			(u.fechahorabajausuario IS NULL) AS activo,
			r.codrol,
			r.nombrerol
		FROM usuario u
		INNER JOIN integrante i
			ON i.codigousuario = u.codigousuario
		LEFT JOIN usuariorol ur
			ON ur.codigousuario = u.codigousuario
			AND ur.ambitorol = 'SISTEMA'
			AND ur.fechahorabajausuariorol IS NULL
		LEFT JOIN rol r
			ON r.codrol = ur.codrol
			AND r.ambitorol = 'SISTEMA'
			AND r.fechahorabajarol IS NULL
		ORDER BY
			i.nombreintegrante,
			u.codigousuario,
			r.nombrerol
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usuarios := make([]dto.UsuarioAdministracionResponse, 0)

	var usuarioActual *dto.UsuarioAdministracionResponse

	for rows.Next() {
		var (
			codigoUsuario         int64
			codIntegrante         int64
			email                 string
			nombreIntegrante      string
			descripcionIntegrante *string
			ultimoLogin           sql.NullTime
			activo                bool
			codRol                sql.NullInt64
			nombreRol             sql.NullString
		)

		err = rows.Scan(
			&codigoUsuario,
			&codIntegrante,
			&email,
			&nombreIntegrante,
			&descripcionIntegrante,
			&ultimoLogin,
			&activo,
			&codRol,
			&nombreRol,
		)
		if err != nil {
			return nil, err
		}

		if usuarioActual == nil ||
			usuarioActual.CodigoUsuario != codigoUsuario {

			usuario := dto.UsuarioAdministracionResponse{
				CodigoUsuario:         codigoUsuario,
				CodIntegrante:         codIntegrante,
				Email:                 email,
				NombreIntegrante:      nombreIntegrante,
				DescripcionIntegrante: descripcionIntegrante,
				Activo:                activo,
				RolesSistema:          make([]dto.RolSistemaUsuarioResponse, 0),
			}

			if ultimoLogin.Valid {
				usuario.UltimoLogin = &ultimoLogin.Time
			}

			usuarios = append(usuarios, usuario)
			usuarioActual = &usuarios[len(usuarios)-1]
		}

		if codRol.Valid && nombreRol.Valid {
			usuarioActual.RolesSistema = append(
				usuarioActual.RolesSistema,
				dto.RolSistemaUsuarioResponse{
					CodRol:    codRol.Int64,
					NombreRol: nombreRol.String,
				},
			)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return usuarios, nil
}

func (r *usuarioAdministracionRepository) ObtenerPorID(
	codigoUsuario int64,
) (*dto.UsuarioAdministracionResponse, error) {
	const query = `
		SELECT
			u.codigousuario,
			i.codintegrante,
			u.email,
			i.nombreintegrante,
			i.descripcionintegrante,
			u.ultimologin,
			(u.fechahorabajausuario IS NULL) AS activo,
			rol.codrol,
			rol.nombrerol
		FROM usuario u
		INNER JOIN integrante i
			ON i.codigousuario = u.codigousuario
		LEFT JOIN usuariorol ur
			ON ur.codigousuario = u.codigousuario
			AND ur.ambitorol = 'SISTEMA'
			AND ur.fechahorabajausuariorol IS NULL
		LEFT JOIN rol
			ON rol.codrol = ur.codrol
			AND rol.ambitorol = 'SISTEMA'
			AND rol.fechahorabajarol IS NULL
		WHERE u.codigousuario = $1
		ORDER BY rol.nombrerol
	`

	rows, err := r.db.Query(query, codigoUsuario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usuario *dto.UsuarioAdministracionResponse

	for rows.Next() {
		var (
			codigoUsuarioDB       int64
			codIntegrante         int64
			email                 string
			nombreIntegrante      string
			descripcionIntegrante *string
			ultimoLogin           sql.NullTime
			activo                bool
			codRol                sql.NullInt64
			nombreRol             sql.NullString
		)

		err = rows.Scan(
			&codigoUsuarioDB,
			&codIntegrante,
			&email,
			&nombreIntegrante,
			&descripcionIntegrante,
			&ultimoLogin,
			&activo,
			&codRol,
			&nombreRol,
		)
		if err != nil {
			return nil, err
		}

		if usuario == nil {
			usuario = &dto.UsuarioAdministracionResponse{
				CodigoUsuario:         codigoUsuarioDB,
				CodIntegrante:         codIntegrante,
				Email:                 email,
				NombreIntegrante:      nombreIntegrante,
				DescripcionIntegrante: descripcionIntegrante,
				Activo:                activo,
				RolesSistema:          make([]dto.RolSistemaUsuarioResponse, 0),
			}

			if ultimoLogin.Valid {
				ultimoLoginTime := ultimoLogin.Time
				usuario.UltimoLogin = &ultimoLoginTime
			}
		}

		if codRol.Valid && nombreRol.Valid {
			usuario.RolesSistema = append(
				usuario.RolesSistema,
				dto.RolSistemaUsuarioResponse{
					CodRol:    codRol.Int64,
					NombreRol: nombreRol.String,
				},
			)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if usuario == nil {
		return nil, sql.ErrNoRows
	}

	return usuario, nil
}

func (r *usuarioAdministracionRepository) ObtenerRolesSistema(
	codigoUsuario int64,
) (*dto.UsuarioRolesResponse, error) {
	const queryUsuario = `
		SELECT EXISTS (
			SELECT 1
			FROM usuario
			WHERE codigousuario = $1
		)
	`

	var existe bool

	err := r.db.QueryRow(
		queryUsuario,
		codigoUsuario,
	).Scan(&existe)
	if err != nil {
		return nil, err
	}

	if !existe {
		return nil, sql.ErrNoRows
	}

	const queryRoles = `
		SELECT
			r.codrol,
			r.nombrerol,
			r.descripcionrol,
			EXISTS (
				SELECT 1
				FROM usuariorol ur
				WHERE ur.codigousuario = $1
				  AND ur.codrol = r.codrol
				  AND ur.ambitorol = 'SISTEMA'
				  AND ur.fechahorabajausuariorol IS NULL
			) AS asignado
		FROM rol r
		WHERE r.ambitorol = 'SISTEMA'
		  AND r.fechahorabajarol IS NULL
		ORDER BY r.nombrerol, r.codrol
	`

	rows, err := r.db.Query(
		queryRoles,
		codigoUsuario,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	response := &dto.UsuarioRolesResponse{
		CodigoUsuario: codigoUsuario,
		Roles:         make([]dto.RolUsuarioAdministracionResponse, 0),
	}

	for rows.Next() {
		var rol dto.RolUsuarioAdministracionResponse

		err = rows.Scan(
			&rol.CodRol,
			&rol.NombreRol,
			&rol.DescripcionRol,
			&rol.Asignado,
		)
		if err != nil {
			return nil, err
		}

		response.Roles = append(
			response.Roles,
			rol,
		)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *usuarioAdministracionRepository) ObtenerProyectosPorUsuario(
	codigoUsuario int64,
) (*dto.UsuarioProyectosResponse, error) {
	const queryUsuario = `
		SELECT EXISTS (
			SELECT 1
			FROM usuario
			WHERE codigousuario = $1
		)
	`

	var existe bool

	err := r.db.QueryRow(
		queryUsuario,
		codigoUsuario,
	).Scan(&existe)
	if err != nil {
		return nil, err
	}

	if !existe {
		return nil, sql.ErrNoRows
	}

	const queryProyectos = `
		SELECT
			p.codigoproyecto,
			p.nombreproyecto,
			r.codrol,
			r.nombrerol,
			ip.espropietario
		FROM integrante i
		INNER JOIN integranteproyecto ip
			ON ip.codintegrante = i.codintegrante
			AND ip.fechahorabajaintegranteproy IS NULL
		INNER JOIN proyecto p
			ON p.codigoproyecto = ip.codigoproyecto
			AND p.fechahorabajaproyecto IS NULL
		INNER JOIN rol r
			ON r.codrol = ip.codrol
			AND r.ambitorol = ip.ambitorol
			AND r.ambitorol = 'PROYECTO'
			AND r.fechahorabajarol IS NULL
		WHERE i.codigousuario = $1
		  AND i.fechahorabajaintegrante IS NULL
		ORDER BY
			p.nombreproyecto,
			p.codigoproyecto
	`

	rows, err := r.db.Query(
		queryProyectos,
		codigoUsuario,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	response := &dto.UsuarioProyectosResponse{
		CodigoUsuario: codigoUsuario,
		Proyectos:     make([]dto.ProyectoUsuarioResponse, 0),
	}

	for rows.Next() {
		var proyecto dto.ProyectoUsuarioResponse

		err = rows.Scan(
			&proyecto.CodigoProyecto,
			&proyecto.NombreProyecto,
			&proyecto.CodRol,
			&proyecto.NombreRol,
			&proyecto.EsPropietario,
		)
		if err != nil {
			return nil, err
		}

		response.Proyectos = append(
			response.Proyectos,
			proyecto,
		)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *usuarioAdministracionRepository) ActualizarAdministracion(
	codigoUsuario int64,
	esAdministradorSistema bool,
	activo bool,
) (*dto.ActualizarAdministracionUsuarioResponse, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Actualizar estado activo/inactivo del usuario.
	result, err := tx.Exec(`
		UPDATE usuario
		SET fechahorabajausuario =
			CASE
				WHEN $2 THEN NULL
				ELSE COALESCE(
					fechahorabajausuario,
					CURRENT_TIMESTAMP
				)
			END
		WHERE codigousuario = $1
	`,
		codigoUsuario,
		activo,
	)
	if err != nil {
		return nil, err
	}

	filas, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if filas == 0 {
		return nil, sql.ErrNoRows
	}

	// 2. Administrar únicamente el rol global
	//    "Administrador del sistema".
	if esAdministradorSistema {

		// Si ya lo tiene activo, no vuelve a insertarlo.
		_, err = tx.Exec(`
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
			WHERE LOWER(r.nombrerol) =
				  LOWER($2)
			  AND r.ambitorol = 'SISTEMA'
			  AND r.fechahorabajarol IS NULL
			  AND NOT EXISTS (
					SELECT 1
					FROM usuariorol ur
					WHERE ur.codigousuario = $1
					  AND ur.codrol = r.codrol
					  AND ur.ambitorol = 'SISTEMA'
					  AND ur.fechahorabajausuariorol IS NULL
			  )
		`,
			codigoUsuario,
			model.NombreRolAdministradorSistema,
		)
		if err != nil {
			return nil, err
		}

	} else {

		// Quitar el rol significa darle fecha de baja.
		// No eliminamos físicamente el registro.
		_, err = tx.Exec(`
			UPDATE usuariorol ur
			SET fechahorabajausuariorol = CURRENT_TIMESTAMP
			FROM rol r
			WHERE ur.codigousuario = $1
			  AND ur.codrol = r.codrol
			  AND ur.ambitorol = r.ambitorol
			  AND ur.ambitorol = 'SISTEMA'
			  AND ur.fechahorabajausuariorol IS NULL
			  AND LOWER(r.nombrerol) =
				  LOWER($2)
			  AND r.fechahorabajarol IS NULL
		`,
			codigoUsuario,
			model.NombreRolAdministradorSistema,
		)
		if err != nil {
			return nil, err
		}
	}

	// 3. Si las dos operaciones salieron bien,
	//    recién ahí confirmamos la transacción.
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &dto.ActualizarAdministracionUsuarioResponse{
		CodigoUsuario:          codigoUsuario,
		EsAdministradorSistema: esAdministradorSistema,
		Activo:                 activo,
	}, nil
}

func (r *usuarioAdministracionRepository) CambiarEstado(
	codigoUsuario int64,
	activo bool,
) (*dto.CambiarEstadoUsuarioResponse, error) {

	result, err := r.db.Exec(`
		UPDATE usuario
		SET fechahorabajausuario =
			CASE
				WHEN $2 THEN NULL
				ELSE COALESCE(
					fechahorabajausuario,
					CURRENT_TIMESTAMP
				)
			END
		WHERE codigousuario = $1
	`,
		codigoUsuario,
		activo,
	)
	if err != nil {
		return nil, err
	}

	filas, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if filas == 0 {
		return nil, sql.ErrNoRows
	}

	return &dto.CambiarEstadoUsuarioResponse{
		CodigoUsuario: codigoUsuario,
		Activo:        activo,
	}, nil
}
