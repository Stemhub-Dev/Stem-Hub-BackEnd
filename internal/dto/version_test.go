package dto

import "testing"

func TestEtiquetaVersion(t *testing.T) {
	casos := map[int64]string{1: "v1", 2: "v2", 12: "v12"}

	for numero, quiero := range casos {
		if got := EtiquetaVersion(numero); got != quiero {
			t.Errorf("EtiquetaVersion(%d) = %q, quería %q", numero, got, quiero)
		}
	}
}
