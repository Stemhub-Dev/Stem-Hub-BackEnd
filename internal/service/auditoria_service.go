package service

import (
	"context"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var ErrFiltroAuditoriaInvalido = errors.New(
	"Los filtros de auditoría son inválidos",
)

type AuditoriaService interface {
	Listar(
		ctx context.Context,
		filtro dto.AuditoriaFiltro,
	) (*dto.AuditoriaPaginadaResponse, error)
}

type auditoriaService struct {
	repository repository.AuditoriaConsultaRepository
}

func NewAuditoriaService(
	repo repository.AuditoriaConsultaRepository,
) AuditoriaService {
	return &auditoriaService{repository: repo}
}

func (s *auditoriaService) Listar(
	ctx context.Context,
	filtro dto.AuditoriaFiltro,
) (*dto.AuditoriaPaginadaResponse, error) {
	filtro.Entidad = strings.ToLower(strings.TrimSpace(filtro.Entidad))
	filtro.Evento = strings.ToUpper(strings.TrimSpace(filtro.Evento))

	if filtro.Pagina < 1 ||
		filtro.TamanoPagina < 1 ||
		filtro.TamanoPagina > 100 {
		return nil, ErrFiltroAuditoriaInvalido
	}

	if len(filtro.Entidad) > 100 || len(filtro.Evento) > 30 {
		return nil, ErrFiltroAuditoriaInvalido
	}

	if filtro.CodigoUsuario != nil && *filtro.CodigoUsuario <= 0 {
		return nil, ErrFiltroAuditoriaInvalido
	}

	if filtro.CodigoProyecto != nil && *filtro.CodigoProyecto <= 0 {
		return nil, ErrFiltroAuditoriaInvalido
	}

	if filtro.Desde != nil &&
		filtro.Hasta != nil &&
		!filtro.Desde.Before(*filtro.Hasta) {
		return nil, ErrFiltroAuditoriaInvalido
	}

	return s.repository.Listar(ctx, filtro)
}
