package repository

import (
	"database/sql"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type ReporteFiltros struct {
	FechaDesde       *time.Time
	FechaHasta       *time.Time
	CodigoCancion    *int64
	CodigoVersion    *int64
	EstadoComentario *string
}

type ReporteRepository interface {
	ObtenerProyecto(codigoProyecto int64) (dto.ReporteProyectoResponse, error)
	ObtenerActividad(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error)
	ObtenerHistorialVersiones(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error)
	ObtenerParticipacionColaboradores(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error)
	ObtenerEstadoCanciones(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error)
}

type reporteRepository struct {
	db *sql.DB
}

func NewReporteRepository(db *sql.DB) ReporteRepository {
	return &reporteRepository{db: db}
}

func (r *reporteRepository) ObtenerProyecto(codigoProyecto int64) (dto.ReporteProyectoResponse, error) {
	var proyecto dto.ReporteProyectoResponse
	err := r.db.QueryRow(`
		SELECT p.codigoproyecto, p.nombreproyecto, ep.nombreestadoproy, tp.nombretipoproy
		FROM proyecto p
		JOIN estadoproyecto ep ON ep.codestadoproy = p.codestadoproy
		JOIN tipoproyecto tp ON tp.codtipoproy = p.codtipoproy
		WHERE p.codigoproyecto = $1 AND p.fechahorabajaproyecto IS NULL
	`, codigoProyecto).Scan(&proyecto.CodigoProyecto, &proyecto.Nombre, &proyecto.Estado, &proyecto.Tipo)
	return proyecto, err
}

func (r *reporteRepository) ObtenerActividad(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error) {
	var fila map[string]interface{}
	var err error
	var canciones, versiones, comentarios, colaboradores int64
	err = r.db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM cancion WHERE codigoproyecto = $1 AND fechahorabajacancion IS NULL),
			(SELECT COUNT(*) FROM cancion c JOIN cancionversion cv ON cv.codigocancion = c.codigocancion
			 WHERE c.codigoproyecto = $1 AND c.fechahorabajacancion IS NULL AND cv.fechahorabajaversion IS NULL
			 AND ($2::timestamptz IS NULL OR cv.fechahoraaltaversion >= $2) AND ($3::timestamptz IS NULL OR cv.fechahoraaltaversion <= $3)),
			(SELECT COUNT(*) FROM comentario co WHERE co.codigoproyecto = $1 AND co.fechahorabajacomentario IS NULL
			 AND ($2::timestamptz IS NULL OR co.fechahoraaltacomentario >= $2) AND ($3::timestamptz IS NULL OR co.fechahoraaltacomentario <= $3)),
			(SELECT COUNT(*) FROM integranteproyecto WHERE codigoproyecto = $1 AND fechahorabajaintegranteproy IS NULL)
	`, codigoProyecto, filtros.FechaDesde, filtros.FechaHasta).Scan(&canciones, &versiones, &comentarios, &colaboradores)
	if err != nil {
		return nil, err
	}

	fila = map[string]interface{}{"canciones": canciones, "versiones": versiones, "comentarios": comentarios, "colaboradores": colaboradores}
	return []map[string]interface{}{fila}, nil
}

func (r *reporteRepository) ObtenerHistorialVersiones(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT c.codigocancion, c.nombrecancion, cv.codigocancionversion, cv.numeroversion,
		       cv.fechahoraaltaversion, cv.formatoarchivocancionver, cv.notasversion
		FROM cancion c JOIN cancionversion cv ON cv.codigocancion = c.codigocancion
		WHERE c.codigoproyecto = $1 AND c.fechahorabajacancion IS NULL AND cv.fechahorabajaversion IS NULL
		  AND ($2::timestamptz IS NULL OR cv.fechahoraaltaversion >= $2)
		  AND ($3::timestamptz IS NULL OR cv.fechahoraaltaversion <= $3)
		  AND ($4::bigint IS NULL OR c.codigocancion = $4)
		  AND ($5::bigint IS NULL OR cv.codigocancionversion = $5)
		ORDER BY c.nombrecancion, cv.numeroversion
	`, codigoProyecto, filtros.FechaDesde, filtros.FechaHasta, filtros.CodigoCancion, filtros.CodigoVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := make([]map[string]interface{}, 0)
	for rows.Next() {
		var idCancion, idVersion int64
		var nombre, formato string
		var numero int
		var fecha time.Time
		var notas sql.NullString
		if err := rows.Scan(&idCancion, &nombre, &idVersion, &numero, &fecha, &formato, &notas); err != nil {
			return nil, err
		}
		fila := map[string]interface{}{"codigo_cancion": idCancion, "cancion": nombre, "codigo_version": idVersion, "numero_version": numero, "fecha": fecha, "formato": formato}
		if notas.Valid {
			fila["notas"] = notas.String
		}
		resultado = append(resultado, fila)
	}
	return resultado, rows.Err()
}

