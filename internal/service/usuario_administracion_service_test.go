package service

import (
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type usuarioAdministracionRepositoryFalso struct {
	repository.UsuarioAdministracionRepository

	llamado bool
	err     error
}

func (r *usuarioAdministracionRepositoryFalso) ActualizarAdministracion(
	codigoUsuario int64,
	esAdministradorSistema bool,
	activo bool,
) (*dto.ActualizarAdministracionUsuarioResponse, error) {
	r.llamado = true
	if r.err != nil {
		return nil, r.err
	}
	return &dto.ActualizarAdministracionUsuarioResponse{
		CodigoUsuario:          codigoUsuario,
		EsAdministradorSistema: esAdministradorSistema,
		Activo:                 activo,
	}, nil
}

func (r *usuarioAdministracionRepositoryFalso) CambiarEstado(
	codigoUsuario int64,
	activo bool,
) (*dto.CambiarEstadoUsuarioResponse, error) {
	r.llamado = true
	if r.err != nil {
		return nil, r.err
	}
	return &dto.CambiarEstadoUsuarioResponse{CodigoUsuario: codigoUsuario, Activo: activo}, nil
}

func ptr(valor bool) *bool { return &valor }

const (
	codigoAdmin int64 = 10
	codigoOtro  int64 = 20
)

func TestCambiarEstado_AdminNoPuedeDarseDeBaja(t *testing.T) {
	repo := &usuarioAdministracionRepositoryFalso{}
	s := NewUsuarioAdministracionService(repo)

	_, err := s.CambiarEstado(codigoAdmin, codigoAdmin, dto.CambiarEstadoUsuarioRequest{Activo: ptr(false)})

	if !errors.Is(err, ErrAutoBajaNoPermitida) {
		t.Fatalf("err = %v, quería ErrAutoBajaNoPermitida", err)
	}
	if repo.llamado {
		t.Fatal("no debería llegar al repositorio")
	}
}

func TestCambiarEstado_AdminPuedeDarDeBajaAOtro(t *testing.T) {
	repo := &usuarioAdministracionRepositoryFalso{}
	s := NewUsuarioAdministracionService(repo)

	if _, err := s.CambiarEstado(codigoAdmin, codigoOtro, dto.CambiarEstadoUsuarioRequest{Activo: ptr(false)}); err != nil {
		t.Fatalf("err = %v, quería nil", err)
	}
}

func TestCambiarEstado_SinAdministradoresRestantes(t *testing.T) {
	repo := &usuarioAdministracionRepositoryFalso{err: repository.ErrSinAdministradoresActivos}
	s := NewUsuarioAdministracionService(repo)

	_, err := s.CambiarEstado(codigoAdmin, codigoOtro, dto.CambiarEstadoUsuarioRequest{Activo: ptr(false)})

	if !errors.Is(err, ErrUltimoAdministradorSistema) {
		t.Fatalf("err = %v, quería ErrUltimoAdministradorSistema", err)
	}
}

func TestActualizarAdministracion_ProtegeAlPropioAdmin(t *testing.T) {
	casos := []struct {
		nombre string
		admin  bool
		activo bool
		quiero error
	}{
		{"quitarse el rol", false, true, ErrAutoQuitarAdministradorNoPermitido},
		{"darse de baja", true, false, ErrAutoBajaNoPermitida},
		{"ambas", false, false, ErrAutoBajaNoPermitida},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &usuarioAdministracionRepositoryFalso{}
			s := NewUsuarioAdministracionService(repo)

			_, err := s.ActualizarAdministracion(codigoAdmin, codigoAdmin, dto.ActualizarAdministracionUsuarioRequest{
				EsAdministradorSistema: ptr(caso.admin),
				Activo:                 ptr(caso.activo),
			})

			if !errors.Is(err, caso.quiero) {
				t.Fatalf("err = %v, quería %v", err, caso.quiero)
			}
			if repo.llamado {
				t.Fatal("no debería llegar al repositorio")
			}
		})
	}
}

func TestActualizarAdministracion_QuitarRolAOtro(t *testing.T) {
	repo := &usuarioAdministracionRepositoryFalso{}
	s := NewUsuarioAdministracionService(repo)

	_, err := s.ActualizarAdministracion(codigoAdmin, codigoOtro, dto.ActualizarAdministracionUsuarioRequest{
		EsAdministradorSistema: ptr(false),
		Activo:                 ptr(true),
	})
	if err != nil {
		t.Fatalf("err = %v, quería nil", err)
	}

	repo.err = repository.ErrSinAdministradoresActivos
	_, err = s.ActualizarAdministracion(codigoAdmin, codigoOtro, dto.ActualizarAdministracionUsuarioRequest{
		EsAdministradorSistema: ptr(false),
		Activo:                 ptr(true),
	})
	if !errors.Is(err, ErrUltimoAdministradorSistema) {
		t.Fatalf("err = %v, quería ErrUltimoAdministradorSistema", err)
	}
}
