package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

// ArchivoStemGuardado es el archivo ya subido a MinIO que se asocia a un
// stem: la key del objeto, el formato y el nombre original.
type ArchivoStemGuardado struct {
	URL            string
	Formato        string
	NombreOriginal string
}

// CambiosStem es el estado final de un stem editado, ya resuelto por el
// service: lo que no cambia viaja con su valor actual.
type CambiosStem struct {
	Nombre           string
	NombreArchivo    string
	CodCategoriaStem int64
	// Si no es nil se crea la categoría en el proyecto y CodCategoriaStem se
	// ignora, igual que en Crear.
	NuevaCategoria *string
	// nil = se conserva el archivo actual.
	Archivo *ArchivoStemGuardado
}

type StemRepository interface {
	// Categorías por defecto + las del proyecto, activas. Primero las por
	// defecto (en su orden de alta), después las del proyecto.
	ListarCategorias(
		codigoProyecto int64,
	) ([]model.CategoriaStem, error)

	// Busca una categoría activa utilizable en el proyecto: por defecto o
	// propia del proyecto. sql.ErrNoRows si no existe.
	BuscarCategoria(
		codigoProyecto int64,
		codCategoriaStem int64,
	) (*model.CategoriaStem, error)

	// Compara sin distinguir mayúsculas ni espacios de los extremos, contra
	// las por defecto y las del proyecto.
	ExisteCategoriaConNombre(
		codigoProyecto int64,
		nombre string,
	) (bool, error)

	ListarPorVersion(
		codigoCancionVersion int64,
	) ([]model.Stem, error)

	// sql.ErrNoRows si no existe o no pertenece a la versión.
	BuscarPorCodigo(
		codigoCancionVersion int64,
		codStem int64,
	) (*model.Stem, error)

	// codStemExcluido permite editar un stem sin chocar con su propio
	// nombre; 0 para no excluir ninguno.
	ExisteNombreEnVersion(
		codigoCancionVersion int64,
		nombre string,
		codStemExcluido int64,
	) (bool, error)

	// Crea el stem y, si nuevaCategoria no es nil, la categoría del proyecto
	// en la misma transacción (codCategoriaStem se ignora en ese caso).
	Crear(
		codigoProyecto int64,
		codigoCancionVersion int64,
		nombre string,
		codCategoriaStem int64,
		nuevaCategoria *string,
		archivo ArchivoStemGuardado,
	) (int64, error)

	// Si cambios.Archivo no es nil lo reemplaza y el stem pierde la marca de
	// IA. La versión (y con ella la canción) del stem no cambia nunca.
	Actualizar(
		codigoProyecto int64,
		codStem int64,
		cambios CambiosStem,
	) error

	// Borrado real de la fila, junto con sus comentarios y las respuestas a
	// esos comentarios (HU-ABM-04-03 CA2).
	Eliminar(
		codStem int64,
	) error

	ContarGlobalPorProductor(
		codigoIntegrante int64,
		filtro dto.ListarStemsGlobalFiltro,
	) (int, error)

	ListarGlobalPorProductor(
		codigoIntegrante int64,
		filtro dto.ListarStemsGlobalFiltro,
		desplazamiento int,
	) ([]dto.StemGlobalListadoResponse, error)
}

type stemRepository struct {
	db *sql.DB
}

func NewStemRepository(db *sql.DB) StemRepository {
	return &stemRepository{
		db: db,
	}
}

const consultaContarStemsGlobales = `
	SELECT COUNT(*)
	FROM integranteproyecto ip
	INNER JOIN rol r
		ON r.codrol = ip.codrol
	   AND r.ambitorol = ip.ambitorol
	INNER JOIN proyecto p
		ON p.codigoproyecto = ip.codigoproyecto
	INNER JOIN cancion c
		ON c.codigoproyecto = p.codigoproyecto
	INNER JOIN cancionversion cv
		ON cv.codigocancion = c.codigocancion
	INNER JOIN stem s
		ON s.codigocancionversion = cv.codigocancionversion
	INNER JOIN categoriastem cs
		ON cs.codcategoriastem = s.codcategoriastem
	WHERE ip.codintegrante = $1
	  AND ip.fechahorabajaintegranteproy IS NULL
	  AND ip.ambitorol = 'PROYECTO'
	  AND r.fechahorabajarol IS NULL
	  AND LOWER(r.nombrerol) = LOWER($2)
	  AND p.fechahorabajaproyecto IS NULL
	  AND c.fechahorabajacancion IS NULL
	  AND cv.fechahorabajaversion IS NULL
	  AND s.fechahorabajastem IS NULL
	  AND cs.fechahorabajacategoriastem IS NULL
	  AND (
			$3::text = ''
			OR s.nombrestem ILIKE '%' || $3::text || '%'
	  )
	  AND (
			$4::bigint IS NULL
			OR s.codcategoriastem = $4::bigint
	  )
`

