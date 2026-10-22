package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// HU-SEG-B02: token de una sesión cerrada o inactiva → 401.

type sesionesDePrueba struct {
	estado      repository.EstadoSesion
	err         error
	revocadas   []string
	globales    []string
	actividad   []string
	inactividad time.Duration
}

func (s *sesionesDePrueba) RegistrarActividad(idSesion, _ string, _ time.Time, inactividadMaxima time.Duration) (repository.EstadoSesion, error) {
	s.actividad = append(s.actividad, idSesion)
	s.inactividad = inactividadMaxima
	return s.estado, s.err
}

func (s *sesionesDePrueba) Revocar(idSesion, _ string, _ time.Time) error {
	s.revocadas = append(s.revocadas, idSesion)
	s.estado = repository.SesionCerrada
	return s.err
}

func (s *sesionesDePrueba) RevocarTodas(idAutenticacion string, _ time.Time) error {
	s.globales = append(s.globales, idAutenticacion)
	s.estado = repository.SesionCerrada
	return s.err
}

func codigoDeError(t *testing.T, cuerpo []byte) string {
	t.Helper()
	var respuesta map[string]string
	if err := json.Unmarshal(cuerpo, &respuesta); err != nil {
		t.Fatalf("cuerpo no es JSON: %v", err)
	}
	return respuesta["error"]
}

func TestValidarJWT_SesionCerradaOInactivaResponde401ConSuCodigo(t *testing.T) {
	casos := map[repository.EstadoSesion]string{
		repository.SesionCerrada:  CodigoSesionCerrada,
		repository.SesionInactiva: CodigoSesionInactiva,
	}

	for estado, codigo := range casos {
		t.Run(codigo, func(t *testing.T) {
			clave := nuevaClave(t)
			sesiones := &sesionesDePrueba{estado: estado}

			grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodGet, "/protegido",
				"Bearer "+firmar(t, clave, claimsValidos()))

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, quería 401", grabador.Code)
			}
			if got := codigoDeError(t, grabador.Body.Bytes()); got != codigo {
				t.Fatalf("error = %q, quería %q", got, codigo)
			}
		})
	}
}

func TestValidarJWT_RegistraLaActividadConInactividadDe45Minutos(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{estado: repository.SesionVigente}
	claims := claimsValidos()

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodGet, "/protegido",
		"Bearer "+firmar(t, clave, claims))

	if grabador.Code != http.StatusNoContent {
		t.Fatalf("status = %d, quería 204", grabador.Code)
	}
	if len(sesiones.actividad) != 1 || sesiones.actividad[0] != claims.SessionID {
		t.Fatalf("actividad = %v, quería [%s]", sesiones.actividad, claims.SessionID)
	}
	if sesiones.inactividad != 45*time.Minute {
		t.Fatalf("inactividad = %v, quería 45m", sesiones.inactividad)
	}
}

func TestValidarJWT_FallaDelRegistroResponde500(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{err: errors.New("sin base")}

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodGet, "/protegido",
		"Bearer "+firmar(t, clave, claimsValidos()))

	if grabador.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, quería 500", grabador.Code)
	}
}

func TestCerrarSesion_RevocaYElMismoTokenDejaDeServir(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{estado: repository.SesionVigente}
	claims := claimsValidos()
	token := "Bearer " + firmar(t, clave, claims)

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodPost, "/auth/logout", token)
	if grabador.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, quería 204", grabador.Code)
	}
	if len(sesiones.revocadas) != 1 || sesiones.revocadas[0] != claims.SessionID {
		t.Fatalf("revocadas = %v, quería [%s]", sesiones.revocadas, claims.SessionID)
	}

	grabador = ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodGet, "/protegido", token)
	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("después del logout: status = %d, quería 401", grabador.Code)
	}
}

func TestCerrarSesion_SinTokenResponde401(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{estado: repository.SesionVigente}

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodPost, "/auth/logout", "")

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quería 401", grabador.Code)
	}
	if len(sesiones.revocadas) != 0 {
		t.Fatalf("revocó sin token: %v", sesiones.revocadas)
	}
}

func TestCerrarSesion_ScopeGlobalRevocaTodasLasDelUsuario(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{estado: repository.SesionVigente}
	claims := claimsValidos()

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodPost, "/auth/logout?scope=global",
		"Bearer "+firmar(t, clave, claims))

	if grabador.Code != http.StatusNoContent {
		t.Fatalf("status = %d, quería 204", grabador.Code)
	}
	if len(sesiones.globales) != 1 || sesiones.globales[0] != claims.Subject {
		t.Fatalf("globales = %v, quería [%s]", sesiones.globales, claims.Subject)
	}
	if len(sesiones.revocadas) != 0 {
		t.Fatalf("revocó solo una sesión: %v", sesiones.revocadas)
	}
}
