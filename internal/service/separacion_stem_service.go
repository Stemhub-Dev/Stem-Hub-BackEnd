package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mlservice"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const cantidadStemsPorDefecto = 4

var (
	ErrSeparacionCantidadInvalida  = errors.New("la cantidad de stems debe ser 2, 4 o 5")
	ErrSeparacionVersionSinArchivo = errors.New("la versión no tiene un archivo de audio para separar")
	ErrSeparacionEnCurso           = errors.New("la versión ya tiene una separación en curso")
	ErrSeparacionYaGenerada        = errors.New("la versión ya tiene stems generados con IA")
	ErrSeparacionNoEncontrada      = errors.New("la versión no tiene separaciones")
)

// Mensajes que quedan guardados en la separación fallida y ve el usuario;
// el detalle técnico va al log.
const (
	mensajeSeparacionTimeout       = "La separación tardó demasiado. Probá de nuevo más tarde."
	mensajeSeparacionFallida       = "No se pudo separar la pista. Probá de nuevo más tarde."
	mensajeSeparacionInterrumpida  = "La separación se interrumpió porque el servidor se reinició. Probá de nuevo."
	mensajeSeparacionSinCategorias = "Faltan las categorías por defecto de los stems generados."
)

// Nombre del stem en StemHub y categoría por defecto (019/021) para cada
// stem que devuelve Spleeter.
var stemsSpleeter = map[string]struct {
	nombre    string
	categoria string
}{
	"vocals":        {"Voz", "Voz"},
	"drums":         {"Batería", "Batería"},
	"bass":          {"Bajo", "Bajo"},
	"piano":         {"Piano", "Piano"},
	"other":         {"Otros", "Otros"},
	"accompaniment": {"Acompañamiento", "Otros"},
}

type SeparacionStemService interface {
	// Registra la separación y la procesa en segundo plano: vuelve enseguida
	// con la separación PENDIENTE.
	Solicitar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		cantidadStems *int,
	) (*dto.SeparacionStemsResponse, error)

	ObtenerUltima(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
	) (*dto.SeparacionStemsResponse, error)

	// Al arrancar: las que quedaron en curso no las procesa nadie.
	FallarInterrumpidas() error
}

type separacionStemService struct {
	separacionRepository repository.SeparacionStemRepository
	stemRepository       repository.StemRepository
	cancionRepository    repository.CancionRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	audioStorage         storage.AudioStorage
	mlCliente            mlservice.Cliente

	// El worker de Spleeter separa de a una pista (un solo proceso de
	// gunicorn): las demás esperan su turno acá, todavía PENDIENTE.
	turnos chan struct{}

	// go f() en producción; los tests lo corren en el momento.
	enSegundoPlano func(func())
}

func NewSeparacionStemService(
	separacionRepository repository.SeparacionStemRepository,
	stemRepository repository.StemRepository,
	cancionRepository repository.CancionRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
	mlCliente mlservice.Cliente,
) SeparacionStemService {

	return &separacionStemService{
		separacionRepository: separacionRepository,
		stemRepository:       stemRepository,
		cancionRepository:    cancionRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		audioStorage:         audioStorage,
		mlCliente:            mlCliente,
		turnos:               make(chan struct{}, 1),
		enSegundoPlano:       func(f func()) { go f() },
	}
}

func separacionAResponse(separacion model.SeparacionStem) dto.SeparacionStemsResponse {
	return dto.SeparacionStemsResponse{
		CodSeparacionStem:     separacion.CodSeparacionStem,
		CodigoCancionVersion:  separacion.CodigoCancionVersion,
		CantidadStems:         separacion.CantidadStems,
		Estado:                separacion.EstadoSeparacion,
		MensajeError:          separacion.MensajeError,
		TiempoProcesamientoMs: separacion.TiempoProcesamientoMs,
		FechaHoraSolicitud:    separacion.FechaHoraSolicitud,
		FechaHoraFin:          separacion.FechaHoraFin,
	}
}

