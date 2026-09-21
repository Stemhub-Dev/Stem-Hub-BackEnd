package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const (
	codigoEstadoProyectoInicial int64 = 1
	ambitoRolProyecto                 = "PROYECTO"
)

const (
	TamanoMaximoLogoProyecto int64 = 5 * 1024 * 1024 // 5 MB
	VigenciaURLLogoProyecto        = 1 * time.Hour
)

var (
	ErrPerfilRequerido = errors.New(
		"el usuario debe completar su perfil",
	)

	ErrTipoProyectoNoValido = errors.New(
		"el tipo de proyecto no es válido",
	)

	ErrRolProyectoNoValido = errors.New(
		"el rol no corresponde al ámbito proyecto",
	)

	ErrGenerosNoValidos = errors.New(
		"uno o más géneros no son válidos",
	)

	ErrNombreProyectoObligatorio = errors.New(
		"el nombre del proyecto es obligatorio",
	)

	ErrProyectoNoEncontrado = errors.New(
		"el proyecto no existe",
	)

	ErrProyectoSinAcceso = errors.New(
		"el usuario no pertenece al proyecto",
	)

	ErrEstadoProyectoNoValido = errors.New(
		"el estado del proyecto no es válido",
	)

	ErrProyectoSoloPropietario = errors.New(
		"solo el propietario puede modificar el proyecto",
	)

	ErrProyectoLogoFormatoInvalido = errors.New(
		"el formato del logo no está soportado",
	)

	ErrProyectoLogoDemasiadoGrande = errors.New(
		"el logo supera el tamaño máximo permitido",
	)

	ErrProyectoErrorAlmacenamiento = errors.New(
		"error al almacenar el logo del proyecto",
	)
)

type ProyectoService interface {
	CrearProyecto(
		codigoUsuario int64,
		request dto.CrearProyectoRequest,
	) (*dto.CrearProyectoResponse, error)

	ListarProyectos(
		codigoUsuario int64,
	) ([]dto.ProyectoListadoResponse, error)

	ListarColaboradores(
		codigoUsuario int64,
		codigoProyecto int64,
	) ([]dto.ColaboradorProyectoResponse, error)

	ObtenerDetalleProyecto(
		codigoUsuario int64,
		codigoProyecto int64,
	) (*dto.ProyectoDetalleResponse, error)

	EditarProyecto(
		codigoUsuario int64,
		codigoProyecto int64,
		request dto.EditarProyectoRequest,
		logo *ArchivoImagen,
	) (*dto.EditarProyectoResponse, error)

	DarDeBaja(
		codigoUsuario int64,
		codigoProyecto int64,
	) error
}

type proyectoService struct {
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	audioStorage         storage.AudioStorage
}

func NewProyectoService(
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
) ProyectoService {

	return &proyectoService{
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		audioStorage:         audioStorage,
	}
}

func (s *proyectoService) CrearProyecto(
	codigoUsuario int64,
	request dto.CrearProyectoRequest,
) (*dto.CrearProyectoResponse, error) {

	request.Nombre = strings.TrimSpace(request.Nombre)

	if request.Nombre == "" {
		return nil, ErrNombreProyectoObligatorio
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrPerfilRequerido
	}

	existeTipo, err :=
		s.proyectoRepository.ExisteTipoProyectoActivo(
			request.CodigoTipoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeTipo {
		return nil, ErrTipoProyectoNoValido
	}

	ambitoRol, err :=
		s.proyectoRepository.ObtenerAmbitoRolActivo(
			request.CodRol,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRolProyectoNoValido
	}

	if err != nil {
		return nil, err
	}

	if ambitoRol != ambitoRolProyecto {
		return nil, ErrRolProyectoNoValido
	}

	existenGeneros, err :=
		s.proyectoRepository.ExistenGeneros(
			request.CodigosGeneros,
		)

	if err != nil {
		return nil, err
	}

	if !existenGeneros {
		return nil, ErrGenerosNoValidos
	}

	codigoProyecto, err :=
		s.proyectoRepository.Crear(
			integrante.CodIntegrante,
			request.Nombre,
			request.Descripcion,
			codigoEstadoProyectoInicial,
			request.CodigoTipoProyecto,
			request.CodigosGeneros,
			request.CodRol,
			ambitoRol,
		)

	if err != nil {
		return nil, err
	}

	return &dto.CrearProyectoResponse{
		CodigoProyecto:     codigoProyecto,
		Nombre:             request.Nombre,
		Descripcion:        request.Descripcion,
		CodigoTipoProyecto: request.CodigoTipoProyecto,
		CodigosGeneros:     request.CodigosGeneros,
		CodRol:             request.CodRol,
		EsPropietario:      true,
	}, nil
}

func (s *proyectoService) ListarProyectos(
	codigoUsuario int64,
) ([]dto.ProyectoListadoResponse, error) {

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return []dto.ProyectoListadoResponse{}, nil
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return []dto.ProyectoListadoResponse{}, nil
	}

	return s.proyectoRepository.ListarPorIntegrante(
		integrante.CodIntegrante,
	)
}