const consultaListarStemsGlobales = `
	SELECT
		s.codstem,
		s.nombrestem,
		s.codcategoriastem,
		cs.nombrecategoriastem,
		s.generadoconia,
		s.nombrearchivostem,
		s.formatoarchivostem,

		p.codigoproyecto,
		p.nombreproyecto,

		c.codigocancion,
		c.nombrecancion,

		cv.codigocancionversion,
		cv.numeroversion

	FROM integranteproyecto ip
	INNER JOIN rol r
		ON r.codrol = ip.codrol
	   AND r.ambitorol = ip.ambitorol
	INNER JOIN proyecto p
		ON p.codigoproyecto = ip.codigoproyecto
	INNER JOIN cancion c
		ON c.codigoproyecto = p.codigoproyecto
	INNER JOIN cancionversion cv
		ON cv.codigocancion = c.codigocancion
	INNER JOIN stem s
		ON s.codigocancionversion = cv.codigocancionversion
	INNER JOIN categoriastem cs
		ON cs.codcategoriastem = s.codcategoriastem
	WHERE ip.codintegrante = $1
	  AND ip.fechahorabajaintegranteproy IS NULL
	  AND ip.ambitorol = 'PROYECTO'
	  AND r.fechahorabajarol IS NULL
	  AND LOWER(r.nombrerol) = LOWER($2)
	  AND p.fechahorabajaproyecto IS NULL
	  AND c.fechahorabajacancion IS NULL
	  AND cv.fechahorabajaversion IS NULL
	  AND s.fechahorabajastem IS NULL
	  AND cs.fechahorabajacategoriastem IS NULL
	  AND (
			$3::text = ''
			OR s.nombrestem ILIKE '%' || $3::text || '%'
	  )
	  AND (
			$4::bigint IS NULL
			OR s.codcategoriastem = $4::bigint
	  )
	ORDER BY s.codstem DESC
	LIMIT $5 OFFSET $6
`

func (r *stemRepository) ListarCategorias(
	codigoProyecto int64,
) ([]model.CategoriaStem, error) {

	rows, err := r.db.Query(`
		SELECT
			codcategoriastem,
			nombrecategoriastem,
			codigoproyecto
		FROM categoriastem
		WHERE (codigoproyecto IS NULL OR codigoproyecto = $1)
		  AND fechahorabajacategoriastem IS NULL
		ORDER BY
			codigoproyecto IS NOT NULL,
			fechahoraaltacategoriastem,
			codcategoriastem
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	categorias := []model.CategoriaStem{}

	for rows.Next() {

		var categoria model.CategoriaStem
		var proyecto sql.NullInt64

		if err := rows.Scan(
			&categoria.CodCategoriaStem,
			&categoria.NombreCategoriaStem,
			&proyecto,
		); err != nil {
			return nil, err
		}

		if proyecto.Valid {
			categoria.CodigoProyecto = &proyecto.Int64
		}

		categorias = append(categorias, categoria)
	}

	return categorias, rows.Err()
}

func (r *stemRepository) BuscarCategoria(
	codigoProyecto int64,
	codCategoriaStem int64,
) (*model.CategoriaStem, error) {

	var categoria model.CategoriaStem
	var proyecto sql.NullInt64

	err := r.db.QueryRow(`
		SELECT
			codcategoriastem,
			nombrecategoriastem,
			codigoproyecto
		FROM categoriastem
		WHERE codcategoriastem = $1
		  AND (codigoproyecto IS NULL OR codigoproyecto = $2)
		  AND fechahorabajacategoriastem IS NULL
	`,
		codCategoriaStem,
		codigoProyecto,
	).Scan(
		&categoria.CodCategoriaStem,
		&categoria.NombreCategoriaStem,
		&proyecto,
	)

	if err != nil {
		return nil, err
	}

	if proyecto.Valid {
		categoria.CodigoProyecto = &proyecto.Int64
	}

	return &categoria, nil
}

func (r *stemRepository) ExisteCategoriaConNombre(
	codigoProyecto int64,
	nombre string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM categoriastem
			WHERE (codigoproyecto IS NULL OR codigoproyecto = $1)
			  AND fechahorabajacategoriastem IS NULL
			  AND lower(btrim(nombrecategoriastem)) = lower(btrim($2))
		)
	`,
		codigoProyecto,
		nombre,
	).Scan(&existe)

	return existe, err
}