func (s *separacionStemService) Solicitar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	cantidadStems *int,
) (*dto.SeparacionStemsResponse, error) {

	cantidad := cantidadStemsPorDefecto

	if cantidadStems != nil {
		cantidad = *cantidadStems
	}

	if _, ok := mlservice.NombresStemPorCantidad[cantidad]; !ok {
		return nil, ErrSeparacionCantidadInvalida
	}

	integrante, err := validarAccesoVersionStem(
		s.proyectoRepository,
		s.cancionRepository,
		s.integranteRepository,
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		permisoGestionarStems,
	)

	if err != nil {
		return nil, err
	}

	version, err := s.cancionRepository.BuscarVersionPorCodigo(codigoCancion, codigoVersion)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStemVersionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	if version.URLArchivoCancionVer == nil || *version.URLArchivoCancionVer == "" {
		return nil, ErrSeparacionVersionSinArchivo
	}

	// Volver a separar duplicaría los stems: primero hay que eliminar los
	// generados antes (los que el usuario reemplazó ya no cuentan como IA).
	yaGenerada, err := s.separacionRepository.ExisteStemGeneradoConIA(codigoVersion)

	if err != nil {
		return nil, err
	}

	if yaGenerada {
		return nil, ErrSeparacionYaGenerada
	}

	separacion, err := s.separacionRepository.Crear(codigoVersion, integrante.CodIntegrante, cantidad)

	if errors.Is(err, repository.ErrSeparacionEnCurso) {
		return nil, ErrSeparacionEnCurso
	}

	if err != nil {
		return nil, err
	}

	fuente := *version.URLArchivoCancionVer
	pendiente := *separacion

	s.enSegundoPlano(func() {
		s.procesar(pendiente, codigoProyecto, codigoCancion, fuente)
	})

	respuesta := separacionAResponse(*separacion)

	return &respuesta, nil
}

func (s *separacionStemService) ObtenerUltima(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) (*dto.SeparacionStemsResponse, error) {

	if _, err := validarAccesoVersionStem(
		s.proyectoRepository,
		s.cancionRepository,
		s.integranteRepository,
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		"",
	); err != nil {
		return nil, err
	}

	separacion, err := s.separacionRepository.BuscarUltimaPorVersion(codigoVersion)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSeparacionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	respuesta := separacionAResponse(*separacion)

	return &respuesta, nil
}

func (s *separacionStemService) FallarInterrumpidas() error {

	cantidad, err := s.separacionRepository.FallarInterrumpidas(mensajeSeparacionInterrumpida)

	if err != nil {
		return err
	}

	if cantidad > 0 {
		log.Printf("%d separaciones de stems interrumpidas quedaron en ERROR", cantidad)
	}

	return nil
}

// procesar corre fuera de la request: llama al microservicio de IA (que
// sube los stems a MinIO) y registra el resultado.
func (s *separacionStemService) procesar(
	separacion model.SeparacionStem,
	codigoProyecto int64,
	codigoCancion int64,
	fuente string,
) {

	codSeparacion := separacion.CodSeparacionStem

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic en la separación de stems %d: %v", codSeparacion, r)
			s.fallar(codSeparacion, mensajeSeparacionFallida)
		}
	}()

	s.turnos <- struct{}{}
	defer func() { <-s.turnos }()

	if err := s.separacionRepository.MarcarProcesando(codSeparacion); err != nil {
		log.Printf("No se pudo marcar la separación %d como PROCESANDO: %v", codSeparacion, err)
		s.fallar(codSeparacion, mensajeSeparacionFallida)
		return
	}

	destinos := map[string]string{}

	for _, nombre := range mlservice.NombresStemPorCantidad[separacion.CantidadStems] {

		clave, err := claveObjetoStem(codigoProyecto, codigoCancion, separacion.CodigoCancionVersion, "wav")

		if err != nil {
			log.Printf("No se pudo generar la key del stem %s (separación %d): %v", nombre, codSeparacion, err)
			s.fallar(codSeparacion, mensajeSeparacionFallida)
			return
		}

		destinos[nombre] = clave
	}

	resultado, err := s.mlCliente.SepararStems(context.Background(), mlservice.SeparacionRequest{
		SourceObjectKey:       fuente,
		StemCount:             separacion.CantidadStems,
		DestinationObjectKeys: destinos,
	})

	if err != nil {
		log.Printf("Falló la separación de stems %d: %v", codSeparacion, err)
		s.fallar(codSeparacion, mensajeErrorSeparacion(err))
		return
	}

	stems, err := s.armarStemsGenerados(separacion.CodigoCancionVersion, resultado.Stems)

	if err == nil {
		err = s.separacionRepository.Completar(
			codSeparacion,
			separacion.CodigoCancionVersion,
			stems,
			resultado.ProcessingTimeMs,
		)
	}

	if err != nil {
		log.Printf("No se pudieron guardar los stems de la separación %d: %v", codSeparacion, err)

		// El microservicio ya los subió: sin fila que los referencie quedarían
		// huérfanos en MinIO.
		for _, clave := range destinos {
			if errEliminar := s.audioStorage.Eliminar(context.Background(), clave); errEliminar != nil {
				log.Println("No se pudo eliminar el stem generado en MinIO:", clave, errEliminar)
			}
		}

		mensaje := mensajeSeparacionFallida

		if errors.Is(err, errSinCategoriasPorDefecto) {
			mensaje = mensajeSeparacionSinCategorias
		}

		s.fallar(codSeparacion, mensaje)
	}
}

