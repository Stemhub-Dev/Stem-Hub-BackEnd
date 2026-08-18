package model

import "time"

type Auditoria struct {
	CodigoAuditoria    int64     `json:"codigoAuditoria"`
	CodigoUsuario      int64     `json:"codigoUsuario"`
	Entidad            string    `json:"entidad"`
	CodigoRegistro     int64     `json:"codigoRegistro"`
	Accion             string    `json:"accion"`
	FechaHoraAuditoria time.Time `json:"fechaHoraAuditoria"`
}
