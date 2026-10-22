package dto

import "time"

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
	TotalStems                  IndicadorEnteroResponse  `json:"totalStems"`
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

// Novedades del Inicio ("Mientras no estabas"): lo que hicieron los demás
// integrantes en los proyectos del usuario, de lo más reciente a lo más viejo.
const (
	NovedadVersion    = "VERSION"
	NovedadComentario = "COMENTARIO"
	NovedadRespuesta  = "RESPUESTA"
	// Stems subidos a mano, agrupados por autor, versión y día.
	NovedadStems = "STEMS"
	// Separación con IA terminada. Es la única que incluye las del propio
	// usuario: avisa que un proceso en segundo plano terminó.
	NovedadSeparacion = "SEPARACION"
)

type TableroNovedadAutorResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
}

type TableroNovedadResponse struct {
	Tipo           string                      `json:"tipo"`
	Fecha          time.Time                   `json:"fecha"`
	Autor          TableroNovedadAutorResponse `json:"autor"`
	CodigoProyecto int64                       `json:"codigoProyecto"`
	NombreProyecto string                      `json:"nombreProyecto"`
	CodigoCancion  int64                       `json:"codigoCancion"`
	NombreCancion  string                      `json:"nombreCancion"`
	// Stems subidos (STEMS) o generados (SEPARACION); 1 en el resto.
	Cantidad int64 `json:"cantidad"`
}

type TableroNovedadesResponse struct {
	Items []TableroNovedadResponse `json:"items"`
	// Novedades posteriores a ?desde= (todas, no solo las de Items); 0 si
	// no se pidió.
	NovedadesDesde int64 `json:"novedadesDesde"`
}

// TableroVisitaResponse devuelve la visita registrada antes de la actual
// (nil en la primera), para contar las novedades desde entonces.
type TableroVisitaResponse struct {
	VisitaAnterior *time.Time `json:"visitaAnterior"`
}
