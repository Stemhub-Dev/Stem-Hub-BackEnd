package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// Solo hay un reporte (estado del proyecto) y su consulta usa únicamente
// estos dos filtros: es una foto del estado actual, sin recorte por fecha.
type ReporteFiltros struct {
	CodigoCancion    *int64
	EstadoComentario *string
}

type ReporteRepository interface {
	ObtenerProyecto(codigoProyecto int64) (dto.ReporteProyectoResponse, error)
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
		SELECT p.codigoproyecto, p.nombreproyecto, ep.nombreestadoproy, tp.nombretipoproy,
		       NULLIF(TRIM(p.logoproyecto), '')
		FROM proyecto p
		JOIN estadoproyecto ep ON ep.codestadoproy = p.codestadoproy
		JOIN tipoproyecto tp ON tp.codtipoproy = p.codtipoproy
		WHERE p.codigoproyecto = $1 AND p.fechahorabajaproyecto IS NULL
	`, codigoProyecto).Scan(&proyecto.CodigoProyecto, &proyecto.Nombre, &proyecto.Estado, &proyecto.Tipo, &proyecto.Logo)
	return proyecto, err
}

func (r *reporteRepository) ObtenerEstadoCanciones(codigoProyecto int64, filtros ReporteFiltros) ([]map[string]interface{}, error) {
	// Pendientes y resueltos como números (no la lista de estados distintos):
	// es lo que dice qué canción necesita atención. Las que más pendientes
	// tienen van primero.
	rows, err := r.db.Query(`
		SELECT c.codigocancion, c.nombrecancion,
		       COALESCE(MAX(cv.numeroversion), 0), COUNT(DISTINCT cv.codigocancionversion),
		       COUNT(DISTINCT co.codigocomentario),
		       COUNT(DISTINCT co.codigocomentario) FILTER (WHERE LOWER(ec.nombreestadocom) = 'pendiente'),
		       COUNT(DISTINCT co.codigocomentario) FILTER (WHERE LOWER(ec.nombreestadocom) = 'resuelto')
		FROM cancion c
		LEFT JOIN cancionversion cv ON cv.codigocancion = c.codigocancion AND cv.fechahorabajaversion IS NULL
		LEFT JOIN comentario co ON co.codigocancionversion = cv.codigocancionversion AND co.fechahorabajacomentario IS NULL
		  AND ($3::text IS NULL OR EXISTS (SELECT 1 FROM estadocomentario ecf WHERE ecf.codestadocom = co.codestadocom AND ecf.nombreestadocom = $3))
		LEFT JOIN estadocomentario ec ON ec.codestadocom = co.codestadocom
		WHERE c.codigoproyecto = $1 AND c.fechahorabajacancion IS NULL
		  AND ($2::bigint IS NULL OR c.codigocancion = $2)
		GROUP BY c.codigocancion, c.nombrecancion
		ORDER BY 6 DESC, c.nombrecancion
	`, codigoProyecto, filtros.CodigoCancion, filtros.EstadoComentario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var nombre string
		var ultima, versiones, comentarios, pendientes, resueltos int64
		if err := rows.Scan(&id, &nombre, &ultima, &versiones, &comentarios, &pendientes, &resueltos); err != nil {
			return nil, err
		}
		resultado = append(resultado, map[string]interface{}{"codigo_cancion": id, "cancion": nombre, "ultima_version": ultima, "total_versiones": versiones, "comentarios": comentarios, "comentarios_pendientes": pendientes, "comentarios_resueltos": resueltos})
	}
	return resultado, rows.Err()
}
