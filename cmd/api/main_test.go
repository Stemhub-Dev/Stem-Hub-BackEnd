package main

import (
	"testing"
	"time"
)

func TestResolverZonaHoraria(t *testing.T) {
	casos := []struct {
		nombre   string
		esperado string
	}{
		{"", zonaHorariaPorDefecto},
		{"UTC", "UTC"},
		{"Europe/Madrid", "Europe/Madrid"},
	}

	for _, caso := range casos {
		zona, err := resolverZonaHoraria(caso.nombre)
		if err != nil {
			t.Fatalf("resolverZonaHoraria(%q): error inesperado: %v", caso.nombre, err)
		}
		if zona.String() != caso.esperado {
			t.Errorf("resolverZonaHoraria(%q) = %q, se esperaba %q", caso.nombre, zona, caso.esperado)
		}
	}
}

func TestResolverZonaHoraria_Invalida(t *testing.T) {
	if _, err := resolverZonaHoraria("Marte/Olympus"); err == nil {
		t.Fatal("se esperaba un error para una zona inexistente")
	}
}

func TestConfigurarZonaHoraria(t *testing.T) {
	anterior := time.Local
	t.Cleanup(func() { time.Local = anterior })
	t.Setenv("TZ", "UTC")

	configurarZonaHoraria()

	if time.Local.String() != "UTC" {
		t.Errorf("time.Local = %q, se esperaba UTC", time.Local)
	}
}
