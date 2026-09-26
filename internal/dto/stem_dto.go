package dto

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