const columnasStem = `
	s.codstem,
	s.codigocancionversion,
	s.nombrestem,
	s.codcategoriastem,
	c.nombrecategoriastem,
	s.urlarchivostem,
	s.formatoarchivostem,
	s.nombrearchivostem,
	s.generadoconia
`

type escaneable interface {
	Scan(dest ...any) error
}

func escanearStem(fila escaneable) (*model.Stem, error) {

	var stem model.Stem

	err := fila.Scan(
		&stem.CodStem,
		&stem.CodigoCancionVersion,
		&stem.NombreStem,
		&stem.CodCategoriaStem,
		&stem.NombreCategoriaStem,
		&stem.URLArchivoStem,
		&stem.FormatoArchivoStem,
		&stem.NombreArchivoStem,
		&stem.GeneradoConIA,
	)

	if err != nil {
		return nil, err
	}

	return &stem, nil
}

func (r *stemRepository) ListarPorVersion(
	codigoCancionVersion int64,
) ([]model.Stem, error) {

	rows, err := r.db.Query(`
		SELECT `+columnasStem+`
		FROM stem s
		JOIN categoriastem c
		  ON c.codcategoriastem = s.codcategoriastem
		WHERE s.codigocancionversion = $1
		  AND s.fechahorabajastem IS NULL
		ORDER BY s.codstem
	`,
		codigoCancionVersion,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	stems := []model.Stem{}

	for rows.Next() {

		stem, err := escanearStem(rows)

		if err != nil {
			return nil, err
		}

		stems = append(stems, *stem)
	}

	return stems, rows.Err()
}

func (r *stemRepository) BuscarPorCodigo(
	codigoCancionVersion int64,
	codStem int64,
) (*model.Stem, error) {

	return escanearStem(r.db.QueryRow(`
		SELECT `+columnasStem+`
		FROM stem s
		JOIN categoriastem c
		  ON c.codcategoriastem = s.codcategoriastem
		WHERE s.codstem = $1
		  AND s.codigocancionversion = $2
		  AND s.fechahorabajastem IS NULL
	`,
		codStem,
		codigoCancionVersion,
	))
}

func (r *stemRepository) ExisteNombreEnVersion(
	codigoCancionVersion int64,
	nombre string,
	codStemExcluido int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM stem
			WHERE codigocancionversion = $1
			  AND fechahorabajastem IS NULL
			  AND lower(btrim(nombrestem)) = lower(btrim($2))
			  AND codstem <> $3
		)
	`,
		codigoCancionVersion,
		nombre,
		codStemExcluido,
	).Scan(&existe)

	return existe, err
}

func (r *stemRepository) Crear(
	codigoProyecto int64,
	codigoCancionVersion int64,
	nombre string,
	codCategoriaStem int64,
	nuevaCategoria *string,
	archivo ArchivoStemGuardado,
) (int64, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return 0, err
	}

	defer tx.Rollback()

	if nuevaCategoria != nil {
		if codCategoriaStem, err = insertarCategoriaProyecto(tx, codigoProyecto, *nuevaCategoria); err != nil {
			return 0, err
		}
	}

	var codStem int64

	if err := tx.QueryRow(`
		INSERT INTO stem (
			codigocancionversion,
			nombrestem,
			codcategoriastem,
			urlarchivostem,
			formatoarchivostem,
			nombrearchivostem
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING codstem
	`,
		codigoCancionVersion,
		nombre,
		codCategoriaStem,
		archivo.URL,
		archivo.Formato,
		archivo.NombreOriginal,
	).Scan(&codStem); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return codStem, nil
}

func insertarCategoriaProyecto(
	tx *sql.Tx,
	codigoProyecto int64,
	nombre string,
) (int64, error) {

	var codCategoriaStem int64

	err := tx.QueryRow(`
		INSERT INTO categoriastem (
			nombrecategoriastem,
			codigoproyecto
		)
		VALUES ($1, $2)
		RETURNING codcategoriastem
	`,
		nombre,
		codigoProyecto,
	).Scan(&codCategoriaStem)

	return codCategoriaStem, err
}

func (r *stemRepository) Actualizar(
	codigoProyecto int64,
	codStem int64,
	cambios CambiosStem,
) error {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	codCategoriaStem := cambios.CodCategoriaStem

	if cambios.NuevaCategoria != nil {
		if codCategoriaStem, err = insertarCategoriaProyecto(tx, codigoProyecto, *cambios.NuevaCategoria); err != nil {
			return err
		}
	}

	if cambios.Archivo == nil {
		_, err = tx.Exec(`
			UPDATE stem
			SET nombrestem = $2,
				nombrearchivostem = $3,
				codcategoriastem = $4
			WHERE codstem = $1
		`,
			codStem,
			cambios.Nombre,
			cambios.NombreArchivo,
			codCategoriaStem,
		)
	} else {
		_, err = tx.Exec(`
			UPDATE stem
			SET nombrestem = $2,
				nombrearchivostem = $3,
				codcategoriastem = $4,
				urlarchivostem = $5,
				formatoarchivostem = $6,
				generadoconia = false
			WHERE codstem = $1
		`,
			codStem,
			cambios.Nombre,
			cambios.NombreArchivo,
			codCategoriaStem,
			cambios.Archivo.URL,
			cambios.Archivo.Formato,
		)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *stemRepository) Eliminar(
	codStem int64,
) error {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	// Las FK son RESTRICT: primero las respuestas, después los comentarios
	// y recién ahí el stem. Incluye los comentarios ya dados de baja, que
	// también referencian al stem.
	sentencias := []string{
		`DELETE FROM comentariorespuesta
		 WHERE codigocomentario IN (
			SELECT codigocomentario FROM comentario WHERE codstem = $1
		 )`,
		`DELETE FROM comentario WHERE codstem = $1`,
		`DELETE FROM stem WHERE codstem = $1`,
	}

	for _, sentencia := range sentencias {
		if _, err := tx.Exec(sentencia, codStem); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *stemRepository) ContarGlobalPorProductor(
	codigoIntegrante int64,
	filtro dto.ListarStemsGlobalFiltro,
) (int, error) {

	var categoria any

	if filtro.Categoria != nil {
		categoria = *filtro.Categoria
	}

	var total int

	err := r.db.QueryRow(
		consultaContarStemsGlobales,
		codigoIntegrante,
		model.NombreRolProductor,
		escaparPatronLike(
			filtro.Busqueda,
		),
		categoria,
	).Scan(&total)

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *stemRepository) ListarGlobalPorProductor(
	codigoIntegrante int64,
	filtro dto.ListarStemsGlobalFiltro,
	desplazamiento int,
) ([]dto.StemGlobalListadoResponse, error) {

	var categoria any

	if filtro.Categoria != nil {
		categoria = *filtro.Categoria
	}

	rows, err := r.db.Query(
		consultaListarStemsGlobales,
		codigoIntegrante,
		model.NombreRolProductor,
		escaparPatronLike(
			filtro.Busqueda,
		),
		categoria,
		filtro.TamanoPagina,
		desplazamiento,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	stems :=
		make(
			[]dto.StemGlobalListadoResponse,
			0,
		)

	for rows.Next() {

		var stem dto.StemGlobalListadoResponse

		err := rows.Scan(
			&stem.CodStem,
			&stem.Nombre,
			&stem.CodCategoriaStem,
			&stem.NombreCategoria,
			&stem.GeneradoConIA,
			&stem.NombreArchivo,
			&stem.FormatoArchivo,
			&stem.CodigoProyecto,
			&stem.NombreProyecto,
			&stem.CodigoCancion,
			&stem.NombreCancion,
			&stem.CodigoCancionVersion,
			&stem.NumeroVersion,
		)

		if err != nil {
			return nil, err
		}

		stems = append(
			stems,
			stem,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stems, nil
}
