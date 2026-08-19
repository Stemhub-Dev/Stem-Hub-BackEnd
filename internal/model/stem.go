package model

import "time"

type Stem struct {
	CodStem              int64      `json:"codStem"`
	CodigoCancionVersion int64      `json:"codigoCancionVersion"`
	NombreStem           string     `json:"nombreStem"`
	DescripcionStem      *string    `json:"descripcionStem"`
	URLVersionWav        *string    `json:"urlVersionWav"`
	URLVersionMp3        *string    `json:"urlVersionMp3"`
	FechaHoraBajaStem    *time.Time `json:"fechaHoraBajaStem"`
}
