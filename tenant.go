package common

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxPrefeituraUUID = "prefeituraUUID"
	ctxSubject        = "subject"
)

// TenantMiddleware valida o JWT RS256 Bearer, extrai cpa_prefeitura_id e sub,
// injeta ambos no contexto gin e enriquece o logger estruturado.
// Quando svc-token trocar chaves: só pubKey muda aqui, handlers intocados.
func TenantMiddleware(pubKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token não fornecido.")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == "" {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token inválido.")
			c.Abort()
			return
		}
		claims, err := ParseJWT(tokenStr, pubKey)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token inválido.")
			c.Abort()
			return
		}
		prefeituraUUID := claims.CpaPrefeituraID
		if prefeituraUUID == "" {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token sem cpa_prefeitura_id.")
			c.Abort()
			return
		}
		c.Set(ctxPrefeituraUUID, prefeituraUUID)
		sub := claims.RegisteredClaims.Subject
		c.Set(ctxSubject, sub)
		logger := LoggerFromCtx(c)
		enriched := logger.With().
			Str("prefeitura_uuid", prefeituraUUID).
			Str("sub", sub).
			Logger()
		c.Set(ctxLogger, &enriched)
		c.Next()
	}
}

func PrefeituraUUIDFromCtx(c *gin.Context) string { return c.GetString(ctxPrefeituraUUID) }
func SubFromCtx(c *gin.Context) string             { return c.GetString(ctxSubject) }
