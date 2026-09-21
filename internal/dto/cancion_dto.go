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
	CodigoCancionVersion int64          `json:"codigoCancionVersion"`
	CodigoCancion        int64          `json:"codigoCancion"`
	NumeroVersion        int            `json:"numeroVersion"`
	EtiquetaVersion      string         `json:"etiquetaVersion"`
	Notas                *string        `json:"notas"`
	Stems                []StemResponse `json:"stems"`
}

type StemResponse struct {
	CodStem int64  `json:"codStem"`
	Nombre  string `json:"nombre"`
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

type VersionCancionListadoResponse struct {
	CodigoCancionVersion int64     `json:"codigoCancionVersion"`
	NumeroVersion        int       `json:"numeroVersion"`
	EtiquetaVersion      string    `json:"etiquetaVersion"`
	FechaHoraAlta        time.Time `json:"fechaHoraAlta"`
	URLArchivo           *string   `json:"urlArchivo"`
	FormatoArchivo       *string   `json:"formatoArchivo"`
	Notas                *string   `json:"notas"`
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
}

type EditarCancionRequest struct {
	Nombre string `json:"nombre"`
}

type EditarCancionResponse struct {
	CodigoCancion  int64  `json:"codigoCancion"`
	CodigoProyecto int64  `json:"codigoProyecto"`
	Nombre         string `json:"nombre"`
}
