package service

import (
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// Tablero (HU-DASH-B01/B02).

const (
	AgrupacionSemanal = "semanal"
	AgrupacionMensual = "mensual"

	// Período contra el que se calcula la variación de los indicadores. El
	// mismo que el de versionesUltimos30Dias; queda pendiente con PO
	// confirmar si debe ser otro (ver HU-DASH-B01).
	diasPeriodoIndicadores = 30

	// Ventana del gráfico de actividad (HU-DASH-B02).
	mesesActividad = 3

	// Etiqueta para las canciones cuya versión actual no tiene etapa.
	EtapaSinAsignar = "Sin etapa"
)

var (
	ErrTableroProyectoNoEncontrado = errors.New("el proyecto no existe")
	ErrTableroSinAcceso            = errors.New("el usuario no participa del proyecto")
	ErrTableroAgrupacionInvalida   = errors.New("la agrupación debe ser semanal o mensual")
)

type TableroService interface {
	ObtenerIndicadores(codigoUsuario int64, codigoProyecto *int64) (*dto.TableroIndicadoresResponse, error)
	ObtenerVersionesPorProyecto(codigoUsuario int64, codigoProyecto *int64) ([]dto.TableroVersionesPorProyectoResponse, error)
	ObtenerCancionesPorEtapa(codigoUsuario int64, codigoProyecto *int64) ([]dto.TableroCancionesPorEtapaResponse, error)
	ObtenerActividad(codigoUsuario int64, codigoProyecto *int64, agrupacion string) ([]dto.TableroActividadResponse, error)
}

type tableroService struct {
	tableroRepository    repository.TableroRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository

	// Reemplazable en tests.
	ahora func() time.Time
}

func NewTableroService(
	tableroRepository repository.TableroRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
) TableroService {
	return &tableroService{
		tableroRepository:    tableroRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		ahora:                time.Now,
	}
}

// resolverIntegrante devuelve el integrante sobre cuyos proyectos se calcula
// el Tablero. Con proyectoId valida además que el proyecto exista (404) y que
// el usuario participe de él (403), en ese orden.
//
// Un usuario sin perfil de integrante no participa de ningún proyecto: se
// devuelve el código 0, que no corresponde a ningún integrante (es una
// identidad), y las consultas dan todo en 0 / vacío.
func (s *tableroService) resolverIntegrante(
	codigoUsuario int64,
	codigoProyecto *int64,
) (int64, error) {

	if codigoProyecto != nil {
		existe, err := s.proyectoRepository.ExisteProyectoActivo(*codigoProyecto)
		if err != nil {
			return 0, err
		}
		if !existe {
			return 0, ErrTableroProyectoNoEncontrado
		}
	}

	integrante, err := s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		if codigoProyecto != nil {
			return 0, ErrTableroSinAcceso
		}
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	if codigoProyecto != nil {
		participa, err := s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			*codigoProyecto,
		)
		if err != nil {
			return 0, err
		}
		if !participa {
			return 0, ErrTableroSinAcceso
		}
	}

	return integrante.CodIntegrante, nil
}

func (s *tableroService) ObtenerIndicadores(
	codigoUsuario int64,
	codigoProyecto *int64,
) (*dto.TableroIndicadoresResponse, error) {

	codigoIntegrante, err := s.resolverIntegrante(codigoUsuario, codigoProyecto)
	if err != nil {
		return nil, err
	}

	ahora := s.ahora()
	inicioPeriodo := ahora.AddDate(0, 0, -diasPeriodoIndicadores)
	inicioPeriodoAnterior := inicioPeriodo.AddDate(0, 0, -diasPeriodoIndicadores)

	conteos, err := s.tableroRepository.ObtenerConteos(
		codigoIntegrante,
		codigoProyecto,
		inicioPeriodo,
		ahora,
		inicioPeriodoAnterior,
	)
	if err != nil {
		return nil, err
	}

	return calcularIndicadores(conteos), nil
}

// calcularIndicadores arma la respuesta a partir de los conteos crudos. La
// variación es la diferencia absoluta contra el inicio del período (o contra
// el período anterior, para versionesUltimos30Dias).
func calcularIndicadores(c repository.TableroConteos) *dto.TableroIndicadoresResponse {

	promedioActual := promedioVersiones(c.VersionesActuales, c.CancionesActuales)
	promedioInicio := promedioVersiones(c.VersionesInicio, c.CancionesInicio)

	return &dto.TableroIndicadoresResponse{
		TotalProyectosActivos: dto.IndicadorEnteroResponse{
			Valor:     c.ProyectosActuales,
			Variacion: c.ProyectosActuales - c.ProyectosInicio,
		},
		TotalCanciones: dto.IndicadorEnteroResponse{
			Valor:     c.CancionesActuales,
			Variacion: c.CancionesActuales - c.CancionesInicio,
		},
		VersionesUltimos30Dias: dto.IndicadorEnteroResponse{
			Valor:     c.VersionesPeriodo,
			Variacion: c.VersionesPeriodo - c.VersionesPeriodoAnterior,
		},
		ComentariosPendientes: dto.IndicadorEnteroResponse{
			Valor:     c.PendientesActuales,
			Variacion: c.PendientesActuales - c.PendientesInicio,
		},
		PromedioVersionesPorCancion: dto.IndicadorDecimalResponse{
			Valor:     promedioActual,
			Variacion: redondear2(promedioActual - promedioInicio),
		},
	}
}

func promedioVersiones(versiones, canciones int64) float64 {
	if canciones == 0 {
		return 0
	}
	return redondear2(float64(versiones) / float64(canciones))
}

func redondear2(valor float64) float64 {
	return math.Round(valor*100) / 100
}

