package model

import "time"

// Nombres exactos de los roles como están en la tabla rol (se comparan sin
// distinguir mayúsculas). Tienen que coincidir carácter por carácter con lo
// que dejan las migraciones: un nombre desfasado no rompe la compilación, la
// consulta simplemente no matchea ninguna fila y el filtro queda vacío en
// silencio.
const (
	NombreRolAdministradorSistema = "Administrador"
	NombreRolProductor            = "Productor"
	// 002datosiniciales.sql lo creó como "Artista Musical" y
	// 004datosseguridadl.sql lo renombró a "Músico (Artista)".
	NombreRolMusicoArtista = "Músico (Artista)"
)

type Rol struct {
	//el asterisco indica que el campo puede ser nulo
	//el json es como se debe escribir en el frontend
	CodRol           int64      `json:"codRol"`
	NombreRol        string     `json:"nombreRol"`
	DescripcionRol   *string    `json:"descripcionRol"`
	AmbitoRol        string     `json:"ambitoRol"`
	FechaHoraBajaRol *time.Time `json:"fechaHoraBajaRol"`
}
