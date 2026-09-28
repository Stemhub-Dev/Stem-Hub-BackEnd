package dto

import "time"

type CategoriaStemResponse struct {
	CodCategoriaStem int64  `json:"codCategoriaStem"`
	Nombre           string `json:"nombre"`
	PorDefecto       bool   `json:"porDefecto"`
}

type StemListadoResponse struct {
	CodStem          int64  `json:"codStem"`
	Nombre           string `json:"nombre"`
	CodCategoriaStem int64  `json:"codCategoriaStem"`
	NombreCategoria  string `json:"nombreCategoria"`
	GeneradoConIA    bool   `json:"generadoConIA"`
	NombreArchivo    string `json:"nombreArchivo"`
	FormatoArchivo   string `json:"formatoArchivo"`
}

// URL presignada de MinIO para reproducir el stem, igual que
// AudioVersionResponse.
type AudioStemResponse struct {
	CodStem          int64  `json:"codStem"`
	URL              string `json:"url"`
	ExpiraEnSegundos int    `json:"expiraEnSegundos"`
}

// Cuerpo de POST .../stems/separacion. Sin cantidadStems se separa en 4
// (voz, batería, bajo, otros).
type SolicitarSeparacionStemsRequest struct {
	CantidadStems *int `json:"cantidadStems"`
}

// Estado de "Separar Pistas" sobre una versión. Estado: PENDIENTE,
// PROCESANDO, COMPLETADA o ERROR; al completarse, los stems generados
// aparecen en el listado de stems de la versión con generadoConIA = true.
type SeparacionStemsResponse struct {
	CodSeparacionStem     int64      `json:"codSeparacionStem"`
	CodigoCancionVersion  int64      `json:"codigoCancionVersion"`
	CantidadStems         int        `json:"cantidadStems"`
	Estado                string     `json:"estado"`
	MensajeError          *string    `json:"mensajeError"`
	TiempoProcesamientoMs *int64     `json:"tiempoProcesamientoMs"`
	FechaHoraSolicitud    time.Time  `json:"fechaHoraSolicitud"`
	FechaHoraFin          *time.Time `json:"fechaHoraFin"`
}
