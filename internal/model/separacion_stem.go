package model

import "time"

const (
	EstadoSeparacionPendiente  = "PENDIENTE"
	EstadoSeparacionProcesando = "PROCESANDO"
	EstadoSeparacionCompletada = "COMPLETADA"
	EstadoSeparacionError      = "ERROR"
)

// SeparacionStem es un pedido de "Separar Pistas" sobre una versión, que el
// backend procesa en segundo plano llamando al microservicio de IA.
type SeparacionStem struct {
	CodSeparacionStem     int64
	CodigoCancionVersion  int64
	CodIntegrante         int64
	CantidadStems         int
	EstadoSeparacion      string
	MensajeError          *string
	TiempoProcesamientoMs *int64
	FechaHoraSolicitud    time.Time
	FechaHoraFin          *time.Time
}
