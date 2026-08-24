package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

const (
	UsuarioContextKey = "usuario"
	ClaimsContextKey  = "supabaseClaims"
)

type SupabaseClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	usuarioService service.UsuarioService
	keyFunc        jwt.Keyfunc
	issuer         string
	audience       string
}

func NewAuthMiddleware(
	usuarioService service.UsuarioService,
	supabaseURL string,
	audience string,
	jwksBaseURLOverride string,
) (*AuthMiddleware, error) {

	supabaseURL = strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	audience = strings.TrimSpace(audience)
	jwksBaseURLOverride = strings.TrimRight(strings.TrimSpace(jwksBaseURLOverride), "/")

	if supabaseURL == "" {
		return nil, errors.New("SUPABASE_URL es requerido")
	}

	if audience == "" {
		return nil, errors.New("SUPABASE_AUDIENCE es requerido")
	}

	// El claim "iss" del token siempre trae supabaseURL (la URL pública que
	// el cliente usó para autenticarse), pero el backend puede necesitar
	// descargar el JWKS por otra ruta de red (p. ej. un nombre de servicio
	// Docker en vez de "localhost"). jwksBaseURLOverride cubre ese caso; si
	// no se pasa, JWKS e issuer se derivan de la misma URL como siempre.
	issuer := supabaseURL + "/auth/v1"

	jwksBaseURL := supabaseURL
	if jwksBaseURLOverride != "" {
		jwksBaseURL = jwksBaseURLOverride
	}

	jwksURL := jwksBaseURL + "/auth/v1/.well-known/jwks.json"

	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar JWKS de Supabase: %w", err)
	}

	return &AuthMiddleware{
		usuarioService: usuarioService,
		keyFunc:        jwks.Keyfunc,
		issuer:         issuer,
		audience:       audience,
	}, nil
}

func (m *AuthMiddleware) ValidarJWT(c *gin.Context) {
	authorization := c.GetHeader("Authorization")

	if authorization == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "token de autenticación requerido",
		})
		return
	}

	partes := strings.Fields(authorization)

	if len(partes) != 2 ||
		!strings.EqualFold(partes[0], "Bearer") ||
		partes[1] == "" {

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "formato de token inválido",
		})
		return
	}

	claims := &SupabaseClaims{}

	token, err := jwt.ParseWithClaims(
		partes[1],
		claims,
		m.keyFunc,
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "token inválido o expirado",
		})
		return
	}

	if claims.Subject == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "token sin identificación de usuario",
		})
		return
	}

	c.Set(ClaimsContextKey, claims)

	c.Next()
}

func (m *AuthMiddleware) UsuarioActivo(c *gin.Context) {
	claimsValue, existe := c.Get(ClaimsContextKey)

	if !existe {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "identidad de autenticación no disponible",
		})
		return
	}

	claims, ok := claimsValue.(*SupabaseClaims)

	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	usuario, err := m.usuarioService.
		ObtenerUsuarioActivoPorIDAutenticacion(claims.Subject)

	if err != nil {
		if errors.Is(err, service.ErrUsuarioNoEncontrado) ||
			errors.Is(err, service.ErrUsuarioInactivo) {

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "usuario sin acceso a StemHub",
			})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	c.Set(UsuarioContextKey, usuario)

	c.Next()
}
