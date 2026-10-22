package middleware

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// HU-SEG-B02: token de una sesión cerrada o inactiva → 401.

type sesionesDePrueba struct {
	vigente     bool
	err         error
	revocadas   []string
	actividad   []string
	inactividad time.Duration
}

func (s *sesionesDePrueba) RegistrarActividad(idSesion, _ string, _ time.Time, inactividadMaxima time.Duration) (bool, error) {
	s.actividad = append(s.actividad, idSesion)
	s.inactividad = inactividadMaxima
	return s.vigente, s.err
}

func (s *sesionesDePrueba) Revocar(idSesion, _ string, _ time.Time) error {
	s.revocadas = append(s.revocadas, idSesion)
	s.vigente = false
	return s.err
}

func TestValidarJWT_SesionCerradaOInactivaResponde401(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{vigente: false}

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodGet, "/protegido",
		"Bearer "+firmar(t, clave, claimsValidos()))

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quería 401", grabador.Code)
	}
}

func TestValidarJWT_RegistraLaActividadConInactividadDe45Minutos(t *testing.T) {
	clave := nuevaClave(t)
	sesiones := &sesionesDePrueba{vigente: true}
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
	sesiones := &sesionesDePrueba{vigente: true}
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
	sesiones := &sesionesDePrueba{vigente: true}

	grabador := ejecutarConSesiones(t, clave, usuarioActivo, sesiones, http.MethodPost, "/auth/logout", "")

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quería 401", grabador.Code)
	}
	if len(sesiones.revocadas) != 0 {
		t.Fatalf("revocó sin token: %v", sesiones.revocadas)
	}
}
