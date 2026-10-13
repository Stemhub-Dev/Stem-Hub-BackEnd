package service

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// Falsos para TableroService: embeben la interfaz real (nil) e implementan
// solo lo que usa el service.

type proyectoTableroFalso struct {
	repository.ProyectoRepository

	noExiste    bool
	noParticipa bool
}

func (p *proyectoTableroFalso) ExisteProyectoActivo(int64) (bool, error) {
	return !p.noExiste, nil
}

func (p *proyectoTableroFalso) EsIntegranteActivo(int64, int64) (bool, error) {
	return !p.noParticipa, nil
}

type tableroRepositoryFalso struct {
	repository.TableroRepository

	conteos       repository.TableroConteos
	etapas        []repository.TableroEtapaConteo
	total         int64
	actividad     map[int]repository.TableroActividadConteo
	err           error
	integrante    int64
	limites       []time.Time
	inicioPeriodo time.Time
	periodoAnt    time.Time
}

func (r *tableroRepositoryFalso) ObtenerConteos(integrante int64, _ *int64, inicio, _, anterior time.Time) (repository.TableroConteos, error) {
	r.integrante, r.inicioPeriodo, r.periodoAnt = integrante, inicio, anterior
	return r.conteos, r.err
}

func (r *tableroRepositoryFalso) ListarVersionesPorProyecto(integrante int64, _ *int64) ([]dto.TableroVersionesPorProyectoResponse, error) {
	r.integrante = integrante
	return []dto.TableroVersionesPorProyectoResponse{}, r.err
}

func (r *tableroRepositoryFalso) ContarCancionesPorEtapa(integrante int64, _ *int64) ([]repository.TableroEtapaConteo, int64, error) {
	r.integrante = integrante
	return r.etapas, r.total, r.err
}

func (r *tableroRepositoryFalso) ContarActividad(integrante int64, _ *int64, limites []time.Time) (map[int]repository.TableroActividadConteo, error) {
	r.integrante, r.limites = integrante, limites
	return r.actividad, r.err
}

// Miércoles 7 de octubre de 2026, 15:30 en Buenos Aires.
var ahoraTablero = time.Date(2026, time.October, 7, 15, 30, 0, 0, time.FixedZone("ART", -3*60*60))

func nuevoServicioTablero(
	repo *tableroRepositoryFalso,
	proyectos *proyectoTableroFalso,
	integrantes *integranteRepositoryFalso,
) *tableroService {
	servicio := NewTableroService(repo, proyectos, integrantes).(*tableroService)
	servicio.ahora = func() time.Time { return ahoraTablero }
	return servicio
}

func integranteTablero() *integranteRepositoryFalso {
	return &integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 10}}
}

func proyectoTablero(v int64) *int64 { return &v }

func TestTablero_ValidacionDeProyecto(t *testing.T) {
	casos := []struct {
		nombre      string
		proyectos   *proyectoTableroFalso
		integrantes *integranteRepositoryFalso
		proyecto    *int64
		esperado    error
	}{
		{"proyecto inexistente es 404", &proyectoTableroFalso{noExiste: true}, integranteTablero(), proyectoTablero(3), ErrTableroProyectoNoEncontrado},
		{"no participa es 403", &proyectoTableroFalso{noParticipa: true}, integranteTablero(), proyectoTablero(3), ErrTableroSinAcceso},
		{"sin perfil con proyecto es 403", &proyectoTableroFalso{}, &integranteRepositoryFalso{err: sql.ErrNoRows}, proyectoTablero(3), ErrTableroSinAcceso},
		{"inexistente gana a sin acceso", &proyectoTableroFalso{noExiste: true, noParticipa: true}, integranteTablero(), proyectoTablero(3), ErrTableroProyectoNoEncontrado},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := nuevoServicioTablero(&tableroRepositoryFalso{}, caso.proyectos, caso.integrantes)

			llamadas := map[string]func() error{
				"indicadores": func() error { _, err := servicio.ObtenerIndicadores(1, caso.proyecto); return err },
				"versiones":   func() error { _, err := servicio.ObtenerVersionesPorProyecto(1, caso.proyecto); return err },
				"etapas":      func() error { _, err := servicio.ObtenerCancionesPorEtapa(1, caso.proyecto); return err },
				"actividad": func() error {
					_, err := servicio.ObtenerActividad(1, caso.proyecto, AgrupacionSemanal)
					return err
				},
			}

			for nombre, llamar := range llamadas {
				if err := llamar(); !errors.Is(err, caso.esperado) {
					t.Errorf("%s: se esperaba %v, se obtuvo %v", nombre, caso.esperado, err)
				}
			}
		})
	}
}

