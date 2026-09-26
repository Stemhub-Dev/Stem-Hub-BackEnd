package model

// Stem activo de una versión, con el nombre de su categoría resuelto.
// Mismo criterio de archivo único que CancionVersion: la key del objeto en
// MinIO + el formato, más el nombre original para mostrarlo en la UI.
type Stem struct {
	CodStem              int64
	CodigoCancionVersion int64
	NombreStem           string
	CodCategoriaStem     int64
	NombreCategoriaStem  string
	URLArchivoStem       string
	FormatoArchivoStem   string
	NombreArchivoStem    string
	GeneradoConIA        bool
}

// CategoriaStem agrupa stems. CodigoProyecto nil = categoría por defecto
// (Batería, Bajo, Guitarra, Voz), disponible en todos los proyectos.
type CategoriaStem struct {
	CodCategoriaStem    int64
	NombreCategoriaStem string
	CodigoProyecto      *int64
}
