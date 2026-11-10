package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type RegistroAuditoria struct {
	CodigoUsuario  int64
	Entidad        string
	CodigoRegistro int64
	Accion         string
	Evento         string
	CodigoProyecto *int64
	Contexto       map[string]any
}

// Usa la transacción de la operación de negocio.
// No abre otra transacción ni hace Commit.
func RegistrarAuditoriaTx(
	tx *sql.Tx,
	registro RegistroAuditoria,
) error {
	if tx == nil {
		return fmt.Errorf("registrar auditoría: transacción requerida")
	}

	contexto := registro.Contexto
	if contexto == nil {
		contexto = map[string]any{}
	}

	contextoJSON, err := json.Marshal(contexto)
	if err != nil {
		return fmt.Errorf("serializar contexto de auditoría: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO public.auditoria (
			codigousuario,
			entidad,
			codigoregistro,
			accion,
			evento,
			codigoproyecto,
			contexto
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`,
		registro.CodigoUsuario,
		registro.Entidad,
		registro.CodigoRegistro,
		registro.Accion,
		registro.Evento,
		registro.CodigoProyecto,
		string(contextoJSON),
	)
	if err != nil {
		return fmt.Errorf("insertar auditoría: %w", err)
	}

	return nil
}

// Registra el alta de una versión ya insertada en la misma transacción.
// No confirma ni revierte la transacción: eso corresponde al llamador.
func RegistrarAltaVersionTx(
	tx *sql.Tx,
	codigoUsuario int64,
	codigoVersion int64,
) error {
	var codigoProyecto int64
	var codigoCancion int64
	var nombreProyecto string
	var nombreCancion string
	var numeroVersion int
	var formatoArchivo string
	var notas *string

	err := tx.QueryRow(`
		SELECT
			c.codigoproyecto,
			c.codigocancion,
			p.nombreproyecto,
			c.nombrecancion,
			cv.numeroversion,
			cv.formatoarchivocancionver,
			cv.notasversion
		FROM cancionversion cv
		JOIN cancion c
		  ON c.codigocancion = cv.codigocancion
		JOIN proyecto p
		  ON p.codigoproyecto = c.codigoproyecto
		WHERE cv.codigocancionversion = $1
	`, codigoVersion).Scan(
		&codigoProyecto,
		&codigoCancion,
		&nombreProyecto,
		&nombreCancion,
		&numeroVersion,
		&formatoArchivo,
		&notas,
	)
	if err != nil {
		return fmt.Errorf("leer contexto de alta de versión: %w", err)
	}

	return RegistrarAuditoriaTx(tx, RegistroAuditoria{
		CodigoUsuario:  codigoUsuario,
		Entidad:        "cancionversion",
		CodigoRegistro: codigoVersion,
		Accion:         "INSERT",
		Evento:         "ALTA",
		CodigoProyecto: &codigoProyecto,
		Contexto: map[string]any{
			"nombreProyecto": nombreProyecto,
			"codigoCancion":  codigoCancion,
			"nombreCancion":  nombreCancion,
			"numeroVersion":  numeroVersion,
			"formatoArchivo": formatoArchivo,
			"notas":          notas,
		},
	})
}

// Registra el alta de un stem ya insertado.
// Puede reutilizarse para stems manuales y generados con IA.
func RegistrarAltaStemTx(
	tx *sql.Tx,
	codigoUsuario int64,
	codStem int64,
) error {
	var codigoProyecto int64
	var codigoCancion int64
	var codigoVersion int64
	var nombreProyecto string
	var nombreCancion string
	var numeroVersion int
	var nombreStem string
	var nombreCategoria string
	var nombreArchivo string
	var formatoArchivo string
	var generadoConIA bool

	err := tx.QueryRow(`
		SELECT
			c.codigoproyecto,
			c.codigocancion,
			cv.codigocancionversion,
			p.nombreproyecto,
			c.nombrecancion,
			cv.numeroversion,
			s.nombrestem,
			cat.nombrecategoriastem,
			s.nombrearchivostem,
			s.formatoarchivostem,
			s.generadoconia
		FROM stem s
		JOIN cancionversion cv
		  ON cv.codigocancionversion = s.codigocancionversion
		JOIN cancion c
		  ON c.codigocancion = cv.codigocancion
		JOIN proyecto p
		  ON p.codigoproyecto = c.codigoproyecto
		JOIN categoriastem cat
		  ON cat.codcategoriastem = s.codcategoriastem
		WHERE s.codstem = $1
	`, codStem).Scan(
		&codigoProyecto,
		&codigoCancion,
		&codigoVersion,
		&nombreProyecto,
		&nombreCancion,
		&numeroVersion,
		&nombreStem,
		&nombreCategoria,
		&nombreArchivo,
		&formatoArchivo,
		&generadoConIA,
	)
	if err != nil {
		return fmt.Errorf("leer contexto de alta de stem: %w", err)
	}

	return RegistrarAuditoriaTx(tx, RegistroAuditoria{
		CodigoUsuario:  codigoUsuario,
		Entidad:        "stem",
		CodigoRegistro: codStem,
		Accion:         "INSERT",
		Evento:         "ALTA",
		CodigoProyecto: &codigoProyecto,
		Contexto: map[string]any{
			"nombreProyecto":       nombreProyecto,
			"codigoCancion":        codigoCancion,
			"nombreCancion":        nombreCancion,
			"codigoCancionVersion": codigoVersion,
			"numeroVersion":        numeroVersion,
			"nombreStem":           nombreStem,
			"nombreCategoria":      nombreCategoria,
			"nombreArchivo":        nombreArchivo,
			"formatoArchivo":       formatoArchivo,
			"generadoConIA":        generadoConIA,
		},
	})
}

func RegistrarAltaComentarioTx(
	tx *sql.Tx,
	codigoUsuario int64,
	codigoComentario int64,
) error {
	var codigoProyecto int64
	var codigoCancion int64
	var codigoVersion int64
	var numeroVersion int
	var nombreProyecto string
	var nombreCancion string
	var codStem *int64
	var nombreStem *string
	var texto string
	var estado string
	var inicio *float64
	var fin *float64

	err := tx.QueryRow(`
		SELECT
			c.codigoproyecto,
			c.codigocancion,
			cv.codigocancionversion,
			cv.numeroversion,
			p.nombreproyecto,
			c.nombrecancion,
			com.codstem,
			s.nombrestem,
			com.descripcioncomentario,
			ec.nombreestadocom,
			com.tiempoiniciosegundos,
			com.tiempofinsegundos
		FROM comentario com
		JOIN cancionversion cv
		  ON cv.codigocancionversion = com.codigocancionversion
		JOIN cancion c
		  ON c.codigocancion = cv.codigocancion
		JOIN proyecto p
		  ON p.codigoproyecto = c.codigoproyecto
		JOIN estadocomentario ec
		  ON ec.codestadocom = com.codestadocom
		LEFT JOIN stem s
		  ON s.codstem = com.codstem
		WHERE com.codigocomentario = $1
	`, codigoComentario).Scan(
		&codigoProyecto,
		&codigoCancion,
		&codigoVersion,
		&numeroVersion,
		&nombreProyecto,
		&nombreCancion,
		&codStem,
		&nombreStem,
		&texto,
		&estado,
		&inicio,
		&fin,
	)
	if err != nil {
		return fmt.Errorf("leer contexto de alta de comentario: %w", err)
	}

	return RegistrarAuditoriaTx(tx, RegistroAuditoria{
		CodigoUsuario:  codigoUsuario,
		Entidad:        "comentario",
		CodigoRegistro: codigoComentario,
		Accion:         "INSERT",
		Evento:         "ALTA",
		CodigoProyecto: &codigoProyecto,
		Contexto: map[string]any{
			"nombreProyecto":       nombreProyecto,
			"codigoCancion":        codigoCancion,
			"nombreCancion":        nombreCancion,
			"codigoCancionVersion": codigoVersion,
			"numeroVersion":        numeroVersion,
			"codStem":              codStem,
			"nombreStem":           nombreStem,
			"texto":                texto,
			"estado":               estado,
			"tiempoInicioSegundos": inicio,
			"tiempoFinSegundos":    fin,
		},
	})
}
