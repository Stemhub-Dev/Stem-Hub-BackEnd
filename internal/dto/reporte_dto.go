package dto

import "time"

type ReporteCatalogoResponse struct {
	IDReporte               string `json:"id_reporte"`
	TipoReporte             string `json:"tipo_reporte"`
	FuncionEjecucionReporte string `json:"funcion_ejecucion_reporte"`
}

type GenerarReporteRequest struct {
	CodigoProyecto   int64      `json:"codigo_proyecto" binding:"required"`
	FechaDesde       *time.Time `json:"fecha_desde"`
	FechaHasta       *time.Time `json:"fecha_hasta"`
	CodigoCancion    *int64     `json:"codigo_cancion"`
	CodigoVersion    *int64     `json:"codigo_version"`
	EstadoComentario *string    `json:"estado_comentario"`
}

type ReporteProyectoResponse struct {
	CodigoProyecto int64  `json:"codigo_proyecto"`
	Nombre         string `json:"nombre"`
	Estado         string `json:"estado"`
	Tipo           string `json:"tipo"`
}

type ReporteRespuesta struct {
	IDReporte       string                   `json:"id_reporte"`
	TipoReporte     string                   `json:"tipo_reporte"`
	Parametros      GenerarReporteRequest    `json:"parametros"`
	Proyecto        ReporteProyectoResponse  `json:"proyecto"`
	Datos           []map[string]interface{} `json:"datos"`
	FechaGeneracion time.Time                `json:"fecha_generacion"`
}