func TestTablero_SinPerfilSinProyectoConsultaConIntegranteCero(t *testing.T) {
	repo := &tableroRepositoryFalso{integrante: -1}
	servicio := nuevoServicioTablero(repo, &proyectoTableroFalso{}, &integranteRepositoryFalso{err: sql.ErrNoRows})

	indicadores, err := servicio.ObtenerIndicadores(1, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.integrante != 0 {
		t.Errorf("integrante = %d, se esperaba 0", repo.integrante)
	}
	if *indicadores != (dto.TableroIndicadoresResponse{}) {
		t.Errorf("se esperaban todos los indicadores en 0, se obtuvo %+v", *indicadores)
	}
}

func TestTablero_PropagaErrorDelRepositorio(t *testing.T) {
	falla := errors.New("falla de base")
	servicio := nuevoServicioTablero(&tableroRepositoryFalso{err: falla}, &proyectoTableroFalso{}, integranteTablero())

	if _, err := servicio.ObtenerIndicadores(1, nil); !errors.Is(err, falla) {
		t.Errorf("se esperaba el error del repositorio, se obtuvo %v", err)
	}
	if _, err := servicio.ObtenerActividad(1, nil, AgrupacionMensual); !errors.Is(err, falla) {
		t.Errorf("se esperaba el error del repositorio, se obtuvo %v", err)
	}
}

func TestObtenerIndicadores_CalculaValoresYVariaciones(t *testing.T) {
	repo := &tableroRepositoryFalso{conteos: repository.TableroConteos{
		ProyectosActuales: 3, ProyectosInicio: 2,
		CancionesActuales: 3, CancionesInicio: 4,
		VersionesActuales: 7, VersionesInicio: 4,
		VersionesPeriodo: 3, VersionesPeriodoAnterior: 5,
		PendientesActuales: 6, PendientesInicio: 1,
	}}
	servicio := nuevoServicioTablero(repo, &proyectoTableroFalso{}, integranteTablero())

	indicadores, err := servicio.ObtenerIndicadores(1, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	esperado := dto.TableroIndicadoresResponse{
		TotalProyectosActivos:  dto.IndicadorEnteroResponse{Valor: 3, Variacion: 1},
		TotalCanciones:         dto.IndicadorEnteroResponse{Valor: 3, Variacion: -1},
		VersionesUltimos30Dias: dto.IndicadorEnteroResponse{Valor: 3, Variacion: -2},
		ComentariosPendientes:  dto.IndicadorEnteroResponse{Valor: 6, Variacion: 5},
		// 7/3 = 2.33; 4/4 = 1.
		PromedioVersionesPorCancion: dto.IndicadorDecimalResponse{Valor: 2.33, Variacion: 1.33},
	}
	if *indicadores != esperado {
		t.Errorf("indicadores = %+v\nse esperaba   %+v", *indicadores, esperado)
	}

	if repo.integrante != 10 {
		t.Errorf("integrante = %d, se esperaba 10", repo.integrante)
	}
	if !repo.inicioPeriodo.Equal(ahoraTablero.AddDate(0, 0, -30)) || !repo.periodoAnt.Equal(ahoraTablero.AddDate(0, 0, -60)) {
		t.Errorf("períodos inesperados: inicio %v, anterior %v", repo.inicioPeriodo, repo.periodoAnt)
	}
}

func TestObtenerIndicadores_PromedioEsCeroSinCanciones(t *testing.T) {
	repo := &tableroRepositoryFalso{conteos: repository.TableroConteos{VersionesActuales: 2}}
	servicio := nuevoServicioTablero(repo, &proyectoTableroFalso{}, integranteTablero())

	indicadores, err := servicio.ObtenerIndicadores(1, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if indicadores.PromedioVersionesPorCancion != (dto.IndicadorDecimalResponse{}) {
		t.Errorf("promedio = %+v, se esperaba 0", indicadores.PromedioVersionesPorCancion)
	}
}

func TestObtenerCancionesPorEtapa(t *testing.T) {
	casos := []struct {
		nombre   string
		etapas   []repository.TableroEtapaConteo
		total    int64
		esperado []dto.TableroCancionesPorEtapaResponse
	}{
		{
			nombre:   "sin canciones devuelve vacío",
			etapas:   []repository.TableroEtapaConteo{{Etapa: "Maquetación", Activa: true}},
			total:    0,
			esperado: []dto.TableroCancionesPorEtapaResponse{},
		},
		{
			nombre: "incluye etapas en 0 y agrega Sin etapa",
			etapas: []repository.TableroEtapaConteo{
				{Etapa: "Maquetación", Cantidad: 2, Activa: true},
				{Etapa: "Composición", Cantidad: 0, Activa: true},
				{Etapa: "Mezcla", Cantidad: 1, Activa: true},
			},
			total: 5,
			esperado: []dto.TableroCancionesPorEtapaResponse{
				{Etapa: "Maquetación", Cantidad: 2},
				{Etapa: "Composición", Cantidad: 0},
				{Etapa: "Mezcla", Cantidad: 1},
				{Etapa: EtapaSinAsignar, Cantidad: 2},
			},
		},
		{
			nombre: "etapa dada de baja solo si tiene canciones",
			etapas: []repository.TableroEtapaConteo{
				{Etapa: "Idea", Cantidad: 0, Activa: false},
				{Etapa: "Demo", Cantidad: 1, Activa: false},
				{Etapa: "Mezcla", Cantidad: 0, Activa: true},
			},
			total: 1,
			esperado: []dto.TableroCancionesPorEtapaResponse{
				{Etapa: "Demo", Cantidad: 1},
				{Etapa: "Mezcla", Cantidad: 0},
			},
		},
		{
			nombre:   "catálogo vacío cuenta todo como Sin etapa",
			etapas:   []repository.TableroEtapaConteo{},
			total:    4,
			esperado: []dto.TableroCancionesPorEtapaResponse{{Etapa: EtapaSinAsignar, Cantidad: 4}},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &tableroRepositoryFalso{etapas: caso.etapas, total: caso.total}
			servicio := nuevoServicioTablero(repo, &proyectoTableroFalso{}, integranteTablero())

			datos, err := servicio.ObtenerCancionesPorEtapa(1, nil)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if !reflect.DeepEqual(datos, caso.esperado) {
				t.Errorf("datos = %+v\nse esperaba %+v", datos, caso.esperado)
			}
		})
	}
}

func TestObtenerActividad_AgrupacionInvalida(t *testing.T) {
	servicio := nuevoServicioTablero(&tableroRepositoryFalso{}, &proyectoTableroFalso{}, integranteTablero())

	for _, agrupacion := range []string{"diaria", "SEMANAL", ""} {
		if _, err := servicio.ObtenerActividad(1, nil, agrupacion); !errors.Is(err, ErrTableroAgrupacionInvalida) {
			t.Errorf("agrupacion %q: se esperaba ErrTableroAgrupacionInvalida, se obtuvo %v", agrupacion, err)
		}
	}
}

func TestObtenerActividad_SinActividadDevuelveVacio(t *testing.T) {
	servicio := nuevoServicioTablero(&tableroRepositoryFalso{}, &proyectoTableroFalso{}, integranteTablero())

	datos, err := servicio.ObtenerActividad(1, nil, AgrupacionSemanal)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if datos == nil || len(datos) != 0 {
		t.Errorf("se esperaba un arreglo vacío (no nil), se obtuvo %#v", datos)
	}
}

func TestObtenerActividad_MensualCompletaPeriodosConCero(t *testing.T) {
	repo := &tableroRepositoryFalso{actividad: map[int]repository.TableroActividadConteo{
		2: {Versiones: 3, Comentarios: 1},
		4: {Versiones: 0, Comentarios: 2},
	}}
	servicio := nuevoServicioTablero(repo, &proyectoTableroFalso{}, integranteTablero())

	datos, err := servicio.ObtenerActividad(1, nil, AgrupacionMensual)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	esperado := []dto.TableroActividadResponse{
		{Periodo: "2026-07"},
		{Periodo: "2026-08", Versiones: 3, Comentarios: 1},
		{Periodo: "2026-09"},
		{Periodo: "2026-10", Comentarios: 2},
	}
	if !reflect.DeepEqual(datos, esperado) {
		t.Errorf("datos = %+v\nse esperaba %+v", datos, esperado)
	}
}

func TestLimitesActividad_Semanal(t *testing.T) {
	limites := limitesActividad(ahoraTablero, AgrupacionSemanal)

	// Hace 3 meses fue el martes 7/7, cuya semana empieza el lunes 6/7. La
	// semana en curso empieza el lunes 5/10 y termina el lunes 12/10.
	primero := time.Date(2026, time.July, 6, 0, 0, 0, 0, ahoraTablero.Location())
	ultimo := time.Date(2026, time.October, 12, 0, 0, 0, 0, ahoraTablero.Location())

	if !limites[0].Equal(primero) {
		t.Errorf("primer límite = %v, se esperaba %v", limites[0], primero)
	}
	if !limites[len(limites)-1].Equal(ultimo) {
		t.Errorf("último límite = %v, se esperaba %v", limites[len(limites)-1], ultimo)
	}
	if len(limites) != 15 {
		t.Errorf("se esperaban 15 límites (14 semanas), se obtuvieron %d", len(limites))
	}
	for i := 1; i < len(limites); i++ {
		if limites[i].Weekday() != time.Monday || limites[i].Sub(limites[i-1]) != 7*24*time.Hour {
			t.Errorf("límite %d inválido: %v", i, limites[i])
		}
	}
}

func TestLimitesActividad_MensualNoSeSaltaFebrero(t *testing.T) {
	// 31/5 - 3 meses normalizaría a marzo si se restara antes de ir al
	// primer día del mes.
	ahora := time.Date(2026, time.May, 31, 10, 0, 0, 0, time.UTC)

	limites := limitesActividad(ahora, AgrupacionMensual)

	esperado := []time.Time{
		time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(limites, esperado) {
		t.Errorf("límites = %v\nse esperaba %v", limites, esperado)
	}
}
