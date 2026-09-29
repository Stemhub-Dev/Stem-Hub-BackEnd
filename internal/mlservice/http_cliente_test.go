package mlservice

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func nuevoClienteDePrueba(t *testing.T, handler http.HandlerFunc, timeout time.Duration) Cliente {
	t.Helper()

	servidor := httptest.NewServer(handler)
	t.Cleanup(servidor.Close)

	cliente, err := NewHTTPCliente(servidor.URL+"/", timeout, timeout)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	return cliente
}

func TestNewHTTPCliente_SinURL(t *testing.T) {
	if _, err := NewHTTPCliente("  ", time.Second, time.Second); err == nil {
		t.Fatal("se esperaba error sin ML_SERVICE_URL")
	}
}

func TestSepararStems_EnviaContratoYLeeRespuesta(t *testing.T) {
	var recibido SeparacionRequest
	var ruta string

	cliente := nuevoClienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		ruta = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&recibido); err != nil {
			t.Errorf("cuerpo inválido: %v", err)
		}
		w.Write([]byte(`{"status":"completed","stems":[{"stem_name":"vocals","object_key":"k/v.wav","duration_seconds":12.5}],"processing_time_ms":900}`))
	}, time.Second)

	respuesta, err := cliente.SepararStems(context.Background(), SeparacionRequest{
		SourceObjectKey:       "proyectos/1/canciones/2/v1.mp3",
		StemCount:             2,
		DestinationObjectKeys: map[string]string{"vocals": "k/v.wav", "accompaniment": "k/a.wav"},
	})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ruta != "/v1/separation/tracks" {
		t.Fatalf("ruta = %q", ruta)
	}
	if recibido.SourceObjectKey != "proyectos/1/canciones/2/v1.mp3" || recibido.StemCount != 2 || len(recibido.DestinationObjectKeys) != 2 {
		t.Fatalf("request recibido = %+v", recibido)
	}
	if len(respuesta.Stems) != 1 || respuesta.Stems[0].DurationSeconds != 12.5 || respuesta.ProcessingTimeMs != 900 {
		t.Fatalf("respuesta = %+v", respuesta)
	}
}

func TestResumirComentarios_EnviaContratoYLeeRespuesta(t *testing.T) {
	var cuerpo map[string]any

	cliente := nuevoClienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/comments/insights" {
			t.Errorf("ruta = %q", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&cuerpo)
		w.Write([]byte(`{"provider_used":"gemini","summary":"Bien","key_points":["a"],"agreements":[],"disagreements":["b"],"overall_sentiment":"positivo","processing_time_ms":50}`))
	}, time.Second)

	respuesta, err := cliente.ResumirComentarios(context.Background(), ResumenRequest{
		Context:  &ContextoResumen{SongName: "Tema", VersionLabel: "v2"},
		Comments: []ComentarioResumen{{Author: "Ana", Text: "Subir la voz"}},
	})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if respuesta.Summary != "Bien" || respuesta.ProviderUsed != "gemini" || len(respuesta.Disagreements) != 1 {
		t.Fatalf("respuesta = %+v", respuesta)
	}

	comentarios, _ := cuerpo["comments"].([]any)
	if len(comentarios) != 1 {
		t.Fatalf("comments enviados = %v", cuerpo["comments"])
	}
	if _, tieneTimestamp := comentarios[0].(map[string]any)["timestamp"]; tieneTimestamp {
		t.Fatal("timestamp nil no debería enviarse")
	}
	if contexto, _ := cuerpo["context"].(map[string]any); contexto["song_name"] != "Tema" {
		t.Fatalf("context enviado = %v", cuerpo["context"])
	}
}

func TestPost_ErrorConCodigoDelServicio(t *testing.T) {
	cliente := nuevoClienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte(`{"error":"LLM_TIMEOUT","message":"El proveedor tardó demasiado","detail":{"provider":"gemini"}}`))
	}, time.Second)

	_, err := cliente.ResumirComentarios(context.Background(), ResumenRequest{})

	var errorML *ErrorServicioML
	if !errors.As(err, &errorML) {
		t.Fatalf("err = %v, se esperaba ErrorServicioML", err)
	}
	if errorML.Status != http.StatusGatewayTimeout || errorML.Codigo != CodigoLLMTimeout {
		t.Fatalf("errorML = %+v", errorML)
	}
	if errorML.Detalle != `{"provider":"gemini"}` {
		t.Fatalf("detalle = %q", errorML.Detalle)
	}
}

func TestPost_ErrorSinFormatoDelServicio(t *testing.T) {
	cliente := nuevoClienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"detail":[{"msg":"field required"}]}`))
	}, time.Second)

	_, err := cliente.SepararStems(context.Background(), SeparacionRequest{})

	var errorML *ErrorServicioML
	if !errors.As(err, &errorML) || errorML.Status != http.StatusUnprocessableEntity || errorML.Codigo != "" {
		t.Fatalf("err = %v", err)
	}
}

func TestPost_Timeout(t *testing.T) {
	cliente := nuevoClienteDePrueba(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(300 * time.Millisecond):
		}
	}, 50*time.Millisecond)

	_, err := cliente.SepararStems(context.Background(), SeparacionRequest{})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, se esperaba context.DeadlineExceeded", err)
	}
}