func (r *reporteRepository) ObtenerParticipacionColaboradores(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT i.codintegrante, i.nombreintegrante, r.nombrerol,
		       COUNT(DISTINCT cv.codigocancionversion), COUNT(DISTINCT co.codigocomentario)
		FROM integranteproyecto ip
		JOIN integrante i ON i.codintegrante = ip.codintegrante
		JOIN rol r ON r.codrol = ip.codrol AND r.ambitorol = ip.ambitorol
		LEFT JOIN comentario co ON co.codintegrante = i.codintegrante AND co.fechahorabajacomentario IS NULL
		  AND ($2::timestamptz IS NULL OR co.fechahoraaltacomentario >= $2)
		  AND ($3::timestamptz IS NULL OR co.fechahoraaltacomentario <= $3)
		  AND ($4::text IS NULL OR EXISTS (SELECT 1 FROM estadocomentario ecf WHERE ecf.codestadocom = co.codestadocom AND ecf.nombreestadocom = $4))
		LEFT JOIN cancionversion cv ON cv.codigocancionversion = co.codigocancionversion
		WHERE ip.codigoproyecto = $1 AND ip.fechahorabajaintegranteproy IS NULL
		GROUP BY i.codintegrante, i.nombreintegrante, r.nombrerol
		ORDER BY i.nombreintegrante
	`, codigoProyecto, filtros.FechaDesde, filtros.FechaHasta, filtros.EstadoComentario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var nombre, rol string
		var versiones, comentarios int64
		if err := rows.Scan(&id, &nombre, &rol, &versiones, &comentarios); err != nil {
			return nil, err
		}
		resultado = append(resultado, map[string]interface{}{"codigo_integrante": id, "colaborador": nombre, "rol": rol, "versiones_comentadas": versiones, "comentarios": comentarios})
	}
	return resultado, rows.Err()
}

func (r *reporteRepository) ObtenerEstadoCanciones(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT c.codigocancion, c.nombrecancion,
		       COALESCE(MAX(cv.numeroversion), 0), COUNT(DISTINCT cv.codigocancionversion),
		       COUNT(DISTINCT co.codigocomentario), COALESCE(STRING_AGG(DISTINCT ec.nombreestadocom, ', '), '')
		FROM cancion c
		LEFT JOIN cancionversion cv ON cv.codigocancion = c.codigocancion AND cv.fechahorabajaversion IS NULL
		LEFT JOIN comentario co ON co.codigocancionversion = cv.codigocancionversion AND co.fechahorabajacomentario IS NULL
		  AND ($3::text IS NULL OR EXISTS (SELECT 1 FROM estadocomentario ecf WHERE ecf.codestadocom = co.codestadocom AND ecf.nombreestadocom = $3))
		LEFT JOIN estadocomentario ec ON ec.codestadocom = co.codestadocom
		WHERE c.codigoproyecto = $1 AND c.fechahorabajacancion IS NULL
		  AND ($2::bigint IS NULL OR c.codigocancion = $2)
		GROUP BY c.codigocancion, c.nombrecancion
		ORDER BY c.nombrecancion
	`, codigoProyecto, filtros.CodigoCancion, filtros.EstadoComentario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var nombre, estados string
		var ultima, versiones, comentarios int64
		if err := rows.Scan(&id, &nombre, &ultima, &versiones, &comentarios, &estados); err != nil {
			return nil, err
		}
		resultado = append(resultado, map[string]interface{}{"codigo_cancion": id, "cancion": nombre, "ultima_version": ultima, "total_versiones": versiones, "comentarios": comentarios, "estados_comentarios": estados})
	}
	return resultado, rows.Err()
}
