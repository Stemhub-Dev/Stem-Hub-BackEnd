package model

import "time"

type CancionVersion struct {
	CodigoCancionVersion    int64      `json:"codigoCancionVersion"`
	CodigoCancion           int64      `json:"codigoCancion"`
	NumeroVersion           int        `json:"numeroVersion"`
	FechaHoraAltaVersion    time.Time  `json:"fechaHoraAltaVersion"`
	FechaHoraBajaVersion    *time.Time `json:"fechaHoraBajaVersion"`
	URLVersionWavCancionVer *string    `json:"urlVersionWavCancionVer"`
	URLVersionMp3CancionVer *string    `json:"urlVersionMp3CancionVer"`
}