func (s *separacionStemService) fallar(codSeparacion int64, mensaje string) {
	if err := s.separacionRepository.Fallar(codSeparacion, mensaje); err != nil {
		log.Printf("No se pudo marcar la separación %d como ERROR: %v", codSeparacion, err)
	}
}

func mensajeErrorSeparacion(err error) string {

	var errorML *mlservice.ErrorServicioML

	if errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &errorML) && errorML.Codigo == mlservice.CodigoSeparacionTimeout) {
		return mensajeSeparacionTimeout
	}

	return mensajeSeparacionFallida
}

var errSinCategoriasPorDefecto = errors.New("falta una categoría por defecto para los stems generados")

// armarStemsGenerados le da a cada stem separado su nombre y categoría en
// StemHub. Si el nombre ya lo usa otro stem de la versión (ej. uno "Voz"
// cargado a mano) se agrega "(IA)", porque el nombre es único por versión.
func (s *separacionStemService) armarStemsGenerados(
	codigoVersion int64,
	separados []mlservice.StemSeparado,
) ([]repository.StemGenerado, error) {

	categorias, err := s.separacionRepository.CategoriasPorDefecto()

	if err != nil {
		return nil, err
	}

	existentes, err := s.stemRepository.ListarPorVersion(codigoVersion)

	if err != nil {
		return nil, err
	}

	usados := map[string]bool{}

	for _, stem := range existentes {
		usados[strings.ToLower(strings.TrimSpace(stem.NombreStem))] = true
	}

	stems := make([]repository.StemGenerado, 0, len(separados))

	for _, separado := range separados {

		datos, ok := stemsSpleeter[separado.StemName]

		if !ok {
			datos.nombre = separado.StemName
			datos.categoria = "Otros"
		}

		codCategoria, ok := categorias[datos.categoria]

		if !ok {
			return nil, fmt.Errorf("%w: %s", errSinCategoriasPorDefecto, datos.categoria)
		}

		nombre := nombreStemLibre(datos.nombre, usados)
		usados[strings.ToLower(nombre)] = true

		stems = append(stems, repository.StemGenerado{
			Nombre:           nombre,
			CodCategoriaStem: codCategoria,
			Archivo: repository.ArchivoStemGuardado{
				URL:            separado.ObjectKey,
				Formato:        "wav",
				NombreOriginal: separado.StemName + ".wav",
			},
		})
	}

	return stems, nil
}

func nombreStemLibre(base string, usados map[string]bool) string {

	candidato := base

	for intento := 1; usados[strings.ToLower(candidato)]; intento++ {
		if intento == 1 {
			candidato = base + " (IA)"
		} else {
			candidato = fmt.Sprintf("%s (IA %d)", base, intento)
		}
	}

	return candidato
}
