package middleware

import (
	"net/http"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type PermisoMiddleware struct {
	permisoService service.PermisoService
}

func NewPermisoMiddleware(
	permisoService service.PermisoService,
) *PermisoMiddleware {
	return &PermisoMiddleware{
		permisoService: permisoService,
	}
}

func (m *PermisoMiddleware) RequerirPermiso(
	codigoPermiso string,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		usuarioContexto, existe := c.Get(UsuarioContextKey)

		if !existe {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "Usuario no autenticado"},
			)
			return
		}

		usuario, ok := usuarioContexto.(*model.Usuario)

		if !ok || usuario == nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "Usuario no autenticado"},
			)
			return
		}

		tienePermiso, err := m.permisoService.TienePermiso(
			usuario.CodigoUsuario,
			codigoPermiso,
		)

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": "Error al verificar permisos"},
			)
			return
		}

		if !tienePermiso {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{"error": "No tiene permisos para realizar esta acción"},
			)
			return
		}

		c.Next()
	}
}
