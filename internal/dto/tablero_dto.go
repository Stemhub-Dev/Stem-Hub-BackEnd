package dto

// Tablero (HU-DASH-B01/B02). Todos los datos se calculan sobre los proyectos
// en los que el usuario participa, o sobre uno solo si se filtra por
// proyectoId.

// IndicadorEnteroResponse es un indicador de conteo. Variacion es la
// diferencia absoluta (no porcentual) contra el período anterior: puede ser
// negativa.
type IndicadorEnteroResponse struct {
	Valor     int64 `json:"valor"`
	Variacion int64 `json:"variacion"`
}

// IndicadorDecimalResponse es un indicador con decimales, redondeado a 2.
type IndicadorDecimalResponse struct {
	Valor     float64 `json:"valor"`
	Variacion float64 `json:"variacion"`
}

type TableroIndicadoresResponse struct {
	TotalProyectosActivos       IndicadorEnteroResponse  `json:"totalProyectosActivos"`
	TotalCanciones              IndicadorEnteroResponse  `json:"totalCanciones"`
	VersionesUltimos30Dias      IndicadorEnteroResponse  `json:"versionesUltimos30Dias"`
	ComentariosPendientes       IndicadorEnteroResponse  `json:"comentariosPendientes"`
	PromedioVersionesPorCancion IndicadorDecimalResponse `json:"promedioVersionesPorCancion"`
}

type TableroVersionesPorProyectoResponse struct {
	ProyectoID        int64  `json:"proyectoId"`
	Nombre            string `json:"nombre"`
	CantidadVersiones int64  `json:"cantidadVersiones"`
}

type TableroCancionesPorEtapaResponse struct {
	Etapa    string `json:"etapa"`
	Cantidad int64  `json:"cantidad"`
}

type TableroActividadResponse struct {
	// Inicio del período: "2026-09-28" (semana, empieza el lunes) o
	// "2026-09" (mes).
	Periodo     string `json:"periodo"`
	Versiones   int64  `json:"versiones"`
	Comentarios int64  `json:"comentarios"`
}