func (s *tableroService) ObtenerVersionesPorProyecto(
	codigoUsuario int64,
	codigoProyecto *int64,
) ([]dto.TableroVersionesPorProyectoResponse, error) {

	codigoIntegrante, err := s.resolverIntegrante(codigoUsuario, codigoProyecto)
	if err != nil {
		return nil, err
	}

	return s.tableroRepository.ListarVersionesPorProyecto(codigoIntegrante, codigoProyecto)
}

func (s *tableroService) ObtenerCancionesPorEtapa(
	codigoUsuario int64,
	codigoProyecto *int64,
) ([]dto.TableroCancionesPorEtapaResponse, error) {

	codigoIntegrante, err := s.resolverIntegrante(codigoUsuario, codigoProyecto)
	if err != nil {
		return nil, err
	}

	etapas, totalCanciones, err := s.tableroRepository.ContarCancionesPorEtapa(
		codigoIntegrante,
		codigoProyecto,
	)
	if err != nil {
		return nil, err
	}

	return distribuirPorEtapa(etapas, totalCanciones), nil
}

// distribuirPorEtapa incluye las etapas activas del catálogo aunque estén en
// 0 (HU-DASH-B02 #2), las dadas de baja solo si todavía tienen canciones, y
// agrega "Sin etapa" con las canciones restantes si las hay. Sin canciones
// no hay nada que graficar: devuelve [] (HU-DASH-B02 #6).
func distribuirPorEtapa(
	etapas []repository.TableroEtapaConteo,
	totalCanciones int64,
) []dto.TableroCancionesPorEtapaResponse {

	resultado := make([]dto.TableroCancionesPorEtapaResponse, 0)

	if totalCanciones == 0 {
		return resultado
	}

	var conEtapa int64

	for _, etapa := range etapas {
		conEtapa += etapa.Cantidad

		if !etapa.Activa && etapa.Cantidad == 0 {
			continue
		}

		resultado = append(resultado, dto.TableroCancionesPorEtapaResponse{
			Etapa:    etapa.Etapa,
			Cantidad: etapa.Cantidad,
		})
	}

	if sinEtapa := totalCanciones - conEtapa; sinEtapa > 0 {
		resultado = append(resultado, dto.TableroCancionesPorEtapaResponse{
			Etapa:    EtapaSinAsignar,
			Cantidad: sinEtapa,
		})
	}

	return resultado
}

func (s *tableroService) ObtenerActividad(
	codigoUsuario int64,
	codigoProyecto *int64,
	agrupacion string,
) ([]dto.TableroActividadResponse, error) {

	if agrupacion != AgrupacionSemanal && agrupacion != AgrupacionMensual {
		return nil, ErrTableroAgrupacionInvalida
	}

	codigoIntegrante, err := s.resolverIntegrante(codigoUsuario, codigoProyecto)
	if err != nil {
		return nil, err
	}

	limites := limitesActividad(s.ahora(), agrupacion)

	conteos, err := s.tableroRepository.ContarActividad(
		codigoIntegrante,
		codigoProyecto,
		limites,
	)
	if err != nil {
		return nil, err
	}

	return armarActividad(limites, agrupacion, conteos), nil
}

// limitesActividad devuelve los límites de los períodos completos (semanas
// de lunes a domingo o meses calendario, en la zona horaria de ahora) que
// cubren los últimos 3 meses, incluido el período en curso. n+1 límites
// delimitan n períodos; el último límite es el fin del período en curso.
func limitesActividad(ahora time.Time, agrupacion string) []time.Time {

	var inicio, fin time.Time
	var siguiente func(time.Time) time.Time

	if agrupacion == AgrupacionMensual {
		// Se resta desde el primer día del mes: restarle meses a un 31
		// normaliza hacia adelante (31/5 - 3 meses = 3/3) y se saltearía
		// febrero.
		inicio = inicioMes(ahora).AddDate(0, -mesesActividad, 0)
		fin = inicioMes(ahora).AddDate(0, 1, 0)
		siguiente = func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
	} else {
		inicio = inicioSemana(ahora.AddDate(0, -mesesActividad, 0))
		fin = inicioSemana(ahora).AddDate(0, 0, 7)
		siguiente = func(t time.Time) time.Time { return t.AddDate(0, 0, 7) }
	}

	limites := []time.Time{inicio}
	for t := siguiente(inicio); !t.After(fin); t = siguiente(t) {
		limites = append(limites, t)
	}

	return limites
}

func inicioSemana(t time.Time) time.Time {
	// time.Weekday arranca en domingo; la semana del Tablero, en lunes.
	diasDesdeLunes := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-diasDesdeLunes, 0, 0, 0, 0, t.Location())
}

func inicioMes(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// armarActividad completa con 0 los períodos sin actividad, para que la
// línea temporal no tenga huecos. Si no hubo ninguna actividad en toda la
// ventana devuelve [] (HU-DASH-B02 #6).
func armarActividad(
	limites []time.Time,
	agrupacion string,
	conteos map[int]repository.TableroActividadConteo,
) []dto.TableroActividadResponse {

	resultado := make([]dto.TableroActividadResponse, 0)

	if len(conteos) == 0 {
		return resultado
	}

	formato := "2006-01-02"
	if agrupacion == AgrupacionMensual {
		formato = "2006-01"
	}

	for i := 0; i < len(limites)-1; i++ {
		conteo := conteos[i+1]

		resultado = append(resultado, dto.TableroActividadResponse{
			Periodo:     limites[i].Format(formato),
			Versiones:   conteo.Versiones,
			Comentarios: conteo.Comentarios,
		})
	}

	return resultado
}
