package model

import "time"

// NombreRolAdministradorSistema es el nombre exacto (comparado sin
// distinguir mayúsculas) del rol de ámbito SISTEMA con permisos de
// administración global, tal como se lo crea en la migración
// 004datosseguridadl.sql.
const NombreRolAdministradorSistema = "Administrador del sistema"

type Rol struct {
	//el asterisco indica que el campo puede ser nulo
	//el json es como se debe escribir en el frontend
	CodRol           int64      `json:"codRol"`
	NombreRol        string     `json:"nombreRol"`
	DescripcionRol   *string    `json:"descripcionRol"`
	AmbitoRol        string     `json:"ambitoRol"`
	FechaHoraBajaRol *time.Time `json:"fechaHoraBajaRol"`
}
