package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type AuditoriaConsultaRepository interface {
	Listar(
		ctx context.Context,
		filtro dto.AuditoriaFiltro,
	) (*dto.AuditoriaPaginadaResponse, error)
}

type auditoriaConsultaRepository struct {
	db *sql.DB
}

func NewAuditoriaConsultaRepository(
	db *sql.DB,
) AuditoriaConsultaRepository {
	return &auditoriaConsultaRepository{db: db}
}

// Estos parámetros se comparten entre el conteo y el listado:
// $1 entidad, $2 evento, $3 usuario, $4 proyecto,
// $5 desde inclusivo y $6 hasta exclusivo.
const filtroConsultaAuditoria = `
	WHERE ($1::text = '' OR a.entidad = $1)
	  AND ($2::text = '' OR a.evento = $2)
	  AND ($3::bigint IS NULL OR a.codigousuario = $3)
	  AND ($4::bigint IS NULL OR a.codigoproyecto = $4)
	  AND ($5::timestamptz IS NULL OR a.fechahoraauditoria >= $5)
	  AND ($6::timestamptz IS NULL OR a.fechahoraauditoria < $6)
`

func (r *auditoriaConsultaRepository) Listar(
	ctx context.Context,
	filtro dto.AuditoriaFiltro,
) (*dto.AuditoriaPaginadaResponse, error) {
	if filtro.Pagina < 1 ||
		filtro.TamanoPagina < 1 ||
		filtro.TamanoPagina > 100 {
		return nil, fmt.Errorf("paginación de auditoría inválida")
	}

	// Conteo y filas se leen sobre la misma vista de la base.
	// Un evento nuevo entre ambas consultas no desajusta la respuesta.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("iniciar consulta de auditoría: %w", err)
	}
	defer tx.Rollback()

	args := []any{
		filtro.Entidad,
		filtro.Evento,
		filtro.CodigoUsuario,
		filtro.CodigoProyecto,
		filtro.Desde,
		filtro.Hasta,
	}

	respuesta := &dto.AuditoriaPaginadaResponse{
		Data:        make([]dto.AuditoriaItemResponse, 0),
		CurrentPage: filtro.Pagina,
		PageSize:    filtro.TamanoPagina,
	}

	err = tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM public.auditoria a `+
			filtroConsultaAuditoria,
		args...,
	).Scan(&respuesta.TotalItems)
	if err != nil {
		return nil, fmt.Errorf("contar auditoría: %w", err)
	}

	tamano := int64(filtro.TamanoPagina)
	respuesta.TotalPages = respuesta.TotalItems / tamano
	if respuesta.TotalItems%tamano != 0 {
		respuesta.TotalPages++
	}

	// Evita calcular un OFFSET innecesario o desbordado.
	if int64(filtro.Pagina) > respuesta.TotalPages {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("finalizar consulta de auditoría: %w", err)
		}
		return respuesta, nil
	}

	desplazamiento := (int64(filtro.Pagina) - 1) * tamano

	args = append(args, tamano, desplazamiento)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			a.codigoauditoria,
			a.codigousuario,
			COALESCE(
				(
					SELECT i.nombreintegrante
					FROM public.integrante i
					WHERE i.codigousuario = a.codigousuario
					ORDER BY i.codintegrante
					LIMIT 1
				),
				u.email
			),
			u.email,
			a.entidad,
			a.codigoregistro,
			a.accion,
			a.evento,
			a.codigoproyecto,
			a.contexto,
			a.fechahoraauditoria
		FROM public.auditoria a
		JOIN public.usuario u
		  ON u.codigousuario = a.codigousuario
	`+filtroConsultaAuditoria+`
		ORDER BY
			a.fechahoraauditoria DESC,
			a.codigoauditoria DESC
		LIMIT $7 OFFSET $8
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("listar auditoría: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item dto.AuditoriaItemResponse
		var contexto []byte

		err := rows.Scan(
			&item.CodigoAuditoria,
			&item.CodigoUsuario,
			&item.NombreUsuario,
			&item.EmailUsuario,
			&item.Entidad,
			&item.CodigoRegistro,
			&item.Accion,
			&item.Evento,
			&item.CodigoProyecto,
			&contexto,
			&item.FechaHoraAuditoria,
		)
		if err != nil {
			return nil, fmt.Errorf("leer evento de auditoría: %w", err)
		}

		item.Contexto = json.RawMessage(contexto)
		respuesta.Data = append(respuesta.Data, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer auditoría: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("cerrar filas de auditoría: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finalizar consulta de auditoría: %w", err)
	}

	return respuesta, nil
}
