package service

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

const (
	ReporteActividad       = "actividad-proyecto"
	ReporteHistorial       = "historial-versiones"
	ReporteParticipacion   = "participacion-colaboradores"
	ReporteEstadoCanciones = "estado-canciones"
)

var (
	ErrReporteTipoNoEncontrado     = errors.New("el tipo de reporte no existe")
	ErrReporteProyectoNoEncontrado = errors.New("el proyecto no existe")
	ErrReporteSinPermiso           = errors.New("el usuario no es administrador del proyecto")
	ErrReporteFiltrosInvalidos     = errors.New("los filtros del reporte no son válidos")
	ErrReporteExportacion          = errors.New("no se pudo exportar el reporte")
)

type ReporteService interface {
	ListarCatalogo(codigoUsuario, codigoProyecto int64) ([]dto.ReporteCatalogoResponse, error)
	Generar(codigoUsuario int64, tipo string, request dto.GenerarReporteRequest) (*dto.ReporteRespuesta, error)
	ExportarPDF(codigoUsuario int64, tipo string, request dto.GenerarReporteRequest) ([]byte, string, error)
}

type reporteService struct {
	reporteRepository    repository.ReporteRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
}

func NewReporteService(reporteRepository repository.ReporteRepository, proyectoRepository repository.ProyectoRepository, integranteRepository repository.IntegranteRepository) ReporteService {
	return &reporteService{reporteRepository: reporteRepository, proyectoRepository: proyectoRepository, integranteRepository: integranteRepository}
}

func catalogoReportes() []dto.ReporteCatalogoResponse {
	return []dto.ReporteCatalogoResponse{
		{IDReporte: "REP-001", TipoReporte: ReporteActividad, FuncionEjecucionReporte: "generarActividadProyecto"},
		{IDReporte: "REP-002", TipoReporte: ReporteHistorial, FuncionEjecucionReporte: "generarHistorialVersiones"},
		{IDReporte: "REP-003", TipoReporte: ReporteParticipacion, FuncionEjecucionReporte: "generarParticipacionColaboradores"},
		{IDReporte: "REP-004", TipoReporte: ReporteEstadoCanciones, FuncionEjecucionReporte: "generarEstadoCanciones"},
	}
}

func (s *reporteService) validarAcceso(codigoUsuario, codigoProyecto int64) error {
	existe, err := s.proyectoRepository.ExisteProyectoActivo(codigoProyecto)
	if err != nil {
		return err
	}
	if !existe {
		return ErrReporteProyectoNoEncontrado
	}
	integrante, err := s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)
	if err != nil {
		return err
	}
	propietario, err := s.proyectoRepository.EsPropietarioActivo(integrante.CodIntegrante, codigoProyecto)
	if err != nil {
		return err
	}
	if !propietario {
		return ErrReporteSinPermiso
	}
	return nil
}

func (s *reporteService) ListarCatalogo(codigoUsuario, codigoProyecto int64) ([]dto.ReporteCatalogoResponse, error) {
	if err := s.validarAcceso(codigoUsuario, codigoProyecto); err != nil {
		return nil, err
	}
	return catalogoReportes(), nil
}

func (s *reporteService) Generar(codigoUsuario int64, tipo string, request dto.GenerarReporteRequest) (*dto.ReporteRespuesta, error) {
	if !esTipoReporteValido(tipo) {
		return nil, ErrReporteTipoNoEncontrado
	}
	if request.CodigoProyecto <= 0 || (request.FechaDesde != nil && request.FechaHasta != nil && request.FechaDesde.After(*request.FechaHasta)) {
		return nil, ErrReporteFiltrosInvalidos
	}
	if err := s.validarAcceso(codigoUsuario, request.CodigoProyecto); err != nil {
		return nil, err
	}
	proyecto, err := s.reporteRepository.ObtenerProyecto(request.CodigoProyecto)
	if err != nil {
		return nil, err
	}
	filtros := repository.ReporteFiltros{FechaDesde: request.FechaDesde, FechaHasta: request.FechaHasta, CodigoCancion: request.CodigoCancion, CodigoVersion: request.CodigoVersion, EstadoComentario: request.EstadoComentario}
	var datos []map[string]interface{}
	switch tipo {
	case ReporteActividad:
		datos, err = s.reporteRepository.ObtenerActividad(request.CodigoProyecto, filtros)
	case ReporteHistorial:
		datos, err = s.reporteRepository.ObtenerHistorialVersiones(request.CodigoProyecto, filtros)
	case ReporteParticipacion:
		datos, err = s.reporteRepository.ObtenerParticipacionColaboradores(request.CodigoProyecto, filtros)
	case ReporteEstadoCanciones:
		datos, err = s.reporteRepository.ObtenerEstadoCanciones(request.CodigoProyecto, filtros)
	}
	if err != nil {
		return nil, err
	}
	return &dto.ReporteRespuesta{IDReporte: idReporte(tipo), TipoReporte: tipo, Parametros: request, Proyecto: proyecto, Datos: datos, FechaGeneracion: time.Now().UTC()}, nil
}

func (s *reporteService) ExportarPDF(codigoUsuario int64, tipo string, request dto.GenerarReporteRequest) ([]byte, string, error) {
	reporte, err := s.Generar(codigoUsuario, tipo, request)
	if err != nil {
		return nil, "", err
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Reporte "+reporte.TipoReporte, true)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Reporte "+reporte.TipoReporte, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.MultiCell(0, 6, fmt.Sprintf("Proyecto: %s\nFecha: %s\nFiltros: proyecto=%d fecha_desde=%v fecha_hasta=%v cancion=%v version=%v", reporte.Proyecto.Nombre, reporte.FechaGeneracion.Format("2006-01-02 15:04:05 UTC"), request.CodigoProyecto, request.FechaDesde, request.FechaHasta, request.CodigoCancion, request.CodigoVersion), "", "L", false)
	pdf.Ln(4)
	for indice, fila := range reporte.Datos {
		pdf.MultiCell(0, 6, fmt.Sprintf("%d. %v", indice+1, fila), "", "L", false)
	}
	var contenido bytes.Buffer
	if err := pdf.Output(&contenido); err != nil {
		log.Printf("Error al generar PDF de reporte: tipo=%s proyecto=%d error=%v", tipo, request.CodigoProyecto, err)
		return nil, "", ErrReporteExportacion
	}
	log.Printf("Reporte PDF generado: tipo=%s proyecto=%d filas=%d", tipo, request.CodigoProyecto, len(reporte.Datos))
	nombre := fmt.Sprintf("%s_%s_%s.pdf", sanitizarNombre(reporte.Proyecto.Nombre), tipo, reporte.FechaGeneracion.Format("20060102"))
	return contenido.Bytes(), nombre, nil
}

func esTipoReporteValido(tipo string) bool {
	return tipo == ReporteActividad || tipo == ReporteHistorial || tipo == ReporteParticipacion || tipo == ReporteEstadoCanciones
}
func idReporte(tipo string) string {
	for _, reporte := range catalogoReportes() {
		if reporte.TipoReporte == tipo {
			return reporte.IDReporte
		}
	}
	return ""
}
func sanitizarNombre(nombre string) string {
	nombre = strings.TrimSpace(strings.ToLower(nombre))
	nombre = strings.NewReplacer(" ", "_", "/", "-", "\\", "-").Replace(nombre)
	return nombre
}
