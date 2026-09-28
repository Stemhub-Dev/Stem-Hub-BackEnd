package mlservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type httpCliente struct {
	baseURL           string
	http              *http.Client
	timeoutSeparacion time.Duration
	timeoutResumen    time.Duration
}

// NewHTTPCliente recibe la URL base del microservicio (ej.
// http://ml-service:8000). Cada llamada lleva su propio timeout: el de
// separación tiene que superar el SEPARATION_TIMEOUT_SECONDS del
// microservicio (300 s por defecto) para que sea él quien corte primero y
// devuelva un error con código.
func NewHTTPCliente(
	baseURL string,
	timeoutSeparacion time.Duration,
	timeoutResumen time.Duration,
) (Cliente, error) {

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	if baseURL == "" {
		return nil, errors.New("ML_SERVICE_URL es obligatoria")
	}

	return &httpCliente{
		baseURL:           baseURL,
		http:              &http.Client{},
		timeoutSeparacion: timeoutSeparacion,
		timeoutResumen:    timeoutResumen,
	}, nil
}

func (c *httpCliente) SepararStems(
	ctx context.Context,
	request SeparacionRequest,
) (*SeparacionResponse, error) {

	var respuesta SeparacionResponse

	if err := c.post(ctx, "/v1/separation/tracks", c.timeoutSeparacion, request, &respuesta); err != nil {
		return nil, err
	}

	return &respuesta, nil
}

func (c *httpCliente) ResumirComentarios(
	ctx context.Context,
	request ResumenRequest,
) (*ResumenResponse, error) {

	var respuesta ResumenResponse

	if err := c.post(ctx, "/v1/comments/insights", c.timeoutResumen, request, &respuesta); err != nil {
		return nil, err
	}

	return &respuesta, nil
}

func (c *httpCliente) post(
	ctx context.Context,
	ruta string,
	timeout time.Duration,
	cuerpo any,
	destino any,
) error {

	ctx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	payload, err := json.Marshal(cuerpo)

	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ruta, bytes.NewReader(payload))

	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	respuesta, err := c.http.Do(request)

	if err != nil {
		return fmt.Errorf("no se pudo llamar al servicio de IA (%s): %w", ruta, err)
	}

	defer respuesta.Body.Close()

	contenido, err := io.ReadAll(respuesta.Body)

	if err != nil {
		return fmt.Errorf("no se pudo leer la respuesta del servicio de IA (%s): %w", ruta, err)
	}

	if respuesta.StatusCode < 200 || respuesta.StatusCode > 299 {
		return errorDesdeRespuesta(respuesta.StatusCode, contenido)
	}

	if err := json.Unmarshal(contenido, destino); err != nil {
		return fmt.Errorf("respuesta inválida del servicio de IA (%s): %w", ruta, err)
	}

	return nil
}

func errorDesdeRespuesta(status int, contenido []byte) error {

	var cuerpo struct {
		Error   string          `json:"error"`
		Message string          `json:"message"`
		Detail  json.RawMessage `json:"detail"`
	}

	errorML := &ErrorServicioML{Status: status}

	if json.Unmarshal(contenido, &cuerpo) == nil && cuerpo.Error != "" {
		errorML.Codigo = cuerpo.Error
		errorML.Mensaje = cuerpo.Message

		if detalle := strings.TrimSpace(string(cuerpo.Detail)); detalle != "" && detalle != "{}" && detalle != "null" {
			errorML.Detalle = recortar(detalle)
		}

		return errorML
	}

	errorML.Mensaje = recortar(strings.TrimSpace(string(contenido)))

	return errorML
}

// Para que un cuerpo enorme (ej. la respuesta cruda del LLM) no inunde el log.
func recortar(texto string) string {
	if len(texto) > 1000 {
		return texto[:1000] + "..."
	}
	return texto
}
