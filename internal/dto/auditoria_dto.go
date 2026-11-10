package dto

import (
	"encoding/json"
	"time"
)

type AuditoriaFiltro struct {
	Pagina         int
	TamanoPagina   int
	Entidad        string
	Evento         string
	CodigoUsuario  *int64
	CodigoProyecto *int64
	Desde          *time.Time
	Hasta          *time.Time
}

type AuditoriaItemResponse struct {
	CodigoAuditoria    int64           `json:"codigoAuditoria"`
	CodigoUsuario      int64           `json:"codigoUsuario"`
	NombreUsuario      string          `json:"nombreUsuario"`
	EmailUsuario       string          `json:"emailUsuario"`
	Entidad            string          `json:"entidad"`
	CodigoRegistro     int64           `json:"codigoRegistro"`
	Accion             string          `json:"accion"`
	Evento             *string         `json:"evento"`
	CodigoProyecto     *int64          `json:"codigoProyecto"`
	Contexto           json.RawMessage `json:"contexto"`
	FechaHoraAuditoria time.Time       `json:"fechaHoraAuditoria"`
}

type AuditoriaPaginadaResponse struct {
	Data        []AuditoriaItemResponse `json:"data"`
	CurrentPage int                     `json:"currentPage"`
	PageSize    int                     `json:"pageSize"`
	TotalItems  int64                   `json:"totalItems"`
	TotalPages  int64                   `json:"totalPages"`
}
