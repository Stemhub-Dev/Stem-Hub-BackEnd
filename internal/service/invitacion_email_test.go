package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// HU-NOT-B01: "El email es inválido" para cualquier dirección mal formada.
func TestEsEmailValido(t *testing.T) {
	largo := strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 63)

	validos := []string{
		"ana@test.local", "a.b+c@sub.ejemplo.com", "x@ejemplo-uno.com.ar", "a1@b2.io",
		// TLD internacionalizado en Punycode (.рф).
		"ana@example.xn--p1ai",
	}
	invalidos := []string{
		"", "ana", "ana@", "@ejemplo.com", "a@b", "Ana <ana@test.local>",
		"persona@-ejemplo.com", "persona@ejemplo-.com", "persona@ejemplo_.com",
		"persona@ejemplo..com", "persona@ejemplo.c0m", "persona@.ejemplo.com", largo,
		// Punycode mal formado o que no decodifica a letras.
		"ana@example.xn--", "ana@example.xn--zz-", "ana@example.xn--123",
	}

	for _, email := range validos {
		if !esEmailValido(email) {
			t.Errorf("%q debería ser válido", email)
		}
	}
	for _, email := range invalidos {
		if esEmailValido(email) {
			t.Errorf("%q debería ser inválido", email)
		}
	}
}

type invitacionesQueCambian struct {
	repository.InvitacionProyectoRepository
	lecturas []*model.InvitacionProyecto
	leidas   int
}

func (r *invitacionesQueCambian) BuscarPorToken(string) (*model.InvitacionProyecto, error) {
	i := r.leidas
	if i >= len(r.lecturas) {
		i = len(r.lecturas) - 1
	}
	r.leidas++
	copia := *r.lecturas[i]
	return &copia, nil
}

func (r *invitacionesQueCambian) Aceptar(int64, int64, int64, int64, string) error {
	return repository.ErrInvitacionNoDisponible
}

type proyectoParaAceptar struct {
	repository.ProyectoRepository
}

func (proyectoParaAceptar) ExisteProyectoActivo(int64) (bool, error) { return true, nil }

func (proyectoParaAceptar) EsIntegranteActivo(int64, int64) (bool, error) { return false, nil }

// Si la invitación deja de estar pendiente entre la validación y el reclamo,
// se responde el motivo real (cancelada, vencida) y no "ya fue aceptada".
func TestAceptar_InvitacionQueCambiaEnElMedio(t *testing.T) {
	ahora := time.Now()
	pendiente := &model.InvitacionProyecto{CodigoInvitacionProy: 1, CodigoProyecto: 2, EmailInvitado: "", FechaHoraExpiracion: ahora.Add(time.Hour)}
	cancelada := *pendiente
	cancelada.FechaHoraBajaInvitacionProy = &ahora
	vencida := *pendiente
	vencida.FechaHoraExpiracion = ahora.Add(-time.Minute)
	aceptada := *pendiente
	aceptada.FechaHoraAceptacion = &ahora

	casos := []struct {
		nombre  string
		despues *model.InvitacionProyecto
		quiero  error
	}{
		{"cancelada", &cancelada, ErrInvitacionCancelada},
		{"vencida", &vencida, ErrInvitacionVencida},
		{"aceptada", &aceptada, ErrInvitacionYaAceptada},
		{"sigue pendiente", pendiente, ErrInvitacionYaAceptada},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			svc := &invitacionProyectoService{
				invitacionRepository: &invitacionesQueCambian{lecturas: []*model.InvitacionProyecto{pendiente, caso.despues}},
				proyectoRepository:   proyectoParaAceptar{},
				integranteRepository: &integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 9}},
			}

			_, err := svc.Aceptar(&model.Usuario{CodigoUsuario: 5, Email: "x@test.local"}, "tok")

			if !errors.Is(err, caso.quiero) {
				t.Fatalf("err = %v, quería %v", err, caso.quiero)
			}
		})
	}
}
