package service

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// Falsos para ReporteService: embeben la interfaz real (nil) e implementan
// solo lo que usa el service.

type proyectoReporteFalso struct {
	repository.ProyectoRepository

	noPropietario bool
}

func (p *proyectoReporteFalso) ExisteProyectoActivo(int64) (bool, error) {
	return true, nil
}

func (p *proyectoReporteFalso) EsPropietarioActivo(int64, int64) (bool, error) {
	return !p.noPropietario, nil
}

type reporteRepositoryFalso struct {
	repository.ReporteRepository

	datos []map[string]interface{}
}

func (r *reporteRepositoryFalso) ObtenerProyecto(codigoProyecto int64) (dto.ReporteProyectoResponse, error) {
	return dto.ReporteProyectoResponse{CodigoProyecto: codigoProyecto, Nombre: "Maqueta Solista", Estado: "En Progreso", Tipo: "EP"}, nil
}

func (r *reporteRepositoryFalso) ObtenerEstadoCanciones(int64, repository.ReporteFiltros) ([]map[string]interface{}, error) {
	return r.datos, nil
}

func nuevoServicioReportes(proyectos *proyectoReporteFalso) ReporteService {
	reportes := &reporteRepositoryFalso{datos: []map[string]interface{}{
		{"codigo_cancion": int64(1), "cancion": "Balada", "comentarios_pendientes": int64(2)},
	}}
	integrantes := &integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 10}}
	return NewReporteService(reportes, proyectos, integrantes)
}

func TestExportarPDF_DevuelvePDFConNombreDescriptivo(t *testing.T) {
	servicio := nuevoServicioReportes(&proyectoReporteFalso{})

	contenido, nombre, err := servicio.ExportarPDF(1, ReporteEstadoCanciones, dto.GenerarReporteRequest{CodigoProyecto: 3})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !bytes.HasPrefix(contenido, []byte("%PDF-")) {
		t.Errorf("el contenido no es un PDF")
	}
	if !strings.HasPrefix(nombre, "maqueta-solista_estado-de-canciones_") || !strings.HasSuffix(nombre, ".pdf") {
		t.Errorf("nombre de archivo inesperado: %q", nombre)
	}
}

func TestExportarPDF_PropagaErrorDeGenerar(t *testing.T) {
	servicio := nuevoServicioReportes(&proyectoReporteFalso{noPropietario: true})

	contenido, nombre, err := servicio.ExportarPDF(1, ReporteEstadoCanciones, dto.GenerarReporteRequest{CodigoProyecto: 3})
	if !errors.Is(err, ErrReporteSinPermiso) {
		t.Fatalf("se esperaba ErrReporteSinPermiso, se obtuvo %v", err)
	}
	if contenido != nil || nombre != "" {
		t.Errorf("no debería devolver contenido ni nombre ante un error")
	}
}
