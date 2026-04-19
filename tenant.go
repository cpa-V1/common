package common

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ctxPrefeituraUUID = "prefeituraUUID"

// TenantMiddleware extrai o prefeitura_uuid do Bearer token e injeta no contexto.
// Stub: o token É o uuid. Quando auth chegar, só troca o parsing aqui.
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token não fornecido.")
			c.Abort()
			return
		}
		prefeituraUUID := strings.TrimPrefix(auth, "Bearer ")
		if prefeituraUUID == "" {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token inválido.")
			c.Abort()
			return
		}
		c.Set(ctxPrefeituraUUID, prefeituraUUID)
		c.Next()
	}
}

func PrefeituraUUIDFromCtx(c *gin.Context) string {
	return c.GetString(ctxPrefeituraUUID)
}
