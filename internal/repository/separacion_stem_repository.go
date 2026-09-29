package repository

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

// ErrSeparacionEnCurso: la versión ya tiene una separación PENDIENTE o
// PROCESANDO (índice uqseparacionstemencurso).
var ErrSeparacionEnCurso = errors.New("la versión ya tiene una separación en curso")

const codigoViolacionUnica = "23505"

// StemGenerado es un stem que devolvió el microservicio de IA, ya subido a
// MinIO, listo para insertarse con generadoconia = true.
type StemGenerado struct {
	Nombre           string
	CodCategoriaStem int64
	Archivo          ArchivoStemGuardado
}

type SeparacionStemRepository interface {
	Crear(
		codigoCancionVersion int64,
		codIntegrante int64,
		cantidadStems int,
	) (*model.SeparacionStem, error)

	MarcarProcesando(
		codSeparacionStem int64,
	) error

	// Inserta los stems y pasa la separación a COMPLETADA en una misma
	// transacción.
	Completar(
		codSeparacionStem int64,
		codigoCancionVersion int64,
		stems []StemGenerado,
		tiempoProcesamientoMs int64,
	) error

	Fallar(
		codSeparacionStem int64,
		mensaje string,
	) error

	// La más reciente de la versión. sql.ErrNoRows si nunca se separó.
	BuscarUltimaPorVersion(
		codigoCancionVersion int64,
	) (*model.SeparacionStem, error)

	// Pasa a ERROR las que quedaron PENDIENTE o PROCESANDO: al reiniciar el
	// backend la goroutine que las procesaba ya no existe.
	FallarInterrumpidas(
		mensaje string,
	) (int64, error)

	// Nombre de la categoría por defecto → código, para asignarle una
	// categoría a cada stem generado.
	CategoriasPorDefecto() (map[string]int64, error)

	ExisteStemGeneradoConIA(
		codigoCancionVersion int64,
	) (bool, error)
}

type separacionStemRepository struct {
	db *sql.DB
}

func NewSeparacionStemRepository(db *sql.DB) SeparacionStemRepository {
	return &separacionStemRepository{
		db: db,
	}
}

const columnasSeparacionStem = `
	codseparacionstem,
	codigocancionversion,
	codintegrante,
	cantidadstems,
	estadoseparacion,
	mensajeerror,
	tiempoprocesamientoms,
	fechahorasolicitud,
	fechahorafin
`

func escanearSeparacionStem(fila escaneable) (*model.SeparacionStem, error) {

	var separacion model.SeparacionStem
	var mensaje sql.NullString
	var tiempo sql.NullInt64
	var fin sql.NullTime

	if err := fila.Scan(
		&separacion.CodSeparacionStem,
		&separacion.CodigoCancionVersion,
		&separacion.CodIntegrante,
		&separacion.CantidadStems,
		&separacion.EstadoSeparacion,
		&mensaje,
		&tiempo,
		&separacion.FechaHoraSolicitud,
		&fin,
	); err != nil {
		return nil, err
	}

	if mensaje.Valid {
		separacion.MensajeError = &mensaje.String
	}

	if tiempo.Valid {
		separacion.TiempoProcesamientoMs = &tiempo.Int64
	}

	if fin.Valid {
		separacion.FechaHoraFin = &fin.Time
	}

	return &separacion, nil
}

func (r *separacionStemRepository) Crear(
	codigoCancionVersion int64,
	codIntegrante int64,
	cantidadStems int,
) (*model.SeparacionStem, error) {

	separacion, err := escanearSeparacionStem(r.db.QueryRow(`
		INSERT INTO separacionstem (
			codigocancionversion,
			codintegrante,
			cantidadstems
		)
		VALUES ($1, $2, $3)
		RETURNING `+columnasSeparacionStem,
		codigoCancionVersion,
		codIntegrante,
		cantidadStems,
	))

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == codigoViolacionUnica {
		return nil, ErrSeparacionEnCurso
	}

	return separacion, err
}

func (r *separacionStemRepository) MarcarProcesando(
	codSeparacionStem int64,
) error {

	_, err := r.db.Exec(`
		UPDATE separacionstem
		SET estadoseparacion = 'PROCESANDO'
		WHERE codseparacionstem = $1
	`,
		codSeparacionStem,
	)

	return err
}

func (r *separacionStemRepository) Completar(
	codSeparacionStem int64,
	codigoCancionVersion int64,
	stems []StemGenerado,
	tiempoProcesamientoMs int64,
) error {

	tx, err := r.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	for _, stem := range stems {
		if _, err := tx.Exec(`
			INSERT INTO stem (
				codigocancionversion,
				nombrestem,
				codcategoriastem,
				urlarchivostem,
				formatoarchivostem,
				nombrearchivostem,
				generadoconia
			)
			VALUES ($1, $2, $3, $4, $5, $6, true)
		`,
			codigoCancionVersion,
			stem.Nombre,
			stem.CodCategoriaStem,
			stem.Archivo.URL,
			stem.Archivo.Formato,
			stem.Archivo.NombreOriginal,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
		UPDATE separacionstem
		SET estadoseparacion = 'COMPLETADA',
			tiempoprocesamientoms = $2,
			fechahorafin = CURRENT_TIMESTAMP
		WHERE codseparacionstem = $1
	`,
		codSeparacionStem,
		tiempoProcesamientoMs,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *separacionStemRepository) Fallar(
	codSeparacionStem int64,
	mensaje string,
) error {

	_, err := r.db.Exec(`
		UPDATE separacionstem
		SET estadoseparacion = 'ERROR',
			mensajeerror = $2,
			fechahorafin = CURRENT_TIMESTAMP
		WHERE codseparacionstem = $1
	`,
		codSeparacionStem,
		mensaje,
	)

	return err
}

func (r *separacionStemRepository) BuscarUltimaPorVersion(
	codigoCancionVersion int64,
) (*model.SeparacionStem, error) {

	return escanearSeparacionStem(r.db.QueryRow(`
		SELECT `+columnasSeparacionStem+`
		FROM separacionstem
		WHERE codigocancionversion = $1
		ORDER BY fechahorasolicitud DESC, codseparacionstem DESC
		LIMIT 1
	`,
		codigoCancionVersion,
	))
}

func (r *separacionStemRepository) FallarInterrumpidas(
	mensaje string,
) (int64, error) {

	resultado, err := r.db.Exec(`
		UPDATE separacionstem
		SET estadoseparacion = 'ERROR',
			mensajeerror = $1,
			fechahorafin = CURRENT_TIMESTAMP
		WHERE estadoseparacion IN ('PENDIENTE', 'PROCESANDO')
	`,
		mensaje,
	)

	if err != nil {
		return 0, err
	}

	return resultado.RowsAffected()
}

func (r *separacionStemRepository) CategoriasPorDefecto() (map[string]int64, error) {

	rows, err := r.db.Query(`
		SELECT
			codcategoriastem,
			nombrecategoriastem
		FROM categoriastem
		WHERE codigoproyecto IS NULL
		  AND fechahorabajacategoriastem IS NULL
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	categorias := map[string]int64{}

	for rows.Next() {

		var codigo int64
		var nombre string

		if err := rows.Scan(&codigo, &nombre); err != nil {
			return nil, err
		}

		categorias[nombre] = codigo
	}

	return categorias, rows.Err()
}

func (r *separacionStemRepository) ExisteStemGeneradoConIA(
	codigoCancionVersion int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM stem
			WHERE codigocancionversion = $1
			  AND fechahorabajastem IS NULL
			  AND generadoconia
		)
	`,
		codigoCancionVersion,
	).Scan(&existe)

	return existe, err
}
