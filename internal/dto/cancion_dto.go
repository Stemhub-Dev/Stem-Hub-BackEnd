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
	CodigoCancionVersion int64  `json:"codigoCancionVersion"`
	CodigoCancion        int64  `json:"codigoCancion"`
	NumeroVersion        int    `json:"numeroVersion"`
	EtiquetaVersion      string `json:"etiquetaVersion"`
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
}
