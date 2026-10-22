package dto

import "time"

// El texto de comentarios y respuestas no lleva binding: vacío y más de 200
// caracteres los valida el service, que recorta espacios, cuenta runas y
// devuelve el mensaje de cada caso (con binding salía "Solicitud inválida").

type CrearComentarioRequest struct {
	Texto                string   `json:"texto"`
	TiempoInicioSegundos *float64 `json:"tiempoInicioSegundos" binding:"omitempty,min=0"`
	TiempoFinSegundos    *float64 `json:"tiempoFinSegundos" binding:"omitempty,min=0"`
	// Opcional: el comentario es de ese stem de la versión. Sin él, es de
	// la versión completa.
	CodStem *int64 `json:"codStem" binding:"omitempty,min=1"`
}

type AutorComentarioResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
	// URL presignada del avatar; null si el integrante no subió uno.
	AvatarUrl *string `json:"avatarUrl"`
	// Object key del avatar en el storage. Lo completa el repositorio y el
	// service lo convierte en AvatarUrl; no viaja en la respuesta.
	AvatarObjectKey *string `json:"-"`
}

type CrearComentarioResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	CodStem              *int64                  `json:"codStem"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	Autor                AutorComentarioResponse `json:"autor"`
}

type ComentarioListadoResponse struct {
	CodigoComentario     int64                         `json:"codigoComentario"`
	CodStem              *int64                        `json:"codStem"`
	Texto                string                        `json:"texto"`
	Estado               string                        `json:"estado"`
	TiempoInicioSegundos *float64                      `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                      `json:"tiempoFinSegundos"`
	FechaHoraAlta        time.Time                     `json:"fechaHoraAlta"`
	Autor                AutorComentarioResponse       `json:"autor"`
	EsPropio             bool                          `json:"esPropio"`
	Respuestas           []RespuestaComentarioResponse `json:"respuestas"`
}

type CrearRespuestaComentarioRequest struct {
	Texto string `json:"texto"`
}

type RespuestaComentarioResponse struct {
	CodigoRespuesta int64                   `json:"codigoRespuesta"`
	Texto           string                  `json:"texto"`
	FechaHoraAlta   time.Time               `json:"fechaHoraAlta"`
	Autor           AutorComentarioResponse `json:"autor"`
	EsPropia        bool                    `json:"esPropia"`
}

type ModificarComentarioRequest struct {
	Texto string `json:"texto"`
}

type ModificarComentarioResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	Autor                AutorComentarioResponse `json:"autor"`
}

type CambiarEstadoComentarioRequest struct {
	Estado string `json:"estado" binding:"required"`
}

type CambiarEstadoComentarioResponse struct {
	CodigoComentario int64  `json:"codigoComentario"`
	Estado           string `json:"estado"`
}

// Resumen de los comentarios de una versión generado con IA
// (stemhub-microservicio-IA). No se guarda: cada pedido lo genera de nuevo.
type ResumenComentariosResponse struct {
	Resumen               string   `json:"resumen"`
	PuntosClave           []string `json:"puntosClave"`
	Acuerdos              []string `json:"acuerdos"`
	Desacuerdos           []string `json:"desacuerdos"`
	SentimientoGeneral    string   `json:"sentimientoGeneral"`
	CantidadComentarios   int      `json:"cantidadComentarios"`
	Proveedor             string   `json:"proveedor"`
	TiempoProcesamientoMs int64    `json:"tiempoProcesamientoMs"`
}
