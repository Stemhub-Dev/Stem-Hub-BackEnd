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
