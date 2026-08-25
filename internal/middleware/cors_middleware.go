package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cors permite que el frontend (origen distinto: localhost:5173 en dev)
// pueda llamar a esta API, incluyendo el preflight OPTIONS que el navegador
// manda antes de cualquier request con header Authorization. Sin este
// middleware, Gin responde 404 a OPTIONS y el navegador nunca llega a
// mandar el request real (ej. POST /usuarios/registrar).
func Cors(allowedOrigins []string) gin.HandlerFunc {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if _, permitido := origins[origin]; permitido {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
