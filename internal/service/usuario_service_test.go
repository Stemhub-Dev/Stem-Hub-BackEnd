package service

import (
	"strings"
	"testing"
)

// La validación del nombre corre antes de tocar el repositorio, por eso
// alcanza con un service sin repositorio.
func TestRegistrarUsuario_NombreLargo(t *testing.T) {
	svc := NewUsuarioService(nil)

	_, _, _, err := svc.RegistrarUsuario(
		"auth-id",
		"ana@test.com",
		strings.Repeat("a", LargoMaximoNombreIntegrante+1),
		nil,
	)

	if err != ErrNombreIntegranteLargo {
		t.Fatalf("esperaba ErrNombreIntegranteLargo, obtuve %v", err)
	}
}
