package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mlservice"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type comentariosResumenFalso struct {
	comentarioRepositoryFalso

	comentarios []dto.ComentarioListadoResponse
}

func (r *comentariosResumenFalso) ListarPorVersion(int64, int64) ([]dto.ComentarioListadoResponse, error) {
	return r.comentarios, nil
}

func (r *comentariosResumenFalso) ObtenerContextoVersion(int64) (string, int, error) {
	return "Tema", 2, nil
}

type mlResumenFalso struct {
	mlClienteFalso

	err       error
	recibido  *mlservice.ResumenRequest
	respuesta *mlservice.ResumenResponse
}

func (m *mlResumenFalso) ResumirComentarios(_ context.Context, request mlservice.ResumenRequest) (*mlservice.ResumenResponse, error) {
	m.recibido = &request
	if m.err != nil {
		return nil, m.err
	}
	return m.respuesta, nil
}

func nuevoServicioResumen(repo *comentariosResumenFalso, ml *mlResumenFalso) ComentarioService {
	return NewComentarioService(
		repo,
		&proyectoAccesoFalso{},
		&cancionExistenciaFalsa{},
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		ml,
	)
}

func segundos(v float64) *float64 { return &v }

func TestResumirPorVersion_ArmaElPedidoYTraduceLaRespuesta(t *testing.T) {
	alta := time.Date(2026, 9, 1, 12, 0, 0, 0, time.FixedZone("ART", -3*3600))
	repo := &comentariosResumenFalso{comentarios: []dto.ComentarioListadoResponse{
		// El listado viene del más nuevo al más viejo.
		{Texto: "Me gusta el final", Autor: dto.AutorComentarioResponse{Nombre: "Beto"}, FechaHoraAlta: alta.Add(time.Hour)},
		{
			Texto:                "La voz está baja",
			TiempoInicioSegundos: segundos(65),
			TiempoFinSegundos:    segundos(72.5),
			Autor:                dto.AutorComentarioResponse{Nombre: "Ana"},
			FechaHoraAlta:        alta,
			Respuestas: []dto.RespuestaComentarioResponse{
				{Texto: "Coincido", Autor: dto.AutorComentarioResponse{Nombre: "Beto"}, FechaHoraAlta: alta.Add(time.Minute)},
			},
		},
	}}
	ml := &mlResumenFalso{respuesta: &mlservice.ResumenResponse{
		ProviderUsed:     "gemini",
		Summary:          "Subir la voz",
		KeyPoints:        []string{"voz baja"},
		OverallSentiment: "positivo",
		ProcessingTimeMs: 800,
	}}

	resumen, err := nuevoServicioResumen(repo, ml).ResumirPorVersion(1, 7, 8, 9)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	enviados := ml.recibido.Comments
	if len(enviados) != 3 {
		t.Fatalf("comentarios enviados = %+v", enviados)
	}
	if enviados[0].Text != "[1:05–1:12] La voz está baja" || enviados[0].Author != "Ana" {
		t.Fatalf("primero = %+v", enviados[0])
	}
	if enviados[1].Text != "(respuesta a Ana) Coincido" || enviados[1].Author != "Beto" {
		t.Fatalf("respuesta = %+v", enviados[1])
	}
	if enviados[2].Text != "Me gusta el final" {
		t.Fatalf("último = %+v", enviados[2])
	}
	if enviados[0].Timestamp == nil || enviados[0].Timestamp.Location() != time.UTC {
		t.Fatalf("timestamp = %v", enviados[0].Timestamp)
	}
	if ml.recibido.Context.SongName != "Tema" || ml.recibido.Context.VersionLabel != "v2" {
		t.Fatalf("context = %+v", ml.recibido.Context)
	}

	if resumen.Resumen != "Subir la voz" || resumen.Proveedor != "gemini" || resumen.CantidadComentarios != 3 {
		t.Fatalf("resumen = %+v", resumen)
	}
	if resumen.Acuerdos == nil || resumen.Desacuerdos == nil {
		t.Fatal("las listas vacías deberían ser [] y no null")
	}
}

func TestResumirPorVersion_SinComentariosNoLlamaAlServicio(t *testing.T) {
	ml := &mlResumenFalso{}

	_, err := nuevoServicioResumen(&comentariosResumenFalso{}, ml).ResumirPorVersion(1, 7, 8, 9)

	if !errors.Is(err, ErrResumenSinComentarios) {
		t.Fatalf("err = %v", err)
	}
	if ml.recibido != nil {
		t.Fatal("no debería llamar al servicio de IA")
	}
}

func TestResumirPorVersion_ErroresDelServicio(t *testing.T) {
	casos := []struct {
		nombre   string
		err      error
		esperado error
	}{
		{"timeout del LLM", &mlservice.ErrorServicioML{Status: 504, Codigo: mlservice.CodigoLLMTimeout}, ErrResumenTimeout},
		{"timeout del backend", context.DeadlineExceeded, ErrResumenTimeout},
		{"error del proveedor", &mlservice.ErrorServicioML{Status: 502, Codigo: "LLM_PROVIDER_ERROR"}, ErrResumenNoDisponible},
		{"servicio caído", errors.New("connection refused"), ErrResumenNoDisponible},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &comentariosResumenFalso{comentarios: []dto.ComentarioListadoResponse{{Texto: "Hola"}}}

			_, err := nuevoServicioResumen(repo, &mlResumenFalso{err: caso.err}).ResumirPorVersion(1, 7, 8, 9)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}
