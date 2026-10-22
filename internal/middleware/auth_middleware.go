package middleware

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

const (
	UsuarioContextKey = "usuario"
	ClaimsContextKey  = "supabaseClaims"

	// Debe coincidir con el literal que usa el frontend
	// (stemhub-frontend/src/lib/http/interceptors.ts).
	CodigoUsuarioInactivo = "usuario_inactivo"

	// HU-SEG-B02: una sesión sin requests durante este tiempo deja de
	// servir, aunque el access token no haya vencido.
	InactividadMaximaSesion = 45 * time.Minute
)

type SupabaseClaims struct {
	Email        string           `json:"email"`
	Role         string           `json:"role"`
	SessionID    string           `json:"session_id"`
	UserMetadata MetadatosUsuario `json:"user_metadata"`
	jwt.RegisteredClaims
}

// GoTrue marca email_verified al confirmar el email y en los proveedores
// OAuth que lo verifican (Google).
type MetadatosUsuario struct {
	EmailVerified bool `json:"email_verified"`
}

// RegistroSesiones lo implementa repository.SesionRepository.
type RegistroSesiones interface {
	RegistrarActividad(idSesion, idAutenticacion string, ahora time.Time, inactividadMaxima time.Duration) (bool, error)
	Revocar(idSesion, idAutenticacion string, ahora time.Time) error
}

type AuthMiddleware struct {
	usuarioService service.UsuarioService
	sesiones       RegistroSesiones
	keyFunc        jwt.Keyfunc
	issuer         string
	audience       string
}

func NewAuthMiddleware(
	usuarioService service.UsuarioService,
	sesiones RegistroSesiones,
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
		sesiones:       sesiones,
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

	// GoTrue emite siempre session_id y sub como UUID; sin ellos no se
	// puede saber si la sesión se cerró.
	if uuid.Validate(claims.SessionID) != nil || uuid.Validate(claims.Subject) != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "token sin sesión válida",
		})
		return
	}

	vigente, err := m.sesiones.RegistrarActividad(
		claims.SessionID,
		claims.Subject,
		time.Now(),
		InactividadMaximaSesion,
	)

	if err != nil {
		log.Printf("ValidarJWT: no se pudo registrar la actividad de la sesión %s: %v", claims.SessionID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	if !vigente {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "sesión cerrada o vencida por inactividad",
		})
		return
	}

	c.Set(ClaimsContextKey, claims)

	c.Next()
}

// CerrarSesion revoca la sesión del token (va después de ValidarJWT). El
// front lo llama antes de cerrar la sesión en GoTrue: GoTrue revoca el
// refresh token, pero el access token seguiría sirviendo hasta su "exp".
func (m *AuthMiddleware) CerrarSesion(c *gin.Context) {
	claimsValue, _ := c.Get(ClaimsContextKey)
	claims, ok := claimsValue.(*SupabaseClaims)

	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "identidad de autenticación no disponible",
		})
		return
	}

	if err := m.sesiones.Revocar(claims.SessionID, claims.Subject, time.Now()); err != nil {
		log.Printf("CerrarSesion: no se pudo revocar la sesión %s: %v", claims.SessionID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	c.Status(http.StatusNoContent)
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
		log.Printf("UsuarioActivo: claims de tipo inesperado %T", claimsValue)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	usuario, err := m.usuarioService.
		ObtenerUsuarioActivoPorIDAutenticacion(claims.Subject)

	if err != nil {
		if errors.Is(err, service.ErrUsuarioNoEncontrado) {
			// El frontend distingue este caso (usuario autenticado en
			// Supabase pero todavía no registrado en StemHub) por el 404
			// para redirigir a /registro — ver SessionProvider.tsx.
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": "usuario no registrado en StemHub",
			})
			return
		}

		if errors.Is(err, service.ErrUsuarioInactivo) {
			// "codigo" le permite al frontend distinguir la cuenta dada de
			// baja de un 403 de permisos y cerrar la sesión en vez de
			// mostrar "Acceso denegado" dentro de la app.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":  "usuario sin acceso a StemHub",
				"codigo": CodigoUsuarioInactivo,
			})
			return
		}

		// Sin este log el 500 quedaba sin rastro: el error no llega a
		// ningún handler.
		log.Printf("UsuarioActivo: no se pudo obtener el usuario %s: %v", claims.Subject, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	c.Set(UsuarioContextKey, usuario)

	c.Next()
}
