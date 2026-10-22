package dto

import "time"

type CrearCancionResponse struct {
	CodigoCancion        int64  `json:"codigoCancion"`
	NombreCancion        string `json:"nombreCancion"`
	CodigoCancionVersion int64  `json:"codigoCancionVersion"`
	NumeroVersion        int    `json:"numeroVersion"`
	EtiquetaVersion      string `json:"etiquetaVersion"`
}

type CrearVersionCancionResponse struct {
	CodigoCancionVersion int64   `json:"codigoCancionVersion"`
	CodigoCancion        int64   `json:"codigoCancion"`
	NumeroVersion        int     `json:"numeroVersion"`
	EtiquetaVersion      string  `json:"etiquetaVersion"`
	Notas                *string `json:"notas"`
}

// EditarVersionRequest: PATCH .../versiones/:versionId (HU-ABM-03-02). Solo
// las notas: la pista no se cambia (para otra pista se crea otra versión).
// Notas vacías o solo espacios borran las notas.
type EditarVersionRequest struct {
	Notas *string `json:"notas" binding:"required"`
}

type EditarVersionResponse struct {
	CodigoCancionVersion int64   `json:"codigoCancionVersion"`
	Notas                *string `json:"notas"`
}

// BajaVersionResponse: DELETE .../versiones/:versionId (HU-ABM-03-03).
type BajaVersionResponse struct {
	CodigoCancionVersion int64 `json:"codigoCancionVersion"`
	NumeroVersion        int   `json:"numeroVersion"`
}

type VersionActualCancionResponse struct {
	CodigoCancionVersion int64   `json:"codigoCancionVersion"`
	NumeroVersion        int     `json:"numeroVersion"`
	EtiquetaVersion      string  `json:"etiquetaVersion"`
	URLArchivo           *string `json:"urlArchivo"`
	FormatoArchivo       *string `json:"formatoArchivo"`
}

type CancionListadoResponse struct {
	CodigoCancion int64                         `json:"codigoCancion"`
	Nombre        string                        `json:"nombre"`
	VersionActual *VersionActualCancionResponse `json:"versionActual"`
}

type AutorVersionResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
}

type VersionCancionListadoResponse struct {
	CodigoCancionVersion int64     `json:"codigoCancionVersion"`
	NumeroVersion        int       `json:"numeroVersion"`
	EtiquetaVersion      string    `json:"etiquetaVersion"`
	FechaHoraAlta        time.Time `json:"fechaHoraAlta"`
	URLArchivo           *string   `json:"urlArchivo"`
	FormatoArchivo       *string   `json:"formatoArchivo"`
	Notas                *string   `json:"notas"`
	// Autor es nil en las versiones subidas antes de que se guardara.
	Autor *AutorVersionResponse `json:"autor"`
	// EsActual marca la última versión activa de la canción.
	EsActual bool `json:"esActual"`
}

type AudioVersionResponse struct {
	CodigoCancionVersion int64  `json:"codigoCancionVersion"`
	URL                  string `json:"url"`
	FormatoArchivo       string `json:"formatoArchivo"`
	ExpiraEnSegundos     int    `json:"expiraEnSegundos"`
}

type MiCancionListadoResponse struct {
	CodigoCancion  int64  `json:"codigoCancion"`
	Nombre         string `json:"nombre"`
	CodigoProyecto int64  `json:"codigoProyecto"`
	NombreProyecto string `json:"nombreProyecto"`

	VersionActual *VersionActualCancionResponse `json:"versionActual"`

	// Actividad de la versión actual, para que Mis Canciones muestre qué se
	// movió sin abrir cada canción. Todo queda en nil/0 si la canción no
	// tiene versiones; el autor también es nil en las versiones subidas antes
	// de que se guardara, y la etapa mientras el alta de versión no la pida.
	FechaVersionActual    *time.Time            `json:"fechaVersionActual"`
	AutorVersionActual    *AutorVersionResponse `json:"autorVersionActual"`
	EtapaVersionActual    *string               `json:"etapaVersionActual"`
	ComentariosPendientes int                   `json:"comentariosPendientes"`
}

// Valores admitidos por el parámetro sort de GET /canciones. El repositorio
// los recibe tal cual y los compara dentro de la consulta, así que cualquier
// otro valor debe rechazarse en el handler.
const (
	OrdenMisCancionesReciente   = "reciente"
	OrdenMisCancionesNombreAsc  = "nombreAsc"
	OrdenMisCancionesNombreDesc = "nombreDesc"
)

// ListarMisCancionesFiltro agrupa los criterios de búsqueda, filtrado,
// orden y paginación de GET /canciones. Pagina y TamanoPagina ya vienen
// validados (>= 1) y Orden es uno de los OrdenMisCanciones*.
type ListarMisCancionesFiltro struct {
	Busqueda  string
	Proyectos []int64
	// SoloConComentariosPendientes deja las canciones cuya versión actual
	// tiene al menos un comentario en estado Pendiente.
	SoloConComentariosPendientes bool
	Orden                        string
	Pagina                       int
	TamanoPagina                 int
}

type MisCancionesPaginadasResponse struct {
	Data        []MiCancionListadoResponse `json:"data"`
	TotalItems  int                        `json:"totalItems"`
	TotalPages  int                        `json:"totalPages"`
	CurrentPage int                        `json:"currentPage"`
}

type EditarCancionRequest struct {
	Nombre string `json:"nombre"`
}

type EditarCancionResponse struct {
	CodigoCancion  int64  `json:"codigoCancion"`
	CodigoProyecto int64  `json:"codigoProyecto"`
	Nombre         string `json:"nombre"`
}