func (s *proyectoService) ListarColaboradores(
	codigoUsuario int64,
	codigoProyecto int64,
) ([]dto.ColaboradorProyectoResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoSinAcceso
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrProyectoSinAcceso
	}

	colaboradores, err :=
		s.proyectoRepository.ListarColaboradores(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.ColaboradorProyectoResponse,
		0,
		len(colaboradores),
	)

	for _, colaborador := range colaboradores {

		var avatarUrl *string

		if colaborador.AvatarObjectKey != nil {

			url, err := s.audioStorage.ObtenerURLDescarga(
				context.Background(),
				*colaborador.AvatarObjectKey,
				VigenciaURLAvatar,
			)

			if err != nil {
				return nil, err
			}

			avatarUrl = &url
		}

		respuesta = append(
			respuesta,
			dto.ColaboradorProyectoResponse{
				CodigoIntegrante: colaborador.CodIntegrante,
				Nombre:           colaborador.NombreIntegrante,
				AvatarUrl:        avatarUrl,
				CodRol:           colaborador.CodRol,
				NombreRol:        colaborador.NombreRol,
				EsPropietario:    colaborador.EsPropietario,
			},
		)
	}

	return respuesta, nil
}

func (s *proyectoService) resolverLogoProyectoURL(
	logoObjectKey *string,
) (*string, error) {

	if logoObjectKey == nil {
		return nil, nil
	}

	url, err := s.audioStorage.ObtenerURLDescarga(
		context.Background(),
		*logoObjectKey,
		VigenciaURLLogoProyecto,
	)

	if err != nil {
		return nil, err
	}

	return &url, nil
}

