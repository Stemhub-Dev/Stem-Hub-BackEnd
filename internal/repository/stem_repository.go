package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

// ArchivoStemGuardado es el archivo ya subido a MinIO que se asocia a un
// stem: la key del objeto, el formato y el nombre original.
type ArchivoStemGuardado struct {
	URL            string
	Formato        string
	NombreOriginal string
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

	// Si archivo no es nil lo reemplaza y el stem pierde la marca de IA.
	Actualizar(
		codStem int64,
		nombre string,
		archivo *ArchivoStemGuardado,
	) error

	// Borrado real de la fila, junto con sus comentarios y las respuestas a
	// esos comentarios (HU-ABM-04-03 CA2).
	Eliminar(
		codStem int64,
	) error
}

type stemRepository struct {
	db *sql.DB
}

func NewStemRepository(db *sql.DB) StemRepository {
	return &stemRepository{
		db: db,
	}
}

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
		if err := tx.QueryRow(`
			INSERT INTO categoriastem (
				nombrecategoriastem,
				codigoproyecto
			)
			VALUES ($1, $2)
			RETURNING codcategoriastem
		`,
			*nuevaCategoria,
			codigoProyecto,
		).Scan(&codCategoriaStem); err != nil {
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

func (r *stemRepository) Actualizar(
	codStem int64,
	nombre string,
	archivo *ArchivoStemGuardado,
) error {

	if archivo == nil {
		_, err := r.db.Exec(`
			UPDATE stem
			SET nombrestem = $2
			WHERE codstem = $1
		`,
			codStem,
			nombre,
		)
		return err
	}

	_, err := r.db.Exec(`
		UPDATE stem
		SET nombrestem = $2,
			urlarchivostem = $3,
			formatoarchivostem = $4,
			nombrearchivostem = $5,
			generadoconia = false
		WHERE codstem = $1
	`,
		codStem,
		nombre,
		archivo.URL,
		archivo.Formato,
		archivo.NombreOriginal,
	)

	return err
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
