package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// HU-SEG-B01: sin token o con token inválido → 401; usuario dado de baja → 403.

const (
	issuerPrueba   = "https://supabase.test/auth/v1"
	audiencePrueba = "authenticated"
)

type usuarioServiceDePrueba struct {
	usuario *model.Usuario
	err     error
}

func (s usuarioServiceDePrueba) ObtenerUsuarioActivoPorIDAutenticacion(string) (*model.Usuario, error) {
	return s.usuario, s.err
}

func (s usuarioServiceDePrueba) RegistrarUsuario(string, string, bool, string, *string) (*model.Usuario, *model.Integrante, bool, error) {
	return nil, nil, false, nil
}

func nuevaClave(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return clave
}

func firmar(t *testing.T, clave *ecdsa.PrivateKey, claims SupabaseClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(clave)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func claimsValidos() SupabaseClaims {
	return SupabaseClaims{
		Email: "ana@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth-123",
			Issuer:    issuerPrueba,
			Audience:  jwt.ClaimStrings{audiencePrueba},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

func ejecutarProtegido(t *testing.T, clave *ecdsa.PrivateKey, servicio usuarioServiceDePrueba, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	m := &AuthMiddleware{
		usuarioService: servicio,
		keyFunc:        func(*jwt.Token) (any, error) { return &clave.PublicKey, nil },
		issuer:         issuerPrueba,
		audience:       audiencePrueba,
	}

	router := gin.New()
	router.GET("/protegido", m.ValidarJWT, m.UsuarioActivo, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	grabador := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	router.ServeHTTP(grabador, request)
	return grabador
}

var usuarioActivo = usuarioServiceDePrueba{usuario: &model.Usuario{CodigoUsuario: 7}}

func TestValidarJWT_TokenFaltanteOInvalidoResponde401(t *testing.T) {
	clave := nuevaClave(t)
	otraClave := nuevaClave(t)

	vencido := claimsValidos()
	vencido.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))

	otraAudiencia := claimsValidos()
	otraAudiencia.Audience = jwt.ClaimStrings{"otra"}

	casos := map[string]string{
		"sin header":     "",
		"sin Bearer":     firmar(t, clave, claimsValidos()),
		"firma de otro":  "Bearer " + firmar(t, otraClave, claimsValidos()),
		"vencido":        "Bearer " + firmar(t, clave, vencido),
		"otra audiencia": "Bearer " + firmar(t, clave, otraAudiencia),
		"basura":         "Bearer no-es-un-jwt",
	}

	for caso, authorization := range casos {
		t.Run(caso, func(t *testing.T) {
			grabador := ejecutarProtegido(t, clave, usuarioActivo, authorization)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, quería 401", grabador.Code)
			}
		})
	}
}

func TestValidarJWT_TokenValidoPasa(t *testing.T) {
	clave := nuevaClave(t)

	grabador := ejecutarProtegido(t, clave, usuarioActivo, "Bearer "+firmar(t, clave, claimsValidos()))

	if grabador.Code != http.StatusNoContent {
		t.Fatalf("status = %d, quería 204", grabador.Code)
	}
}

func TestUsuarioActivo_DadoDeBajaResponde403(t *testing.T) {
	clave := nuevaClave(t)

	grabador := ejecutarProtegido(
		t,
		clave,
		usuarioServiceDePrueba{err: service.ErrUsuarioInactivo},
		"Bearer "+firmar(t, clave, claimsValidos()),
	)

	if grabador.Code != http.StatusForbidden {
		t.Fatalf("status = %d, quería 403", grabador.Code)
	}

	var cuerpo map[string]string
	if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("cuerpo no es JSON: %v", err)
	}
	if cuerpo["codigo"] != CodigoUsuarioInactivo {
		t.Fatalf("codigo = %q, quería %q", cuerpo["codigo"], CodigoUsuarioInactivo)
	}
}

func TestUsuarioActivo_SinRegistrarResponde404(t *testing.T) {
	clave := nuevaClave(t)

	grabador := ejecutarProtegido(
		t,
		clave,
		usuarioServiceDePrueba{err: service.ErrUsuarioNoEncontrado},
		"Bearer "+firmar(t, clave, claimsValidos()),
	)

	if grabador.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quería 404", grabador.Code)
	}
}