func (s *proyectoService) ObtenerDetalleProyecto(
	codigoUsuario int64,
	codigoProyecto int64,
) (*dto.ProyectoDetalleResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoSinAcceso
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrProyectoSinAcceso
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrProyectoSinAcceso
	}

	proyecto,
		nombreTipoProyecto,
		nombreEstadoProyecto,
		err :=
		s.proyectoRepository.ObtenerDetalle(
			codigoProyecto,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	generos, err :=
		s.proyectoRepository.ListarGenerosProyecto(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	esPropietario, err :=
		s.proyectoRepository.EsPropietarioActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	logoUrl, err :=
		s.resolverLogoProyectoURL(
			proyecto.LogoProyecto,
		)

	if err != nil {
		return nil, err
	}

	return &dto.ProyectoDetalleResponse{
		CodigoProyecto: proyecto.CodigoProyecto,
		Nombre:         proyecto.NombreProyecto,
		Descripcion:    proyecto.DescripcionProyecto,

		LogoUrl: logoUrl,

		CodigoTipoProyecto: proyecto.CodTipoProy,
		NombreTipoProyecto: nombreTipoProyecto,

		CodigoEstadoProyecto: proyecto.CodEstadoProy,
		NombreEstadoProyecto: nombreEstadoProyecto,

		Generos: generos,

		EsPropietario: esPropietario,
	}, nil
}

func claveObjetoLogoProyecto(
	codigoProyecto int64,
	formato string,
) string {

	return fmt.Sprintf(
		"proyectos/%d/logo.%s",
		codigoProyecto,
		formato,
	)
}

func (s *proyectoService) EditarProyecto(
	codigoUsuario int64,
	codigoProyecto int64,
	request dto.EditarProyectoRequest,
	logo *ArchivoImagen,
) (*dto.EditarProyectoResponse, error) {

	// 1. Verificar que el proyecto exista y esté activo.
	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrProyectoNoEncontrado
	}

	// 2. Obtener integrante asociado al usuario.
	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoSinAcceso
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrProyectoSinAcceso
	}

	// 3. Solo el propietario activo puede modificar.
	esPropietario, err :=
		s.proyectoRepository.EsPropietarioActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esPropietario {
		return nil, ErrProyectoSoloPropietario
	}

	// 4. Obtener valores actuales.
	proyectoActual,
		_,
		_,
		err :=
		s.proyectoRepository.ObtenerDetalle(
			codigoProyecto,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	generosActuales, err :=
		s.proyectoRepository.ListarGenerosProyecto(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	// ---------------------------------------
	// NOMBRE
	// ---------------------------------------

	nombre := proyectoActual.NombreProyecto

	if request.Nombre != nil {

		nombre = strings.TrimSpace(
			*request.Nombre,
		)

		if nombre == "" {
			return nil, ErrNombreProyectoObligatorio
		}
	}

	// ---------------------------------------
	// DESCRIPCIÓN
	// ---------------------------------------

	descripcion :=
		proyectoActual.DescripcionProyecto

	if request.Descripcion != nil {
		descripcion = request.Descripcion
	}

	// ---------------------------------------
	// TIPO DE PROYECTO
	// ---------------------------------------

	codigoTipoProyecto :=
		proyectoActual.CodTipoProy

	if request.CodigoTipoProyecto != nil {

		existeTipo, err :=
			s.proyectoRepository.
				ExisteTipoProyectoActivo(
					*request.CodigoTipoProyecto,
				)

		if err != nil {
			return nil, err
		}

		if !existeTipo {
			return nil, ErrTipoProyectoNoValido
		}

		codigoTipoProyecto =
			*request.CodigoTipoProyecto
	}

	// ---------------------------------------
	// ESTADO
	// ---------------------------------------

	codigoEstadoProyecto :=
		proyectoActual.CodEstadoProy

	if request.CodigoEstadoProyecto != nil {

		existeEstado, err :=
			s.proyectoRepository.
				ExisteEstadoProyectoActivo(
					*request.CodigoEstadoProyecto,
				)

		if err != nil {
			return nil, err
		}

		if !existeEstado {
			return nil, ErrEstadoProyectoNoValido
		}

		codigoEstadoProyecto =
			*request.CodigoEstadoProyecto
	}

	// ---------------------------------------
	// GÉNEROS
	// ---------------------------------------

	codigosGeneros := make(
		[]int64,
		0,
		len(generosActuales),
	)

	for _, genero := range generosActuales {

		codigosGeneros = append(
			codigosGeneros,
			genero.CodigoGenero,
		)
	}

	// Si no vienen géneros, conservamos los actuales.
	// Si vienen, reemplazamos la selección.
	if len(request.CodigosGeneros) > 0 {

		existenGeneros, err :=
			s.proyectoRepository.ExistenGeneros(
				request.CodigosGeneros,
			)

		if err != nil {
			return nil, err
		}

		if !existenGeneros {
			return nil, ErrGenerosNoValidos
		}

		codigosGeneros =
			request.CodigosGeneros
	}

	// ---------------------------------------
	// LOGO
	// ---------------------------------------

	logoObjectKey :=
		proyectoActual.LogoProyecto

	if logo != nil &&
		logo.Contenido != nil {

		if logo.Tamano >
			TamanoMaximoLogoProyecto {

			return nil,
				ErrProyectoLogoDemasiadoGrande
		}

		formato, err :=
			formatoImagenDesdeNombreArchivo(
				logo.NombreOriginal,
			)

		if err != nil {
			return nil,
				ErrProyectoLogoFormatoInvalido
		}

		objectKey :=
			claveObjetoLogoProyecto(
				codigoProyecto,
				formato,
			)

		err = s.audioStorage.Subir(
			context.Background(),
			objectKey,
			logo.Contenido,
			logo.Tamano,
			formatosImagenPermitidos[formato],
		)

		if err != nil {
			return nil,
				ErrProyectoErrorAlmacenamiento
		}

		logoObjectKey = &objectKey
	}

	// 5. Repository recibe finalmente todos los valores,
	// tanto modificados como conservados.
	err = s.proyectoRepository.Actualizar(
		codigoProyecto,
		nombre,
		descripcion,
		logoObjectKey,
		codigoEstadoProyecto,
		codigoTipoProyecto,
		codigosGeneros,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	logoUrl, err :=
		s.resolverLogoProyectoURL(
			logoObjectKey,
		)

	if err != nil {
		return nil, err
	}

	return &dto.EditarProyectoResponse{
		CodigoProyecto: codigoProyecto,
		Nombre:         nombre,
		Descripcion:    descripcion,

		LogoUrl: logoUrl,

		CodigoTipoProyecto: codigoTipoProyecto,

		CodigoEstadoProyecto: codigoEstadoProyecto,

		CodigosGeneros: codigosGeneros,
	}, nil
}

func (s *proyectoService) DarDeBaja(
	codigoUsuario int64,
	codigoProyecto int64,
) error {

	// 1. El proyecto debe existir y estar activo.
	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return err
	}

	if !existeProyecto {
		return ErrProyectoNoEncontrado
	}

	// 2. Obtener el perfil Stem-Hub del usuario.
	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrProyectoSinAcceso
	}

	if err != nil {
		return err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return ErrProyectoSinAcceso
	}

	// 3. Solamente el propietario activo
	//    puede dar de baja el proyecto.
	esPropietario, err :=
		s.proyectoRepository.EsPropietarioActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return err
	}

	if !esPropietario {
		return ErrProyectoSoloPropietario
	}

	// 4. Baja lógica únicamente del proyecto.
	err = s.proyectoRepository.DarDeBaja(
		codigoProyecto,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrProyectoNoEncontrado
	}

	if err != nil {
		return err
	}

	return nil
}
