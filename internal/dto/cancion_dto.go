package dto

type CrearCancionRequest struct {
	Nombre string `json:"nombre" binding:"required"`

	URLVersionWAV *string `json:"urlVersionWav"`
	URLVersionMP3 *string `json:"urlVersionMp3"`
}

type CrearCancionResponse struct {
	CodigoCancion        int64  `json:"codigoCancion"`
	NombreCancion        string `json:"nombreCancion"`
	CodigoCancionVersion int64  `json:"codigoCancionVersion"`
	NumeroVersion        int    `json:"numeroVersion"`
	EtiquetaVersion      string `json:"etiquetaVersion"`
}

type CrearVersionCancionRequest struct {
	URLVersionWAV *string `json:"urlVersionWav"`
	URLVersionMP3 *string `json:"urlVersionMp3"`
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
	URLVersionWAV        *string `json:"urlVersionWav"`
	URLVersionMP3        *string `json:"urlVersionMp3"`
}

type CancionListadoResponse struct {
	CodigoCancion int64                         `json:"codigoCancion"`
	Nombre        string                        `json:"nombre"`
	VersionActual *VersionActualCancionResponse `json:"versionActual"`
}
