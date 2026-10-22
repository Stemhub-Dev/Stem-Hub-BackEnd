package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// Los largos se cuentan en caracteres: "ñ" ocupa 2 bytes y cuenta 1.
func textoDeLargo(largo int) string {
	return strings.Repeat("ñ", largo)
}

func TestCrearProyecto_NombreAlLimiteSeAcepta(t *testing.T) {
	_, _, servicio := nuevoServicioPortada(nil)
	pedido := pedidoAltaProyecto()
	pedido.Nombre = textoDeLargo(LargoMaximoNombreProyecto)

	if _, err := servicio.CrearProyecto(1, pedido, nil); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestCrearProyecto_NombreLargoNoCreaNada(t *testing.T) {
	repo, _, servicio := nuevoServicioPortada(nil)
	pedido := pedidoAltaProyecto()
	pedido.Nombre = textoDeLargo(LargoMaximoNombreProyecto + 1)

	_, err := servicio.CrearProyecto(1, pedido, nil)

	if !errors.Is(err, ErrNombreProyectoLargo) || repo.crearLlamado {
		t.Fatalf("err = %v, crear llamado = %v", err, repo.crearLlamado)
	}
}

func TestCrearProyecto_DescripcionLargaNoCreaNada(t *testing.T) {
	repo, _, servicio := nuevoServicioPortada(nil)
	pedido := pedidoAltaProyecto()
	descripcion := textoDeLargo(LargoMaximoDescripcionProyecto + 1)
	pedido.Descripcion = &descripcion

	_, err := servicio.CrearProyecto(1, pedido, nil)

	if !errors.Is(err, ErrDescripcionProyectoLarga) || repo.crearLlamado {
		t.Fatalf("err = %v, crear llamado = %v", err, repo.crearLlamado)
	}
}

func TestEditarProyecto_NombreYDescripcionLargos(t *testing.T) {
	nombre := textoDeLargo(LargoMaximoNombreProyecto + 1)
	descripcion := textoDeLargo(LargoMaximoDescripcionProyecto + 1)

	casos := []struct {
		nombre   string
		pedido   dto.EditarProyectoRequest
		esperado error
	}{
		{"nombre", dto.EditarProyectoRequest{Nombre: &nombre}, ErrNombreProyectoLargo},
		{"descripción", dto.EditarProyectoRequest{Descripcion: &descripcion}, ErrDescripcionProyectoLarga},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, _, servicio := nuevoServicioPortada(nil)

			_, err := servicio.EditarProyecto(1, 9, caso.pedido, nil, false)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}

func TestCancion_NombreLargoSeRechazaAntesDeTocarLaBase(t *testing.T) {
	// Sin repositorios: si la validación no cortara antes, el test paniquea.
	servicio := NewCancionService(nil, nil, nil, nil)
	nombre := textoDeLargo(LargoMaximoNombreCancion + 1)

	if _, err := servicio.Crear(1, 7, nombre, wavValido()); !errors.Is(err, ErrCancionNombreLargo) {
		t.Fatalf("crear: err = %v", err)
	}

	_, err := servicio.Editar(1, 7, 8, dto.EditarCancionRequest{Nombre: nombre})
	if !errors.Is(err, ErrCancionNombreLargo) {
		t.Fatalf("editar: err = %v", err)
	}
}

func TestCrearVersion_NotasLargasNoAbrenTransaccion(t *testing.T) {
	e := nuevoEscenarioVersion()
	notas := textoDeLargo(LargoMaximoNotasVersion + 1)

	err := e.crear(wavValido(), &notas)

	if !errors.Is(err, ErrVersionNotasLargas) {
		t.Fatalf("err = %v, se esperaba ErrVersionNotasLargas", err)
	}
	if e.canciones.iniciarLlamado {
		t.Fatal("no debería iniciar la creación de la versión")
	}
}

func TestValidarNombreStem_Largo(t *testing.T) {
	if _, err := validarNombreStem(textoDeLargo(largoMaximoNombreStem)); err != nil {
		t.Fatalf("al límite: err = %v", err)
	}
	if _, err := validarNombreStem(textoDeLargo(largoMaximoNombreStem + 1)); !errors.Is(err, ErrStemNombreLargo) {
		t.Fatalf("largo: err = %v", err)
	}
}
